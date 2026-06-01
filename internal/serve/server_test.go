package serve

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall"
	"testing"
)

type stubListener struct{}

func (stubListener) Accept() (net.Conn, error) { return nil, errors.New("not implemented") }
func (stubListener) Close() error              { return nil }
func (stubListener) Addr() net.Addr            { return stubAddr("127.0.0.1:1313") }

type stubAddr string

func (a stubAddr) Network() string { return "tcp" }
func (a stubAddr) String() string  { return string(a) }

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

func TestListenForServePromptsForAlternatePort(t *testing.T) {
	originalListen := listenTCP
	defer func() { listenTCP = originalListen }()

	calls := []string{}
	listenTCP = func(addr string) (net.Listener, error) {
		calls = append(calls, addr)
		if addr == "127.0.0.1:1313" {
			return nil, syscall.EADDRINUSE
		}
		return stubListener{}, nil
	}

	var out bytes.Buffer
	listener, addr, err := listenForServe(Options{
		Input:       strings.NewReader("1414\n"),
		Output:      &out,
		Interactive: true,
	}, &out)
	if err != nil {
		t.Fatalf("listenForServe returned error: %v", err)
	}
	defer listener.Close()
	if addr != "127.0.0.1:1414" {
		t.Fatalf("expected alternate addr %q, got %q", "127.0.0.1:1414", addr)
	}
	if len(calls) != 2 || calls[0] != "127.0.0.1:1313" || calls[1] != "127.0.0.1:1414" {
		t.Fatalf("unexpected listen attempts %#v", calls)
	}
	if !strings.Contains(out.String(), "default port 1313 is unavailable") {
		t.Fatalf("expected prompt output, got %q", out.String())
	}
}

func TestListenForServeReturnsHelpfulNonInteractivePortConflict(t *testing.T) {
	originalListen := listenTCP
	defer func() { listenTCP = originalListen }()

	listenTCP = func(addr string) (net.Listener, error) {
		return nil, syscall.EADDRINUSE
	}

	var out bytes.Buffer
	_, _, err := listenForServe(Options{Output: &out}, &out)
	if err == nil {
		t.Fatal("expected port conflict error")
	}
	if !strings.Contains(err.Error(), "--port <port>") {
		t.Fatalf("expected helpful port guidance, got %v", err)
	}
}
