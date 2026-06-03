package html

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

func TestReferenceDeckBuildFlow(t *testing.T) {
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
	if got, want := len(slides), 8; got != want {
		t.Fatalf("expected %d slide bundles, got %d", want, got)
	}

	filtered := deck.FilterSlides(slides, deck.FilterOptions{})
	if got, want := len(filtered), 6; got != want {
		t.Fatalf("expected %d build-visible slides, got %d", want, got)
	}
	shaped := deck.ApplySectionDividers(filtered)
	if got, want := len(shaped), 7; got != want {
		t.Fatalf("expected %d build-rendered slides, got %d", want, got)
	}

	sections := deck.BuildSections(shaped)
	if got, want := len(sections), 1; got != want {
		t.Fatalf("expected %d section, got %d", want, got)
	}
	if sections[0].Title != "Strategy" {
		t.Fatalf("expected section title %q, got %q", "Strategy", sections[0].Title)
	}

	why := findSlideByID(t, slides, "02-why")
	if got, want := len(why.Notes), 3; got != want {
		t.Fatalf("expected %d notes on 02-why, got %d: %#v", want, got, why.Notes)
	}
	if strings.Contains(why.BodyMarkdown, "Remind the audience that editable PPTX remains a later goal.") {
		t.Fatalf("expected body notes to be removed from rendered body: %q", why.BodyMarkdown)
	}

	activeTheme, err := theme.Load(projectRoot, parsed.Config.Theme.Name)
	if err != nil {
		t.Fatalf("load theme: %v", err)
	}
	parsed.Config.Theme.Options, err = theme.ResolveOptions(activeTheme, parsed.Config.Theme.Options)
	if err != nil {
		t.Fatalf("resolve theme options: %v", err)
	}
	if got, want := parsed.Config.Theme.Options["color_mode"], "dark"; got != want {
		t.Fatalf("resolved color_mode = %v, want %q", got, want)
	}
	if got, want := parsed.Config.Theme.Options["typography"], "executive"; got != want {
		t.Fatalf("resolved typography = %v, want %q", got, want)
	}

	model := deck.Model{
		Config:   parsed.Config,
		Sections: sections,
		Slides:   shaped,
	}
	report, err := Write(projectRoot, model, activeTheme)
	if err != nil {
		t.Fatalf("write html: %v", err)
	}
	if len(report.Items) != 0 {
		t.Fatalf("expected reference deck to build without warnings, got %#v", report.Items)
	}

	htmlPath := filepath.Join(projectRoot, OutputFile)
	rendered, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read rendered html: %v", err)
	}
	out := string(rendered)

	for _, needle := range []string{"Margo Reference Deck", "Strategy", "Why Margo", "Customer Story", "Architecture View", "Mermaid QA", "Chart QA", "Export PDF", "MARGO", "Product Strategy", "Customer Momentum", "Diagram Regression Check", "Chart Regression Check", "Authoring and output model overview", "image-fit-contain", "#4db6ac", "color-scheme: dark", "\"Avenir Next\", \"Helvetica Neue\", sans-serif", "class=\"slide-shell content-slide\"", "class=\"slide-shell title-slide\"", "class=\"slide-shell section-slide\"", "class=\"slide-shell split-layout\"", "class=\"slide-body section-stack\"", "slides/02-why/diagram.svg", "slides/02-why/backdrop.svg", "slides/05-customer-story/spotlight.svg", "assets/shared-grid.svg", "assets/video-poster.svg", "https://example.com/demo.mp4", "class=\"callout", "class=\"shortcode-columns\"", "class=\"shortcode-stat\"", "class=\"shortcode-video\"", "class=\"shortcode-figure shortcode-figure-fit-contain\"", "class=\"shortcode-figure-caption\"", "class=\"shortcode-github-repo\"", "class=\"shortcode-github-repo-card\"", "https://github.com/jjanuszczak/margo", "Static repo card shortcode for product and engineering decks.", "class=\"shortcode-mermaid shortcode-mermaid-align-center\"", "class=\"shortcode-mermaid-definition\"", "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs", "flowchart LR", "A[Markdown] --> B[Build]", "Authoring-to-output flow", "flowchart TD", "A[\"Author writes Markdown\"] --> B[\"Shortcode expands in HTML\"]", "Reference-deck Mermaid smoke test", "class=\"shortcode-chart\"", "class=\"shortcode-chart-canvas\"", "class=\"shortcode-chart-config\"", "themes/default/assets/chart.umd.min.js", "\"type\": \"doughnut\"", "\"label\": \"Margo outputs\"", "Reference-deck Chart.js smoke test", "class=\"nav-hint\"", "class=\"slide-dots\"", "data-slide-dot=\"0\"", "scroll-snap-type: x mandatory", "window.matchMedia('(max-width: 900px)')", "scrollIntoView({ behavior: 'smooth', inline: 'start', block: 'nearest' })", "Source: shared deck assets", "Deck-level asset rendered through the figure shortcode.", "media-split-slide media-right", "media-split-slide media-left", "@media print", "size: 13.333in 7.5in", "display: block !important", "break-after: page", "A reasonably sized deck should still feel fast.", "Shortcodes can carry shared deck assets cleanly.", "Why this exists", "Markdown", "Front matter", "Archetypes", "HTML first", "PDF next", "PPTX later", "<meta name=\"margo-fixture\" content=\"reference-deck\">", "window.__margoFixture = \"reference-deck\""} {
		if !strings.Contains(out, needle) {
			t.Fatalf("expected rendered output to contain %q", needle)
		}
	}
	if strings.Contains(out, "<p>flowchart LR") || strings.Contains(out, "<p>flowchart TD") {
		t.Fatalf("expected rendered output not to leak raw mermaid source paragraph")
	}
	if !strings.Contains(out, "Internal Strategy Review") {
		t.Fatalf("expected rendered output to contain deck footer %q", "Internal Strategy Review")
	}
	if _, err := os.Stat(filepath.Join(projectRoot, OutputDir, "slides", "02-why", "diagram.svg")); err != nil {
		t.Fatalf("expected staged slide asset to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, OutputDir, "slides", "02-why", "backdrop.svg")); err != nil {
		t.Fatalf("expected staged background asset to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, OutputDir, "slides", "05-customer-story", "spotlight.svg")); err != nil {
		t.Fatalf("expected staged media slide asset to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, OutputDir, "assets", "shared-grid.svg")); err != nil {
		t.Fatalf("expected staged deck asset to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, OutputDir, "assets", "video-poster.svg")); err != nil {
		t.Fatalf("expected staged video poster asset to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, OutputDir, "themes", "default", "assets", "chart.umd.min.js")); err != nil {
		t.Fatalf("expected staged theme chart asset to exist: %v", err)
	}

	for _, forbidden := range []string{
		"Draft Slide",
		"Hidden Slide",
		"Remind the audience that editable PPTX remains a later goal.",
		"Mention Hugo mental model",
	} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("expected rendered output to exclude %q", forbidden)
		}
	}
}

func TestReferenceDeckServeFilteringIncludesDrafts(t *testing.T) {
	projectRoot := fixtureProjectRoot(t, "reference-deck")

	slides, err := content.DiscoverSlides(projectRoot)
	if err != nil {
		t.Fatalf("discover slides: %v", err)
	}

	filtered := deck.FilterSlides(slides, deck.FilterOptions{IncludeDrafts: true})
	if got, want := len(filtered), 7; got != want {
		t.Fatalf("expected %d serve-visible slides, got %d", want, got)
	}
	shaped := deck.ApplySectionDividers(filtered)
	if got, want := len(shaped), 8; got != want {
		t.Fatalf("expected %d serve-rendered slides, got %d", want, got)
	}

	for _, slide := range shaped {
		if slide.Title == "Hidden Slide" {
			t.Fatal("hidden slide should not appear in serve-visible set")
		}
	}

	foundDraft := false
	foundDivider := false
	for _, slide := range shaped {
		if slide.Title == "Draft Slide" {
			foundDraft = true
		}
		if slide.Synthetic && slide.Layout == "section" && slide.Title == "Strategy" {
			foundDivider = true
		}
	}
	if !foundDraft {
		t.Fatal("expected draft slide to appear in serve-visible set")
	}
	if !foundDivider {
		t.Fatal("expected synthetic Strategy section divider to appear in serve-rendered set")
	}
}

func findSlideByID(t *testing.T, slides []deck.Slide, id string) deck.Slide {
	t.Helper()
	for _, slide := range slides {
		if slide.ID == id {
			return slide
		}
	}
	t.Fatalf("slide %q not found", id)
	return deck.Slide{}
}
