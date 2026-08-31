package html

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jjanuszczak/margo/internal/config"
	"github.com/jjanuszczak/margo/internal/content"
	"github.com/jjanuszczak/margo/internal/deck"
	"github.com/jjanuszczak/margo/internal/theme"
)

func BenchmarkBenchmarkDeckBuildFlow(b *testing.B) {
	projectRoot := fixtureProjectRoot(b, "benchmark-deck")

	raw, err := config.LoadRaw(filepath.Join(projectRoot, config.DefaultFilename))
	if err != nil {
		b.Fatalf("load raw config: %v", err)
	}
	parsed, err := config.Parse(raw)
	if err != nil {
		b.Fatalf("parse config: %v", err)
	}

	slides, err := content.DiscoverSlides(projectRoot)
	if err != nil {
		b.Fatalf("discover slides: %v", err)
	}
	if got, want := len(slides), 20; got != want {
		b.Fatalf("expected %d slide bundles, got %d", want, got)
	}

	filtered := deck.FilterSlides(slides, deck.FilterOptions{})
	shaped := deck.ApplySectionDividers(filtered)
	sections := deck.BuildSections(shaped)

	activeTheme, err := theme.Load(projectRoot, parsed.Config.Theme.Name)
	if err != nil {
		b.Fatalf("load theme: %v", err)
	}
	parsed.Config.Theme.Options, err = theme.ResolveOptions(activeTheme, parsed.Config.Theme.Options)
	if err != nil {
		b.Fatalf("resolve theme options: %v", err)
	}

	model := deck.Model{
		Config:   parsed.Config,
		Sections: sections,
		Slides:   shaped,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := os.RemoveAll(filepath.Join(projectRoot, "dist")); err != nil {
			b.Fatalf("remove dist: %v", err)
		}
		report, err := Write(projectRoot, model, activeTheme)
		if err != nil {
			b.Fatalf("write html: %v", err)
		}
		if len(report.Items) != 0 {
			b.Fatalf("expected benchmark deck to build without warnings, got %d", len(report.Items))
		}
	}
}
