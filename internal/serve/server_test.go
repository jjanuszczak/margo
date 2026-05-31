package serve

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPDFExportHandlerDisabled(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/__margo/export/pdf", nil)
	rec := httptest.NewRecorder()

	pdfExportHandler(Options{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestPDFExportHandlerMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/__margo/export/pdf", nil)
	rec := httptest.NewRecorder()

	pdfExportHandler(Options{
		PDFEnabled:  true,
		GeneratePDF: func() error { return nil },
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, http.MethodGet) {
		t.Fatalf("expected Allow header to mention GET, got %q", allow)
	}
}

func TestPDFExportHandlerSuccess(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/__margo/export/pdf", nil)
	rec := httptest.NewRecorder()
	called := false
	pdfFile, err := os.CreateTemp(t.TempDir(), "deck-*.pdf")
	if err != nil {
		t.Fatalf("create temp pdf: %v", err)
	}
	if _, err := pdfFile.WriteString("%PDF-1.7"); err != nil {
		t.Fatalf("seed temp pdf: %v", err)
	}
	if err := pdfFile.Close(); err != nil {
		t.Fatalf("close temp pdf: %v", err)
	}

	pdfExportHandler(Options{
		PDFEnabled: true,
		PDFPath:    pdfFile.Name(),
		GeneratePDF: func() error {
			called = true
			return nil
		},
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !called {
		t.Fatal("expected GeneratePDF to be called")
	}
	if got := rec.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("expected application/pdf content type, got %q", got)
	}
	if !strings.Contains(rec.Body.String(), "%PDF-1.7") {
		t.Fatalf("expected response to contain pdf bytes, got %q", rec.Body.String())
	}
}

func TestPDFExportHandlerFailure(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/__margo/export/pdf", nil)
	rec := httptest.NewRecorder()

	pdfExportHandler(Options{
		PDFEnabled:  true,
		GeneratePDF: func() error { return errors.New("boom") },
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("expected failure body to mention error, got %q", rec.Body.String())
	}
}
