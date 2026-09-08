package png

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jjanuszczak/margo/internal/config"
	"github.com/jjanuszczak/margo/internal/content"
	"github.com/jjanuszczak/margo/internal/deck"
	"github.com/jjanuszczak/margo/internal/output/pdf"
	"github.com/jjanuszczak/margo/internal/output/printhtml"
	"github.com/jjanuszczak/margo/internal/theme"
)

func TestOutputFilename(t *testing.T) {
	if got, want := outputFilename(0, deck.Slide{ID: "01 Title & Why"}), "001-01-title-why.png"; got != want {
		t.Fatalf("outputFilename = %q, want %q", got, want)
	}
	if got, want := outputFilename(1, deck.Slide{}), "002-slide.png"; got != want {
		t.Fatalf("outputFilename empty ID = %q, want %q", got, want)
	}
}

func TestIsolatedSlideHTML(t *testing.T) {
	document := []byte("<html><head><title>Deck</title></head><body><main><section class=\"print-slide\"></section><section class=\"slide\"></section></main></body></html>")
	got, err := isolatedSlideHTML(document, 4)
	if err != nil {
		t.Fatalf("isolatedSlideHTML: %v", err)
	}
	for _, needle := range []string{".margo-png-slide[hidden]", ".margo-png-slide:not([hidden])", "main .print-slide, main .slide", "const target = slides[4]", "</head>", "</body>"} {
		if !strings.Contains(string(got), needle) {
			t.Fatalf("expected capture document to contain %q, got %s", needle, got)
		}
	}
	if _, err := isolatedSlideHTML([]byte("<html>"), 0); err == nil {
		t.Fatal("expected malformed print document to fail")
	}
	if _, err := isolatedSlideHTML([]byte("<html><head></head><body>"), 0); err == nil {
		t.Fatal("expected print document without body end to fail")
	}
}

func TestCaptureDocumentLivesBesidePrintHTML(t *testing.T) {
	root := t.TempDir()
	printPath := filepath.Join(root, printhtml.OutputFile)
	capturePath := filepath.Join(filepath.Dir(printPath), ".margo-png-capture-001.html")
	if filepath.Dir(capturePath) != filepath.Dir(printPath) {
		t.Fatalf("capture document must share print asset root: %q", capturePath)
	}
}

func TestBuildScreenshotArgs(t *testing.T) {
	args := strings.Join(buildScreenshotArgs("/tmp/deck/print.html", "/tmp/deck/slide.png", "/tmp/profile"), " ")
	for _, needle := range []string{
		"--headless=new", "--hide-scrollbars", "--virtual-time-budget=10000", "--window-size=1280,720", "--force-device-scale-factor=1.5", "--screenshot=/tmp/deck/slide.png", "file:///tmp/deck/print.html",
	} {
		if !strings.Contains(args, needle) {
			t.Fatalf("expected args to contain %q, got %s", needle, args)
		}
	}
}

func TestValidatePNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "slide.png")
	header := append([]byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 'I', 'H', 'D', 'R'}, []byte{0, 0, 7, 128, 0, 0, 4, 56, 8, 2, 0, 0, 0}...)
	if err := os.WriteFile(path, header, 0o644); err != nil {
		t.Fatalf("write png: %v", err)
	}
	if err := validatePNG(path); err != nil {
		t.Fatalf("validatePNG: %v", err)
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte{0}, 24), 0o644); err != nil {
		t.Fatalf("overwrite png: %v", err)
	}
	if err := validatePNG(path); err == nil {
		t.Fatal("expected invalid PNG to fail")
	}
}

func TestPNGTimeoutDefaultsAndOverride(t *testing.T) {
	if got := pngTimeout(); got != 120*time.Second {
		t.Fatalf("default timeout = %s, want 120s", got)
	}
	t.Setenv(timeoutEnvVar, "75")
	if got := pngTimeout(); got != 75*time.Second {
		t.Fatalf("override timeout = %s, want 75s", got)
	}
}

func TestSortedOutputFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, OutputDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create output dir: %v", err)
	}
	for _, name := range []string{"002-second.png", "001-first.png", "ignore.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write output: %v", err)
		}
	}
	got, err := SortedOutputFiles(root)
	if err != nil {
		t.Fatalf("SortedOutputFiles: %v", err)
	}
	if strings.Join(got, ",") != "001-first.png,002-second.png" {
		t.Fatalf("unexpected output files: %v", got)
	}
}

