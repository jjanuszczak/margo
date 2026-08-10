package deck

import "testing"

func TestApplySectionDividersInsertsSyntheticDivider(t *testing.T) {
	slides := []Slide{
		{ID: "title", FrontMatter: FrontMatter{Title: "Intro", Layout: "title"}},
		{ID: "why", FrontMatter: FrontMatter{Title: "Why", Layout: "content", Section: "Strategy"}},
	}

	shaped := ApplySectionDividers(slides)
	if got, want := len(shaped), 3; got != want {
		t.Fatalf("expected %d shaped slides, got %d", want, got)
	}

	divider := shaped[1]
	if !divider.Synthetic {
		t.Fatal("expected inserted section divider to be synthetic")
	}
	if divider.Layout != "section" {
		t.Fatalf("expected divider layout %q, got %q", "section", divider.Layout)
	}
	if divider.Title != "Strategy" {
		t.Fatalf("expected divider title %q, got %q", "Strategy", divider.Title)
	}
	if divider.BodyMarkdown != "# Strategy" {
		t.Fatalf("expected divider body %q, got %q", "# Strategy", divider.BodyMarkdown)
	}
}

func TestApplySectionDividersSkipsExplicitSectionSlide(t *testing.T) {
	slides := []Slide{
		{ID: "title", FrontMatter: FrontMatter{Title: "Intro", Layout: "title"}},
		{ID: "section", FrontMatter: FrontMatter{Title: "Strategy", Layout: "section", Section: "Strategy"}},
		{ID: "why", FrontMatter: FrontMatter{Title: "Why", Layout: "content", Section: "Strategy"}},
	}

	shaped := ApplySectionDividers(slides)
	if got, want := len(shaped), 3; got != want {
		t.Fatalf("expected explicit section slide to suppress auto-divider, got %d slides", got)
	}

	for _, slide := range shaped {
		if slide.Synthetic {
			t.Fatal("did not expect a synthetic divider when an explicit section slide exists")
		}
	}
}

func TestFilterSlidesFiltersHiddenAndDraftNotes(t *testing.T) {
	slides := []Slide{{
		ID: "title",
		Notes: []Note{
			{ID: "visible"},
			{ID: "hidden", Visibility: "hidden"},
			{ID: "draft", Draft: true},
		},
	}}

	buildNotes := FilterSlides(slides, FilterOptions{})[0].Notes
	if got, want := len(buildNotes), 1; got != want || buildNotes[0].ID != "visible" {
		t.Fatalf("expected only visible note in build, got %#v", buildNotes)
	}
	serveNotes := FilterSlides(slides, FilterOptions{IncludeDrafts: true})[0].Notes
	if got, want := len(serveNotes), 2; got != want || serveNotes[1].ID != "draft" {
		t.Fatalf("expected visible and draft notes in serve, got %#v", serveNotes)
	}
}
