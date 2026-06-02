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

	if err := os.MkdirAll(filepath.Join(projectRoot, "assets"), 0o755); err != nil {
		t.Fatalf("create assets dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "assets", "poster.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatalf("write poster asset: %v", err)
	}

	files := map[string]string{
		"columns.html": `<div class="shortcode-columns">{{ .Inner }}</div>`,
		"column.html":  `<div class="shortcode-column">{{ .Inner }}</div>`,
		"video.html":   `<video controls{{ with index .Params "poster" }} poster="{{ assetRef . }}"{{ end }}><source src="{{ assetRef (index .Params "src") }}"></video>`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(themeRoot, "shortcodes", name), []byte(body), 0o644); err != nil {
			t.Fatalf("write theme shortcode %s: %v", name, err)
		}
	}

	rendered, err := Render(`{{< columns >}}{{< column >}}Left{{< /column >}}{{< column >}}{{< video src="https://example.com/demo.mp4" poster="assets/poster.svg" />}}{{< /column >}}{{< /columns >}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	for _, needle := range []string{"shortcode-columns", "shortcode-column", "<source src=\"https://example.com/demo.mp4\">", "poster=\"assets/poster.svg\""} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("expected rendered shortcode output to contain %q, got %q", needle, rendered)
		}
	}
}

func TestRenderFigureShortcodeValidatesParamsAndResolvesLocalAssets(t *testing.T) {
	projectRoot := t.TempDir()
	themeRoot := filepath.Join(projectRoot, "themes", "default")
	slideRoot := filepath.Join(projectRoot, "slides", "02-why")
	if err := os.MkdirAll(filepath.Join(themeRoot, "shortcodes"), 0o755); err != nil {
		t.Fatalf("create theme shortcode dir: %v", err)
	}
	if err := os.MkdirAll(slideRoot, 0o755); err != nil {
		t.Fatalf("create slide dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(slideRoot, "chart.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatalf("write slide asset: %v", err)
	}

	figure := `{{ validateNoInner .Name .Inner }}{{ validateParams .Name .Params "src" "alt" "caption" "class" "width" "position" "fit" "link" "credit" }}{{ $src := assetRefStrict (requiredParam .Params "src") }}{{ $alt := requiredParam .Params "alt" }}{{ $fit := optionalParamOneOf .Params "fit" "contain" "cover" }}{{ $width := optionalParam .Params "width" }}{{ $position := optionalParam .Params "position" }}{{ $caption := optionalParam .Params "caption" }}{{ $credit := optionalParam .Params "credit" }}{{ $extraClass := optionalParam .Params "class" }}<figure class="shortcode-figure{{ if $fit }} shortcode-figure-fit-{{ $fit }}{{ end }}{{ if $extraClass }} {{ $extraClass }}{{ end }}"{{ if $width }} style="--shortcode-figure-width: {{ $width }};"{{ end }}><img class="shortcode-figure-image" src="{{ $src }}" alt="{{ $alt }}"{{ if $position }} style="object-position: {{ $position }};"{{ end }}>{{ if or $caption $credit }}<figcaption>{{ if $caption }}{{ $caption }}{{ end }}{{ if $credit }}<span class="shortcode-figure-credit">{{ $credit }}</span>{{ end }}</figcaption>{{ end }}</figure>`
	if err := os.WriteFile(filepath.Join(themeRoot, "shortcodes", "figure.html"), []byte(figure), 0o644); err != nil {
		t.Fatalf("write figure shortcode: %v", err)
	}

	rendered, err := Render(`{{< figure src="chart.svg" alt="Quarterly chart" caption="Quarterly growth" width="72%" fit="contain" position="top center" credit="Source: Finance" />}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
		Slide: deck.Slide{
			ID:         "02-why",
			BundlePath: slideRoot,
		},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	for _, needle := range []string{
		`class="shortcode-figure shortcode-figure-fit-contain"`,
		`style="--shortcode-figure-width: 72%;"`,
		`src="slides/02-why/chart.svg"`,
		`alt="Quarterly chart"`,
		`style="object-position: top center;"`,
		`Quarterly growth`,
		`Source: Finance`,
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("expected rendered figure to contain %q, got %q", needle, rendered)
		}
	}

	_, err = Render(`{{< figure alt="Missing src" />}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
		Slide:       deck.Slide{ID: "02-why", BundlePath: slideRoot},
	})
	if err == nil || !strings.Contains(err.Error(), `requires parameter "src"`) {
		t.Fatalf("expected missing src error, got %v", err)
	}

	_, err = Render(`{{< figure src="chart.svg" alt="Bad fit" fit="stretch" />}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
		Slide:       deck.Slide{ID: "02-why", BundlePath: slideRoot},
	})
	if err == nil || !strings.Contains(err.Error(), `parameter "fit" must be one of "contain, cover"`) {
		t.Fatalf("expected invalid fit error, got %v", err)
	}

	_, err = Render(`{{< figure src="missing.svg" alt="Missing asset" />}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
		Slide:       deck.Slide{ID: "02-why", BundlePath: slideRoot},
	})
	if err == nil || !strings.Contains(err.Error(), `asset "missing.svg" not found`) {
		t.Fatalf("expected missing asset error, got %v", err)
	}

	_, err = Render(`{{< figure src="chart.svg" alt="Oops" tone="info" />}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
		Slide:       deck.Slide{ID: "02-why", BundlePath: slideRoot},
	})
	if err == nil || !strings.Contains(err.Error(), `does not support parameter "tone"`) {
		t.Fatalf("expected unsupported parameter error, got %v", err)
	}
}

