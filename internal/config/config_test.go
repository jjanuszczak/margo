package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSnippets(t *testing.T) {
	raw := RawConfig{
		Path: "margo.yaml",
		Bytes: []byte(`version: 1

deck:
  title: Sample

theme:
  name: default

snippets:
  head: |
    <meta name="sample" content="x">
  body_end: |
    <script>window.sample = true;</script>
`),
	}

	parsed, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if parsed.Config.Snippets.Head != "<meta name=\"sample\" content=\"x\">\n" {
		t.Fatalf("unexpected head snippet %q", parsed.Config.Snippets.Head)
	}
	if parsed.Config.Snippets.BodyEnd != "<script>window.sample = true;</script>\n" {
		t.Fatalf("unexpected body_end snippet %q", parsed.Config.Snippets.BodyEnd)
	}
}

func TestParsePresentationNavigationNotes(t *testing.T) {
	raw := RawConfig{
		Path: "margo.yaml",
		Bytes: []byte(`version: 1

deck:
  title: Sample

theme:
  name: default

presentation:
  navigation:
    notes: true
`),
	}

	parsed, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !parsed.Config.Presentation.Navigation.Notes {
		t.Fatal("expected presentation.navigation.notes to be enabled")
	}
}

func TestParsePNGOutput(t *testing.T) {
	raw := RawConfig{
		Path: "margo.yaml",
		Bytes: []byte(`version: 1

deck:
  title: Sample

theme:
  name: default

outputs:
  png: true
`),
	}

	parsed, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !parsed.Config.Outputs.PNG {
		t.Fatal("expected png output to be enabled")
	}
	if parsed.Config.Outputs.HTML {
		t.Fatal("png-only configuration should not implicitly enable html output")
	}
}

func TestSetThemeNamePreservesThemeOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "margo.yaml")
	contents := "version: 1\ndeck:\n  title: Test\ntheme:\n  name: default\n  mode: dark\n  logo: assets/logo.svg\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetThemeName(path, "client-brand"); err != nil {
		t.Fatalf("SetThemeName returned error: %v", err)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"name: client-brand", "mode: dark", "logo: assets/logo.svg"} {
		if !strings.Contains(string(updated), expected) {
			t.Fatalf("expected updated config to contain %q, got %s", expected, updated)
		}
	}
}
