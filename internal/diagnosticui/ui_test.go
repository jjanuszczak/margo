package diagnosticui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jjanuszczak/margo/internal/diagnostics"
)

func TestWriteReportAndInjectPanel(t *testing.T) {
	root := t.TempDir()
	htmlPath := filepath.Join(root, "index.html")
	if err := os.WriteFile(htmlPath, []byte("<html><body><main><div class=\"controls\"></div><section class=\"slide active\"><h1>Interview map</h1></section></main></body></html>"), 0o644); err != nil {
		t.Fatalf("write HTML fixture: %v", err)
	}
	report := diagnostics.Report{}
	report.Add(diagnostics.Diagnostic{
		Severity: diagnostics.SeverityWarning,
		Code:     "layout_overflow",
		Message:  "Interview map (desktop interactive) overflows bottom by 39px (descendant)",
		Meta: map[string]any{
			"slide_index": 0,
			"title":       "Interview map",
			"split_proposals": []map[string]any{{
				"boundary": "before Control and field execution",
				"command":  "margo slide split 04-customer-story --proposal 1",
			}},
		},
	})
	if err := WriteReport(root, report); err != nil {
		t.Fatalf("WriteReport() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, reportPath)); err != nil {
		t.Fatalf("diagnostics report missing: %v", err)
	}
	if err := InjectPanel(htmlPath, report); err != nil {
		t.Fatalf("InjectPanel() error = %v", err)
	}
	output, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read HTML fixture: %v", err)
	}
	for _, needle := range []string{"data-margo-diagnostics-ui", "Layout diagnostics", "Interview map", "data-margo-diagnostics", "data-margo-slide-index=\"0\"", "margo slide split 04-customer-story --proposal 1"} {
		if !strings.Contains(string(output), needle) {
			t.Fatalf("expected injected HTML to contain %q", needle)
		}
	}
}

func TestInjectPanelSkipsNonLayoutDiagnostics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.html")
	if err := os.WriteFile(path, []byte("<html>"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := InjectPanel(path, diagnostics.Report{Items: []diagnostics.Diagnostic{{Code: "asset_missing"}}}); err != nil {
		t.Fatalf("InjectPanel() error = %v", err)
	}
}

func TestInjectPanelIncludesStructuralLayoutWarnings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.html")
	if err := os.WriteFile(path, []byte("<html><body><main><div class=\"controls\"></div></main></body></html>"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	report := diagnostics.Report{Items: []diagnostics.Diagnostic{{
		Severity: diagnostics.SeverityWarning,
		Code:     "layout_structure",
		Message:  "Interview map has a structural layout issue",
		Meta:     map[string]any{"slide_index": 3, "title": "Interview map"},
	}}}
	if err := InjectPanel(path, report); err != nil {
		t.Fatalf("InjectPanel() error = %v", err)
	}
	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	for _, needle := range []string{"layout_structure", "full-width layout", "data-margo-slide-index=\"3\""} {
		if !strings.Contains(string(output), needle) {
			t.Fatalf("expected injected HTML to contain %q", needle)
		}
	}
}
