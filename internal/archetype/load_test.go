package archetype

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesSnakeCaseMetadata(t *testing.T) {
	projectRoot := t.TempDir()
	archDir := filepath.Join(projectRoot, ArchetypesDirName, "section")
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		t.Fatalf("mkdir archetype dir: %v", err)
	}

	content := `name: section
description: Explicit section divider slide
default_layout: section
default_type: section
`
	if err := os.WriteFile(filepath.Join(archDir, MetadataFileName), []byte(content), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	meta, err := Load(projectRoot, "section")
	if err != nil {
		t.Fatalf("load metadata: %v", err)
	}

	if meta.DefaultLayout != "section" {
		t.Fatalf("expected default layout %q, got %q", "section", meta.DefaultLayout)
	}
	if meta.DefaultType != "section" {
		t.Fatalf("expected default type %q, got %q", "section", meta.DefaultType)
	}
}

func TestListSortsBuiltInArchetypesFirst(t *testing.T) {
	projectRoot := t.TempDir()
	writeArchetypeFixture(t, projectRoot, "section", "Explicit section divider slide", "section", "section")
	writeArchetypeFixture(t, projectRoot, "custom", "Custom slide", "custom", "custom")
	writeArchetypeFixture(t, projectRoot, "default", "Default slide", "content", "basic")
	writeArchetypeFixture(t, projectRoot, "title", "Title slide", "title", "title")
	writeArchetypeFixture(t, projectRoot, "agenda", "Agenda slide", "agenda", "agenda")
	writeArchetypeFixture(t, projectRoot, "metric", "Metric slide", "metric", "metric")
	writeArchetypeFixture(t, projectRoot, "quote", "Quote slide", "quote", "quote")
	writeArchetypeFixture(t, projectRoot, "closing", "Closing slide", "closing", "closing")

	list, err := List(projectRoot)
	if err != nil {
		t.Fatalf("list archetypes: %v", err)
	}

	got := make([]string, 0, len(list))
	for _, meta := range list {
		got = append(got, meta.Name)
	}
	want := []string{"default", "title", "section", "agenda", "metric", "quote", "closing", "custom"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected archetype order %v, got %v", want, got)
		}
	}
}

func writeArchetypeFixture(t *testing.T, projectRoot string, name string, description string, layout string, slideType string) {
	t.Helper()
	archDir := filepath.Join(projectRoot, ArchetypesDirName, name)
	if err := os.MkdirAll(archDir, 0o755); err != nil {
		t.Fatalf("mkdir archetype dir: %v", err)
	}

	content := "name: " + name + "\n" +
		"description: " + description + "\n" +
		"default_layout: " + layout + "\n" +
		"default_type: " + slideType + "\n"
	if err := os.WriteFile(filepath.Join(archDir, MetadataFileName), []byte(content), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
}
