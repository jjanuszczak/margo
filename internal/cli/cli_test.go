package cli

import "testing"

func TestParseBuildLikeArgs(t *testing.T) {
	includeDrafts, openBrowser, err := parseBuildLikeArgs("serve", []string{"--include-drafts", "--no-open"})
	if err != nil {
		t.Fatalf("parseBuildLikeArgs returned error: %v", err)
	}
	if !includeDrafts {
		t.Fatal("expected includeDrafts to be true")
	}
	if openBrowser {
		t.Fatal("expected openBrowser to be false")
	}

	includeDrafts, openBrowser, err = parseBuildLikeArgs("build", []string{"--include-drafts"})
	if err != nil {
		t.Fatalf("parseBuildLikeArgs returned error: %v", err)
	}
	if !includeDrafts {
		t.Fatal("expected build includeDrafts to be true")
	}
	if openBrowser {
		t.Fatal("expected build openBrowser to be false")
	}
}

func TestParseNewSlideArgs(t *testing.T) {
	name, archetype, err := parseNewSlideArgs([]string{"roadmap", "--archetype", "title"})
	if err != nil {
		t.Fatalf("parseNewSlideArgs returned error: %v", err)
	}
	if name != "roadmap" {
		t.Fatalf("expected slide name %q, got %q", "roadmap", name)
	}
	if archetype != "title" {
		t.Fatalf("expected archetype %q, got %q", "title", archetype)
	}
}

func TestParseNewThemeArgs(t *testing.T) {
	name, blank, err := parseNewThemeArgs([]string{"custom", "--blank"})
	if err != nil {
		t.Fatalf("parseNewThemeArgs returned error: %v", err)
	}
	if name != "custom" {
		t.Fatalf("expected theme name %q, got %q", "custom", name)
	}
	if !blank {
		t.Fatal("expected blank theme mode")
	}
}
