package serve

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"margo/internal/watch"
)

const DefaultPort = "1313"

var listenTCP = func(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

type Options struct {
	OpenBrowser bool
	Port        string
	Input       io.Reader
	Output      io.Writer
	Interactive bool
	PDFEnabled  bool
	PDFPath     string
	GeneratePDF func() error
}

func Start(projectRoot string, rebuild func() error, opts Options) error {
	distDir := filepath.Join(projectRoot, "dist", "html")
	output := opts.Output
	if output == nil {
		output = io.Discard
	}
	listener, addr, err := listenForServe(opts, output)
	if err != nil {
		return err
	}
	var revision atomic.Uint64
	revision.Store(1)
	defer listener.Close()

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(distDir)))
	mux.HandleFunc("/__margo/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		fmt.Fprintf(w, "%d", revision.Load())
	})
	mux.HandleFunc("/__margo/export/pdf", pdfExportHandler(opts))

	url := fmt.Sprintf("http://%s", addr)
	fmt.Fprintf(output, "serve: preview available at http://%s\n", addr)
	if opts.PDFEnabled {
		fmt.Fprintf(output, "serve: pdf export available at http://%s/__margo/export/pdf\n", addr)
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
			fmt.Fprintln(output, "serve: change detected, rebuilding")
			if err := rebuild(); err != nil {
				fmt.Fprintf(output, "serve: rebuild failed: %v\n", err)
				return nil
			}
			revision.Add(1)
			fmt.Fprintln(output, "serve: rebuild complete")
			return nil
		})
		if err != nil {
			serverErr <- err
		}
	}()

	return <-serverErr
}

func listenForServe(opts Options, output io.Writer) (net.Listener, string, error) {
	port := strings.TrimSpace(opts.Port)
	if port == "" {
		port = DefaultPort
	}
	addr := "127.0.0.1:" + port
	listener, err := listenTCP(addr)
	if err == nil {
		return listener, addr, nil
	}

	if !isAddrInUse(err) {
		return nil, "", fmt.Errorf("listen on %s: %w", addr, err)
	}
	if port != DefaultPort {
		return nil, "", fmt.Errorf("listen on %s: %w", addr, err)
	}
	if !opts.Interactive || opts.Input == nil {
		return nil, "", fmt.Errorf("listen on %s: %w (port %s is unavailable; rerun with --port <port> or start serve interactively to choose another port)", addr, err, DefaultPort)
	}

	reader := bufio.NewReader(opts.Input)
	for {
		fmt.Fprintf(output, "serve: default port %s is unavailable\n", DefaultPort)
		fmt.Fprint(output, "serve: enter another port or press return to cancel: ")

		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return nil, "", fmt.Errorf("read port selection: %w", readErr)
		}
		choice := strings.TrimSpace(line)
		if choice == "" {
			return nil, "", fmt.Errorf("serve canceled: no alternate port selected")
		}
		if !isValidPort(choice) {
			fmt.Fprintf(output, "serve: invalid port %q; enter a value between 1 and 65535\n", choice)
			if errors.Is(readErr, io.EOF) {
				return nil, "", fmt.Errorf("invalid alternate port %q", choice)
			}
			continue
		}

		addr = "127.0.0.1:" + choice
		listener, err = listenTCP(addr)
		if err == nil {
			return listener, addr, nil
		}
		if isAddrInUse(err) {
			fmt.Fprintf(output, "serve: port %s is unavailable\n", choice)
			if errors.Is(readErr, io.EOF) {
				return nil, "", fmt.Errorf("listen on %s: %w", addr, err)
			}
			continue
		}
		return nil, "", fmt.Errorf("listen on %s: %w", addr, err)
	}
}

func isAddrInUse(err error) bool {
	return errors.Is(err, syscall.EADDRINUSE) || strings.Contains(strings.ToLower(err.Error()), "address already in use")
}

func isValidPort(value string) bool {
	port, err := strconv.Atoi(value)
	return err == nil && port >= 1 && port <= 65535
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

		if opts.PDFPath == "" {
			http.Error(w, "pdf export completed but no output path is configured", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		http.ServeFile(w, r, opts.PDFPath)
	}
}
