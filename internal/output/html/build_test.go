package html

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"margo/internal/deck"
	"margo/internal/output/render"
)

func TestRenderBodyColumnsForTwoColumnLayout(t *testing.T) {
	slide := deck.Slide{
		FrontMatter: deck.FrontMatter{
			Layout: "two-column",
		},
		BodyMarkdown: "Left side\n\n<!-- column-break -->\n\nRight side",
	}

	columns := renderBodyColumns(slide, slide.BodyMarkdown, template.HTML("<p>ignored</p>"))
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

	columns := renderBodyColumns(slide, slide.BodyMarkdown, body)
	if got, want := len(columns), 1; got != want {
		t.Fatalf("expected %d fallback column, got %d", want, got)
	}
	if columns[0] != body {
		t.Fatalf("expected fallback body %q, got %q", body, columns[0])
	}
}

func TestRenderBodyColumnsUsesExpandedMarkdown(t *testing.T) {
	slide := deck.Slide{
		FrontMatter: deck.FrontMatter{
			Layout: "two-column",
		},
		BodyMarkdown: "{{< stat value=\"$1\" label=\"Raw\" />}}\n\n<!-- column-break -->\n\nRight side",
	}

	expanded := "<div class=\"shortcode-stat\"><div class=\"shortcode-stat-value\">$1</div></div>\n\n<!-- column-break -->\n\nRight side"
	columns := renderBodyColumns(slide, expanded, template.HTML("<p>ignored</p>"))
	if got, want := len(columns), 2; got != want {
		t.Fatalf("expected %d columns, got %d", want, got)
	}
	if !strings.Contains(string(columns[0]), "shortcode-stat") {
		t.Fatalf("expected expanded shortcode html in first column, got %q", columns[0])
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
	got, warning := resolveAssetReference(projectRoot, slide, "assets/shared-grid.svg", "slide markdown asset")
	if warning != nil {
		t.Fatalf("expected no warning, got %#v", warning)
	}
	if want := "assets/shared-grid.svg"; got != want {
		t.Fatalf("resolveAssetReference() = %q, want %q", got, want)
	}
}

func TestResolveAssetReferenceWarnsForMissingLocalAsset(t *testing.T) {
	projectRoot := t.TempDir()
	bundleDir := filepath.Join(projectRoot, "slides", "02-why")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}

	slide := deck.Slide{ID: "02-why", BundlePath: bundleDir}
	got, warning := resolveAssetReference(projectRoot, slide, "missing.svg", "slide markdown asset")
	if got != "missing.svg" {
		t.Fatalf("expected unresolved asset to remain unchanged, got %q", got)
	}
	if warning == nil || warning.Code != "asset_missing" {
		t.Fatalf("expected asset_missing warning, got %#v", warning)
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

func TestResolveDeckLogoKeepsPlainText(t *testing.T) {
	logo, report := resolveDeckLogo(t.TempDir(), "MARGO")
	if len(report.Items) != 0 {
		t.Fatalf("expected no warnings, got %#v", report.Items)
	}
	if logo.IsImage {
		t.Fatal("expected plain text logo, got image")
	}
	if logo.Text != "MARGO" {
		t.Fatalf("expected text logo %q, got %q", "MARGO", logo.Text)
	}
}

func TestResolveDeckLogoResolvesDeckAsset(t *testing.T) {
	projectRoot := t.TempDir()
	assetsDir := filepath.Join(projectRoot, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("create assets dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "logo.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatalf("write logo asset: %v", err)
	}

	logo, report := resolveDeckLogo(projectRoot, "assets/logo.svg")
	if len(report.Items) != 0 {
		t.Fatalf("expected no warnings, got %#v", report.Items)
	}
	if !logo.IsImage {
		t.Fatal("expected image logo")
	}
	if logo.ImageSrc != "assets/logo.svg" {
		t.Fatalf("expected image src %q, got %q", "assets/logo.svg", logo.ImageSrc)
	}
}

func TestResolveDeckLogoWarnsOnMissingAsset(t *testing.T) {
	projectRoot := t.TempDir()

	logo, report := resolveDeckLogo(projectRoot, "assets/missing.svg")
	if len(report.Items) != 1 {
		t.Fatalf("expected one warning, got %#v", report.Items)
	}
	if report.Items[0].Code != "asset_missing" {
		t.Fatalf("expected asset_missing warning, got %#v", report.Items[0])
	}
	if logo.IsImage {
		t.Fatal("expected unresolved logo to fall back to text mode")
	}
	if logo.Text != "assets/missing.svg" {
		t.Fatalf("expected fallback text %q, got %q", "assets/missing.svg", logo.Text)
	}
}

func TestMarkdownToHTMLRendersTables(t *testing.T) {
	source := "| A | B |\n|---|---|\n| 1 | 2 |"
	rendered := string(markdownToHTML(source))
	if !strings.Contains(rendered, "<table>") {
		t.Fatalf("expected markdown table to render as table, got %q", rendered)
	}
	if !strings.Contains(rendered, "<td>1</td>") {
		t.Fatalf("expected markdown table cell content in rendered html, got %q", rendered)
	}
}

func TestHasChartsDetectsRenderedChartShortcodes(t *testing.T) {
	slides := []render.RenderedSlide{
		{Body: template.HTML(`<div class="shortcode-chart"></div>`)},
	}
	if !hasCharts(slides) {
		t.Fatal("expected hasCharts to detect chart shortcode markup")
	}
	if hasCharts([]render.RenderedSlide{{Body: template.HTML(`<div class="shortcode-mermaid"></div>`)}}) {
		t.Fatal("expected hasCharts to ignore non-chart shortcode markup")
	}
}