func TestWriteCapturesPrintSlidesWithChrome(t *testing.T) {
	if os.Getenv("MARGO_TEST_PNG_EXPORT") != "1" {
		t.Skip("set MARGO_TEST_PNG_EXPORT=1 to run the local Chrome integration check")
	}
	if _, err := pdf.DetectBrowser(); err != nil {
		t.Skipf("Chrome is unavailable: %v", err)
	}

	root := t.TempDir()
	printDir := filepath.Join(root, printhtml.OutputDir)
	if err := os.MkdirAll(printDir, 0o755); err != nil {
		t.Fatalf("create print output: %v", err)
	}
	printHTML := `<!doctype html><html><head><style>
html, body { margin: 0; }
.print-slide { width: 13.333in; height: 7.5in; background: #164e63; color: white; font: 48px sans-serif; }
</style></head><body><main class="print-deck"><section class="print-slide" data-slide-index="0">First</section><section class="print-slide" data-slide-index="1">Second</section></main></body></html>`
	if err := os.WriteFile(filepath.Join(root, printhtml.OutputFile), []byte(printHTML), 0o644); err != nil {
		t.Fatalf("write print HTML: %v", err)
	}
	if err := Write(root, []deck.Slide{{ID: "first"}, {ID: "second"}}); err != nil {
		t.Fatalf("write png slides: %v", err)
	}
	files, err := SortedOutputFiles(root)
	if err != nil {
		t.Fatalf("list png slides: %v", err)
	}
	if strings.Join(files, ",") != "001-first.png,002-second.png" {
		t.Fatalf("unexpected png files: %v", files)
	}
}

func TestWriteCapturesReferenceDeckWithChrome(t *testing.T) {
	if os.Getenv("MARGO_TEST_PNG_FIXTURE") != "1" {
		t.Skip("set MARGO_TEST_PNG_FIXTURE=1 to run the reference-deck Chrome integration check")
	}
	if _, err := pdf.DetectBrowser(); err != nil {
		t.Skipf("Chrome is unavailable: %v", err)
	}
	projectRoot := copyReferenceDeck(t)
	raw, err := config.LoadRaw(filepath.Join(projectRoot, config.DefaultFilename))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	parsed, err := config.Parse(raw)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	slides, err := content.DiscoverSlides(projectRoot)
	if err != nil {
		t.Fatalf("discover slides: %v", err)
	}
	slides = deck.ApplySectionDividers(deck.FilterSlides(slides, deck.FilterOptions{}))
	activeTheme, err := theme.Load(projectRoot, parsed.Config.Theme.Name)
	if err != nil {
		t.Fatalf("load theme: %v", err)
	}
	parsed.Config.Theme.Options, err = theme.ResolveOptions(activeTheme, parsed.Config.Theme.Options)
	if err != nil {
		t.Fatalf("resolve theme options: %v", err)
	}
	model := deck.Model{Config: parsed.Config, Slides: slides, Sections: deck.BuildSections(slides)}
	if _, err := printhtml.Write(projectRoot, model, activeTheme); err != nil {
		t.Fatalf("write print html: %v", err)
	}
	if err := Write(projectRoot, slides); err != nil {
		t.Fatalf("write reference deck png slides: %v", err)
	}
	files, err := SortedOutputFiles(projectRoot)
	if err != nil {
		t.Fatalf("list png slides: %v", err)
	}
	if len(files) != len(slides) {
		t.Fatalf("generated %d PNGs for %d slides: %v", len(files), len(slides), files)
	}
}

func copyReferenceDeck(t *testing.T) string {
	t.Helper()
	source, err := filepath.Abs(filepath.Join("..", "..", "..", "examples", "reference-deck"))
	if err != nil {
		t.Fatalf("resolve reference deck: %v", err)
	}
	destination := filepath.Join(t.TempDir(), "reference-deck")
	err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(destination, 0o755)
		}
		if entry.IsDir() && entry.Name() == "dist" {
			return filepath.SkipDir
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.Create(target)
		if err != nil {
			return err
		}
		defer output.Close()
		_, err = io.Copy(output, input)
		return err
	})
	if err != nil {
		t.Fatalf("copy reference deck: %v", err)
	}
	return destination
}
