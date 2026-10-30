package desktop

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jjanuszczak/margo/internal/scaffold"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	root := filepath.Join(t.TempDir(), "deck")
	if err := scaffold.CreateDeck(scaffold.DeckOptions{Name: "Test", TargetDir: root}); err != nil {
		t.Fatal(err)
	}
	return &Engine{root: root, temp: t.TempDir(), origin: "http://127.0.0.1:1/token/"}
}
func call(t *testing.T, e *Engine, r Request) Response {
	t.Helper()
	r.Protocol = Protocol
	response := e.Handle(r)
	if response.Error != "" {
		t.Fatal(response.Error)
	}
	return response
}
func TestSourceSavesAreLosslessAndRejectConcurrentEdits(t *testing.T) {
	e := testEngine(t)
	snapshot := call(t, e, Request{Op: "snapshot"}).Snapshot
	path := snapshot.Slides[0].Path
	source := "---\n# preserve comment\ntitle: A\nunknown: [one, two]\norder: 1\n---\n\nBody  \n"
	if err := os.WriteFile(filepath.Join(e.root, path), []byte(source), 0o640); err != nil {
		t.Fatal(err)
	}
	file := call(t, e, Request{Op: "read", Path: path}).File
	os.Chmod(filepath.Join(e.root, path), 0o640)
	edited := strings.Replace(source, "Body", "Changed", 1)
	saved := call(t, e, Request{Op: "save", Path: path, Content: edited, Revision: file.Revision}).File
	if saved.Content != edited {
		t.Fatal("save changed unrelated source")
	}
	info, _ := os.Stat(filepath.Join(e.root, path))
	if info.Mode().Perm() != 0o640 {
		t.Fatal("save changed permissions")
	}
	external := edited + "agent edit\n"
	os.WriteFile(filepath.Join(e.root, path), []byte(external), 0o640)
	rejected := e.Handle(Request{Protocol: Protocol, Op: "save", Path: path, Revision: saved.Revision, Content: "overwrite"})
	if !rejected.Conflict || rejected.Error == "" || rejected.File.Content != external {
		t.Fatalf("missing conflict: %+v", rejected)
	}
	data, _ := os.ReadFile(filepath.Join(e.root, path))
	if string(data) != external {
		t.Fatal("external work overwritten")
	}
}
func TestPathsAndProtocolAreScoped(t *testing.T) {
	e := testEngine(t)
	for _, path := range []string{"../secret.md", "/tmp/secret.md", "dist/html/index.html", "themes/default/theme.yaml"} {
		result := e.Handle(Request{Protocol: Protocol, Op: "read", Path: path})
		if result.Error == "" {
			t.Fatalf("accepted %s", path)
		}
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(e.root, "slides", "alias")); err != nil {
		t.Fatal(err)
	}
	if _, err := confined(e.root, "slides/alias/test.md"); err == nil {
		t.Fatal("followed symlink")
	}
	if result := e.Handle(Request{Protocol: 99, Op: "hello"}); result.Error == "" {
		t.Fatal("accepted wrong protocol")
	}
}
func TestStructuralChangesRequireCurrentProjectRevision(t *testing.T) {
	e := testEngine(t)
	called := false
	e.callbacks.Mutate = func(string, Request) (string, error) { called = true; return "", nil }
	snapshot := call(t, e, Request{Op: "snapshot"}).Snapshot
	os.WriteFile(filepath.Join(e.root, "shared.md"), []byte("external"), 0o644)
	result := e.Handle(Request{Protocol: Protocol, Op: "delete", Slide: snapshot.Slides[0].ID, Revision: snapshot.Revision})
	if !result.Conflict || called {
		t.Fatal("mutation ran against stale project")
	}
}
func TestPreviewUsesCanonicalRendererWithoutTouchingProject(t *testing.T) {
	for _, fixture := range []string{"reference-deck", "benchmark-deck", "omarchy-theme-showcase", "arca-investor-memo"} {
		t.Run(fixture, func(t *testing.T) {
			root, err := filepath.Abs(filepath.Join("..", "..", "examples", fixture))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(root); os.IsNotExist(err) {
				t.Skip("optional dogfood fixture unavailable")
			}
			if _, err := os.Stat(filepath.Join(root, "margo.yaml")); os.IsNotExist(err) {
				t.Skip("optional dogfood fixture config unavailable")
			}
			e := &Engine{root: root, temp: t.TempDir(), origin: "http://127.0.0.1:1/token/"}
			before, err := projectRevision(root)
			if err != nil {
				t.Fatal(err)
			}
			snapshot := call(t, e, Request{Op: "snapshot"}).Snapshot
			response := call(t, e, Request{Op: "preview", Slide: snapshot.Slides[0].ID})
			if !strings.Contains(response.Preview, "/1/preview.html") {
				t.Fatal(response.Preview)
			}
			data, err := os.ReadFile(filepath.Join(e.temp, "1", "index.html"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "<html") {
				t.Fatal("not HTML")
			}
			call(t, e, Request{Op: "preview", Presentation: true})
			after, err := projectRevision(root)
			if err != nil {
				t.Fatal(err)
			}
			if before != after {
				t.Fatal("preview changed authored files")
			}
		})
	}
}
func TestNotesMediaAndPreviewFailure(t *testing.T) {
	e := testEngine(t)
	s := call(t, e, Request{Op: "snapshot"}).Snapshot
	slide := s.Slides[0]
	note := call(t, e, Request{Op: "note", Slide: slide.ID, Name: "Research", Revision: s.Revision}).File
	if !strings.Contains(note.Path, "/notes/") {
		t.Fatal(note.Path)
	}
	s = call(t, e, Request{Op: "snapshot"}).Snapshot
	media := filepath.Join(t.TempDir(), "image with spaces.svg")
	os.WriteFile(media, []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), 0o644)
	imported := call(t, e, Request{Op: "media", Slide: slide.ID, Source: media, Revision: s.Revision})
	if !strings.Contains(imported.Markdown, "media-") {
		t.Fatal(imported.Markdown)
	}
	call(t, e, Request{Op: "preview", Slide: slide.ID})
	os.WriteFile(filepath.Join(e.root, slide.Path), []byte("---\ntitle: [broken YAML\n---\nBroken"), 0o644)
	result := e.Handle(Request{Protocol: Protocol, Op: "preview", Slide: slide.ID})
	if result.Error == "" {
		t.Fatal("bad layout rendered")
	}
	if _, err := os.Stat(filepath.Join(e.temp, "1", "index.html")); err != nil {
		t.Fatal("lost last good preview")
	}
}
func TestProtocolServerIsReadOnlyAndClosesOnEOF(t *testing.T) {
	e := testEngine(t)
	inputReader, inputWriter := io.Pipe()
	outputReader, outputWriter := io.Pipe()
	done := make(chan error, 1)
	go func() {
		defer inputReader.Close()
		done <- Run(inputReader, outputWriter, Callbacks{})
		outputWriter.Close()
	}()
	encoder := json.NewEncoder(inputWriter)
	decoder := json.NewDecoder(outputReader)
	exchange := func(r Request) Response {
		t.Helper()
		r.Protocol = Protocol
		if err := encoder.Encode(r); err != nil {
			t.Fatal(err)
		}
		var response Response
		if err := decoder.Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.Error != "" {
			t.Fatal(response.Error)
		}
		return response
	}
	opened := exchange(Request{Op: "open", Root: e.root})
	preview := exchange(Request{Op: "preview", Slide: opened.Snapshot.Slides[0].ID})
	client := http.Client{Timeout: 3 * time.Second}
	for _, tc := range []struct {
		method, url string
		status      int
	}{{"GET", preview.Preview, 200}, {"POST", preview.Preview, 405}, {"GET", strings.Split(preview.Preview, "/")[0] + "//" + strings.Split(preview.Preview, "/")[2] + "/missing/index.html", 404}} {
		request, _ := http.NewRequest(tc.method, tc.url, nil)
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != tc.status {
			t.Fatalf("%s got %d", tc.url, response.StatusCode)
		}
	}
	inputWriter.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("engine did not stop on EOF")
	}
	if _, err := client.Get(preview.Preview); err == nil {
		t.Fatal("listener stayed alive")
	}
}
