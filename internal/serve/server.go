package serve

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"

	pdfoutput "margo/internal/output/pdf"
	"margo/internal/watch"
)

const DefaultPort = "1313"

type Options struct {
	OpenBrowser bool
	PDFEnabled  bool
	GeneratePDF func() error
}

func Start(projectRoot string, rebuild func() error, opts Options) error {
	distDir := filepath.Join(projectRoot, "dist", "html")
	addr := "127.0.0.1:" + DefaultPort
	var revision atomic.Uint64
	revision.Store(1)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(distDir)))
	mux.HandleFunc("/__margo/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		fmt.Fprintf(w, "%d", revision.Load())
	})
	mux.HandleFunc("/__margo/export/pdf", pdfExportHandler(opts))

	url := fmt.Sprintf("http://%s", addr)
	fmt.Printf("serve: preview available at http://%s\n", addr)
	if opts.PDFEnabled {
		fmt.Printf("serve: pdf export available at http://%s/__margo/export/pdf\n", addr)
	}
	if opts.OpenBrowser {
		go openBrowser(url)
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- http.Serve(listener, mux)
	}()

	go func() {
		err := watch.Poll(projectRoot, 750*time.Millisecond, func() error {
			fmt.Println("serve: change detected, rebuilding")
			if err := rebuild(); err != nil {
				fmt.Printf("serve: rebuild failed: %v\n", err)
				return nil
			}
			revision.Add(1)
			fmt.Println("serve: rebuild complete")
			return nil
		})
		if err != nil {
			serverErr <- err
		}
	}()

	return <-serverErr
}

func openBrowser(url string) {
	cmd := exec.Command("open", url)
	if err := cmd.Start(); err != nil {
		fmt.Printf("serve: browser auto-open failed: %v\n", err)
	}
}

func pdfExportHandler(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !opts.PDFEnabled || opts.GeneratePDF == nil {
			http.Error(w, "pdf export is not enabled for this deck", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := opts.GeneratePDF(); err != nil {
			http.Error(w, fmt.Sprintf("pdf export failed: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "wrote %s\n", filepath.ToSlash(pdfoutput.OutputFile))
	}
}
