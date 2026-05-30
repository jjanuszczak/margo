package watch

import (
	"os"
	"path/filepath"
	"testing"

	"margo/internal/ignore"
)

func TestSnapshotProjectHonorsIgnoreFile(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, "slides", "visible"), 0o755); err != nil {
		t.Fatalf("create visible dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "slides", "ignored"), 0o755); err != nil {
		t.Fatalf("create ignored dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, ignore.DefaultFilename), []byte("slides/ignored/\n"), 0o644); err != nil {
		t.Fatalf("write ignore file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "margo.yaml"), []byte("deck:\n  title: Test\n\ntheme:\n  name: default\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "slides", "visible", "index.md"), []byte("visible"), 0o644); err != nil {
		t.Fatalf("write visible file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "slides", "ignored", "index.md"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("write ignored file: %v", err)
	}

	snapshot, err := SnapshotProject(projectRoot)
	if err != nil {
		t.Fatalf("SnapshotProject returned error: %v", err)
	}

	paths := map[string]bool{}
	for _, entry := range snapshot.Entries {
		paths[entry.Path] = true
	}
	if !paths["margo.yaml"] || !paths[filepath.Join("slides", "visible", "index.md")] {
		t.Fatalf("expected snapshot to contain config and visible slide, got %#v", paths)
	}
	if paths[filepath.Join("slides", "ignored", "index.md")] {
		t.Fatalf("expected ignored file to be excluded from snapshot, got %#v", paths)
	}
}
