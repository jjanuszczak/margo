package printhtml

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"margo/internal/deck"
	"margo/internal/diagnostics"
	"margo/internal/output/render"
	"margo/internal/theme"
)

const (
	OutputDir  = "dist/pdf"
	OutputFile = "dist/pdf/print.html"
)

type pageData struct {
	Deck         deck.DeckMetadata
	DeckLogo     render.RenderedLogo
	Snippets     renderedSnippets
	PDFEnabled   bool
	HasCharts    bool
	Theme        theme.Metadata
	ThemeOptions map[string]any
	Slides       []render.RenderedSlide
}

type renderedSnippets struct {
	Head    template.HTML
	BodyEnd template.HTML
}

const defaultTemplate = `<!doctype html>
<html lang="en" class="margo-print {{ colorModeClass .ThemeOptions }}">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{{ .Deck.Title }}</title>
    <style>
      :root {
        color-scheme: {{ themeOption .ThemeOptions "color_mode" "light" }};
        --margo-print-accent: {{ themeOption .ThemeOptions "accent_color" "#8f6f33" }};
        --margo-print-page-width: 13.333in;
        --margo-print-page-height: 7.5in;
        --margo-print-bg: {{ pageBackground .ThemeOptions }};
        --margo-print-fg: {{ pageForeground .ThemeOptions }};
        --margo-print-muted: {{ pageMuted .ThemeOptions }};
      }
      * { box-sizing: border-box; }
      html, body { margin: 0; padding: 0; background: var(--margo-print-bg); color: var(--margo-print-fg); }
      body {
        font-family: {{ bodyFont .ThemeOptions }};
        line-height: 1.45;
        -webkit-print-color-adjust: exact;
        print-color-adjust: exact;
      }
      .print-deck {
        display: flex;
        flex-direction: column;
        gap: 0;
      }
      .print-slide {
        position: relative;
        width: var(--margo-print-page-width);
        min-height: var(--margo-print-page-height);
        margin: 0 auto;
        padding: 0;
        background: var(--margo-print-bg);
        color: var(--margo-print-fg);
        page-break-after: always;
        break-after: page;
        overflow: hidden;
      }
      .print-slide-frame {
        width: 100%;
        min-height: var(--margo-print-page-height);
        padding: 0.6in 0.7in;
      }
      .print-slide-frame > *:first-child { margin-top: 0; }
      .print-slide-frame > *:last-child { margin-bottom: 0; }
      .print-slide-body img,
      .print-slide-body video {
        max-width: 100%;
        height: auto;
      }
      .print-slide-body a { color: var(--margo-print-accent); }
      .print-slide-body blockquote {
        margin: 1.25rem 0;
        padding-left: 1rem;
        border-left: 4px solid var(--margo-print-accent);
      }
      .print-slide-body code,
      .print-slide-body pre {
        font-family: "SFMono-Regular", "Menlo", "Consolas", monospace;
      }
      .print-slide-body table {
        width: 100%;
        border-collapse: collapse;
      }
      .print-slide-body th,
      .print-slide-body td {
        padding: 0.35rem 0.5rem;
        border: 1px solid rgba(127, 127, 127, 0.35);
        text-align: left;
      }
      .print-meta {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 1rem;
        margin-bottom: 0.35in;
        color: var(--margo-print-muted);
        font-size: 0.82rem;
        text-transform: uppercase;
        letter-spacing: 0.08em;
      }
      .print-logo {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        max-width: 50%;
        font-weight: 700;
      }
      .print-logo img {
        display: block;
        max-height: 0.45in;
        max-width: 100%;
      }
      .print-footer {
        margin-top: 0.3in;
        display: flex;
        justify-content: space-between;
        gap: 1rem;
        color: var(--margo-print-muted);
        font-size: 0.78rem;
      }
      .print-title {
        font-family: {{ headingFont .ThemeOptions }};
      }
      .typography-executive .print-title {
        letter-spacing: -0.03em;
      }
      @page {
        size: 13.333in 7.5in;
        margin: 0;
      }
      @media print {
        .print-slide {
          margin: 0;
        }
      }
    </style>
  </head>
  <body class="{{ typographyClass .ThemeOptions }}">
    <main class="print-deck">
      {{ range .Slides }}
      <section class="print-slide print-slide-layout-{{ .Layout }}" data-slide-index="{{ .Index }}">
        <div class="print-slide-frame"{{ if .StyleAttr }} style="{{ .StyleAttr }}"{{ end }}>
          <div class="print-meta">
            <div class="print-logo">
              {{ if and $.DeckLogo.IsImage (not .HideLogo) }}
              <img src="{{ $.DeckLogo.ImageSrc }}" alt="{{ $.Deck.Title }}">
              {{ else if and $.DeckLogo.Text (not .HideLogo) }}
              <span>{{ $.DeckLogo.Text }}</span>
              {{ end }}
            </div>
            <div>{{ if .SectionTitle }}{{ .SectionTitle }}{{ else }}{{ $.Deck.Title }}{{ end }}</div>
          </div>
          <div class="print-slide-body">{{ .Body }}</div>
          {{ if or (not .HideFooter) .ResolvedFooterText }}
          <div class="print-footer">
            <div>{{ if not .HideFooter }}{{ .ResolvedFooterText }}{{ end }}</div>
            <div>{{ .Title }}</div>
          </div>
          {{ end }}
        </div>
      </section>
      {{ end }}
    </main>
  </body>
</html>
`

