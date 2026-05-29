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
