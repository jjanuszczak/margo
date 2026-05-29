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
