package scaffold

import (
	"path/filepath"
	"strings"
	"testing"
)

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