func TestRenderMermaidShortcodeSupportsInnerContentAndValidatesAlign(t *testing.T) {
	projectRoot := t.TempDir()
	themeRoot := filepath.Join(projectRoot, "themes", "default")
	if err := os.MkdirAll(filepath.Join(themeRoot, "shortcodes"), 0o755); err != nil {
		t.Fatalf("create theme shortcode dir: %v", err)
	}

	mermaid := `{{ requiredInner .Name .Inner }}{{ validateParams .Name .Params "caption" "align" "class" }}{{ $align := optionalParamOneOf .Params "align" "left" "center" "right" }}{{ $caption := optionalParam .Params "caption" }}{{ $extraClass := optionalParam .Params "class" }}<figure class="shortcode-mermaid{{ if $align }} shortcode-mermaid-align-{{ $align }}{{ else }} shortcode-mermaid-align-center{{ end }}{{ if $extraClass }} {{ $extraClass }}{{ end }}"><div class="shortcode-mermaid-render" aria-hidden="true"></div><pre class="shortcode-mermaid-definition">{{ .Inner }}</pre>{{ if $caption }}<figcaption class="shortcode-mermaid-caption">{{ $caption }}</figcaption>{{ end }}</figure>`
	if err := os.WriteFile(filepath.Join(themeRoot, "shortcodes", "mermaid.html"), []byte(mermaid), 0o644); err != nil {
		t.Fatalf("write mermaid shortcode: %v", err)
	}

	rendered, err := Render(`{{< mermaid caption="Authoring flow" align="left" >}}
flowchart LR
  A[Markdown] --> B[Build]
{{< /mermaid >}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	for _, needle := range []string{
		`class="shortcode-mermaid shortcode-mermaid-align-left"`,
		`class="shortcode-mermaid-definition"`,
		`flowchart LR`,
		`A[Markdown] --> B[Build]`,
		`Authoring flow`,
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("expected rendered mermaid to contain %q, got %q", needle, rendered)
		}
	}

	_, err = Render(`{{< mermaid />}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
	})
	if err == nil || !strings.Contains(err.Error(), `requires inner content`) {
		t.Fatalf("expected missing inner content error, got %v", err)
	}

	_, err = Render(`{{< mermaid align="wide" >}}flowchart LR
A --> B
{{< /mermaid >}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
	})
	if err == nil || !strings.Contains(err.Error(), `parameter "align" must be one of "left, center, right"`) {
		t.Fatalf("expected invalid align error, got %v", err)
	}
}

func TestRenderGitHubRepoShortcodeValidatesRepoAndRendersCard(t *testing.T) {
	projectRoot := t.TempDir()
	themeRoot := filepath.Join(projectRoot, "themes", "default")
	if err := os.MkdirAll(filepath.Join(themeRoot, "shortcodes"), 0o755); err != nil {
		t.Fatalf("create theme shortcode dir: %v", err)
	}

	repoCard := `{{ validateNoInner .Name .Inner }}{{ validateParams .Name .Params "repo" "caption" "class" }}{{ $repo := mustMatch .Name (requiredParam .Params "repo") "repo" "^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$" }}{{ $caption := optionalParam .Params "caption" }}{{ $extraClass := optionalParam .Params "class" }}{{ $url := printf "https://github.com/%s" $repo }}<figure class="shortcode-github-repo{{ if $extraClass }} {{ $extraClass }}{{ end }}"><a class="shortcode-github-repo-card" href="{{ $url }}"><div class="shortcode-github-repo-label">GitHub Repo</div><div class="shortcode-github-repo-name">{{ $repo }}</div><div class="shortcode-github-repo-url">{{ $url }}</div></a>{{ if $caption }}<figcaption class="shortcode-github-repo-caption">{{ $caption }}</figcaption>{{ end }}</figure>`
	if err := os.WriteFile(filepath.Join(themeRoot, "shortcodes", "github-repo.html"), []byte(repoCard), 0o644); err != nil {
		t.Fatalf("write github repo shortcode: %v", err)
	}

	rendered, err := Render(`{{< github-repo repo="jjanuszczak/margo" caption="Markdown-first deck authoring" />}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	for _, needle := range []string{
		`class="shortcode-github-repo"`,
		`class="shortcode-github-repo-card"`,
		`href="https://github.com/jjanuszczak/margo"`,
		`GitHub Repo`,
		`jjanuszczak/margo`,
		`Markdown-first deck authoring`,
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("expected rendered github repo card to contain %q, got %q", needle, rendered)
		}
	}

	_, err = Render(`{{< github-repo repo="bad repo" />}}`, Context{
		ProjectRoot: projectRoot,
		Theme:       theme.Metadata{RootDir: themeRoot},
	})
	if err == nil || !strings.Contains(err.Error(), `repo must match`) {
		t.Fatalf("expected invalid repo error, got %v", err)
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
