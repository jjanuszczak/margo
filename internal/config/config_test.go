package config

import "testing"

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
