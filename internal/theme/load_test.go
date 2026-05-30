package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingThemeMetadataReturnsStructuredError(t *testing.T) {
	projectRoot := t.TempDir()
	themeDir := filepath.Join(projectRoot, ThemesDirName, "custom")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		t.Fatalf("mkdir theme dir: %v", err)
	}

	_, err := Load(projectRoot, "custom")
	if err == nil {
		t.Fatal("expected Load to fail")
	}
	themeErr, ok := AsError(err)
	if !ok {
		t.Fatalf("expected structured theme error, got %T: %v", err, err)
	}
	if themeErr.Path != filepath.Join(themeDir, ThemeMetadataFile) {
		t.Fatalf("unexpected error path %q", themeErr.Path)
	}
	if !strings.Contains(themeErr.Message, "read theme metadata") {
		t.Fatalf("unexpected error message %q", themeErr.Message)
	}
}
