package deploy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubPagesWritesReleaseAndManualWorkflow(t *testing.T) {
	root := t.TempDir()
	if output, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	result, err := GitHubPages(root, PagesOptions{MargoVersion: "v0.3.0"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(result.WorkflowPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"tags: ['v*']", "workflow_dispatch:", "go-version: '1.22.x'", "margo@v0.3.0", "path: dist/html", "actions/deploy-pages@v4"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("workflow missing %q", want)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".nojekyll")); err != nil {
		t.Fatalf(".nojekyll missing: %v", err)
	}
	if _, err := GitHubPages(root, PagesOptions{}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected collision, got %v", err)
	}
}

func TestGitHubPagesRequiresGitRepository(t *testing.T) {
	if _, err := GitHubPages(t.TempDir(), PagesOptions{}); err == nil || !strings.Contains(err.Error(), "Git repository") {
		t.Fatalf("expected Git error, got %v", err)
	}
}
