package render

import (
	"strings"
	"testing"

	"github.com/jjanuszczak/margo/internal/deck"
)

func TestRenderNotesPreservesMetadata(t *testing.T) {
	rendered := renderNotes([]deck.Note{{
		ID:       "speaker-script",
		Name:     "Speaker script",
		Path:     "notes/speaker-script.md",
		Order:    10,
		Draft:    true,
		Kind:     "speaker_script",
		Tags:     []string{"internal"},
		Language: "en",
		Markdown: "Open with the customer story.",
	}})
	if got, want := len(rendered), 1; got != want {
		t.Fatalf("expected %d rendered note, got %#v", want, rendered)
	}
	note := rendered[0]
	if note.ID != "speaker-script" || note.Name != "Speaker script" || note.Path != "notes/speaker-script.md" || note.Order != 10 || !note.Draft || note.Kind != "speaker_script" || len(note.Tags) != 1 || note.Language != "en" {
		t.Fatalf("rendered metadata was not preserved: %#v", note)
	}
	if !strings.Contains(string(note.Body), "Open with the customer story.") {
		t.Fatalf("expected rendered note body, got %q", note.Body)
	}
}
