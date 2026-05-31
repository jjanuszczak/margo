package pdf

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBuildPrintArgs(t *testing.T) {
	args := buildPrintArgs("/tmp/deck/index.html", "/tmp/deck/deck.pdf", "/tmp/chrome-profile")
	joined := strings.Join(args, " ")

	for _, needle := range []string{
		"--headless=new",
		"--disable-software-rasterizer",
		"--disable-dev-shm-usage",
		"--run-all-compositor-stages-before-draw",
		"--virtual-time-budget=10000",
		"--print-to-pdf-no-header",
		"--user-data-dir=/tmp/chrome-profile",
		"--print-to-pdf=/tmp/deck/deck.pdf",
		"file:///tmp/deck/index.html",
	} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("expected args to contain %q, got %v", needle, args)
		}
	}
}

func TestFileURLUsesForwardSlashes(t *testing.T) {
	url := fileURL(filepath.Join(string(filepath.Separator), "tmp", "deck", "index.html"))
	if !strings.HasPrefix(url, "file:///") {
		t.Fatalf("expected file URL prefix, got %q", url)
	}
}

func TestBrowserCandidatesIncludeExpectedDefaults(t *testing.T) {
	candidates := browserCandidates()
	if len(candidates) == 0 {
		t.Fatal("expected at least one browser candidate")
	}

	values := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		values = append(values, candidate.value)
	}
	joined := strings.Join(values, ",")

	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(joined, "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome") {
			t.Fatalf("expected darwin candidates to include Google Chrome app path, got %v", values)
		}
	default:
		if !strings.Contains(joined, "google-chrome-stable") {
			t.Fatalf("expected non-darwin candidates to include google-chrome-stable, got %v", values)
		}
	}
}

func TestFindBrowserUsesOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "chrome")
	if err := os.WriteFile(override, []byte(""), 0o755); err != nil {
		t.Fatalf("write override browser: %v", err)
	}

	t.Setenv(browserEnvVar, override)
	info, err := findBrowser()
	if err != nil {
		t.Fatalf("findBrowser returned error: %v", err)
	}
	if info.Path != override {
		t.Fatalf("expected override path %q, got %q", override, info.Path)
	}
	if info.Source != browserEnvVar {
		t.Fatalf("expected override source %q, got %q", browserEnvVar, info.Source)
	}
}

func TestFindBrowserInvalidOverride(t *testing.T) {
	t.Setenv(browserEnvVar, "/tmp/does-not-exist-chrome")
	_, err := findBrowser()
	if err == nil || !strings.Contains(err.Error(), browserEnvVar) {
		t.Fatalf("expected override error mentioning %s, got %v", browserEnvVar, err)
	}
}

func TestCreatePrintHTMLInjectsPrintClassAndStripsScripts(t *testing.T) {
	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "index.html")
	source := `<!doctype html><html lang="en"><head><script>console.log("x")</script></head><body><script>console.log("y")</script><main>ok</main></body></html>`
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source html: %v", err)
	}

	printPath, err := createPrintHTML(sourcePath)
	if err != nil {
		t.Fatalf("createPrintHTML returned error: %v", err)
	}
	defer os.Remove(printPath)

	rendered, err := os.ReadFile(printPath)
	if err != nil {
		t.Fatalf("read print html: %v", err)
	}
	out := string(rendered)
	if !strings.Contains(out, `<html class="margo-print" lang="en">`) {
		t.Fatalf("expected print class injection, got %q", out)
	}
	if strings.Contains(out, "<script>") {
		t.Fatalf("expected scripts to be stripped, got %q", out)
	}
}

func TestPDFTimeoutDefaultsAndOverride(t *testing.T) {
	if got := pdfTimeout(); got != 120*time.Second {
		t.Fatalf("default timeout = %s, want %s", got, 120*time.Second)
	}

	t.Setenv(timeoutEnvVar, "75")
	if got := pdfTimeout(); got != 75*time.Second {
		t.Fatalf("override timeout = %s, want %s", got, 75*time.Second)
	}

	t.Setenv(timeoutEnvVar, "bad")
	if got := pdfTimeout(); got != 120*time.Second {
		t.Fatalf("invalid timeout fallback = %s, want %s", got, 120*time.Second)
	}
}
