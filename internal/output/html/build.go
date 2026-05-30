package html

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	rendererhtml "github.com/yuin/goldmark/renderer/html"
	"margo/internal/deck"
	"margo/internal/shortcode"
	"margo/internal/theme"
)

const (
	OutputDir  = "dist/html"
	OutputFile = "dist/html/index.html"
)

type pageData struct {
	Deck         deck.DeckMetadata
	PDFEnabled   bool
	Theme        theme.Metadata
	ThemeOptions map[string]any
	Sections     []deck.Section
	Slides       []renderedSlide
}

type renderedSlide struct {
	Index              int
	Title              string
	Layout             string
	Body               template.HTML
	IsDraft            bool
	SectionID          string
	SectionTitle       string
	HideLogo           bool
	HideFooter         bool
	ResolvedFooterText string
	StyleAttr          template.CSS
	ImageHintClass     string
	ImageCaption       string
}

const columnBreakMarker = "<!-- column-break -->"

func Write(projectRoot string, model deck.Model, activeTheme theme.Metadata) error {
	if err := os.MkdirAll(filepath.Join(projectRoot, OutputDir), 0o755); err != nil {
		return fmt.Errorf("create html output directory: %w", err)
	}
	if err := stageAssets(projectRoot, model.Slides); err != nil {
		return fmt.Errorf("stage html assets: %w", err)
	}

	file, err := os.Create(filepath.Join(projectRoot, OutputFile))
	if err != nil {
		return fmt.Errorf("create html output file: %w", err)
	}
	defer file.Close()

	slides, err := renderSlides(projectRoot, model.Config.Deck, model.Slides, model.Sections, activeTheme)
	if err != nil {
		return err
	}

	data := pageData{
		Deck:         model.Config.Deck,
		PDFEnabled:   model.Config.Outputs.PDF,
		Theme:        activeTheme,
		ThemeOptions: model.Config.Theme.Options,
		Sections:     model.Sections,
		Slides:       slides,
	}

	if activeTheme.DeckLayout == "" {
		tmpl, err := template.New("deck").Funcs(template.FuncMap{
			"markdownToHTML": markdownToHTML,
			"themeOption":    themeOption,
		}).ParseFiles(activeTheme.DefaultLayout)
		if err != nil {
			return fmt.Errorf("parse html template: %w", err)
		}
		if err := tmpl.ExecuteTemplate(file, filepath.Base(activeTheme.DefaultLayout), data); err != nil {
			return fmt.Errorf("render html output: %w", err)
		}
		return nil
	}

	tmpl, err := template.New("deck").Funcs(template.FuncMap{
		"themeOption": themeOption,
	}).ParseFiles(activeTheme.DeckLayout)
	if err != nil {
		return fmt.Errorf("parse deck layout: %w", err)
	}
	if err := tmpl.ExecuteTemplate(file, filepath.Base(activeTheme.DeckLayout), data); err != nil {
		return fmt.Errorf("render deck layout: %w", err)
	}

	return nil
}

func renderSlides(projectRoot string, deckMeta deck.DeckMetadata, slides []deck.Slide, sections []deck.Section, activeTheme theme.Metadata) ([]renderedSlide, error) {
	result := make([]renderedSlide, 0, len(slides))
	for i, slide := range slides {
		bodySource := rewriteMarkdownAssetRefs(projectRoot, slide, slide.BodyMarkdown)
		expanded, err := shortcode.Render(bodySource, shortcode.Context{
			ProjectRoot: projectRoot,
			Theme:       activeTheme,
			Slide:       slide,
		})
		if err != nil {
			return nil, fmt.Errorf("expand shortcodes for slide %q: %w", slide.ID, err)
		}
		body := markdownToHTML(expanded)
		section, hasSection := deck.FindSection(sections, slide.Section)
		sectionID := ""
		sectionTitle := ""
		if hasSection {
			sectionID = section.ID
			sectionTitle = section.Title
		}
		if activeTheme.DeckLayout == "" {
			result = append(result, renderedSlide{
				Index:              i,
				Title:              slide.Title,
				Layout:             resolveLayoutName(slide),
				Body:               body,
				SectionID:          sectionID,
				SectionTitle:       sectionTitle,
				HideLogo:           slide.HideLogo,
				HideFooter:         slide.HideFooter,
				ResolvedFooterText: resolveFooterText(slide, deckMeta.Footer),
				StyleAttr:          resolveSlideStyle(projectRoot, slide),
				ImageHintClass:     resolveImageHintClass(slide.ImageHints),
				ImageCaption:       resolveImageCaption(slide.ImageHints),
			})
			continue
		}

		layoutPath := resolveLayoutPath(activeTheme, slide)
		rendered, err := executeSlideLayout(projectRoot, layoutPath, slide, i, body, sectionID, sectionTitle, deckMeta.Footer)
		if err != nil {
			return nil, err
		}
		result = append(result, rendered)
	}
	return result, nil
}

