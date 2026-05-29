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
	if got, want := len(slides), 4; got != want {
		t.Fatalf("expected %d slide bundles, got %d", want, got)
	}

	filtered := deck.FilterSlides(slides, deck.FilterOptions{})
	if got, want := len(filtered), 2; got != want {
		t.Fatalf("expected %d build-visible slides, got %d", want, got)
	}

	sections := deck.BuildSections(filtered)
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

	model := deck.Model{
		Config:   parsed.Config,
		Sections: sections,
		Slides:   filtered,
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

	for _, needle := range []string{"Margo Reference Deck", "Strategy", "Why Margo"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("expected rendered output to contain %q", needle)
		}
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
	if got, want := len(filtered), 3; got != want {
		t.Fatalf("expected %d serve-visible slides, got %d", want, got)
	}

	for _, slide := range filtered {
		if slide.Title == "Hidden Slide" {
			t.Fatal("hidden slide should not appear in serve-visible set")
		}
	}

	foundDraft := false
	for _, slide := range filtered {
		if slide.Title == "Draft Slide" {
			foundDraft = true
		}
	}
	if !foundDraft {
		t.Fatal("expected draft slide to appear in serve-visible set")
	}
}

func fixtureProjectRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "examples", "reference-deck"))
	if err != nil {
		t.Fatalf("resolve fixture project root: %v", err)
	}
	return root
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
