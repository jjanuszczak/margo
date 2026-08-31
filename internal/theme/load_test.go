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

func TestLoadRejectsIncompleteDeckSlideLayoutPair(t *testing.T) {
	projectRoot := t.TempDir()
	themeDir := filepath.Join(projectRoot, ThemesDirName, "custom")
	if err := os.MkdirAll(filepath.Join(themeDir, "layouts"), 0o755); err != nil {
		t.Fatalf("mkdir layouts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, ThemeMetadataFile), []byte("name: custom\n"), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layouts", "deck.html"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write deck layout: %v", err)
	}

	_, err := Load(projectRoot, "custom")
	if err == nil {
		t.Fatal("expected Load to fail")
	}
	themeErr, ok := AsError(err)
	if !ok {
		t.Fatalf("expected structured theme error, got %T: %v", err, err)
	}
	if themeErr.Path != filepath.Join(themeDir, "layouts", "slide-default.html") {
		t.Fatalf("unexpected error path %q", themeErr.Path)
	}
	if !strings.Contains(themeErr.Message, "incomplete theme layout contract") {
		t.Fatalf("unexpected error message %q", themeErr.Message)
	}
}

func TestLoadRejectsDuplicateConfigOptions(t *testing.T) {
	projectRoot := t.TempDir()
	themeDir := filepath.Join(projectRoot, ThemesDirName, "custom")
	if err := os.MkdirAll(filepath.Join(themeDir, "layouts"), 0o755); err != nil {
		t.Fatalf("mkdir layouts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, ThemeMetadataFile), []byte(`name: custom
config_options:
  - name: color_mode
    type: string
  - name: color_mode
    type: string
`), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layouts", "default.html"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write default layout: %v", err)
	}

	_, err := Load(projectRoot, "custom")
	if err == nil {
		t.Fatal("expected Load to fail")
	}
	themeErr, ok := AsError(err)
	if !ok {
		t.Fatalf("expected structured theme error, got %T: %v", err, err)
	}
	if !strings.Contains(themeErr.Message, `duplicate theme config option "color_mode"`) {
		t.Fatalf("unexpected error message %q", themeErr.Message)
	}
}

func TestLoadRejectsUnsupportedOptionType(t *testing.T) {
	projectRoot := t.TempDir()
	themeDir := filepath.Join(projectRoot, ThemesDirName, "custom")
	if err := os.MkdirAll(filepath.Join(themeDir, "layouts"), 0o755); err != nil {
		t.Fatalf("mkdir layouts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, ThemeMetadataFile), []byte(`name: custom
config_options:
  - name: density
    type: select
`), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layouts", "default.html"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write default layout: %v", err)
	}

	_, err := Load(projectRoot, "custom")
	if err == nil {
		t.Fatal("expected Load to fail")
	}
	themeErr, ok := AsError(err)
	if !ok {
		t.Fatalf("expected structured theme error, got %T: %v", err, err)
	}
	if !strings.Contains(themeErr.Message, `unsupported type "select"`) {
		t.Fatalf("unexpected error message %q", themeErr.Message)
	}
}

func TestLoadReadsNestedPPTXThemeContract(t *testing.T) {
	projectRoot := t.TempDir()
	themeDir := filepath.Join(projectRoot, ThemesDirName, "custom")
	if err := os.MkdirAll(filepath.Join(themeDir, "layouts"), 0o755); err != nil {
		t.Fatalf("mkdir theme dirs: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(themeDir, "pptx"), 0o755); err != nil {
		t.Fatalf("mkdir PPTX theme dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, ThemeMetadataFile), []byte("name: custom\n"), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "pptx", "theme.yaml"), []byte("slide_size: widescreen\ncolors:\n  accent: '#123456'\n"), 0o644); err != nil {
		t.Fatalf("write PPTX contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layouts", "default.html"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write default layout: %v", err)
	}

	meta, err := Load(projectRoot, "custom")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if meta.PPTX == nil || meta.PPTX.Colors["accent"] != "#123456" {
		t.Fatalf("expected nested PPTX contract, got %#v", meta.PPTX)
	}
}

func TestLoadRejectsInvalidPPTXColor(t *testing.T) {
	projectRoot := t.TempDir()
	themeDir := filepath.Join(projectRoot, ThemesDirName, "custom")
	if err := os.MkdirAll(filepath.Join(themeDir, "layouts"), 0o755); err != nil {
		t.Fatalf("mkdir theme dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, ThemeMetadataFile), []byte("name: custom\npptx:\n  colors:\n    accent: nope\n"), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layouts", "default.html"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write default layout: %v", err)
	}
	if _, err := Load(projectRoot, "custom"); err == nil || !strings.Contains(err.Error(), "six-digit hex") {
		t.Fatalf("expected invalid PPTX color error, got %v", err)
	}
}
