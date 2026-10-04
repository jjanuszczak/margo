package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateContractInfersThemeGeometryAndResponsiveProfile(t *testing.T) {
	projectRoot := t.TempDir()
	themeDir := filepath.Join(projectRoot, ThemesDirName, "custom")
	if err := os.MkdirAll(filepath.Join(themeDir, "layouts", "assets"), 0o755); err != nil {
		t.Fatalf("mkdir theme dirs: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(themeDir, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir assets dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, ThemeMetadataFile), []byte("name: custom\npptx:\n  slide_size: widescreen\n"), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layouts", "default.html"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write layout: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "assets", "theme.css"), []byte(`.slide { padding: 72px; aspect-ratio: 16 / 9; }
@media (max-width: 900px) { .slide { min-height: 70vh; } }
body { font-size: 20px; }
`), 0o644); err != nil {
		t.Fatalf("write css: %v", err)
	}

	meta, err := Load(projectRoot, "custom")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	generated, err := GenerateContract(themeDir, meta)
	if err != nil {
		t.Fatalf("GenerateContract returned error: %v", err)
	}
	if !generated {
		t.Fatal("expected contract to be generated")
	}
	updated, err := Load(projectRoot, "custom")
	if err != nil {
		t.Fatalf("Load generated theme returned error: %v", err)
	}
	if updated.Contract == nil || updated.Contract.Source != "inferred" || updated.Contract.Status != "review_required" {
		t.Fatalf("unexpected generated contract metadata: %#v", updated.Contract)
	}
	if updated.LayoutContract == nil || updated.LayoutContract.Slide.Width != 1920 || updated.LayoutContract.Layouts["default"].Reserved.Top != 72 {
		t.Fatalf("unexpected generated layout contract: %#v", updated.LayoutContract)
	}
	if updated.Responsive == nil || len(updated.Responsive.Profiles) != 2 || !updated.Responsive.Profiles[1].AllowVerticalScroll {
		t.Fatalf("unexpected generated responsive contract: %#v", updated.Responsive)
	}

	generated, err = GenerateContract(themeDir, updated)
	if err != nil {
		t.Fatalf("second GenerateContract returned error: %v", err)
	}
	if generated {
		t.Fatal("expected existing contract to remain unchanged")
	}
}

func TestGenerateContractPreservesExistingThemeMetadata(t *testing.T) {
	projectRoot := t.TempDir()
	themeDir := filepath.Join(projectRoot, ThemesDirName, "custom")
	if err := os.MkdirAll(filepath.Join(themeDir, "layouts"), 0o755); err != nil {
		t.Fatalf("mkdir theme dirs: %v", err)
	}
	source := "name: custom\ndescription: Keep this description\nconfig_options:\n  - name: color_mode\n    type: string\n    default: light\n"
	if err := os.WriteFile(filepath.Join(themeDir, ThemeMetadataFile), []byte(source), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layouts", "default.html"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write layout: %v", err)
	}
	meta, err := Load(projectRoot, "custom")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if _, err := GenerateContract(themeDir, meta); err != nil {
		t.Fatalf("GenerateContract returned error: %v", err)
	}
	updated, err := os.ReadFile(filepath.Join(themeDir, ThemeMetadataFile))
	if err != nil {
		t.Fatalf("read generated metadata: %v", err)
	}
	if !strings.Contains(string(updated), "description: Keep this description") || !strings.Contains(string(updated), "name: custom") {
		t.Fatalf("generated metadata did not preserve existing fields: %s", updated)
	}
}
