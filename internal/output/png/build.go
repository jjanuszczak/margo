// Package png exports one raster image for every rendered print slide.
package png

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jjanuszczak/margo/internal/deck"
	"github.com/jjanuszczak/margo/internal/output/pdf"
	"github.com/jjanuszczak/margo/internal/output/printhtml"
)

const (
	OutputDir = "dist/png"

	timeoutEnvVar = "MARGO_PNG_TIMEOUT_SECONDS"
	cssWidth      = 1280
	cssHeight     = 720
	deviceScale   = "1.5"
)

var invalidFilename = regexp.MustCompile(`[^a-z0-9]+`)

// Write captures every included slide from the print artifact. It deliberately
// uses the print document, which is the theme-owned static export surface, not
// the interactive HTML deck.
func Write(projectRoot string, slides []deck.Slide) error {
	printPath := filepath.Join(projectRoot, printhtml.OutputFile)
	printHTML, err := os.ReadFile(printPath)
	if err != nil {
		return fmt.Errorf("read print html output %q: %w", printPath, err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, OutputDir), 0o755); err != nil {
		return fmt.Errorf("create png output directory: %w", err)
	}
	browser, err := pdf.DetectBrowser()
	if err != nil {
		return err
	}
	if err := removePreviousImages(filepath.Join(projectRoot, OutputDir)); err != nil {
		return err
	}

	timeout := pngTimeout()
	for index, slide := range slides {
		outputPath := filepath.Join(projectRoot, OutputDir, outputFilename(index, slide))
		// Keep the temporary document beside print.html. Theme, deck, and
		// slide assets in that artifact use relative URLs rooted at dist/pdf.
		capturePath := filepath.Join(filepath.Dir(printPath), fmt.Sprintf(".margo-png-capture-%03d.html", index+1))
		captureHTML, err := isolatedSlideHTML(printHTML, index)
		if err != nil {
			return err
		}
		if err := os.WriteFile(capturePath, captureHTML, 0o644); err != nil {
			return fmt.Errorf("write png capture document: %w", err)
		}

		err = capture(browser, capturePath, outputPath, timeout)
		removeErr := os.Remove(capturePath)
		if err != nil {
			return fmt.Errorf("capture slide %d (%s): %w", index+1, slide.ID, err)
		}
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("remove temporary png capture document: %w", removeErr)
		}
		if err := validatePNG(outputPath); err != nil {
			return fmt.Errorf("capture slide %d (%s): %w", index+1, slide.ID, err)
		}
	}
	return nil
}

func isolatedSlideHTML(document []byte, index int) ([]byte, error) {
	headNeedle := []byte("</head>")
	headPosition := bytes.LastIndex(document, headNeedle)
	if headPosition < 0 {
		return nil, errors.New("print html does not contain a closing head tag")
	}
	bodyNeedle := []byte("</body>")
	bodyPosition := bytes.LastIndex(document, bodyNeedle)
	if bodyPosition < 0 {
		return nil, errors.New("print html does not contain a closing body tag")
	}
	style := []byte(`<style>
html, body { width: 100vw !important; height: 100vh !important; margin: 0 !important; overflow: hidden !important; }
main, .print-deck, .deck { width: 100vw !important; height: 100vh !important; min-height: 0 !important; margin: 0 !important; padding: 0 !important; overflow: hidden !important; }
.margo-png-slide[hidden] { display: none !important; }
.margo-png-slide:not([hidden]) { display: block !important; }
.margo-png-slide { width: 100vw !important; height: 100vh !important; min-height: 0 !important; margin: 0 !important; }
</style>`)
	script := []byte(fmt.Sprintf(`<script>
(() => {
  const slides = Array.from(document.querySelectorAll('main .print-slide, main .slide'));
  const target = slides[%d];
  for (const slide of slides) {
    slide.classList.add('margo-png-slide');
    slide.hidden = slide !== target;
  }
})();
</script>`, index))

	withStyle := make([]byte, 0, len(document)+len(style)+len(script))
	withStyle = append(withStyle, document[:headPosition]...)
	withStyle = append(withStyle, style...)
	withStyle = append(withStyle, document[headPosition:]...)
	adjustedBodyPosition := bodyPosition + len(style)
	result := make([]byte, 0, len(withStyle)+len(script))
	result = append(result, withStyle[:adjustedBodyPosition]...)
	result = append(result, script...)
	result = append(result, withStyle[adjustedBodyPosition:]...)
	return result, nil
}

