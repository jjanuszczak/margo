package desktop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNamedNoteCreationSaveAndPreview(t *testing.T) {
	e := testEngine(t)
	snapshot := call(t, e, Request{Op: "snapshot"}).Snapshot
	slide := snapshot.Slides[0]
	file := call(t, e, Request{Op: "note", Slide: slide.ID, Name: "Speaker script", Revision: snapshot.Revision}).File
	expected := "slides/" + slide.ID + "/notes/speaker-script.md"
	if file.Path != expected {
		t.Fatalf("note path: %s", file.Path)
	}
	source := strings.Replace(file.Content, "Add notes here.", "## Opening\n\nExplain **the result**.\n\n- First point\n- Second point", 1)
	call(t, e, Request{Op: "save", Path: file.Path, Content: source, Revision: file.Revision})
	snapshot = call(t, e, Request{Op: "snapshot"}).Snapshot
	found := false
	for _, note := range snapshot.Slides[0].Notes {
		if note.Path == expected {
			found = true
		}
	}
	if !found {
		t.Fatal("named note absent from source selector snapshot")
	}
	preview := call(t, e, Request{Op: "preview", Slide: slide.ID, Notes: true, Path: expected})
	if !strings.HasSuffix(preview.Preview, "/notes.html") {
		t.Fatal(preview.Preview)
	}
	data, err := os.ReadFile(filepath.Join(e.temp, "1", "notes.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Speaker script", "<h2>Opening</h2>", "<strong>the result</strong>", "<li>First point</li>"} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("missing rendered note content %q", expected)
		}
	}
	if strings.Contains(string(data), "kind: note") {
		t.Fatal("front matter leaked into note body")
	}
	duplicate := e.Handle(Request{Protocol: Protocol, Op: "note", Slide: slide.ID, Name: "Speaker script", Revision: snapshot.Revision})
	if duplicate.Error == "" {
		t.Fatal("duplicate note replaced authored file")
	}
	disk, _ := os.ReadFile(filepath.Join(e.root, file.Path))
	if string(disk) != source {
		t.Fatal("note source changed during preview/duplicate creation")
	}
	// Theme note controls remain an explicit project option. The desktop reader
	// above works with that option disabled; normal output works when enabled.
	configPath := filepath.Join(e.root, "margo.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	configData = append(configData, []byte("\npresentation:\n  navigation:\n    notes: true\n")...)
	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		t.Fatal(err)
	}
	call(t, e, Request{Op: "preview", Slide: slide.ID})
	data, err = os.ReadFile(filepath.Join(e.temp, "2", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "<strong>the result</strong>") {
		t.Fatal("theme did not receive bundle note")
	}
	rejected := e.Handle(Request{Protocol: Protocol, Op: "preview", Slide: slide.ID, Notes: true, Path: "slides/other/notes/private.md"})
	if rejected.Error == "" {
		t.Fatal("preview accepted note outside selected slide")
	}
}
