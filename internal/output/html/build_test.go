package html

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"margo/internal/deck"
)

func TestRenderBodyColumnsForTwoColumnLayout(t *testing.T) {
	slide := deck.Slide{
		FrontMatter: deck.FrontMatter{
			Layout: "two-column",
		},
		BodyMarkdown: "Left side\n\n<!-- column-break -->\n\nRight side",
	}

	columns := renderBodyColumns(slide, template.HTML("<p>ignored</p>"))
	if got, want := len(columns), 2; got != want {
		t.Fatalf("expected %d columns, got %d", want, got)
	}
	if string(columns[0]) == "" || string(columns[1]) == "" {
		t.Fatalf("expected both columns to render content, got %#v", columns)
	}
}

func TestRenderBodyColumnsFallsBackWhenNoMarker(t *testing.T) {
	body := template.HTML("<p>Whole body</p>")
	slide := deck.Slide{
		FrontMatter: deck.FrontMatter{
			Layout: "two-column",
		},
		BodyMarkdown: "Whole body",
	}

	columns := renderBodyColumns(slide, body)
	if got, want := len(columns), 1; got != want {
		t.Fatalf("expected %d fallback column, got %d", want, got)
	}
	if columns[0] != body {
		t.Fatalf("expected fallback body %q, got %q", body, columns[0])
	}
}

func TestResolveAssetReferenceSupportsDeckAssets(t *testing.T) {
	projectRoot := t.TempDir()
	assetsDir := filepath.Join(projectRoot, "assets")
	bundleDir := filepath.Join(projectRoot, "slides", "02-why")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("create assets dir: %v", err)
	}
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "shared-grid.svg"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write deck asset: %v", err)
	}

	slide := deck.Slide{ID: "02-why", BundlePath: bundleDir}
	got := resolveAssetReference(projectRoot, slide, "assets/shared-grid.svg")
	if want := "assets/shared-grid.svg"; got != want {
		t.Fatalf("resolveAssetReference() = %q, want %q", got, want)
	}
}

func TestRenderLeadMediaAndContentForMediaLayouts(t *testing.T) {
	slide := deck.Slide{
		FrontMatter: deck.FrontMatter{
			Layout: "media-right",
		},
	}
	body := template.HTML("<h2>Why this exists</h2><p><img src=\"slides/02-why/diagram.svg\" alt=\"flow\"></p><ul><li>Point</li></ul>")

	media := renderLeadMedia(slide, body)
	content := renderLeadContent(slide, body)

	if string(media) == "" || !strings.Contains(string(media), "<img") {
		t.Fatalf("expected extracted media html, got %q", media)
	}
	if strings.Contains(string(content), "<img") || !strings.Contains(string(content), "<ul>") {
		t.Fatalf("expected content without lead image and with remaining body, got %q", content)
	}
}
