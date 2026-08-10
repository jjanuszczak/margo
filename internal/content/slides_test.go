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

	if got, want := len(slide.Notes), 1; got != want {
		t.Fatalf("expected %d notes, got %d: %#v", want, got, slide.Notes)
	}

	for _, note := range slide.Notes {
		if note.Name != "Notes" || note.Markdown == "" {
			t.Fatal("expected non-empty notes")
		}
	}

	if strings.Contains(slide.BodyMarkdown, "Remember to mention future PPTX export intent.") {
		t.Fatalf("body markdown still contains notes section: %q", slide.BodyMarkdown)
	}
}

func TestDiscoverSlidesLoadsNamedBundleNotesAndExcludesThemFromAssets(t *testing.T) {
	projectRoot := t.TempDir()
	bundlePath := filepath.Join(projectRoot, "slides", "01-title")
	if err := os.MkdirAll(filepath.Join(bundlePath, "notes"), 0o755); err != nil {
		t.Fatalf("create notes directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "index.md"), []byte("---\ntitle: Title\n---\nSlide body"), 0o644); err != nil {
		t.Fatalf("write slide: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "notes", "speaker-script.md"), []byte("Say this first."), 0o644); err != nil {
		t.Fatalf("write speaker note: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "notes", "sources.md"), []byte("- [Source](https://example.com)"), 0o644); err != nil {
		t.Fatalf("write sources note: %v", err)
	}

	slides, err := DiscoverSlides(projectRoot)
	if err != nil {
		t.Fatalf("DiscoverSlides returned error: %v", err)
	}
	if got, want := len(slides), 1; got != want {
		t.Fatalf("expected %d slide, got %d", want, got)
	}
	if got, want := len(slides[0].Notes), 2; got != want {
		t.Fatalf("expected %d note files, got %#v", want, slides[0].Notes)
	}
	if slides[0].Notes[0].Name != "Sources" || slides[0].Notes[1].Name != "Speaker script" {
		t.Fatalf("unexpected note labels: %#v", slides[0].Notes)
	}
	if len(slides[0].Assets) != 0 {
		t.Fatalf("notes must not be staged as slide assets: %#v", slides[0].Assets)
	}
}

func TestDiscoverSlidesLoadsNoteFrontMatterAndKeepsImplicitNotes(t *testing.T) {
	projectRoot := t.TempDir()
	bundlePath := filepath.Join(projectRoot, "slides", "01-title")
	if err := os.MkdirAll(filepath.Join(bundlePath, "notes"), 0o755); err != nil {
		t.Fatalf("create notes directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "index.md"), []byte("---\ntitle: Title\n---\nSlide body"), 0o644); err != nil {
		t.Fatalf("write slide: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "notes", "research.md"), []byte("Plain legacy note."), 0o644); err != nil {
		t.Fatalf("write implicit note: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "notes", "speaker.md"), []byte(`---
id: keynote-script
title: Keynote script
order: -1
visibility: visible
draft: true
kind: speaker_script
tags: [internal, rehearsal]
language: en
---
Open with the customer story.`), 0o644); err != nil {
		t.Fatalf("write explicit note: %v", err)
	}

	slides, err := DiscoverSlides(projectRoot)
	if err != nil {
		t.Fatalf("DiscoverSlides returned error: %v", err)
	}
	notes := slides[0].Notes
	if got, want := len(notes), 2; got != want {
		t.Fatalf("expected %d notes, got %#v", want, notes)
	}
	if got := notes[0]; got.ID != "keynote-script" || got.Name != "Keynote script" || got.Order != -1 || !got.Draft || got.Kind != "speaker_script" || got.Language != "en" || len(got.Tags) != 2 {
		t.Fatalf("unexpected explicit note: %#v", got)
	}
	if got := notes[1]; got.ID != "research" || got.Name != "Research" || got.Markdown != "Plain legacy note." || got.Visibility != "" {
		t.Fatalf("unexpected implicit note: %#v", got)
	}
}

func TestDiscoverSlidesLoadsNotesForSlideWithoutFrontMatter(t *testing.T) {
	projectRoot := t.TempDir()
	bundlePath := filepath.Join(projectRoot, "slides", "01-title")
	if err := os.MkdirAll(filepath.Join(bundlePath, "notes"), 0o755); err != nil {
		t.Fatalf("create notes directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "index.md"), []byte("# Plain slide"), 0o644); err != nil {
		t.Fatalf("write plain slide: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "notes", "speaker.md"), []byte("Plain note."), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}

	slides, err := DiscoverSlides(projectRoot)
	if err != nil {
		t.Fatalf("DiscoverSlides returned error: %v", err)
	}
	if got, want := len(slides[0].Notes), 1; got != want || slides[0].Notes[0].Name != "Speaker" {
		t.Fatalf("expected implicit note on plain slide, got %#v", slides[0].Notes)
	}
}

func TestDiscoverSlidesRejectsDuplicateNoteIDsAndInvalidVisibility(t *testing.T) {
	projectRoot := t.TempDir()
	bundlePath := filepath.Join(projectRoot, "slides", "01-title")
	if err := os.MkdirAll(filepath.Join(bundlePath, "notes"), 0o755); err != nil {
		t.Fatalf("create notes directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "index.md"), []byte("---\ntitle: Title\n---\nSlide body"), 0o644); err != nil {
		t.Fatalf("write slide: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "notes", "one.md"), []byte("---\nid: shared\n---\nFirst"), 0o644); err != nil {
		t.Fatalf("write first note: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "notes", "two.md"), []byte("---\nid: shared\n---\nSecond"), 0o644); err != nil {
		t.Fatalf("write duplicate note: %v", err)
	}
	if _, err := DiscoverSlides(projectRoot); err == nil || !strings.Contains(err.Error(), "duplicates note id") {
		t.Fatalf("expected duplicate ID error, got %v", err)
	}
	if err := os.Remove(filepath.Join(bundlePath, "notes", "two.md")); err != nil {
		t.Fatalf("remove duplicate note: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundlePath, "notes", "invalid.md"), []byte("---\nvisibility: skipped\n---\nInvalid"), 0o644); err != nil {
		t.Fatalf("write invalid note: %v", err)
	}
	if _, err := DiscoverSlides(projectRoot); err == nil || !strings.Contains(err.Error(), "invalid visibility") {
		t.Fatalf("expected visibility error, got %v", err)
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
