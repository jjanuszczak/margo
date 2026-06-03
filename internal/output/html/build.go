package html

import (
	"fmt"
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
	OutputDir  = "dist/html"
	OutputFile = "dist/html/index.html"
)

type pageData struct {
	Deck         deck.DeckMetadata
	DeckLogo     render.RenderedLogo
	PDFEnabled   bool
	HasCharts    bool
	Theme        theme.Metadata
	ThemeOptions map[string]any
	Snippets     renderedSnippets
	Sections     []deck.Section
	Slides       []render.RenderedSlide
}

type renderedSnippets struct {
	Head    template.HTML
	BodyEnd template.HTML
}

func Write(projectRoot string, model deck.Model, activeTheme theme.Metadata) (diagnostics.Report, error) {
	if err := os.MkdirAll(filepath.Join(projectRoot, OutputDir), 0o755); err != nil {
		return diagnostics.Report{}, fmt.Errorf("create html output directory: %w", err)
	}
	if err := stageAssets(projectRoot, model.Slides); err != nil {
		return diagnostics.Report{}, fmt.Errorf("stage html assets: %w", err)
	}

	file, err := os.Create(filepath.Join(projectRoot, OutputFile))
	if err != nil {
		return diagnostics.Report{}, fmt.Errorf("create html output file: %w", err)
	}
	defer file.Close()

	slides, report, err := render.RenderSlides(projectRoot, model.Config.Deck, model.Slides, model.Sections, activeTheme)
	if err != nil {
		return diagnostics.Report{}, err
	}
	if err := render.StageThemeAssets(projectRoot, OutputDir, activeTheme); err != nil {
		return diagnostics.Report{}, fmt.Errorf("stage theme assets: %w", err)
	}
	logo, logoReport := resolveDeckLogo(projectRoot, model.Config.Deck.Logo)
	report.Items = append(report.Items, logoReport.Items...)

	data := pageData{
		Deck:         model.Config.Deck,
		DeckLogo:     logo,
		PDFEnabled:   model.Config.Outputs.PDF,
		HasCharts:    hasCharts(slides),
		Theme:        activeTheme,
		ThemeOptions: model.Config.Theme.Options,
		Snippets:     resolveSnippets(model.Config.Snippets),
		Sections:     model.Sections,
		Slides:       slides,
	}

	if activeTheme.DeckLayout == "" {
		tmpl, err := theme.ParseTemplateWithPartials(projectRoot, activeTheme, activeTheme.DefaultLayout, template.FuncMap{
			"markdownToHTML": render.MarkdownToHTML,
			"themeOption":    themeOption,
		})
		if err != nil {
			return diagnostics.Report{}, fmt.Errorf("parse html template: %w", err)
		}
		if err := tmpl.ExecuteTemplate(file, filepath.Base(activeTheme.DefaultLayout), data); err != nil {
			return diagnostics.Report{}, fmt.Errorf("render html output: %w", err)
		}
		return report, nil
	}

	tmpl, err := theme.ParseTemplateWithPartials(projectRoot, activeTheme, activeTheme.DeckLayout, template.FuncMap{
		"themeOption": themeOption,
	})
	if err != nil {
		return diagnostics.Report{}, fmt.Errorf("parse deck layout: %w", err)
	}
	if err := tmpl.ExecuteTemplate(file, filepath.Base(activeTheme.DeckLayout), data); err != nil {
		return diagnostics.Report{}, fmt.Errorf("render deck layout: %w", err)
	}

	return report, nil
}

func resolveSnippets(value deck.SnippetSettings) renderedSnippets {
	return renderedSnippets{
		Head:    template.HTML(strings.TrimSpace(value.Head)),
		BodyEnd: template.HTML(strings.TrimSpace(value.BodyEnd)),
	}
}

func themeOption(options map[string]any, key string, fallback string) string {
	if options == nil {
		return fallback
	}
	raw, ok := options[key]
	if !ok {
		return fallback
	}
	if value, ok := raw.(string); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func hasCharts(slides []render.RenderedSlide) bool {
	for _, slide := range slides {
		if strings.Contains(string(slide.Body), "shortcode-chart") {
			return true
		}
	}
	return false
}
