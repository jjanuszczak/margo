package pdf

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"margo/internal/output/html"
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
	htmlPath := filepath.Join(projectRoot, html.OutputFile)
	if _, err := os.Stat(htmlPath); err != nil {
		return fmt.Errorf("stat html output %q: %w", htmlPath, err)
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
	printHTML, err := createPrintHTML(absHTML)
	if err != nil {
		return fmt.Errorf("prepare print html: %w", err)
	}
	defer os.Remove(printHTML)
	absPDF, err := filepath.Abs(filepath.Join(projectRoot, OutputFile))
	if err != nil {
		return fmt.Errorf("resolve pdf path: %w", err)
	}
	userDataDir, err := os.MkdirTemp("", "margo-chrome-profile-*")
	if err != nil {
		return fmt.Errorf("create temporary chrome profile: %w", err)
	}
	defer os.RemoveAll(userDataDir)

	timeout := pdfTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, browser.Path, buildPrintArgs(printHTML, absPDF, userDataDir)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("pdf export timed out after %s using %s (%s)", timeout, browser.Path, browser.Source)
		}
		message := strings.TrimSpace(string(output))
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

var scriptTagPattern = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)

func createPrintHTML(sourcePath string) (string, error) {
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("read source html: %w", err)
	}
	printHTML := scriptTagPattern.ReplaceAllString(string(raw), "")
	switch {
	case strings.Contains(printHTML, "<html class=\"margo-print\""):
		// already marked
	case strings.Contains(printHTML, "<html "):
		printHTML = strings.Replace(printHTML, "<html ", "<html class=\"margo-print\" ", 1)
	case strings.Contains(printHTML, "<html>"):
		printHTML = strings.Replace(printHTML, "<html>", "<html class=\"margo-print\">", 1)
	default:
		return "", fmt.Errorf("could not locate <html> tag in rendered output")
	}

	tmp, err := os.CreateTemp(filepath.Dir(sourcePath), "margo-print-*.html")
	if err != nil {
		return "", fmt.Errorf("create temporary print html: %w", err)
	}
	defer tmp.Close()
	if _, err := tmp.WriteString(printHTML); err != nil {
		return "", fmt.Errorf("write temporary print html: %w", err)
	}
	return tmp.Name(), nil
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
