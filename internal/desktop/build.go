package desktop

import (
	"fmt"
	"path/filepath"

	"github.com/jjanuszczak/margo/internal/deck"
	"github.com/jjanuszczak/margo/internal/diagnostics"
	"github.com/jjanuszczak/margo/internal/layoutaudit"
	htmloutput "github.com/jjanuszczak/margo/internal/output/html"
	"github.com/jjanuszczak/margo/internal/output/pdf"
	"github.com/jjanuszczak/margo/internal/output/pptx"
	"github.com/jjanuszczak/margo/internal/output/printhtml"
)

func (e *Engine) build(format string) (Response, error) {
	if _, err := confined(e.root, "dist"); err != nil {
		return Response{}, err
	}
	model, active, err := e.load()
	if err != nil {
		return Response{}, err
	}
	model.Slides = deck.ApplySectionDividers(deck.FilterSlides(model.Slides, deck.FilterOptions{}))
	model.Sections = deck.BuildSections(model.Slides)
	var report diagnostics.Report
	var artifact string
	switch format {
	case "html":
		report, err = htmloutput.Write(e.root, model, active)
		artifact = htmloutput.OutputFile
	case "pptx":
		report, err = pptx.Write(e.root, model, active)
		artifact = pptx.OutputFile
	case "pdf":
		report, err = printhtml.Write(e.root, model, active)
		artifact = pdf.OutputFile
	default:
		return Response{}, fmt.Errorf("unsupported export format")
	}
	response := Response{Diagnostics: report.Items}
	if err != nil {
		return response, err
	}
	if format == "html" || format == "pdf" {
		path := filepath.Join(e.root, artifact)
		if format == "pdf" {
			path = filepath.Join(e.root, printhtml.OutputFile)
		}
		if err := layoutaudit.ApplyStructuralRemediation(path, active.StructuralRemediationPolicies()); err != nil {
			return response, err
		}
		enabled, min := active.FitSettings()
		if err := layoutaudit.ApplyFit(path, layoutaudit.FitPolicy{Enabled: enabled, MinScale: min}); err != nil {
			return response, err
		}
	}
	if format == "pdf" {
		if err := pdf.Write(e.root); err != nil {
			return response, err
		}
	}
	response.Artifact = filepath.Join(e.root, artifact)
	response.Log = "Built " + response.Artifact
	return response, nil
}
