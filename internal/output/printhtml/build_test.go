package printhtml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jjanuszczak/margo/internal/config"
	"github.com/jjanuszczak/margo/internal/content"
	"github.com/jjanuszczak/margo/internal/deck"
	"github.com/jjanuszczak/margo/internal/theme"
)

func TestReferenceDeckPrintBuildFlow(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "reference-deck")

	raw, err := config.LoadRaw(filepath.Join(projectRoot, config.DefaultFilename))
	if err != nil {
		t.Fatalf("load raw config: %v", err)
	}
	parsed, err := config.Parse(raw)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	slides, err := content.DiscoverSlides(projectRoot)
	if err != nil {
		t.Fatalf("discover slides: %v", err)
	}
	filtered := deck.ApplySectionDividers(deck.FilterSlides(slides, deck.FilterOptions{}))
	model := deck.Model{
		Config:   parsed.Config,
		Sections: deck.BuildSections(filtered),
		Slides:   filtered,
	}

	activeTheme, err := theme.Load(projectRoot, parsed.Config.Theme.Name)
	if err != nil {
		t.Fatalf("load theme: %v", err)
	}
	parsed.Config.Theme.Options, err = theme.ResolveOptions(activeTheme, parsed.Config.Theme.Options)
	if err != nil {
		t.Fatalf("resolve theme options: %v", err)
	}
	model.Config.Theme.Options = parsed.Config.Theme.Options

	report, err := Write(projectRoot, model, activeTheme)
	if err != nil {
		t.Fatalf("write print html: %v", err)
	}
	if len(report.Items) != 0 {
		t.Fatalf("expected no warnings, got %#v", report.Items)
	}

	rendered, err := os.ReadFile(filepath.Join(projectRoot, OutputFile))
	if err != nil {
		t.Fatalf("read print html: %v", err)
	}
	out := string(rendered)

	for _, needle := range []string{
		`class="margo-print color-mode-dark"`,
		`class="slide-shell content-slide"`,
		`class="slide-shell title-slide"`,
		`class="slide-shell section-slide"`,
		`class="slide-body media-split-slide media-right"`,
		`class="slide-body media-split-slide media-left"`,
		`Strategy`,
		`Why Margo`,
		`Customer Story`,
		`Architecture View`,
		`color-scheme: dark;`,
		`A reasonably sized deck should still feel fast.`,
		`assets/shared-grid.svg`,
		`slides/02-why/diagram.svg`,
		`slides/05-customer-story/spotlight.svg`,
		`class="shortcode-figure shortcode-figure-fit-contain"`,
		`class="shortcode-chart"`,
		`class="shortcode-chart-canvas"`,
		`themes/default/assets/chart.umd.min.js`,
		`themes/default/assets/katex.min.css`,
		`themes/default/assets/katex.min.js`,
		`themes/default/assets/theme.css`,
		`"type": "doughnut"`,
		`Reference-deck Chart.js smoke test`,
		`class="shortcode-math"`,
		`class="shortcode-math-render"`,
		`\operatorname{Var}(X)`,
		`Reference-deck KaTeX smoke test`,
		`Deck-level asset rendered through the figure shortcode.`,
		`Source: shared deck assets`,
		`window.__margoFixture`,
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("expected print html to contain %q", needle)
		}
	}

	for _, forbidden := range []string{
		"Export PDF",
		"Draft Slide",
		"Hidden Slide",
		`class="print-slide`,
		`/assets/margo.js`,
		`window.location.search`,
		`fetch('/__margo/export/pdf'`,
		`Open by describing the friction in browser-based slide tools`,
		`The authoring workflow is based on a Hugo-like project model`,
	} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("expected print html to exclude %q", forbidden)
		}
	}

	if _, err := os.Stat(filepath.Join(projectRoot, OutputDir, "assets", "shared-grid.svg")); err != nil {
		t.Fatalf("expected staged print asset to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, OutputDir, "slides", "02-why", "diagram.svg")); err != nil {
		t.Fatalf("expected staged print slide asset to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, OutputDir, "themes", "default", "assets", "chart.umd.min.js")); err != nil {
		t.Fatalf("expected staged print theme asset to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, OutputDir, "themes", "default", "assets", "katex.min.css")); err != nil {
		t.Fatalf("expected staged print katex css asset to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, OutputDir, "themes", "default", "assets", "katex.min.js")); err != nil {
		t.Fatalf("expected staged print katex js asset to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, OutputDir, "themes", "default", "assets", "fonts", "KaTeX_Main-Regular.woff2")); err != nil {
		t.Fatalf("expected staged print katex font asset to exist: %v", err)
	}
	themeCSS, err := os.ReadFile(filepath.Join(projectRoot, OutputDir, "themes", "default", "assets", "theme.css"))
	if err != nil {
		t.Fatalf("read staged theme css: %v", err)
	}
	for _, needle := range []string{`size: 13.333in 7.5in`, `page-break-after: always;`, `display: block !important`} {
		if !strings.Contains(string(themeCSS), needle) {
			t.Fatalf("expected staged theme css to contain %q", needle)
		}
	}
}

func TestThemePrintDeckTemplateOverridesFallback(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "arca-investor-memo")

	raw, err := config.LoadRaw(filepath.Join(projectRoot, config.DefaultFilename))
	if err != nil {
		t.Fatalf("load raw config: %v", err)
	}
	parsed, err := config.Parse(raw)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	slides, err := content.DiscoverSlides(projectRoot)
	if err != nil {
		t.Fatalf("discover slides: %v", err)
	}
	filtered := deck.ApplySectionDividers(deck.FilterSlides(slides, deck.FilterOptions{}))
	model := deck.Model{
		Config:   parsed.Config,
		Sections: deck.BuildSections(filtered),
		Slides:   filtered,
	}

	activeTheme, err := theme.Load(projectRoot, parsed.Config.Theme.Name)
	if err != nil {
		t.Fatalf("load theme: %v", err)
	}
	parsed.Config.Theme.Options, err = theme.ResolveOptions(activeTheme, parsed.Config.Theme.Options)
	if err != nil {
		t.Fatalf("resolve theme options: %v", err)
	}
	model.Config.Theme.Options = parsed.Config.Theme.Options

	if _, err := Write(projectRoot, model, activeTheme); err != nil {
		t.Fatalf("write print html: %v", err)
	}

	rendered, err := os.ReadFile(filepath.Join(projectRoot, OutputFile))
	if err != nil {
		t.Fatalf("read print html: %v", err)
	}
	out := string(rendered)

	for _, needle := range []string{
		`class="margo-print"`,
		`class="slide `,
		`class="slide-frame"`,
		`class="slide-shell fancy-title-slide"`,
		`class="slide-shell content-layout"`,
		`class="brand-mark"`,
		`Confidential - For Internal Use Only`,
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("expected theme print html to contain %q", needle)
		}
	}

	for _, forbidden := range []string{
		`class="print-slide`,
		`deck-control`,
		`window.location.search`,
		`fetch('/__margo/export/pdf'`,
	} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("expected theme print html to exclude %q", forbidden)
		}
	}

	themeCSS, err := os.ReadFile(filepath.Join(projectRoot, OutputDir, "themes", "arca-institutional-refined", "assets", "theme.css"))
	if err != nil {
		t.Fatalf("read staged theme css: %v", err)
	}
	for _, needle := range []string{
		`background: var(--margo-background-overlay, transparent);`,
		`font-family: var(--body-font);`,
	} {
		if !strings.Contains(string(themeCSS), needle) {
			t.Fatalf("expected staged theme css to contain %q", needle)
		}
	}
}
