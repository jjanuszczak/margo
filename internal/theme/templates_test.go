package theme

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTemplateWithPartialsPrefersDeckPartials(t *testing.T) {
	projectRoot := t.TempDir()
	themeRoot := filepath.Join(projectRoot, ThemesDirName, "custom")
	if err := os.MkdirAll(filepath.Join(themeRoot, "partials"), 0o755); err != nil {
		t.Fatalf("mkdir theme partials: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(themeRoot, "layouts"), 0o755); err != nil {
		t.Fatalf("mkdir theme layouts: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "partials"), 0o755); err != nil {
		t.Fatalf("mkdir deck partials: %v", err)
	}

	layoutPath := filepath.Join(themeRoot, "layouts", "deck.html")
	if err := os.WriteFile(layoutPath, []byte(`{{ template "brand-lockup" . }}`), 0o644); err != nil {
		t.Fatalf("write layout: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeRoot, "partials", "brand-lockup.html"), []byte(`theme`), 0o644); err != nil {
		t.Fatalf("write theme partial: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "partials", "brand-lockup.html"), []byte(`deck`), 0o644); err != nil {
		t.Fatalf("write deck partial: %v", err)
	}

	tmpl, err := ParseTemplateWithPartials(projectRoot, Metadata{
		RootDir:    themeRoot,
		Partials:   map[string]string{"brand-lockup": filepath.Join(themeRoot, "partials", "brand-lockup.html")},
		DeckLayout: layoutPath,
	}, layoutPath, template.FuncMap{})
	if err != nil {
		t.Fatalf("ParseTemplateWithPartials returned error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, filepath.Base(layoutPath), nil); err != nil {
		t.Fatalf("ExecuteTemplate returned error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "deck" {
		t.Fatalf("expected deck partial override, got %q", buf.String())
	}
}

func TestDiscoverPartialsIgnoresNonHTMLFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "brand.html"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write html partial: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write non-html file: %v", err)
	}

	partials, err := discoverPartials(root)
	if err != nil {
		t.Fatalf("discoverPartials returned error: %v", err)
	}
	if got, want := len(partials), 1; got != want {
		t.Fatalf("expected %d partial, got %d", want, got)
	}
	if _, ok := partials["brand"]; !ok {
		t.Fatalf("expected brand partial to be discovered, got %#v", partials)
	}
}
