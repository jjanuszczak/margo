package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"margo/internal/output/printhtml"
)

const (
	OutputDir  = "dist/pdf"
	OutputFile = "dist/pdf/deck.pdf"

	browserEnvVar = "MARGO_CHROME_PATH"
	timeoutEnvVar = "MARGO_PDF_TIMEOUT_SECONDS"
)

func DetectBrowser() (BrowserInfo, error) {
	return findBrowser()
}

func Write(projectRoot string) error {
	htmlPath := filepath.Join(projectRoot, printhtml.OutputFile)
	if _, err := os.Stat(htmlPath); err != nil {
		return fmt.Errorf("stat print html output %q: %w", htmlPath, err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, OutputDir), 0o755); err != nil {
		return fmt.Errorf("create pdf output directory: %w", err)
	}

	browser, err := findBrowser()
	if err != nil {
		return err
	}

	absHTML, err := filepath.Abs(htmlPath)
	if err != nil {
		return fmt.Errorf("resolve html path: %w", err)
	}
	absPDF, err := filepath.Abs(filepath.Join(projectRoot, OutputFile))
	if err != nil {
		return fmt.Errorf("resolve pdf path: %w", err)
	}
	if err := os.Remove(absPDF); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove previous pdf output %q: %w", absPDF, err)
	}
	userDataDir, err := os.MkdirTemp("", "margo-chrome-profile-*")
	if err != nil {
		return fmt.Errorf("create temporary chrome profile: %w", err)
	}
	defer os.RemoveAll(userDataDir)

	timeout := pdfTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	output, err := runBrowser(ctx, browser, buildPrintArgs(absHTML, absPDF, userDataDir), absPDF)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("pdf export timed out after %s using %s (%s)", timeout, browser.Path, browser.Source)
		}
		message := strings.TrimSpace(output)
		hint := ""
		if message == "" {
			hint = " headless Chrome may be blocked in this environment; verify the browser can run headless locally or set " + browserEnvVar + " to a compatible binary."
		}
		if message != "" {
			return fmt.Errorf("print pdf with %q (%s): %w: %s", browser.Path, browser.Source, err, message)
		}
		return fmt.Errorf("print pdf with %q (%s): %w.%s", browser.Path, browser.Source, err, hint)
	}

	if _, err := os.Stat(absPDF); err != nil {
		return fmt.Errorf("expected pdf output %q was not created: %w", absPDF, err)
	}
	return nil
}

// runBrowser waits for a completed PDF rather than relying on Chrome to exit
// after --print-to-pdf. Some Chrome installations leave the headless process
// alive after writing the artifact, which used to make margo wait for the
// export timeout even though the PDF was ready.
func runBrowser(ctx context.Context, browser BrowserInfo, args []string, pdfPath string) (string, error) {
	cmd := exec.Command(browser.Path, args...)
	startedAt := time.Now()
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		return output.String(), err
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-waitCh:
			if pdfReady(pdfPath, startedAt) {
				return output.String(), nil
			}
			return output.String(), err
		case <-ticker.C:
			if pdfReady(pdfPath, startedAt) {
				stopProcess(cmd, waitCh)
				return output.String(), nil
			}
		case <-ctx.Done():
			stopProcess(cmd, waitCh)
			return output.String(), ctx.Err()
		}
	}
}

func stopProcess(cmd *exec.Cmd, waitCh <-chan error) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	<-waitCh
}

func pdfReady(path string, startedAt time.Time) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || info.Size() < 10 || info.ModTime().Before(startedAt) {
		return false
	}
	header := make([]byte, 5)
	if _, err := io.ReadFull(file, header); err != nil || string(header) != "%PDF-" {
		return false
	}
	if _, err := file.Seek(-minInt64(info.Size(), 32), io.SeekEnd); err != nil {
		return false
	}
	tail, err := io.ReadAll(file)
	return err == nil && bytes.Contains(tail, []byte("%%EOF"))
}

func minInt64(left, right int64) int64 {
	if left < right {
		return left
	}
	return right
}

func buildPrintArgs(absHTML string, absPDF string, userDataDir string) []string {
	return []string{
		"--headless=new",
		"--disable-gpu",
		"--disable-software-rasterizer",
		"--disable-dev-shm-usage",
		"--allow-file-access-from-files",
		"--no-first-run",
		"--no-default-browser-check",
		"--run-all-compositor-stages-before-draw",
		"--virtual-time-budget=10000",
		"--print-to-pdf-no-header",
		"--user-data-dir=" + userDataDir,
		"--print-to-pdf=" + absPDF,
		fileURL(absHTML),
	}
}

type BrowserInfo struct {
	Path   string
	Source string
}

func findBrowser() (BrowserInfo, error) {
	if override := strings.TrimSpace(os.Getenv(browserEnvVar)); override != "" {
		if _, err := os.Stat(override); err != nil {
			return BrowserInfo{}, fmt.Errorf("%s points to an unavailable browser %q: %w", browserEnvVar, override, err)
		}
		return BrowserInfo{
			Path:   override,
			Source: browserEnvVar,
		}, nil
	}

	attempts := make([]string, 0, len(browserCandidates()))
	for _, candidate := range browserCandidates() {
		attempts = append(attempts, candidate.value)
		if candidate.kind == browserCandidateFile {
			if _, err := os.Stat(candidate.value); err == nil {
				return BrowserInfo{
					Path:   candidate.value,
					Source: "app-path",
				}, nil
			}
			continue
		}
		if resolved, err := exec.LookPath(candidate.value); err == nil {
			return BrowserInfo{
				Path:   resolved,
				Source: "PATH:" + candidate.value,
			}, nil
		}
	}

	return BrowserInfo{}, fmt.Errorf("could not find a Chrome/Chromium browser; tried %s; set %s to override", strings.Join(attempts, ", "), browserEnvVar)
}

type candidateKind int

const (
	browserCandidatePath candidateKind = iota
	browserCandidateFile
)

type browserCandidate struct {
	kind  candidateKind
	value string
}

func browserCandidates() []browserCandidate {
	switch runtime.GOOS {
	case "darwin":
		return []browserCandidate{
			{kind: browserCandidateFile, value: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"},
			{kind: browserCandidateFile, value: "/Applications/Chromium.app/Contents/MacOS/Chromium"},
			{kind: browserCandidatePath, value: "google-chrome"},
			{kind: browserCandidatePath, value: "chromium"},
		}
	default:
		return []browserCandidate{
			{kind: browserCandidatePath, value: "google-chrome-stable"},
			{kind: browserCandidatePath, value: "google-chrome"},
			{kind: browserCandidatePath, value: "chromium-browser"},
			{kind: browserCandidatePath, value: "chromium"},
		}
	}
}

func fileURL(path string) string {
	return "file://" + filepath.ToSlash(path)
}

func pdfTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv(timeoutEnvVar))
	if raw == "" {
		return 120 * time.Second
	}
	seconds, err := time.ParseDuration(raw + "s")
	if err != nil || seconds <= 0 {
		return 120 * time.Second
	}
	return seconds
}
