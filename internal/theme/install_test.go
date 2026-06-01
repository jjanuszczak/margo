package theme

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCopiesThemeAndWritesSourceMetadata(t *testing.T) {
	projectRoot := t.TempDir()
	repoRoot := createThemeRepo(t, "portable")

	installed, err := Install(InstallOptions{
		ProjectRoot: projectRoot,
		Repo:        repoRoot,
	})
	if err != nil {
		t.Fatalf("install returned error: %v", err)
	}
	if installed.Name != "portable" {
		t.Fatalf("expected installed name %q, got %q", "portable", installed.Name)
	}

	targetTheme := filepath.Join(projectRoot, ThemesDirName, "portable")
	if _, err := os.Stat(filepath.Join(targetTheme, "layouts", "default.html")); err != nil {
		t.Fatalf("expected copied layout: %v", err)
	}

	meta, err := Load(projectRoot, "portable")
	if err != nil {
		t.Fatalf("load installed theme: %v", err)
	}
	if meta.Source == nil {
		t.Fatal("expected installed theme source metadata")
	}
	if meta.Source.Type != "git" {
		t.Fatalf("expected source type git, got %q", meta.Source.Type)
	}
	if meta.Source.Repo != repoRoot {
		t.Fatalf("expected repo %q, got %q", repoRoot, meta.Source.Repo)
	}
	if len(meta.Source.ResolvedRef) != 40 {
		t.Fatalf("expected resolved ref hash, got %q", meta.Source.ResolvedRef)
	}
}

func TestInstallUsesExplicitLocalName(t *testing.T) {
	projectRoot := t.TempDir()
	repoRoot := createThemeRepo(t, "upstream-name")

	installed, err := Install(InstallOptions{
		ProjectRoot: projectRoot,
		Repo:        repoRoot,
		Name:        "brand",
	})
	if err != nil {
		t.Fatalf("install returned error: %v", err)
	}
	if installed.Name != "brand" {
		t.Fatalf("expected installed name %q, got %q", "brand", installed.Name)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ThemesDirName, "brand", ThemeMetadataFile)); err != nil {
		t.Fatalf("expected installed theme metadata: %v", err)
	}
}

func TestListIncludesSourceMetadata(t *testing.T) {
	projectRoot := t.TempDir()
	repoRoot := createThemeRepo(t, "portable")

	if _, err := Install(InstallOptions{
		ProjectRoot: projectRoot,
		Repo:        repoRoot,
	}); err != nil {
		t.Fatalf("install returned error: %v", err)
	}

	themes, err := List(projectRoot)
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if len(themes) != 1 {
		t.Fatalf("expected 1 installed theme, got %d", len(themes))
	}
	if themes[0].Name != "portable" {
		t.Fatalf("expected portable theme, got %#v", themes[0])
	}
	if themes[0].Source == nil || themes[0].Source.Repo != repoRoot {
		t.Fatalf("expected source repo %q, got %#v", repoRoot, themes[0].Source)
	}
}

func createThemeRepo(t *testing.T, name string) string {
	t.Helper()
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "layouts"), 0o755); err != nil {
		t.Fatalf("mkdir layouts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "theme.yaml"), []byte("name: "+name+"\nversion: 0.1.0\n"), 0o644); err != nil {
		t.Fatalf("write theme metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "layouts", "default.html"), []byte("{{ .Deck.Title }}"), 0o644); err != nil {
		t.Fatalf("write theme layout: %v", err)
	}

	runGitCommand(t, repoRoot, "init")
	runGitCommand(t, repoRoot, "config", "user.name", "Codex")
	runGitCommand(t, repoRoot, "config", "user.email", "codex@example.com")
	runGitCommand(t, repoRoot, "add", ".")
	runGitCommand(t, repoRoot, "commit", "-m", "add theme")
	return repoRoot
}

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}
