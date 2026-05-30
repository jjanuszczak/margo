package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"margo/internal/ignore"
)

func TestParseSlideExtractsFrontMatterAndBodyNotes(t *testing.T) {
	projectRoot := t.TempDir()
	bundlePath := filepath.Join(projectRoot, "slides", "02-why")
	if err := os.MkdirAll(bundlePath, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}

	source := `---
title: Why Margo
order: 2
layout: content
notes:
  - Mention Hugo mental model
  - Emphasize HTML-first pipeline
---

## Why this exists

- Markdown-first authoring

## Notes

Remember to mention future PPTX export intent.
`

	slide, err := parseSlide(projectRoot, filepath.Join(bundlePath, "index.md"), bundlePath, source, ignore.Matcher{})
	if err != nil {
		t.Fatalf("parseSlide returned error: %v", err)
	}

	if got, want := len(slide.Notes), 3; got != want {
		t.Fatalf("expected %d notes, got %d: %#v", want, got, slide.Notes)
	}

	for _, note := range slide.Notes {
		if note == "" {
			t.Fatal("expected non-empty notes")
		}
	}

	if strings.Contains(slide.BodyMarkdown, "Remember to mention future PPTX export intent.") {
		t.Fatalf("body markdown still contains notes section: %q", slide.BodyMarkdown)
	}
}

func TestParseSlideResolvesNestedIncludes(t *testing.T) {
	projectRoot := t.TempDir()
	bundlePath := filepath.Join(projectRoot, "slides", "01-title")
	if err := os.MkdirAll(bundlePath, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "shared"), 0o755); err != nil {
		t.Fatalf("create shared dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "shared", "summary.md"), []byte("Shared summary"), 0o644); err != nil {
		t.Fatalf("write summary include: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "shared", "agenda.md"), []byte("- Intro\n{{< include \"shared/summary.md\" >}}"), 0o644); err != nil {
		t.Fatalf("write agenda include: %v", err)
	}

	source := `---
title: Included Content
---

## Agenda

{{< include "shared/agenda.md" >}}
`

	slide, err := parseSlide(projectRoot, filepath.Join(bundlePath, "index.md"), bundlePath, source, ignore.Matcher{})
	if err != nil {
		t.Fatalf("parseSlide returned error: %v", err)
	}

	if !strings.Contains(slide.BodyMarkdown, "Shared summary") {
		t.Fatalf("expected nested include content in body: %q", slide.BodyMarkdown)
	}
	if strings.Contains(slide.BodyMarkdown, `{{< include`) {
		t.Fatalf("expected include directives to be expanded: %q", slide.BodyMarkdown)
	}
}

func TestParseSlideRejectsIncludeCycles(t *testing.T) {
	projectRoot := t.TempDir()
	bundlePath := filepath.Join(projectRoot, "slides", "01-title")
	if err := os.MkdirAll(bundlePath, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "shared"), 0o755); err != nil {
		t.Fatalf("create shared dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "shared", "a.md"), []byte("{{< include \"shared/b.md\" >}}"), 0o644); err != nil {
		t.Fatalf("write a include: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "shared", "b.md"), []byte("{{< include \"shared/a.md\" >}}"), 0o644); err != nil {
		t.Fatalf("write b include: %v", err)
	}

	source := `---
title: Cycles
---

{{< include "shared/a.md" >}}
`

	_, err := parseSlide(projectRoot, filepath.Join(bundlePath, "index.md"), bundlePath, source, ignore.Matcher{})
	if err == nil {
		t.Fatal("expected parseSlide to fail on include cycle")
	}
	if !strings.Contains(err.Error(), "include cycle detected") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestParseSlideRejectsEscapingIncludePaths(t *testing.T) {
	projectRoot := t.TempDir()
	bundlePath := filepath.Join(projectRoot, "slides", "01-title")
	if err := os.MkdirAll(bundlePath, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}

	source := `---
title: Escaping Include
---

{{< include "../outside.md" >}}
`

	_, err := parseSlide(projectRoot, filepath.Join(bundlePath, "index.md"), bundlePath, source, ignore.Matcher{})
	if err == nil {
		t.Fatal("expected parseSlide to fail on escaping include path")
	}
	if !strings.Contains(err.Error(), "escapes the project root") {
		t.Fatalf("expected project-root error, got %v", err)
	}
}

func TestDiscoverSlidesHonorsIgnoreFile(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, "slides", "visible"), 0o755); err != nil {
		t.Fatalf("create visible slide dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "slides", "ignored"), 0o755); err != nil {
		t.Fatalf("create ignored slide dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, ignore.DefaultFilename), []byte("slides/ignored/\n"), 0o644); err != nil {
		t.Fatalf("write ignore file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "slides", "visible", "index.md"), []byte("---\ntitle: Visible\norder: 1\n---\n"), 0o644); err != nil {
		t.Fatalf("write visible slide: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "slides", "ignored", "index.md"), []byte("---\ntitle: Ignored\norder: 2\n---\n"), 0o644); err != nil {
		t.Fatalf("write ignored slide: %v", err)
	}

	slides, err := DiscoverSlides(projectRoot)
	if err != nil {
		t.Fatalf("DiscoverSlides returned error: %v", err)
	}
	if got, want := len(slides), 1; got != want {
		t.Fatalf("expected %d visible slide, got %d", want, got)
	}
	if slides[0].Title != "Visible" {
		t.Fatalf("expected visible slide title, got %q", slides[0].Title)
	}
}

func TestDiscoverBundleAssetsHonorsIgnoreFile(t *testing.T) {
	projectRoot := t.TempDir()
	bundlePath := filepath.Join(projectRoot, "slides", "sample")
	if err := os.MkdirAll(bundlePath, 0o755); err != nil {
		t.Fatalf("create bundle dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, ignore.DefaultFilename), []byte("slides/sample/secret.png\n"), 0o644); err != nil {
		t.Fatalf("write ignore file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "public.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write public asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "secret.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write secret asset: %v", err)
	}

	matcher, err := ignore.Load(projectRoot)
	if err != nil {
		t.Fatalf("load ignore matcher: %v", err)
	}
	assets, err := discoverBundleAssets(projectRoot, bundlePath, matcher)
	if err != nil {
		t.Fatalf("discoverBundleAssets returned error: %v", err)
	}
	if got, want := len(assets), 1; got != want {
		t.Fatalf("expected %d visible asset, got %d: %#v", want, got, assets)
	}
	if assets[0] != "public.png" {
		t.Fatalf("expected remaining asset public.png, got %#v", assets)
	}
}
