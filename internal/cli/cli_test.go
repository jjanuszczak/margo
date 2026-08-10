package cli

import (
	"bytes"
	"io"
	"os"
	"os/exec"
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

func TestParseNewNoteArgs(t *testing.T) {
	name, slide, err := parseNewNoteArgs([]string{"speaker-script", "--slide", "02-why"})
	if err != nil {
		t.Fatalf("parseNewNoteArgs returned error: %v", err)
	}
	if name != "speaker-script" || slide != "02-why" {
		t.Fatalf("unexpected note args: name=%q slide=%q", name, slide)
	}
	if _, _, err := parseNewNoteArgs([]string{"speaker-script"}); err != nil {
		t.Fatalf("parseNewNoteArgs should leave missing slide validation to command, got %v", err)
	}
}

func TestRunNewNoteCreatesFrontMatterScaffold(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := scaffold.CreateDeck(scaffold.DeckOptions{Name: "deck", TargetDir: projectRoot}); err != nil {
		t.Fatalf("create deck: %v", err)
	}
	restoreWD := withWorkingDir(t, projectRoot)
	defer restoreWD()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"new", "note", "Speaker Script", "--slide", "02-why"}, &stdout, &stderr); code != 0 {
		t.Fatalf("expected new note to succeed, stderr=%q", stderr.String())
	}
	path := filepath.Join(projectRoot, "slides", "02-why", "notes", "speaker-script.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read created note: %v", err)
	}
	if !strings.Contains(string(raw), "id: speaker-script") || !strings.Contains(stdout.String(), "created note at") {
		t.Fatalf("unexpected new note result: stdout=%q note=%q", stdout.String(), raw)
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

func TestParseThemeAddArgs(t *testing.T) {
	repo, ref, name, err := parseThemeAddArgs([]string{"https://example.com/theme.git", "--ref", "v1.2.0", "--name", "brand"})
	if err != nil {
		t.Fatalf("parseThemeAddArgs returned error: %v", err)
	}
	if repo != "https://example.com/theme.git" || ref != "v1.2.0" || name != "brand" {
		t.Fatalf("unexpected parsed values repo=%q ref=%q name=%q", repo, ref, name)
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

func TestRunPackAndUnpack(t *testing.T) {
	parent := t.TempDir()
	projectRoot := filepath.Join(parent, "deck")
	if err := scaffold.CreateDeck(scaffold.DeckOptions{Name: "deck", TargetDir: projectRoot}); err != nil {
		t.Fatalf("create deck: %v", err)
	}
	var stdout bytes.Buffer
	if err := runPack([]string{projectRoot}, &stdout); err != nil {
		t.Fatalf("runPack returned error: %v", err)
	}
	archivePath := filepath.Join(parent, "deck.margo")
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("expected archive at %s: %v", archivePath, err)
	}
	if !strings.Contains(stdout.String(), "packed project archive") {
		t.Fatalf("unexpected pack output: %q", stdout.String())
	}

	stdout.Reset()
	if err := runUnpack([]string{archivePath, filepath.Join(parent, "restored")}, &stdout); err != nil {
		t.Fatalf("runUnpack returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(parent, "restored", "margo.yaml")); err != nil {
		t.Fatalf("expected restored project config: %v", err)
	}
}

func TestRunThemeAddInstallsVendoredTheme(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := scaffold.CreateDeck(scaffold.DeckOptions{
		Name:      "test-deck",
		TargetDir: projectRoot,
	}); err != nil {
		t.Fatalf("create deck: %v", err)
	}
	repoRoot := createThemeGitRepo(t, "portable")

	restoreWD := withWorkingDir(t, projectRoot)
	defer restoreWD()

	var out bytes.Buffer
	if err := runThemeAdd([]string{repoRoot}, &out); err != nil {
		t.Fatalf("runThemeAdd returned error: %v", err)
	}
	if !strings.Contains(out.String(), "installed theme portable") {
		t.Fatalf("expected install output, got %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "themes", "portable", "theme.yaml")); err != nil {
		t.Fatalf("expected installed vendored theme: %v", err)
	}
}

func TestRunThemeListReportsVendoredSources(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := scaffold.CreateDeck(scaffold.DeckOptions{
		Name:      "test-deck",
		TargetDir: projectRoot,
	}); err != nil {
		t.Fatalf("create deck: %v", err)
	}
	repoRoot := createThemeGitRepo(t, "portable")

	restoreWD := withWorkingDir(t, projectRoot)
	defer restoreWD()

	if err := runThemeAdd([]string{repoRoot}, io.Discard); err != nil {
		t.Fatalf("runThemeAdd returned error: %v", err)
	}

	var out bytes.Buffer
	if err := runThemeList(&out); err != nil {
		t.Fatalf("runThemeList returned error: %v", err)
	}
	if !strings.Contains(out.String(), "portable - "+repoRoot+" @ ") {
		t.Fatalf("expected listed source info, got %q", out.String())
	}
}

func TestRunThemeUpdateRefreshesInstalledTheme(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "deck")
	if err := scaffold.CreateDeck(scaffold.DeckOptions{
		Name:      "test-deck",
		TargetDir: projectRoot,
	}); err != nil {
		t.Fatalf("create deck: %v", err)
	}
	repoRoot := createThemeGitRepo(t, "portable")

	restoreWD := withWorkingDir(t, projectRoot)
	defer restoreWD()

	if err := runThemeAdd([]string{repoRoot, "--name", "brand"}, io.Discard); err != nil {
		t.Fatalf("runThemeAdd returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "layouts", "default.html"), []byte("updated from repo"), 0o644); err != nil {
		t.Fatalf("write updated layout: %v", err)
	}
	runGit(t, repoRoot, "add", ".")
	runGit(t, repoRoot, "commit", "-m", "update theme")

	var out bytes.Buffer
	if err := runThemeUpdate([]string{"brand"}, &out); err != nil {
		t.Fatalf("runThemeUpdate returned error: %v", err)
	}
	if !strings.Contains(out.String(), "updated theme brand") {
		t.Fatalf("expected update output, got %q", out.String())
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, "themes", "brand", "layouts", "default.html"))
	if err != nil {
		t.Fatalf("read updated vendored theme: %v", err)
	}
	if string(data) != "updated from repo" {
		t.Fatalf("expected updated vendored theme, got %q", string(data))
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

func createThemeGitRepo(t *testing.T, name string) string {
	t.Helper()
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "layouts"), 0o755); err != nil {
		t.Fatalf("mkdir layouts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "theme.yaml"), []byte("name: "+name+"\nversion: 0.1.0\n"), 0o644); err != nil {
		t.Fatalf("write theme metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "layouts", "default.html"), []byte("{{ .Deck.Title }}"), 0o644); err != nil {
		t.Fatalf("write theme layout: %v", err)
	}

	runGit(t, repoRoot, "init")
	runGit(t, repoRoot, "config", "user.name", "Codex")
	runGit(t, repoRoot, "config", "user.email", "codex@example.com")
	runGit(t, repoRoot, "add", ".")
	runGit(t, repoRoot, "commit", "-m", "add theme")
	return repoRoot
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
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
