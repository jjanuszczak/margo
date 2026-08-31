package html

import (
	"html/template"

	"github.com/jjanuszczak/margo/internal/deck"
	"github.com/jjanuszczak/margo/internal/diagnostics"
	"github.com/jjanuszczak/margo/internal/output/render"
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

func splitBodyColumns(expandedMarkdown string, body template.HTML) []template.HTML {
	return render.SplitBodyColumns(expandedMarkdown, body)
}

func leadingImage(body template.HTML) template.HTML {
	return render.LeadingImage(body)
}

func withoutLeadingImage(body template.HTML) template.HTML {
	return render.WithoutLeadingImage(body)
}

func resolveDeckLogo(projectRoot string, value string) (render.RenderedLogo, diagnostics.Report) {
	return render.ResolveDeckLogo(projectRoot, value)
}

func markdownToHTML(source string) template.HTML {
	return render.MarkdownToHTML(source)
}
