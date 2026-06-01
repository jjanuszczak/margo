package printhtml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"margo/internal/config"
	"margo/internal/content"
	"margo/internal/deck"
	"margo/internal/theme"
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
		`class="print-slide print-slide-layout-title"`,
		`class="print-slide print-slide-layout-section"`,
		`class="print-slide print-slide-layout-content"`,
		`class="print-slide print-slide-layout-media-right"`,
		`class="print-slide print-slide-layout-media-left"`,
		`Strategy`,
		`Why Margo`,
		`Customer Story`,
		`Architecture View`,
		`color-scheme: dark;`,
		`A reasonably sized deck should still feel fast.`,
		`assets/shared-grid.svg`,
		`slides/02-why/diagram.svg`,
		`slides/05-customer-story/spotlight.svg`,
		`size: 13.333in 7.5in`,
		`page-break-after: always;`,
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("expected print html to contain %q", needle)
		}
	}

	for _, forbidden := range []string{
		"Export PDF",
		"window.__margoFixture",
		"<script",
		"Draft Slide",
		"Hidden Slide",
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
}
