package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"margo/internal/manifest"
	"margo/internal/scaffold"
)

func TestParseBuildLikeArgs(t *testing.T) {
	includeDrafts, openBrowser, port, err := parseBuildLikeArgs("serve", []string{"--include-drafts", "--no-open", "--port", "1414"})
	if err != nil {
		t.Fatalf("parseBuildLikeArgs returned error: %v", err)
	}
	if !includeDrafts {
		t.Fatal("expected includeDrafts to be true")
	}
	if openBrowser {
		t.Fatal("expected openBrowser to be false")
	}
	if port != "1414" {
		t.Fatalf("expected serve port %q, got %q", "1414", port)
	}

	includeDrafts, openBrowser, port, err = parseBuildLikeArgs("build", []string{"--include-drafts"})
	if err != nil {
		t.Fatalf("parseBuildLikeArgs returned error: %v", err)
	}
	if !includeDrafts {
		t.Fatal("expected build includeDrafts to be true")
	}
	if openBrowser {
		t.Fatal("expected build openBrowser to be false")
	}
	if port != "" {
		t.Fatalf("expected empty build port, got %q", port)
	}
}

func TestParseBuildLikeArgsRejectsInvalidServePort(t *testing.T) {
	_, _, _, err := parseBuildLikeArgs("serve", []string{"--port", "70000"})
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("expected invalid port error, got %v", err)
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

func TestRunNewSlideAppendsToManifestWhenPresent(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := scaffold.CreateDeck(scaffold.DeckOptions{
		Name:      "test-deck",
		TargetDir: projectRoot,
	}); err != nil {
		t.Fatalf("create deck: %v", err)
	}
	if err := manifest.Save(projectRoot, manifest.File{
		Slides: []string{"01-title", "02-why"},
	}); err != nil {
		t.Fatalf("save manifest: %v", err)
	}

	restoreWD := withWorkingDir(t, projectRoot)
	defer restoreWD()

	var out bytes.Buffer
	if err := runNewSlide([]string{"roadmap", "--archetype", "default"}, &out); err != nil {
		t.Fatalf("runNewSlide returned error: %v", err)
	}

	file, ok, err := manifest.Load(projectRoot)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if !ok {
		t.Fatal("expected manifest to exist")
	}
	got := strings.Join(file.Slides, ",")
	if got != "01-title,02-why,roadmap" {
		t.Fatalf("unexpected manifest order %q", got)
	}
}

func TestRunNewDeckSupportsAbsoluteTarget(t *testing.T) {
	targetRoot := t.TempDir()
	targetDir := filepath.Join(targetRoot, "absolute-deck")

	restoreWD := withWorkingDir(t, t.TempDir())
	defer restoreWD()

	var out bytes.Buffer
	if err := runNewDeck(targetDir, &out); err != nil {
		t.Fatalf("runNewDeck returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "margo.yaml")); err != nil {
		t.Fatalf("expected deck scaffold at absolute target, stat failed: %v", err)
	}
}

func TestRunBuildReportsConfigFieldLine(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "margo.yaml"), []byte(`version: 1

deck:
  description: Missing title

theme:
  name: default
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	restoreWD := withWorkingDir(t, projectRoot)
	defer restoreWD()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"build"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected build to fail")
	}
	if !strings.Contains(stderr.String(), "margo.yaml:4") {
		t.Fatalf("expected stderr to contain config line reference, got %q", stderr.String())
	}
}

func TestRunBuildReportsThemeLoadPathCleanly(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "margo.yaml"), []byte(`version: 1

deck:
  title: Sample

theme:
  name: missing
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	restoreWD := withWorkingDir(t, projectRoot)
	defer restoreWD()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"build"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected build to fail")
	}
	out := stderr.String()
	if !strings.Contains(out, "[theme_load_failed] error:") {
		t.Fatalf("expected theme load diagnostic, got %q", out)
	}
	if !strings.Contains(out, "read theme metadata") {
		t.Fatalf("expected theme load message, got %q", out)
	}
	if !strings.Contains(out, filepath.Join("themes", "missing", "theme.yaml")) {
		t.Fatalf("expected theme metadata path, got %q", out)
	}
}

func TestRunBuildWarnsOnMissingLocalAsset(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, "themes", "default", "layouts"), 0o755); err != nil {
		t.Fatalf("create theme layouts dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "slides", "01-title"), 0o755); err != nil {
		t.Fatalf("create slides dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "margo.yaml"), []byte(`version: 1

deck:
  title: Sample

theme:
  name: default
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "themes", "default", "theme.yaml"), []byte("name: default\n"), 0o644); err != nil {
		t.Fatalf("write theme metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "themes", "default", "layouts", "default.html"), []byte("{{ .Deck.Title }} {{ range .Slides }}{{ .Body }}{{ end }}"), 0o644); err != nil {
		t.Fatalf("write theme layout: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "slides", "01-title", "index.md"), []byte(`---
title: Sample
order: 1
---

![Missing](missing.svg)
`), 0o644); err != nil {
		t.Fatalf("write slide: %v", err)
	}

	restoreWD := withWorkingDir(t, projectRoot)
	defer restoreWD()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"build"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected build to succeed with warning, stderr=%q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "[asset_missing] warning:") {
		t.Fatalf("expected asset warning in stdout, got %q", stdout.String())
	}
}

func TestRunBuildRejectsInvalidThemeOptionValue(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, "themes", "default", "layouts"), 0o755); err != nil {
		t.Fatalf("create theme layouts dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "slides", "01-title"), 0o755); err != nil {
		t.Fatalf("create slides dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "margo.yaml"), []byte(`version: 1

deck:
  title: Sample

theme:
  name: default
  color_mode: sepia
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "themes", "default", "theme.yaml"), []byte(`name: default
config_options:
  - name: color_mode
    type: string
    default: light
    values:
      - light
      - dark
`), 0o644); err != nil {
		t.Fatalf("write theme metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "themes", "default", "layouts", "default.html"), []byte("{{ .Deck.Title }}"), 0o644); err != nil {
		t.Fatalf("write theme layout: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "slides", "01-title", "index.md"), []byte("---\ntitle: Sample\norder: 1\n---\n"), 0o644); err != nil {
		t.Fatalf("write slide: %v", err)
	}

	restoreWD := withWorkingDir(t, projectRoot)
	defer restoreWD()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"build"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected build to fail")
	}
	if !strings.Contains(stderr.String(), `invalid value "sepia"`) || !strings.Contains(stderr.String(), "allowed: light, dark") {
		t.Fatalf("expected invalid theme option value in stderr, got %q", stderr.String())
	}
}

func TestRunNewDeckThenBuildStarterDeck(t *testing.T) {
	parentDir := t.TempDir()

	restoreWD := withWorkingDir(t, parentDir)
	defer restoreWD()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"new", "starter-deck"}, &stdout, &stderr); code != 0 {
		t.Fatalf("expected new deck to succeed, stderr=%q", stderr.String())
	}

	projectRoot := filepath.Join(parentDir, "starter-deck")
	configPath := filepath.Join(projectRoot, "margo.yaml")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read starter config: %v", err)
	}
	updated := strings.Replace(string(raw), "  pdf: true\n", "  pdf: false\n", 1)
	if err := os.WriteFile(configPath, []byte(updated), 0o644); err != nil {
		t.Fatalf("write starter config: %v", err)
	}

	restoreDeckWD := withWorkingDir(t, projectRoot)
	defer restoreDeckWD()

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"build"}, &stdout, &stderr); code != 0 {
		t.Fatalf("expected starter deck build to succeed, stderr=%q", stderr.String())
	}

	rendered, err := os.ReadFile(filepath.Join(projectRoot, "dist", "html", "index.html"))
	if err != nil {
		t.Fatalf("read starter output: %v", err)
	}
	out := string(rendered)

	for _, needle := range []string{
		"starter-deck",
		"Why Margo",
		"Product Story",
		"Customer Story",
		"Why this exists",
		"class=\"callout",
		"class=\"shortcode-columns\"",
		"class=\"shortcode-stat\"",
		"assets/company-logo.svg",
		"assets/shared-grid.svg",
		"media-split-slide media-right",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("expected starter deck output to contain %q", needle)
		}
	}
	if strings.Contains(out, "Export PDF") {
		t.Fatalf("expected starter deck output to omit PDF control when pdf output is disabled")
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

func withWorkingDir(t *testing.T, dir string) func() {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	return func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}
}
