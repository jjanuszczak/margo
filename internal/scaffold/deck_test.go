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
	if info, err := os.Stat(filepath.Join(targetDir, "partials")); err != nil || !info.IsDir() {
		t.Fatalf("expected scaffolded deck partials directory: %v", err)
	}

	expected := map[string][]string{
		"AGENTS.md":                           {"# Margo Deck Agent Guide", "Do not edit generated dist/ output.", "## Where custom components live", "shortcodes/<name>.html", "partials/<name>.html", "assets/css/<name>.css", "themes/<theme-name>/shortcodes/", "Deck-local shortcodes and partials override theme entries"},
		filepath.Join(".agents", "README.md"): {"margo-deck-authoring", "margo-theme-authoring", "margo-error-triage", "margo-brand-theme"},
		filepath.Join(".agents", "skills", "margo-brand-theme", "SKILL.md"): {
			"name: margo-brand-theme",
			"Do not create theme files until the user approves",
			"Audit inherited theme styles",
		},
		filepath.Join(".agents", "skills", "margo-brand-theme", "references", "component-contract.md"): {
			"Brand component contract",
			"decorative_shadows: prohibited",
			"CSS rules that target selectors absent from rendered markup are defects",
		},
		filepath.Join(".agents", "skills", "margo-deck-authoring", "SKILL.md"): {
			"name: margo-deck-authoring",
			"Create, edit, review, build, upgrade, or package a Margo deck.",
			"Write slide bodies in Markdown first.",
			"Use a partial for reusable template fragments called by layouts or shortcodes.",
			"add a focused layout under the active deck-local theme",
			"Raw HTML is a last resort",
			"margo slide insert, move, and delete",
		},
		filepath.Join(".agents", "skills", "margo-deck-authoring", "references", "commands.md"): {
			"margo new slide roadmap --archetype agenda",
			"margo slide move 12-market-size --after 04-product",
			"margo slide delete 04-product",
			"margo theme import ../brand.margot --name client-brand --activate",
			"margo pack .",
		},
		filepath.Join(".agents", "skills", "margo-deck-authoring", "references", "conventions.md"): {
			"## Slide sequencing",
			"Deletion moves the bundle to .margo-trash/",
		},
		filepath.Join(".agents", "skills", "margo-theme-authoring", "SKILL.md"): {
			"name: margo-theme-authoring",
			"Do not use for ordinary slide-content changes",
		},
		filepath.Join(".agents", "skills", "margo-theme-authoring", "references", "theme-contract.md"): {
			"Keep presentation-specific markup, class composition, and styling in theme templates and CSS.",
		},
		filepath.Join(".agents", "skills", "margo-github-pages", "SKILL.md"): {
			"Configure or review GitHub Pages deployment for a Margo deck.",
			"deploys dist/html on v* tags and manual dispatch",
		},
		filepath.Join(".agents", "skills", "margo-error-triage", "SKILL.md"): {
			"name: margo-error-triage",
			"Do not edit source, configuration, themes, generated output, or external systems while diagnosing.",
		},
		filepath.Join(".agents", "skills", "margo-error-triage", "references", "triage.md"): {
			"Deck-owned: Markdown, front matter, margo.yaml, assets, includes, or deck-local shortcodes.",
			"Theme-owned: layouts, partials, theme shortcodes, CSS, theme assets, or print templates.",
			"Margo-owned: reproducible CLI or engine behavior that persists in a clean or committed fixture.",
		},
		filepath.Join(".agents", "skills", "margo-error-triage", "references", "evidence.md"): {
			"interactive HTML, print HTML, and PDF as separate checkpoints",
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

func TestCreateDeckWritesScaffoldManifest(t *testing.T) {
	targetDir := filepath.Join(t.TempDir(), "manifest-deck")
	if err := CreateDeck(DeckOptions{Name: "Manifest Deck", TargetDir: targetDir}); err != nil {
		t.Fatalf("create deck: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(targetDir, ManifestPath))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(raw), "version: \"1\"") || !strings.Contains(string(raw), "AGENTS.md") {
		t.Fatalf("unexpected manifest: %s", raw)
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
	columnShortcode := files[filepath.Join("themes", "default", "shortcodes", "column.html")]
	for _, needle := range []string{"\"width\"", "\"font-size\"", "shortcode-column-sized", "--shortcode-column-width", "--shortcode-column-font-size"} {
		if !strings.Contains(columnShortcode, needle) {
			t.Fatalf("expected column shortcode to support %q, got %q", needle, columnShortcode)
		}
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
	for _, needle := range []string{"display: flex", "--shortcode-column-gap", "flex: 1 1 0", "container-type: inline-size", "--shortcode-column-font-size", "cqw", "flex-direction: column"} {
		if !strings.Contains(styles, needle) {
			t.Fatalf("expected scaffolded theme styles to contain %q", needle)
		}
	}
}
