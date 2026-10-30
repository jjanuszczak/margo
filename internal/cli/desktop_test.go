package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jjanuszczak/margo/internal/desktop"
	"github.com/jjanuszczak/margo/internal/manifest"
	"github.com/jjanuszczak/margo/internal/scaffold"
)

func TestDesktopSlideOperationsPreserveSourceNotesAndAssets(t *testing.T) {
	root := filepath.Join(t.TempDir(), "deck")
	if err := scaffold.CreateDeck(scaffold.DeckOptions{Name: "Desktop", TargetDir: root}); err != nil {
		t.Fatal(err)
	}
	slides, _, err := loadResolvedSlides(root)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, slide := range slides {
		ids = append(ids, slide.ID)
	}
	if err := manifest.Save(root, manifest.File{Slides: ids}); err != nil {
		t.Fatal(err)
	}
	original := slides[0]
	sourcePath := filepath.Join(original.BundlePath, "index.md")
	source, _ := os.ReadFile(sourcePath)
	source = []byte(strings.Replace(string(source), "---\n", "---\n# untouched\nunknown: custom-value\n", 1))
	os.WriteFile(sourcePath, source, 0o644)
	os.WriteFile(filepath.Join(original.BundlePath, "diagram.svg"), []byte("asset"), 0o644)
	os.MkdirAll(filepath.Join(original.BundlePath, "notes"), 0o755)
	os.WriteFile(filepath.Join(original.BundlePath, "notes", "research.md"), []byte("# Research\nunchanged"), 0o644)
	duplicate, err := desktopMutate(root, desktop.Request{Op: "duplicate", Slide: original.ID, Name: "Duplicate"})
	if err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{"index.md", "diagram.svg", "notes/research.md"} {
		data, err := os.ReadFile(filepath.Join(root, "slides", duplicate, relative))
		if err != nil {
			t.Fatal(err)
		}
		if relative == "index.md" {
			if !strings.Contains(string(data), "# untouched\nunknown: custom-value") {
				t.Fatal("lost unknown metadata")
			}
		} else if len(data) == 0 {
			t.Fatal("lost bundle contents")
		}
	}
	moved, err := desktopMutate(root, desktop.Request{Op: "move", Slide: duplicate, Position: 1})
	if err != nil {
		t.Fatal(err)
	}
	ordered, _, err := loadResolvedSlides(root)
	if err != nil {
		t.Fatal(err)
	}
	if ordered[0].ID != moved {
		t.Fatal("incorrect order")
	}
	if _, err := desktopMutate(root, desktop.Request{Op: "delete", Slide: moved}); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(root, ".margo-trash", "*", moved, "notes", "research.md"))
	if len(matches) != 1 {
		t.Fatal("delete did not preserve notes in trash")
	}
	created, err := desktopMutate(root, desktop.Request{Op: "create", Name: "Another slide"})
	if err != nil {
		t.Fatal(err)
	}
	ordered, _, err = loadResolvedSlides(root)
	if err != nil {
		t.Fatal(err)
	}
	if ordered[len(ordered)-1].ID != created {
		t.Fatal("create did not append")
	}
}