func capture(browser pdf.BrowserInfo, htmlPath, outputPath string, timeout time.Duration) error {
	userDataDir, err := os.MkdirTemp("", "margo-chrome-profile-*")
	if err != nil {
		return fmt.Errorf("create temporary chrome profile: %w", err)
	}
	defer os.RemoveAll(userDataDir)

	args := buildScreenshotArgs(htmlPath, outputPath, userDataDir)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return runBrowser(ctx, browser, args, outputPath)
}

// runBrowser does not rely on Chrome exiting after --screenshot. Some Chrome
// builds retain the headless process after writing the image, just as they do
// after --print-to-pdf. The first complete PNG is the export completion signal.
func runBrowser(ctx context.Context, browser pdf.BrowserInfo, args []string, outputPath string) error {
	cmd := exec.Command(browser.Path, args...)
	startedAt := time.Now()
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		return err
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-waitCh:
			if pngCreated(outputPath, startedAt) {
				return nil
			}
			return browserError(browser, err, output.String())
		case <-ticker.C:
			if pngCreated(outputPath, startedAt) {
				stopProcess(cmd, waitCh)
				return nil
			}
		case <-ctx.Done():
			stopProcess(cmd, waitCh)
			return fmt.Errorf("png export timed out after %s using %s (%s)", time.Since(startedAt).Round(time.Millisecond), browser.Path, browser.Source)
		}
	}
}

func browserError(browser pdf.BrowserInfo, err error, output string) error {
	message := strings.TrimSpace(output)
	if message != "" {
		return fmt.Errorf("render with %q (%s): %w: %s", browser.Path, browser.Source, err, message)
	}
	return fmt.Errorf("render with %q (%s): %w", browser.Path, browser.Source, err)
}

func stopProcess(cmd *exec.Cmd, waitCh <-chan error) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	<-waitCh
}

func pngCreated(path string, startedAt time.Time) bool {
	info, err := os.Stat(path)
	if err != nil || info.Size() < 24 || info.ModTime().Before(startedAt) {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	header := make([]byte, 8)
	if _, err := file.Read(header); err != nil {
		return false
	}
	return bytes.Equal(header, []byte{137, 80, 78, 71, 13, 10, 26, 10})
}

func buildScreenshotArgs(htmlPath, outputPath, userDataDir string) []string {
	return []string{
		"--headless=new",
		"--disable-gpu",
		"--disable-software-rasterizer",
		"--disable-dev-shm-usage",
		"--allow-file-access-from-files",
		"--hide-scrollbars",
		"--no-first-run",
		"--no-default-browser-check",
		"--run-all-compositor-stages-before-draw",
		"--virtual-time-budget=10000",
		"--window-size=1280,720",
		"--force-device-scale-factor=" + deviceScale,
		"--user-data-dir=" + userDataDir,
		"--screenshot=" + outputPath,
		fileURL(htmlPath),
	}
}

func fileURL(path string) string {
	return "file://" + filepath.ToSlash(path)
}

func outputFilename(index int, slide deck.Slide) string {
	name := slug(slide.ID)
	if name == "" {
		name = "slide"
	}
	return fmt.Sprintf("%03d-%s.png", index+1, name)
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = invalidFilename.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func removePreviousImages(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read png output directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".png") {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return fmt.Errorf("remove previous png output %q: %w", entry.Name(), err)
		}
	}
	return nil
}

func validatePNG(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("expected png output %q was not created: %w", path, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() < 24 {
		return errors.New("generated PNG is empty or too small")
	}
	header := make([]byte, 24)
	if _, err := file.Read(header); err != nil {
		return fmt.Errorf("read png header: %w", err)
	}
	if !bytes.Equal(header[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return errors.New("generated file is not a PNG")
	}
	width := int(header[16])<<24 | int(header[17])<<16 | int(header[18])<<8 | int(header[19])
	height := int(header[20])<<24 | int(header[21])<<16 | int(header[22])<<8 | int(header[23])
	if width != 1920 || height != 1080 {
		return fmt.Errorf("generated PNG dimensions are %dx%d, want 1920x1080", width, height)
	}
	return nil
}

func pngTimeout() time.Duration {
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

// SortedOutputFiles returns generated PNG file names in slide order. It is
// intentionally small and test-friendly for callers that need to inspect the
// artifact set.
func SortedOutputFiles(projectRoot string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(projectRoot, OutputDir))
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".png") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}
