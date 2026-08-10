package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateSlideSectionArchetype(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := CreateDeck(DeckOptions{
		Name:      "test-deck",
		TargetDir: projectRoot,
	}); err != nil {
		t.Fatalf("create deck: %v", err)
	}

	indexPath, err := CreateSlide(SlideOptions{
		ProjectRoot: projectRoot,
		Name:        "Strategy",
		Archetype:   "section",
	})
	if err != nil {
		t.Fatalf("create section slide: %v", err)
	}

	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read section slide: %v", err)
	}
	body := string(raw)

	for _, needle := range []string{
		"layout: section",
		"type: section",
		"section: Strategy",
		"# Strategy",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("expected section slide to contain %q, got:\n%s", needle, body)
		}
	}
}

func TestCreateSlideTitleArchetype(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := CreateDeck(DeckOptions{
		Name:      "test-deck",
		TargetDir: projectRoot,
	}); err != nil {
		t.Fatalf("create deck: %v", err)
	}

	indexPath, err := CreateSlide(SlideOptions{
		ProjectRoot: projectRoot,
		Name:        "Launch Update",
		Archetype:   "title",
	})
	if err != nil {
		t.Fatalf("create title slide: %v", err)
	}

	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read title slide: %v", err)
	}
	body := string(raw)

	for _, needle := range []string{
		"layout: title",
		"type: title",
		"# Launch Update",
		"Add a subtitle or opening statement.",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("expected title slide to contain %q, got:\n%s", needle, body)
		}
	}
}

func TestCreateSlideAgendaArchetype(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := CreateDeck(DeckOptions{
		Name:      "test-deck",
		TargetDir: projectRoot,
	}); err != nil {
		t.Fatalf("create deck: %v", err)
	}

	indexPath, err := CreateSlide(SlideOptions{
		ProjectRoot: projectRoot,
		Name:        "Plan",
		Archetype:   "agenda",
	})
	if err != nil {
		t.Fatalf("create agenda slide: %v", err)
	}

	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read agenda slide: %v", err)
	}
	body := string(raw)

	for _, needle := range []string{
		"layout: agenda",
		"type: agenda",
		"1. First topic",
		"3. Third topic",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("expected agenda slide to contain %q, got:\n%s", needle, body)
		}
	}
}

func TestCreateSlideMetricArchetype(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := CreateDeck(DeckOptions{
		Name:      "test-deck",
		TargetDir: projectRoot,
	}); err != nil {
		t.Fatalf("create deck: %v", err)
	}

	indexPath, err := CreateSlide(SlideOptions{
		ProjectRoot: projectRoot,
		Name:        "North Star",
		Archetype:   "metric",
	})
	if err != nil {
		t.Fatalf("create metric slide: %v", err)
	}

	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read metric slide: %v", err)
	}
	body := string(raw)

	for _, needle := range []string{
		"layout: metric",
		"type: metric",
		"# 42%",
		"Primary metric label",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("expected metric slide to contain %q, got:\n%s", needle, body)
		}
	}
}

func TestCreateSlideImageArchetype(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := CreateDeck(DeckOptions{
		Name:      "test-deck",
		TargetDir: projectRoot,
	}); err != nil {
		t.Fatalf("create deck: %v", err)
	}

	indexPath, err := CreateSlide(SlideOptions{
		ProjectRoot: projectRoot,
		Name:        "Hero Visual",
		Archetype:   "image",
	})
	if err != nil {
		t.Fatalf("create image slide: %v", err)
	}

	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read image slide: %v", err)
	}
	body := string(raw)

	for _, needle := range []string{
		"layout: image",
		"type: image",
		"fit: cover",
		"![Describe the visual](image.png)",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("expected image slide to contain %q, got:\n%s", needle, body)
		}
	}
}

func TestCreateSlideTwoColumnArchetype(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := CreateDeck(DeckOptions{
		Name:      "test-deck",
		TargetDir: projectRoot,
	}); err != nil {
		t.Fatalf("create deck: %v", err)
	}

	indexPath, err := CreateSlide(SlideOptions{
		ProjectRoot: projectRoot,
		Name:        "Split View",
		Archetype:   "two-column",
	})
	if err != nil {
		t.Fatalf("create two-column slide: %v", err)
	}

	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read two-column slide: %v", err)
	}
	body := string(raw)

	for _, needle := range []string{
		"layout: two-column",
		"type: two-column",
		"<!-- column-break -->",
		"Left column content goes here.",
		"Right column content goes here.",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("expected two-column slide to contain %q, got:\n%s", needle, body)
		}
	}
}

func TestCreateSlideMediaRightArchetype(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := CreateDeck(DeckOptions{
		Name:      "test-deck",
		TargetDir: projectRoot,
	}); err != nil {
		t.Fatalf("create deck: %v", err)
	}

	indexPath, err := CreateSlide(SlideOptions{
		ProjectRoot: projectRoot,
		Name:        "Customer Story",
		Archetype:   "media-right",
	})
	if err != nil {
		t.Fatalf("create media-right slide: %v", err)
	}

	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read media-right slide: %v", err)
	}
	body := string(raw)

	for _, needle := range []string{
		"layout: media-right",
		"type: media-right",
		"![Describe the visual](image.png)",
		"- Primary point",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("expected media-right slide to contain %q, got:\n%s", needle, body)
		}
	}
}

func TestCreateSlideMediaLeftArchetype(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := CreateDeck(DeckOptions{
		Name:      "test-deck",
		TargetDir: projectRoot,
	}); err != nil {
		t.Fatalf("create deck: %v", err)
	}

	indexPath, err := CreateSlide(SlideOptions{
		ProjectRoot: projectRoot,
		Name:        "Architecture",
		Archetype:   "media-left",
	})
	if err != nil {
		t.Fatalf("create media-left slide: %v", err)
	}

	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read media-left slide: %v", err)
	}
	body := string(raw)

	for _, needle := range []string{
		"layout: media-left",
		"type: media-left",
		"![Describe the visual](image.png)",
		"- Supporting point",
	} {
		if !strings.Contains(body, needle) {
			t.Fatalf("expected media-left slide to contain %q, got:\n%s", needle, body)
		}
	}
}

func TestCreateNote(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := CreateDeck(DeckOptions{Name: "test-deck", TargetDir: projectRoot}); err != nil {
		t.Fatalf("create deck: %v", err)
	}

	path, err := CreateNote(NoteOptions{ProjectRoot: projectRoot, Slide: "02-why", Name: "Speaker Script"})
	if err != nil {
		t.Fatalf("CreateNote returned error: %v", err)
	}
	if want := filepath.Join(projectRoot, "slides", "02-why", "notes", "speaker-script.md"); path != want {
		t.Fatalf("expected note path %q, got %q", want, path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	for _, needle := range []string{"id: speaker-script", "title: Speaker script", "visibility: visible", "kind: note", "Add notes here."} {
		if !strings.Contains(string(raw), needle) {
			t.Fatalf("expected note scaffold to contain %q, got:\n%s", needle, raw)
		}
	}
	if _, err := CreateNote(NoteOptions{ProjectRoot: projectRoot, Slide: "missing", Name: "Research"}); err == nil {
		t.Fatal("expected missing slide bundle to fail")
	}
}
