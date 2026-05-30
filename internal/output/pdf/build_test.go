package pdf

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildPrintArgs(t *testing.T) {
	args := buildPrintArgs("/tmp/deck/index.html", "/tmp/deck/deck.pdf", "/tmp/chrome-profile")
	joined := strings.Join(args, " ")

	for _, needle := range []string{
		"--headless",
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
