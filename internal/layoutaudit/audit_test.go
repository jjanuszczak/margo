package layoutaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendAuditScriptKeepsDocumentAndSelector(t *testing.T) {
	document := []byte("<html><head></head><body><main><section class=\"slide\"></section></main></body></html>")
	got, err := appendAuditScript(document, Artifact{SlideSelector: "main .slide", ViewportWidth: 1920, ViewportHeight: 1080})
	if err != nil {
		t.Fatalf("appendAuditScript() error = %v", err)
	}
	if !strings.Contains(string(got), "main .slide") {
		t.Fatalf("expected selector in audit script, got %q", got)
	}
	if !strings.HasSuffix(string(got), "</body></html>") {
		t.Fatalf("expected audit script to be inserted before body close, got %q", got)
	}
}

func TestAppendAuditScriptIncludesBoundedFitPolicy(t *testing.T) {
	document := []byte("<html><head></head><body><main><section class=\"slide\"><div class=\"slide-body\">text</div></section></main></body></html>")
	got, err := appendAuditScript(document, Artifact{
		SlideSelector: "main .slide",
		Fit:           &FitPolicy{Enabled: true, MinScale: 0.65},
	})
	if err != nil {
		t.Fatalf("appendAuditScript() error = %v", err)
	}
	output := string(got)
	for _, needle := range []string{"const fitEnabled = true", "const fitMinScale = 0.650000", "region.style.zoom"} {
		if !strings.Contains(output, needle) {
			t.Fatalf("expected fitting script to contain %q, got %q", needle, output)
		}
	}
	if !strings.Contains(output, "layout_structure") || !strings.Contains(output, "two-column layout reserves an empty column") {
		t.Fatalf("expected structural layout detection in audit script, got %q", output)
	}
}

func TestApplyFitInjectsRuntimeScript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.html")
	if err := os.WriteFile(path, []byte("<html><head></head><body></body></html>"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := ApplyFit(path, FitPolicy{Enabled: true, MinScale: 0.7}); err != nil {
		t.Fatalf("ApplyFit() error = %v", err)
	}
	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if !strings.Contains(string(output), "data-margo-layout-fit") || !strings.Contains(string(output), "0.700000") {
		t.Fatalf("expected runtime fit script, got %q", output)
	}
}

func TestApplyStructuralRemediationInjectsRuntimeScript(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.html")
	if err := os.WriteFile(path, []byte("<html><body><section class=\"two-column-slide\"></section></body></html>"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := ApplyStructuralRemediation(path, true); err != nil {
		t.Fatalf("ApplyStructuralRemediation() error = %v", err)
	}
	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if !strings.Contains(string(output), "data-margo-structural-remediation") || !strings.Contains(string(output), "collapse_empty_column") {
		t.Fatalf("expected structural remediation script, got %q", output)
	}
}

func TestAppendAuditScriptRequiresBody(t *testing.T) {
	if _, err := appendAuditScript([]byte("<html></html>"), Artifact{SlideSelector: "main .slide"}); err == nil {
		t.Fatal("expected missing body error")
	}
}

func TestParseResults(t *testing.T) {
	output := []byte(`<html data-margo-layout-audit="%7B%22findings%22%3A%5B%7B%22index%22%3A1%2C%22title%22%3A%22Too%20long%22%2C%22direction%22%3A%22bottom%22%2C%22amount%22%3A24%2C%22mechanism%22%3A%22descendant%22%7D%5D%2C%22structures%22%3A%5B%7B%22index%22%3A2%2C%22title%22%3A%22Wrong%20layout%22%2C%22code%22%3A%22layout_structure%22%2C%22reason%22%3A%22empty%20column%22%7D%5D%7D"></html>`)
	parsed, err := parseResults(output)
	if err != nil {
		t.Fatalf("parseResults() error = %v", err)
	}
	if len(parsed.Findings) != 1 || parsed.Findings[0].Index != 1 || parsed.Findings[0].Amount != 24 {
		t.Fatalf("unexpected findings: %#v", parsed.Findings)
	}
	if len(parsed.Structures) != 1 || parsed.Structures[0].Code != "layout_structure" {
		t.Fatalf("unexpected structures: %#v", parsed.Structures)
	}
}

func TestParseResultsRequiresMarker(t *testing.T) {
	if _, err := parseResults([]byte("<html></html>")); err == nil {
		t.Fatal("expected missing audit marker error")
	}
}
