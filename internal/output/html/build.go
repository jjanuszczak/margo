package html

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/yuin/goldmark"
	"margo/internal/deck"
	"margo/internal/shortcode"
	"margo/internal/theme"
)

const (
	OutputDir  = "dist/html"
	OutputFile = "dist/html/index.html"
)

type pageData struct {
	Deck       deck.DeckMetadata
	PDFEnabled bool
	Theme      theme.Metadata
	Sections   []deck.Section
	Slides     []renderedSlide
}

type renderedSlide struct {
	Index        int
	Title        string
	Layout       string
	Body         template.HTML
	IsDraft      bool
	SectionID    string
	SectionTitle string
}

func Write(projectRoot string, model deck.Model, activeTheme theme.Metadata) error {
	if err := os.MkdirAll(filepath.Join(projectRoot, OutputDir), 0o755); err != nil {
		return fmt.Errorf("create html output directory: %w", err)
	}

	file, err := os.Create(filepath.Join(projectRoot, OutputFile))
	if err != nil {
		return fmt.Errorf("create html output file: %w", err)
	}
	defer file.Close()

	slides, err := renderSlides(projectRoot, model.Slides, model.Sections, activeTheme)
	if err != nil {
		return err
	}

	data := pageData{
		Deck:       model.Config.Deck,
		PDFEnabled: model.Config.Outputs.PDF,
		Theme:      activeTheme,
		Sections:   model.Sections,
		Slides:     slides,
	}

	if activeTheme.DeckLayout == "" {
		tmpl, err := template.New("deck").Funcs(template.FuncMap{
			"markdownToHTML": markdownToHTML,
		}).ParseFiles(activeTheme.DefaultLayout)
		if err != nil {
			return fmt.Errorf("parse html template: %w", err)
		}
		if err := tmpl.ExecuteTemplate(file, filepath.Base(activeTheme.DefaultLayout), data); err != nil {
			return fmt.Errorf("render html output: %w", err)
		}
		return nil
	}

	tmpl, err := template.New("deck").ParseFiles(activeTheme.DeckLayout)
	if err != nil {
		return fmt.Errorf("parse deck layout: %w", err)
	}
	if err := tmpl.ExecuteTemplate(file, filepath.Base(activeTheme.DeckLayout), data); err != nil {
		return fmt.Errorf("render deck layout: %w", err)
	}

	return nil
}

func renderSlides(projectRoot string, slides []deck.Slide, sections []deck.Section, activeTheme theme.Metadata) ([]renderedSlide, error) {
	result := make([]renderedSlide, 0, len(slides))
	for i, slide := range slides {
		expanded, err := shortcode.Render(slide.BodyMarkdown, shortcode.Context{
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
				Index:        i,
				Title:        slide.Title,
				Layout:       resolveLayoutName(slide),
				Body:         body,
				SectionID:    sectionID,
				SectionTitle: sectionTitle,
			})
			continue
		}

		layoutPath := resolveLayoutPath(activeTheme, slide)
		rendered, err := executeSlideLayout(layoutPath, slide, i, body, sectionID, sectionTitle)
		if err != nil {
			return nil, err
		}
		result = append(result, rendered)
	}
	return result, nil
}

func executeSlideLayout(layoutPath string, slide deck.Slide, index int, body template.HTML, sectionID string, sectionTitle string) (renderedSlide, error) {
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
		SectionID    string
		SectionTitle string
	}{
		Index:        index,
		Slide:        slide,
		Body:         body,
		SectionID:    sectionID,
		SectionTitle: sectionTitle,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, filepath.Base(layoutPath), data); err != nil {
		return renderedSlide{}, fmt.Errorf("render slide layout %q: %w", layoutPath, err)
	}

	return renderedSlide{
		Index:        index,
		Title:        slide.Title,
		Layout:       resolveLayoutName(slide),
		Body:         template.HTML(buf.String()),
		IsDraft:      slide.Draft,
		SectionID:    sectionID,
		SectionTitle: sectionTitle,
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
	if err := goldmark.Convert([]byte(source), &buf); err != nil {
		escaped := template.HTMLEscapeString(source)
		return template.HTML("<pre>" + escaped + "</pre>")
	}

	return template.HTML(buf.String())
}
