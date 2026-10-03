package layoutaudit

import (
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

func TestAppendAuditScriptRequiresBody(t *testing.T) {
	if _, err := appendAuditScript([]byte("<html></html>"), Artifact{SlideSelector: "main .slide"}); err == nil {
		t.Fatal("expected missing body error")
	}
}

func TestParseResults(t *testing.T) {
	output := []byte(`<html data-margo-layout-audit="%7B%22findings%22%3A%5B%7B%22index%22%3A1%2C%22title%22%3A%22Too%20long%22%2C%22direction%22%3A%22bottom%22%2C%22amount%22%3A24%2C%22mechanism%22%3A%22descendant%22%7D%5D%7D"></html>`)
	findings, err := parseResults(output)
	if err != nil {
		t.Fatalf("parseResults() error = %v", err)
	}
	if len(findings) != 1 || findings[0].Index != 1 || findings[0].Amount != 24 {
		t.Fatalf("unexpected findings: %#v", findings)
	}
}

func TestParseResultsRequiresMarker(t *testing.T) {
	if _, err := parseResults([]byte("<html></html>")); err == nil {
		t.Fatal("expected missing audit marker error")
	}
}