func executeSlideLayout(projectRoot string, layoutPath string, slide deck.Slide, index int, body template.HTML, sectionID string, sectionTitle string, deckFooter string) (renderedSlide, error) {
	tmpl, err := template.New("slide").Funcs(template.FuncMap{
		"markdownToHTML": markdownToHTML,
	}).ParseFiles(layoutPath)
	if err != nil {
		return renderedSlide{}, fmt.Errorf("parse slide layout %q: %w", layoutPath, err)
	}

	data := struct {
		Index        int
		Slide        deck.Slide
		Body         template.HTML
		BodyColumns  []template.HTML
		SectionID    string
		SectionTitle string
	}{
		Index:        index,
		Slide:        slide,
		Body:         body,
		BodyColumns:  renderBodyColumns(slide, body),
		SectionID:    sectionID,
		SectionTitle: sectionTitle,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, filepath.Base(layoutPath), data); err != nil {
		return renderedSlide{}, fmt.Errorf("render slide layout %q: %w", layoutPath, err)
	}

	return renderedSlide{
		Index:              index,
		Title:              slide.Title,
		Layout:             resolveLayoutName(slide),
		Body:               template.HTML(buf.String()),
		IsDraft:            slide.Draft,
		SectionID:          sectionID,
		SectionTitle:       sectionTitle,
		HideLogo:           slide.HideLogo,
		HideFooter:         slide.HideFooter,
		ResolvedFooterText: resolveFooterText(slide, deckFooter),
		StyleAttr:          resolveSlideStyle(projectRoot, slide),
		ImageHintClass:     resolveImageHintClass(slide.ImageHints),
		ImageCaption:       resolveImageCaption(slide.ImageHints),
	}, nil
}

func resolveLayoutPath(activeTheme theme.Metadata, slide deck.Slide) string {
	layoutName := resolveLayoutName(slide)
	if layoutName != "" {
		if path, ok := activeTheme.SlideLayouts[layoutName]; ok {
			return path
		}
	}
	return activeTheme.DefaultLayout
}

func resolveLayoutName(slide deck.Slide) string {
	if slide.Layout != "" {
		return slide.Layout
	}
	if slide.Type != "" {
		return slide.Type
	}
	return "default"
}

func markdownToHTML(source string) template.HTML {
	if source == "" {
		return ""
	}

	var buf bytes.Buffer
	md := goldmark.New(
		goldmark.WithRendererOptions(
			rendererhtml.WithUnsafe(),
		),
	)
	if err := md.Convert([]byte(source), &buf); err != nil {
		escaped := template.HTMLEscapeString(source)
		return template.HTML("<pre>" + escaped + "</pre>")
	}

	return template.HTML(buf.String())
}

func renderBodyColumns(slide deck.Slide, body template.HTML) []template.HTML {
	if resolveLayoutName(slide) != "two-column" {
		return nil
	}

	parts := strings.Split(slide.BodyMarkdown, columnBreakMarker)
	if len(parts) < 2 {
		return []template.HTML{body}
	}

	columns := make([]template.HTML, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			columns = append(columns, "")
			continue
		}
		columns = append(columns, markdownToHTML(trimmed))
	}
	return columns
}

func resolveFooterText(slide deck.Slide, deckFooter string) string {
	if strings.TrimSpace(slide.FooterText) != "" {
		return slide.FooterText
	}
	return deckFooter
}

func resolveBackgroundStyle(projectRoot string, slide deck.Slide) string {
	parts := make([]string, 0, 3)
	if strings.TrimSpace(slide.Background.Color) != "" {
		parts = append(parts, "background-color: "+slide.Background.Color)
	}
	if strings.TrimSpace(slide.Background.Image) != "" {
		imageRef := slide.Background.Image
		if projectRoot != "" {
			imageRef = resolveAssetReference(projectRoot, slide, imageRef)
		}
		parts = append(parts, "background-image: url('"+template.HTMLEscapeString(imageRef)+"')")
		parts = append(parts, "background-size: cover")
		parts = append(parts, "background-position: center")
	}
	if strings.TrimSpace(slide.Background.Overlay) != "" {
		parts = append(parts, "--margo-background-overlay: "+slide.Background.Overlay)
	}
	if slide.Background.Opacity > 0 {
		parts = append(parts, fmt.Sprintf("--margo-background-opacity: %.2f", slide.Background.Opacity))
	}
	return strings.Join(parts, "; ")
}

func resolveSlideStyle(projectRoot string, slide deck.Slide) template.CSS {
	parts := make([]string, 0, 2)
	if background := strings.TrimSpace(resolveBackgroundStyle(projectRoot, slide)); background != "" {
		parts = append(parts, background)
	}
	if imageHints := strings.TrimSpace(resolveImageHintStyle(slide.ImageHints)); imageHints != "" {
		parts = append(parts, imageHints)
	}
	return template.CSS(strings.Join(parts, "; "))
}

func resolveImageHintClass(hints map[string]any) string {
	fit := strings.TrimSpace(imageHintString(hints, "fit"))
	switch fit {
	case "contain", "cover", "inline":
		return "image-fit-" + fit
	default:
		return ""
	}
}

func resolveImageHintStyle(hints map[string]any) string {
	position := strings.TrimSpace(imageHintString(hints, "position"))
	if position == "" {
		return ""
	}
	return "--margo-image-position: " + position
}

func resolveImageCaption(hints map[string]any) string {
	return strings.TrimSpace(imageHintString(hints, "caption"))
}

func imageHintString(hints map[string]any, key string) string {
	if hints == nil {
		return ""
	}
	raw, ok := hints[key]
	if !ok {
		return ""
	}
	if value, ok := raw.(string); ok {
		return value
	}
	return ""
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
