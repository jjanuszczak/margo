package shortcode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"margo/internal/deck"
	"margo/internal/theme"
)

func TestRenderPrefersDeckShortcodesAndSupportsBlocks(t *testing.T) {
	projectRoot := t.TempDir()
	themeRoot := filepath.Join(projectRoot, "themes", "default")
	if err := os.MkdirAll(filepath.Join(projectRoot, "shortcodes"), 0o755); err != nil {
		t.Fatalf("create deck shortcode dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(themeRoot, "shortcodes"), 0o755); err != nil {
		t.Fatalf("create theme shortcode dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(themeRoot, "shortcodes", "callout.html"), []byte("theme {{ .Inner }}"), 0o644); err != nil {
		t.Fatalf("write theme shortcode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "shortcodes", "callout.html"), []byte("deck {{ index .Params \"tone\" }} {{ .Inner }}"), 0o644); err != nil {
		t.Fatalf("write deck shortcode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "shortcodes", "badge.html"), []byte("[{{ index .Params \"label\" }}]"), 0o644); err != nil {
		t.Fatalf("write badge shortcode: %v", err)
	}

	rendered, err := Render(`{{< callout tone="warning" >}}Hello {{< badge label="v1" />}}{{< /callout >}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
		Slide:       deck.Slide{ID: "01-title"},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if got := strings.TrimSpace(rendered); got != "deck warning Hello [v1]" {
		t.Fatalf("unexpected rendered shortcode output: %q", got)
	}
}

func TestRenderFallsBackToThemeShortcodes(t *testing.T) {
	projectRoot := t.TempDir()
	themeRoot := filepath.Join(projectRoot, "themes", "default")
	if err := os.MkdirAll(filepath.Join(themeRoot, "shortcodes"), 0o755); err != nil {
		t.Fatalf("create theme shortcode dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeRoot, "shortcodes", "badge.html"), []byte("<span>{{ index .Params \"label\" }}</span>"), 0o644); err != nil {
		t.Fatalf("write theme shortcode: %v", err)
	}

	rendered, err := Render(`{{< badge label="theme" />}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if strings.TrimSpace(rendered) != "<span>theme</span>" {
		t.Fatalf("unexpected rendered shortcode output: %q", rendered)
	}
}

func TestRenderThemeShortcodeSetSupportsVideoAndNestedColumns(t *testing.T) {
	projectRoot := t.TempDir()
	themeRoot := filepath.Join(projectRoot, "themes", "default")
	if err := os.MkdirAll(filepath.Join(themeRoot, "shortcodes"), 0o755); err != nil {
		t.Fatalf("create theme shortcode dir: %v", err)
	}

	files := map[string]string{
		"columns.html": `<div class="shortcode-columns">{{ .Inner }}</div>`,
		"column.html":  `<div class="shortcode-column">{{ .Inner }}</div>`,
		"video.html":   `<video controls><source src="{{ index .Params "src" }}"></video>`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(themeRoot, "shortcodes", name), []byte(body), 0o644); err != nil {
			t.Fatalf("write theme shortcode %s: %v", name, err)
		}
	}

	rendered, err := Render(`{{< columns >}}{{< column >}}Left{{< /column >}}{{< column >}}{{< video src="demo.mp4" />}}{{< /column >}}{{< /columns >}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	for _, needle := range []string{"shortcode-columns", "shortcode-column", "<source src=\"demo.mp4\">"} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("expected rendered shortcode output to contain %q, got %q", needle, rendered)
		}
	}
}

func TestRenderRejectsUnknownOrMalformedShortcodes(t *testing.T) {
	projectRoot := t.TempDir()

	_, err := Render(`{{< missing />}}`, Context{ProjectRoot: projectRoot})
	if err == nil || !strings.Contains(err.Error(), `unknown shortcode "missing"`) {
		t.Fatalf("expected unknown shortcode error, got %v", err)
	}

	if err := os.MkdirAll(filepath.Join(projectRoot, "shortcodes"), 0o755); err != nil {
		t.Fatalf("create deck shortcode dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "shortcodes", "callout.html"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write deck shortcode: %v", err)
	}

	_, err = Render(`{{< callout tone=warning />}}`, Context{ProjectRoot: projectRoot})
	if err == nil || !strings.Contains(err.Error(), "unsupported shortcode arguments") {
		t.Fatalf("expected malformed shortcode error, got %v", err)
	}

	_, err = Render(`{{< callout >}}body`, Context{ProjectRoot: projectRoot})
	if err == nil || !strings.Contains(err.Error(), `missing closing shortcode tag for "callout"`) {
		t.Fatalf("expected missing closing tag error, got %v", err)
	}
}
