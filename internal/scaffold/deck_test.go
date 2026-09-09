package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateDeckIncludesAgentGuidanceAndSkills(t *testing.T) {
	targetDir := filepath.Join(t.TempDir(), "agent-ready-deck")
	if err := CreateDeck(DeckOptions{Name: "Agent Ready Deck", TargetDir: targetDir}); err != nil {
		t.Fatalf("create deck: %v", err)
	}

	expected := map[string][]string{
		"AGENTS.md":                           {"# Margo Deck Agent Guide", "Do not edit generated dist/ output."},
		filepath.Join(".agents", "README.md"): {"margo-deck-authoring", "margo-theme-authoring"},
		filepath.Join(".agents", "skills", "margo-deck-authoring", "SKILL.md"): {
			"name: margo-deck-authoring",
			"Create, edit, review, build, or package a Margo deck.",
		},
		filepath.Join(".agents", "skills", "margo-deck-authoring", "references", "commands.md"): {
			"margo new slide roadmap --archetype agenda",
			"margo theme import ../brand.margot --name client-brand --activate",
			"margo pack .",
		},
		filepath.Join(".agents", "skills", "margo-theme-authoring", "SKILL.md"): {
			"name: margo-theme-authoring",
			"Do not use for ordinary slide-content changes",
		},
		filepath.Join(".agents", "skills", "margo-theme-authoring", "references", "theme-contract.md"): {
			"Keep presentation-specific markup, class composition, and styling in theme templates and CSS.",
		},
	}

	for path, fragments := range expected {
		raw, err := os.ReadFile(filepath.Join(targetDir, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, fragment := range fragments {
			if !strings.Contains(string(raw), fragment) {
				t.Errorf("%s does not contain %q", path, fragment)
			}
		}
	}
}

func TestCreateDeckRefusesAgentGuidanceCollision(t *testing.T) {
	targetDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(targetDir, "AGENTS.md"), []byte("existing guidance"), 0o644); err != nil {
		t.Fatalf("write existing guide: %v", err)
	}

	err := CreateDeck(DeckOptions{Name: "Collision Deck", TargetDir: targetDir})
	if err == nil || !strings.Contains(err.Error(), "target file already exists") || !strings.Contains(err.Error(), "AGENTS.md") {
		t.Fatalf("expected AGENTS.md collision, got %v", err)
	}
}

func TestThemeFilesIncludeRefinedThemeStructure(t *testing.T) {
	files := ThemeFiles("default", true)

	if _, ok := files[filepath.Join("themes", "default", "partials", "slide-header.html")]; !ok {
		t.Fatal("expected scaffolded theme to include slide-header partial")
	}
	if _, ok := files[filepath.Join("themes", "default", "partials", "slide-annotations.html")]; !ok {
		t.Fatal("expected scaffolded theme to include slide-annotations partial")
	}
	if _, ok := files[filepath.Join("themes", "default", "shortcodes", "math.html")]; !ok {
		t.Fatal("expected scaffolded theme to include math shortcode")
	}
	if _, ok := files[filepath.Join("themes", "default", "assets", "katex.min.css")]; !ok {
		t.Fatal("expected scaffolded theme to include KaTeX css asset")
	}
	if _, ok := files[filepath.Join("themes", "default", "assets", "katex.min.js")]; !ok {
		t.Fatal("expected scaffolded theme to include KaTeX js asset")
	}
	if _, ok := files[filepath.Join("themes", "default", "assets", "fonts", "KaTeX_Main-Regular.woff2")]; !ok {
		t.Fatal("expected scaffolded theme to include KaTeX font assets")
	}

	deckLayout := files[filepath.Join("themes", "default", "layouts", "deck.html")]
	if !strings.Contains(deckLayout, `href="themes/{{ .Theme.Name }}/assets/katex.min.css"`) {
		t.Fatal("expected scaffolded deck layout to link katex.min.css")
	}
	if !strings.Contains(deckLayout, `href="themes/{{ .Theme.Name }}/assets/theme.css"`) {
		t.Fatal("expected scaffolded deck layout to link theme.css")
	}
	if !strings.Contains(deckLayout, `{{ template "slide-annotations" $slide }}`) {
		t.Fatal("expected scaffolded deck layout to render slide-annotations partial")
	}
	if !strings.Contains(deckLayout, `data-slide-previous`) || !strings.Contains(deckLayout, `data-slide-next`) {
		t.Fatal("expected scaffolded deck layout to include previous and next navigation")
	}
	if !strings.Contains(deckLayout, `data-notes-toggle`) || !strings.Contains(deckLayout, `data-slide-notes`) {
		t.Fatal("expected scaffolded deck layout to support optional slide notes")
	}

	twoColumnLayout := files[filepath.Join("themes", "default", "layouts", "slide-two-column.html")]
	if !strings.Contains(twoColumnLayout, `splitBodyColumns .ExpandedMarkdown .Body`) {
		t.Fatal("expected scaffolded two-column layout to use splitBodyColumns helper")
	}

	mediaLayout := files[filepath.Join("themes", "default", "layouts", "slide-media-right.html")]
	if !strings.Contains(mediaLayout, `withoutLeadingImage .Body`) || !strings.Contains(mediaLayout, `leadingImage .Body`) {
		t.Fatal("expected scaffolded media layout to use generic image helpers")
	}

	styles := files[filepath.Join("themes", "default", "assets", "theme.css")]
	if strings.Contains(styles, "Theme-local styles can be added here later.") {
		t.Fatal("expected scaffolded theme.css to contain real default theme styles")
	}
}
