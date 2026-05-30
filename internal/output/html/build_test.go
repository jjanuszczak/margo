package html

import (
	"html/template"
	"testing"

	"margo/internal/deck"
)

func TestRenderBodyColumnsForTwoColumnLayout(t *testing.T) {
	slide := deck.Slide{
		FrontMatter: deck.FrontMatter{
			Layout: "two-column",
		},
		BodyMarkdown: "Left side\n\n<!-- column-break -->\n\nRight side",
	}

	columns := renderBodyColumns(slide, template.HTML("<p>ignored</p>"))
	if got, want := len(columns), 2; got != want {
		t.Fatalf("expected %d columns, got %d", want, got)
	}
	if string(columns[0]) == "" || string(columns[1]) == "" {
		t.Fatalf("expected both columns to render content, got %#v", columns)
	}
}

func TestRenderBodyColumnsFallsBackWhenNoMarker(t *testing.T) {
	body := template.HTML("<p>Whole body</p>")
	slide := deck.Slide{
		FrontMatter: deck.FrontMatter{
			Layout: "two-column",
		},
		BodyMarkdown: "Whole body",
	}

	columns := renderBodyColumns(slide, body)
	if got, want := len(columns), 1; got != want {
		t.Fatalf("expected %d fallback column, got %d", want, got)
	}
	if columns[0] != body {
		t.Fatalf("expected fallback body %q, got %q", body, columns[0])
	}
}
