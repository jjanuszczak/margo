package render

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	rendererhtml "github.com/yuin/goldmark/renderer/html"
	"margo/internal/deck"
	"margo/internal/diagnostics"
	"margo/internal/shortcode"
	"margo/internal/theme"
)

type RenderedLogo struct {
	Text     string
	ImageSrc string
	IsImage  bool
}

type RenderedSlide struct {
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

func RenderSlides(projectRoot string, deckMeta deck.DeckMetadata, slides []deck.Slide, sections []deck.Section, activeTheme theme.Metadata) ([]RenderedSlide, diagnostics.Report, error) {
	result := make([]RenderedSlide, 0, len(slides))
	var report diagnostics.Report
	for i, slide := range slides {
		bodySource, bodyReport := RewriteMarkdownAssetRefs(projectRoot, slide, slide.BodyMarkdown)
		report.Items = append(report.Items, bodyReport.Items...)
		expanded, err := shortcode.Render(bodySource, shortcode.Context{
			ProjectRoot: projectRoot,
			Theme:       activeTheme,
			Slide:       slide,
		})
		if err != nil {
			return nil, diagnostics.Report{}, fmt.Errorf("expand shortcodes for slide %q: %w", slide.ID, err)
		}
		body := MarkdownToHTML(expanded)
		section, hasSection := deck.FindSection(sections, slide.Section)
		sectionID := ""
		sectionTitle := ""
		if hasSection {
			sectionID = section.ID
			sectionTitle = section.Title
		}
		if activeTheme.DeckLayout == "" {
			result = append(result, RenderedSlide{
				Index:              i,
				Title:              slide.Title,
				Layout:             ResolveLayoutName(slide),
				Body:               body,
				SectionID:          sectionID,
				SectionTitle:       sectionTitle,
				HideLogo:           slide.HideLogo,
				HideFooter:         slide.HideFooter,
				ResolvedFooterText: ResolveFooterText(slide, deckMeta.Footer),
				StyleAttr:          ResolveSlideStyle(projectRoot, slide, &report),
				ImageHintClass:     ResolveImageHintClass(slide.ImageHints),
				ImageCaption:       ResolveImageCaption(slide.ImageHints),
			})
			continue
		}

		layoutPath := ResolveLayoutPath(activeTheme, slide)
		rendered, err := executeSlideLayout(projectRoot, activeTheme, layoutPath, slide, i, expanded, body, sectionID, sectionTitle, deckMeta.Footer, &report)
		if err != nil {
			return nil, diagnostics.Report{}, err
		}
		result = append(result, rendered)
	}
	return result, report, nil
}

func executeSlideLayout(projectRoot string, activeTheme theme.Metadata, layoutPath string, slide deck.Slide, index int, expandedMarkdown string, body template.HTML, sectionID string, sectionTitle string, deckFooter string, report *diagnostics.Report) (RenderedSlide, error) {
	tmpl, err := theme.ParseTemplateWithPartials(projectRoot, activeTheme, layoutPath, template.FuncMap{
		"markdownToHTML": MarkdownToHTML,
	})
	if err != nil {
		return RenderedSlide{}, fmt.Errorf("parse slide layout %q: %w", layoutPath, err)
	}

	data := struct {
		Index        int
		Slide        deck.Slide
		Body         template.HTML
		BodyColumns  []template.HTML
		LeadMedia    template.HTML
		LeadContent  template.HTML
		SectionID    string
		SectionTitle string
	}{
		Index:        index,
		Slide:        slide,
		Body:         body,
		BodyColumns:  RenderBodyColumns(slide, expandedMarkdown, body),
		LeadMedia:    RenderLeadMedia(slide, body),
		LeadContent:  RenderLeadContent(slide, body),
		SectionID:    sectionID,
		SectionTitle: sectionTitle,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, filepath.Base(layoutPath), data); err != nil {
		return RenderedSlide{}, fmt.Errorf("render slide layout %q: %w", layoutPath, err)
	}

	return RenderedSlide{
		Index:              index,
		Title:              slide.Title,
		Layout:             ResolveLayoutName(slide),
		Body:               template.HTML(buf.String()),
		IsDraft:            slide.Draft,
		SectionID:          sectionID,
		SectionTitle:       sectionTitle,
		HideLogo:           slide.HideLogo,
		HideFooter:         slide.HideFooter,
		ResolvedFooterText: ResolveFooterText(slide, deckFooter),
		StyleAttr:          ResolveSlideStyle(projectRoot, slide, report),
		ImageHintClass:     ResolveImageHintClass(slide.ImageHints),
		ImageCaption:       ResolveImageCaption(slide.ImageHints),
	}, nil
}

func ResolveLayoutPath(activeTheme theme.Metadata, slide deck.Slide) string {
	layoutName := ResolveLayoutName(slide)
	if layoutName != "" {
		if path, ok := activeTheme.SlideLayouts[layoutName]; ok {
			return path
		}
	}
	return activeTheme.DefaultLayout
}

func ResolveLayoutName(slide deck.Slide) string {
	if slide.Layout != "" {
		return slide.Layout
	}
	if slide.Type != "" {
		return slide.Type
	}
	return "default"
}

func MarkdownToHTML(source string) template.HTML {
	if source == "" {
		return ""
	}

	var buf bytes.Buffer
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
		),
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

func ResolveDeckLogo(projectRoot string, value string) (RenderedLogo, diagnostics.Report) {
	var report diagnostics.Report
	value = strings.TrimSpace(value)
	if value == "" {
		return RenderedLogo{}, report
	}

	baseRef, suffix := splitURLSuffix(value)
	if deckRef, ok := resolveDeckAssetReference(projectRoot, baseRef); ok {
		return RenderedLogo{
			ImageSrc: deckRef + suffix,
			IsImage:  true,
		}, report
	}

	if looksLikeImageURL(value) {
		if isExternalOrSpecialURL(value) {
			return RenderedLogo{
				ImageSrc: value,
				IsImage:  true,
			}, report
		}
		report.Add(diagnostics.Diagnostic{
			Severity: diagnostics.SeverityWarning,
			Code:     "asset_missing",
			Message:  fmt.Sprintf("deck logo asset %q could not be resolved", value),
			Path:     filepath.Join(projectRoot, "margo.yaml"),
		})
		return RenderedLogo{
			Text: value,
		}, report
	}

	return RenderedLogo{
		Text: value,
	}, report
}

func looksLikeImageURL(value string) bool {
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "data:image/") {
		return true
	}
	for _, ext := range []string{".svg", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif"} {
		if strings.Contains(strings.SplitN(lower, "?", 2)[0], ext) {
			return true
		}
	}
	if strings.HasPrefix(lower, "assets/") || strings.HasPrefix(lower, "/assets/") {
		return true
	}
	return false
}

