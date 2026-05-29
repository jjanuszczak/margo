package serve

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"

	"margo/internal/watch"
)

const DefaultPort = "1313"

type Options struct {
	OpenBrowser bool
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

	url := fmt.Sprintf("http://%s", addr)
	fmt.Printf("serve: preview available at http://%s\n", addr)
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
