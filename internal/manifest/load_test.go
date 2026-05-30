package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"margo/internal/deck"
)

func TestApplyUsesManifestOrder(t *testing.T) {
	slides := []deck.Slide{
		{ID: "01-title", FrontMatter: deck.FrontMatter{Title: "Title"}},
		{ID: "02-why", FrontMatter: deck.FrontMatter{Title: "Why"}},
		{ID: "03-roadmap", FrontMatter: deck.FrontMatter{Title: "Roadmap"}},
	}

	ordered, err := Apply(slides, File{
		Slides: []string{"03-roadmap", "01-title", "02-why"},
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	got := []string{ordered[0].ID, ordered[1].ID, ordered[2].ID}
	want := []string{"03-roadmap", "01-title", "02-why"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected slide %d to be %q, got %q", i, want[i], got[i])
		}
	}
}

func TestApplyRejectsIncompleteManifest(t *testing.T) {
	slides := []deck.Slide{
		{ID: "01-title"},
		{ID: "02-why"},
	}

	_, err := Apply(slides, File{Slides: []string{"01-title"}})
	if err == nil {
		t.Fatal("expected incomplete manifest to fail")
	}
	if !strings.Contains(err.Error(), `missing slide "02-why"`) {
		t.Fatalf("expected missing slide error, got %v", err)
	}
}

func TestApplyRejectsUnknownOrDuplicateSlides(t *testing.T) {
	slides := []deck.Slide{
		{ID: "01-title"},
		{ID: "02-why"},
	}

	_, err := Apply(slides, File{Slides: []string{"01-title", "missing"}})
	if err == nil || !strings.Contains(err.Error(), `unknown slide "missing"`) {
		t.Fatalf("expected unknown slide error, got %v", err)
	}

	_, err = Apply(slides, File{Slides: []string{"01-title", "01-title"}})
	if err == nil || !strings.Contains(err.Error(), `duplicate slide "01-title"`) {
		t.Fatalf("expected duplicate slide error, got %v", err)
	}
}

func TestAppendSlideUpdatesManifestFile(t *testing.T) {
	projectRoot := t.TempDir()
	if err := Save(projectRoot, File{Slides: []string{"01-title", "02-why"}}); err != nil {
		t.Fatalf("save manifest: %v", err)
	}

	if err := AppendSlide(projectRoot, "03-roadmap"); err != nil {
		t.Fatalf("append slide: %v", err)
	}

	manifestFile, ok, err := Load(projectRoot)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if !ok {
		t.Fatal("expected manifest to exist")
	}
	got := strings.Join(manifestFile.Slides, ",")
	if got != "01-title,02-why,03-roadmap" {
		t.Fatalf("unexpected manifest order %q", got)
	}

	raw, err := os.ReadFile(filepath.Join(projectRoot, Filename))
	if err != nil {
		t.Fatalf("read manifest file: %v", err)
	}
	if !strings.Contains(string(raw), "- 03-roadmap") {
		t.Fatalf("expected manifest file to contain appended slide, got:\n%s", string(raw))
	}
}