func RenderBodyColumns(slide deck.Slide, expandedMarkdown string, body template.HTML) []template.HTML {
	if ResolveLayoutName(slide) != "two-column" {
		return nil
	}

	parts := strings.Split(expandedMarkdown, columnBreakMarker)
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
		columns = append(columns, MarkdownToHTML(trimmed))
	}
	return columns
}

func RenderLeadMedia(slide deck.Slide, body template.HTML) template.HTML {
	layout := ResolveLayoutName(slide)
	if layout != "media-left" && layout != "media-right" {
		return ""
	}

	match := leadImagePattern.FindString(string(body))
	return template.HTML(match)
}

func RenderLeadContent(slide deck.Slide, body template.HTML) template.HTML {
	layout := ResolveLayoutName(slide)
	if layout != "media-left" && layout != "media-right" {
		return body
	}

	match := leadImagePattern.FindString(string(body))
	if match == "" {
		return body
	}
	return template.HTML(strings.TrimSpace(strings.Replace(string(body), match, "", 1)))
}

func ResolveFooterText(slide deck.Slide, deckFooter string) string {
	if strings.TrimSpace(slide.FooterText) != "" {
		return slide.FooterText
	}
	return deckFooter
}

func ResolveBackgroundStyle(projectRoot string, slide deck.Slide, report *diagnostics.Report) string {
	parts := make([]string, 0, 3)
	if strings.TrimSpace(slide.Background.Color) != "" {
		parts = append(parts, "background-color: "+slide.Background.Color)
	}
	if strings.TrimSpace(slide.Background.Image) != "" {
		imageRef := slide.Background.Image
		if projectRoot != "" {
			resolvedRef, warning := ResolveAssetReference(projectRoot, slide, imageRef, "background image")
			imageRef = resolvedRef
			if warning != nil && report != nil {
				report.Add(*warning)
			}
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

func ResolveSlideStyle(projectRoot string, slide deck.Slide, report *diagnostics.Report) template.CSS {
	parts := make([]string, 0, 2)
	if background := strings.TrimSpace(ResolveBackgroundStyle(projectRoot, slide, report)); background != "" {
		parts = append(parts, background)
	}
	if imageHints := strings.TrimSpace(resolveImageHintStyle(slide.ImageHints)); imageHints != "" {
		parts = append(parts, imageHints)
	}
	return template.CSS(strings.Join(parts, "; "))
}

func ResolveImageHintClass(hints map[string]any) string {
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

func ResolveImageCaption(hints map[string]any) string {
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
