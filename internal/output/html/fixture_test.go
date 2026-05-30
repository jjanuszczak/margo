package html

import (
	"io"
	"io/fs"
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
	projectRoot := fixtureProjectRoot(t)

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
	if got, want := len(slides), 6; got != want {
		t.Fatalf("expected %d slide bundles, got %d", want, got)
	}

	filtered := deck.FilterSlides(slides, deck.FilterOptions{})
	if got, want := len(filtered), 4; got != want {
		t.Fatalf("expected %d build-visible slides, got %d", want, got)
	}
	shaped := deck.ApplySectionDividers(filtered)
	if got, want := len(shaped), 5; got != want {
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
	if err := Write(projectRoot, model, activeTheme); err != nil {
		t.Fatalf("write html: %v", err)
	}

	htmlPath := filepath.Join(projectRoot, OutputFile)
	rendered, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read rendered html: %v", err)
	}
	out := string(rendered)

	for _, needle := range []string{"Margo Reference Deck", "Strategy", "Why Margo", "Customer Story", "Architecture View", "Export PDF", "MARGO", "Product Strategy", "Customer Momentum", "Authoring and output model overview", "image-fit-contain", "#4db6ac", "color-scheme: dark", "\"Avenir Next\", \"Helvetica Neue\", sans-serif", "slides/02-why/diagram.svg", "slides/02-why/backdrop.svg", "slides/05-customer-story/spotlight.svg", "assets/shared-grid.svg", "assets/video-poster.svg", "https://example.com/demo.mp4", "class=\"callout", "class=\"shortcode-columns\"", "class=\"shortcode-stat\"", "class=\"shortcode-video\"", "media-split-slide media-right", "media-split-slide media-left", "@media print", "size: 13.333in 7.5in", "display: block !important", "break-after: page", "A reasonably sized deck should still feel fast.", "Shortcodes can carry shared deck assets cleanly."} {
		if !strings.Contains(out, needle) {
			t.Fatalf("expected rendered output to contain %q", needle)
		}
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
	projectRoot := fixtureProjectRoot(t)

	slides, err := content.DiscoverSlides(projectRoot)
	if err != nil {
		t.Fatalf("discover slides: %v", err)
	}

	filtered := deck.FilterSlides(slides, deck.FilterOptions{IncludeDrafts: true})
	if got, want := len(filtered), 5; got != want {
		t.Fatalf("expected %d serve-visible slides, got %d", want, got)
	}
	shaped := deck.ApplySectionDividers(filtered)
	if got, want := len(shaped), 6; got != want {
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

func fixtureProjectRoot(t *testing.T) string {
	t.Helper()
	sourceRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "examples", "reference-deck"))
	if err != nil {
		t.Fatalf("resolve fixture project root: %v", err)
	}

	projectRoot := filepath.Join(t.TempDir(), "reference-deck")
	if err := copyFixtureDeck(sourceRoot, projectRoot); err != nil {
		t.Fatalf("copy fixture project root: %v", err)
	}
	return projectRoot
}

func copyFixtureDeck(srcRoot string, dstRoot string) error {
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return os.MkdirAll(dstRoot, 0o755)
		}
		if relPath == "dist" && d.IsDir() {
			return filepath.SkipDir
		}
		if strings.HasPrefix(relPath, "dist"+string(filepath.Separator)) {
			return nil
		}
		if !shouldCopyFixturePath(relPath, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		targetPath := filepath.Join(dstRoot, relPath)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return err
		}
		return nil
	})
}

func shouldCopyFixturePath(relPath string, isDir bool) bool {
	exact := map[string]bool{
		"archetypes": true,
		"assets":     true,
		"slides":     true,
		"themes":     true,
		"margo.yaml": true,
	}

	if exact[relPath] {
		return true
	}

	allowedSubtrees := []string{
		"archetypes/agenda",
		"archetypes/closing",
		"archetypes/default",
		"archetypes/image",
		"archetypes/media-left",
		"archetypes/media-right",
		"archetypes/metric",
		"archetypes/quote",
		"archetypes/section",
		"archetypes/title",
		"archetypes/two-column",
		"assets/shared-grid.svg",
		"assets/video-poster.svg",
		"slides/01-title",
		"slides/02-why",
		"slides/03-draft",
		"slides/04-hidden",
		"slides/05-customer-story",
		"slides/06-architecture-view",
		"themes/default",
	}

	for _, prefix := range allowedSubtrees {
		if relPath == prefix {
			return true
		}
		if strings.HasPrefix(relPath, prefix+string(filepath.Separator)) {
			return true
		}
	}

	return false
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
