package html

import (
	"html/template"

	"margo/internal/deck"
	"margo/internal/diagnostics"
	"margo/internal/output/render"
)

func stageAssets(projectRoot string, slides []deck.Slide) error {
	return render.StageAssets(projectRoot, OutputDir, slides)
}

func rewriteMarkdownAssetRefs(projectRoot string, slide deck.Slide, source string) (string, diagnostics.Report) {
	return render.RewriteMarkdownAssetRefs(projectRoot, slide, source)
}

func resolveAssetReference(projectRoot string, slide deck.Slide, ref string, context string) (string, *diagnostics.Diagnostic) {
	return render.ResolveAssetReference(projectRoot, slide, ref, context)
}

func renderBodyColumns(slide deck.Slide, expandedMarkdown string, body template.HTML) []template.HTML {
	return render.RenderBodyColumns(slide, expandedMarkdown, body)
}

func renderLeadMedia(slide deck.Slide, body template.HTML) template.HTML {
	return render.RenderLeadMedia(slide, body)
}

func renderLeadContent(slide deck.Slide, body template.HTML) template.HTML {
	return render.RenderLeadContent(slide, body)
}

func resolveDeckLogo(projectRoot string, value string) (render.RenderedLogo, diagnostics.Report) {
	return render.ResolveDeckLogo(projectRoot, value)
}

func markdownToHTML(source string) template.HTML {
	return render.MarkdownToHTML(source)
}