func Write(projectRoot string, model deck.Model, activeTheme theme.Metadata) (diagnostics.Report, error) {
	if err := os.MkdirAll(filepath.Join(projectRoot, OutputDir), 0o755); err != nil {
		return diagnostics.Report{}, err
	}
	if err := render.StageAssets(projectRoot, OutputDir, model.Slides); err != nil {
		return diagnostics.Report{}, err
	}

	file, err := os.Create(filepath.Join(projectRoot, OutputFile))
	if err != nil {
		return diagnostics.Report{}, err
	}
	defer file.Close()

	slides, report, err := render.RenderSlides(projectRoot, model.Config.Deck, model.Slides, model.Sections, activeTheme)
	if err != nil {
		return diagnostics.Report{}, err
	}
	if err := render.StageThemeAssets(projectRoot, OutputDir, activeTheme); err != nil {
		return diagnostics.Report{}, err
	}
	logo, logoReport := render.ResolveDeckLogo(projectRoot, model.Config.Deck.Logo)
	report.Items = append(report.Items, logoReport.Items...)

	data := pageData{
		Deck:         model.Config.Deck,
		DeckLogo:     logo,
		Snippets:     resolveSnippets(model.Config.Snippets),
		PDFEnabled:   false,
		HasCharts:    hasCharts(slides),
		Theme:        activeTheme,
		ThemeOptions: model.Config.Theme.Options,
		Slides:       slides,
	}

	templateSource := defaultTemplate
	templateName := "print"
	if strings.TrimSpace(activeTheme.PrintDeckLayout) != "" {
		tmpl, err := theme.ParseTemplateWithPartials(projectRoot, activeTheme, activeTheme.PrintDeckLayout, template.FuncMap{
			"themeOption":     themeOption,
			"colorModeClass":  colorModeClass,
			"typographyClass": typographyClass,
			"bodyFont":        bodyFont,
			"headingFont":     headingFont,
			"pageBackground":  pageBackground,
			"pageForeground":  pageForeground,
			"pageMuted":       pageMuted,
		})
		if err != nil {
			return diagnostics.Report{}, err
		}
		if err := tmpl.ExecuteTemplate(file, filepath.Base(activeTheme.PrintDeckLayout), data); err != nil {
			return diagnostics.Report{}, err
		}
		return report, nil
	}

	tmpl, err := template.New(templateName).Funcs(template.FuncMap{
		"themeOption":     themeOption,
		"colorModeClass":  colorModeClass,
		"typographyClass": typographyClass,
		"bodyFont":        bodyFont,
		"headingFont":     headingFont,
		"pageBackground":  pageBackground,
		"pageForeground":  pageForeground,
		"pageMuted":       pageMuted,
	}).Parse(templateSource)
	if err != nil {
		return diagnostics.Report{}, err
	}
	if err := tmpl.Execute(file, data); err != nil {
		return diagnostics.Report{}, err
	}
	return report, nil
}

func resolveSnippets(value deck.SnippetSettings) renderedSnippets {
	return renderedSnippets{
		Head:    template.HTML(strings.TrimSpace(value.Head)),
		BodyEnd: template.HTML(strings.TrimSpace(value.BodyEnd)),
	}
}

func hasCharts(slides []render.RenderedSlide) bool {
	for _, slide := range slides {
		if strings.Contains(string(slide.Body), "shortcode-chart") {
			return true
		}
	}
	return false
}

func themeOption(options map[string]any, key string, fallback string) string {
	if options == nil {
		return fallback
	}
	raw, ok := options[key]
	if !ok {
		return fallback
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func colorModeClass(options map[string]any) string {
	return "color-mode-" + themeOption(options, "color_mode", "light")
}

func typographyClass(options map[string]any) string {
	return "typography-" + themeOption(options, "typography", "editorial")
}

func bodyFont(options map[string]any) string {
	switch themeOption(options, "typography", "editorial") {
	case "executive":
		return `"Avenir Next", "Helvetica Neue", sans-serif`
	default:
		return `Georgia, "Times New Roman", serif`
	}
}

func headingFont(options map[string]any) string {
	switch themeOption(options, "typography", "editorial") {
	case "executive":
		return `"Avenir Next", "Helvetica Neue", sans-serif`
	default:
		return `Georgia, "Times New Roman", serif`
	}
}

func pageBackground(options map[string]any) string {
	if themeOption(options, "color_mode", "light") == "dark" {
		return "#171310"
	}
	return "#f7f3ec"
}

func pageForeground(options map[string]any) string {
	if themeOption(options, "color_mode", "light") == "dark" {
		return "#f5efe6"
	}
	return "#201b16"
}

func pageMuted(options map[string]any) string {
	if themeOption(options, "color_mode", "light") == "dark" {
		return "#c8bcad"
	}
	return "#6f6357"
}
