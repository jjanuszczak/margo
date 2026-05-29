package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseBuildLikeArgs(t *testing.T) {
	includeDrafts, openBrowser, err := parseBuildLikeArgs("serve", []string{"--include-drafts", "--no-open"})
	if err != nil {
		t.Fatalf("parseBuildLikeArgs returned error: %v", err)
	}
	if !includeDrafts {
		t.Fatal("expected includeDrafts to be true")
	}
	if openBrowser {
		t.Fatal("expected openBrowser to be false")
	}

	includeDrafts, openBrowser, err = parseBuildLikeArgs("build", []string{"--include-drafts"})
	if err != nil {
		t.Fatalf("parseBuildLikeArgs returned error: %v", err)
	}
	if !includeDrafts {
		t.Fatal("expected build includeDrafts to be true")
	}
	if openBrowser {
		t.Fatal("expected build openBrowser to be false")
	}
}

func TestParseNewSlideArgs(t *testing.T) {
	name, archetype, err := parseNewSlideArgs([]string{"roadmap", "--archetype", "title"})
	if err != nil {
		t.Fatalf("parseNewSlideArgs returned error: %v", err)
	}
	if name != "roadmap" {
		t.Fatalf("expected slide name %q, got %q", "roadmap", name)
	}
	if archetype != "title" {
		t.Fatalf("expected archetype %q, got %q", "title", archetype)
	}
}

func TestChooseSlideArchetypeInteractive(t *testing.T) {
	projectRoot := t.TempDir()
	writeArchetype(t, projectRoot, "default", "Default slide")
	writeArchetype(t, projectRoot, "section", "Section divider")
	writeArchetype(t, projectRoot, "title", "Title slide")

	inputPath := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(inputPath, []byte("2\n"), 0o644); err != nil {
		t.Fatalf("write input fixture: %v", err)
	}
	input, err := os.Open(inputPath)
	if err != nil {
		t.Fatalf("open input fixture: %v", err)
	}
	defer input.Close()

	var out bytes.Buffer
	name, err := chooseSlideArchetypeFromReader(projectRoot, input, &out, true)
	if err != nil {
		t.Fatalf("chooseSlideArchetypeFromReader returned error: %v", err)
	}
	if name != "title" {
		t.Fatalf("expected interactive choice %q, got %q", "title", name)
	}
	if !strings.Contains(out.String(), "select archetype [1]:") {
		t.Fatalf("expected prompt output, got %q", out.String())
	}
}

func TestChooseSlideArchetypeNonInteractiveDefaultsFirst(t *testing.T) {
	projectRoot := t.TempDir()
	writeArchetype(t, projectRoot, "default", "Default slide")
	writeArchetype(t, projectRoot, "section", "Section divider")

	var out bytes.Buffer
	name, err := chooseSlideArchetypeFromReader(projectRoot, strings.NewReader(""), &out, false)
	if err != nil {
		t.Fatalf("chooseSlideArchetypeFromReader returned error: %v", err)
	}
	if name != "default" {
		t.Fatalf("expected non-interactive fallback %q, got %q", "default", name)
	}
}

func TestParseNewThemeArgs(t *testing.T) {
	name, blank, err := parseNewThemeArgs([]string{"custom", "--blank"})
	if err != nil {
		t.Fatalf("parseNewThemeArgs returned error: %v", err)
	}
	if name != "custom" {
		t.Fatalf("expected theme name %q, got %q", "custom", name)
	}
	if !blank {
		t.Fatal("expected blank theme mode")
	}
}

func writeArchetype(t *testing.T, projectRoot string, name string, description string) {
	t.Helper()
	dir := filepath.Join(projectRoot, "archetypes", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir archetype dir: %v", err)
	}
	content := "name: " + name + "\n" +
		"description: " + description + "\n" +
		"default_layout: " + name + "\n" +
		"default_type: " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, "archetype.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write archetype metadata: %v", err)
	}
}
