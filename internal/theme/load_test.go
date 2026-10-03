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

func TestLoadReadsLayoutAndResponsiveContracts(t *testing.T) {
	projectRoot := t.TempDir()
	themeDir := filepath.Join(projectRoot, ThemesDirName, "custom")
	if err := os.MkdirAll(filepath.Join(themeDir, "layouts"), 0o755); err != nil {
		t.Fatalf("mkdir theme dirs: %v", err)
	}
	metadata := `name: custom
layout_contract:
  slide:
    width: 1920
    height: 1080
  layouts:
    default:
      reserved:
        top: 72
      regions:
        - name: content
          role: body
          min_font_size: 18
          min_scale: 0.8
          overflow_policy: warn
responsive:
  profiles:
    - name: desktop
      width: 1920
      height: 1080
      mode: fixed_canvas
    - name: mobile
      width: 390
      height: 844
      mode: reflow
      allow_vertical_scroll: true
`
	if err := os.WriteFile(filepath.Join(themeDir, ThemeMetadataFile), []byte(metadata), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layouts", "default.html"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write default layout: %v", err)
	}

	meta, err := Load(projectRoot, "custom")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if meta.LayoutContract == nil || meta.LayoutContract.Slide.Width != 1920 {
		t.Fatalf("expected layout contract, got %#v", meta.LayoutContract)
	}
	if meta.Responsive == nil || len(meta.Responsive.Profiles) != 2 || !meta.Responsive.Profiles[1].AllowVerticalScroll {
		t.Fatalf("expected responsive profiles, got %#v", meta.Responsive)
	}
}

func TestLoadRejectsInvalidResponsiveProfile(t *testing.T) {
	projectRoot := t.TempDir()
	themeDir := filepath.Join(projectRoot, ThemesDirName, "custom")
	if err := os.MkdirAll(filepath.Join(themeDir, "layouts"), 0o755); err != nil {
		t.Fatalf("mkdir theme dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, ThemeMetadataFile), []byte(`name: custom
responsive:
  profiles:
    - name: mobile
      width: 390
      height: 844
      mode: elastic
`), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layouts", "default.html"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write default layout: %v", err)
	}
	if _, err := Load(projectRoot, "custom"); err == nil || !strings.Contains(err.Error(), `unsupported mode "elastic"`) {
		t.Fatalf("expected invalid responsive mode error, got %v", err)
	}
}

func TestLoadRejectsInvalidLayoutRegionPolicy(t *testing.T) {
	projectRoot := t.TempDir()
	themeDir := filepath.Join(projectRoot, ThemesDirName, "custom")
	if err := os.MkdirAll(filepath.Join(themeDir, "layouts"), 0o755); err != nil {
		t.Fatalf("mkdir theme dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, ThemeMetadataFile), []byte(`name: custom
layout_contract:
  slide:
    width: 1920
    height: 1080
  layouts:
    default:
      regions:
        - name: content
          overflow_policy: teleport
`), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layouts", "default.html"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write default layout: %v", err)
	}
	if _, err := Load(projectRoot, "custom"); err == nil || !strings.Contains(err.Error(), `unsupported overflow_policy "teleport"`) {
		t.Fatalf("expected invalid overflow policy error, got %v", err)
	}
}
