package pptx

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"margo/internal/deck"
	"margo/internal/theme"
)

func TestWriteCreatesEditablePackageWithNotesAndImage(t *testing.T) {
	projectRoot := t.TempDir()
	slideDir := filepath.Join(projectRoot, "slides", "01-title")
	if err := os.MkdirAll(slideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(slideDir, "hero.png"), []byte("png-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	model := deck.Model{
		Config: deck.ProjectConfig{Deck: deck.DeckMetadata{Title: "Test Deck"}},
		Slides: []deck.Slide{{
			ID:           "01-title",
			BundlePath:   slideDir,
			FrontMatter:  deck.FrontMatter{Title: "Title"},
			BodyMarkdown: "# Title\n\n- Editable text\n\n[Project](https://example.com)\n\n| Metric | Value |\n| --- | --- |\n| Slides | 20 |\n\n![Hero](hero.png)",
			Notes:        []deck.Note{{Name: "Notes", Markdown: "Speaker notes"}},
		}},
	}

	if report, err := Write(projectRoot, model, theme.Metadata{}); err != nil {
		t.Fatalf("Write returned error: %v", err)
	} else if len(report.Items) != 0 {
		t.Fatalf("Write returned unexpected diagnostics: %#v", report.Items)
	}

	path := filepath.Join(projectRoot, OutputFile)
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open generated pptx: %v", err)
	}
	defer archive.Close()
	want := []string{
		"[Content_Types].xml",
		"ppt/presentation.xml",
		"ppt/slides/slide1.xml",
		"ppt/slides/_rels/slide1.xml.rels",
		"ppt/media/slide-01-title-1.png",
		"ppt/notesSlides/notesSlide1.xml",
	}
	seen := make(map[string]bool)
	for _, file := range archive.File {
		seen[file.Name] = true
	}
	for _, name := range want {
		if !seen[name] {
			t.Errorf("generated pptx missing %q", name)
		}
	}
	for _, file := range archive.File {
		if !strings.HasSuffix(file.Name, ".xml") {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		if err := xml.NewDecoder(rc).Decode(&struct{}{}); err != nil {
			rc.Close()
			t.Errorf("generated XML part %q is malformed: %v", file.Name, err)
			continue
		}
		rc.Close()
	}

	slideXML := readZipPart(t, archive, "ppt/slides/slide1.xml")
	if !strings.Contains(slideXML, "Editable text") || !strings.Contains(slideXML, "r:embed=\"rId2\"") || !strings.Contains(slideXML, "hlinkClick r:id=\"rId3\"") || !strings.Contains(slideXML, "Metric") {
		t.Fatalf("slide XML did not contain editable text and image reference: %s", slideXML)
	}
	rels := readZipPart(t, archive, "ppt/slides/_rels/slide1.xml.rels")
	if !strings.Contains(rels, `Target="https://example.com" TargetMode="External"`) {
		t.Fatalf("slide relationships did not contain external hyperlink: %s", rels)
	}
}

func TestResolveLayoutGeometryUsesThemeRecipe(t *testing.T) {
	active := theme.Metadata{PPTX: &theme.PPTXMetadata{Layouts: map[string]theme.PPTXLayout{
		"media-right": {BodyX: 0.8, BodyY: 1.7, BodyWidth: 6.2, ImagePosition: "right", ImageWidth: 5.1, ImageHeight: 3.2},
	}}}
	geometry := resolveLayoutGeometry(active, "media-right")
	if geometry.bodyX != inches(0.8) || geometry.bodyY != inches(1.7) || geometry.bodyWidth != inches(6.2) {
		t.Fatalf("unexpected recipe body geometry: %#v", geometry)
	}
	if geometry.imageX != slideWidth-geometry.imageWidth-700000 || geometry.imageHeight != inches(3.2) {
		t.Fatalf("unexpected recipe image geometry: %#v", geometry)
	}
}

func readZipPart(t *testing.T, archive *zip.ReadCloser, name string) string {
	t.Helper()
	for _, file := range archive.File {
		if file.Name != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	t.Fatalf("missing zip part %q", name)
	return ""
}
