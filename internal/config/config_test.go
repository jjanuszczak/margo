package config

import (
	"strings"
	"testing"
)

func TestParseReportsMissingDeckTitleWithLine(t *testing.T) {
	raw := RawConfig{
		Path: "margo.yaml",
		Bytes: []byte(`version: 1

deck:
  description: Missing title

theme:
  name: default
`),
	}

	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected Parse to fail")
	}
	fieldErr, ok := AsFieldError(err)
	if !ok {
		t.Fatalf("expected FieldError, got %T: %v", err, err)
	}
	if fieldErr.Field != "deck.title" {
		t.Fatalf("expected field %q, got %q", "deck.title", fieldErr.Field)
	}
	if fieldErr.Line == 0 {
		t.Fatal("expected non-zero line number")
	}
	if !strings.Contains(fieldErr.Error(), "deck.title is required") {
		t.Fatalf("unexpected error text: %v", fieldErr)
	}
}

func TestParseReportsMissingThemeNameWithLine(t *testing.T) {
	raw := RawConfig{
		Path: "margo.yaml",
		Bytes: []byte(`version: 1

deck:
  title: Sample

theme:
  color_mode: dark
`),
	}

	_, err := Parse(raw)
	if err == nil {
		t.Fatal("expected Parse to fail")
	}
	fieldErr, ok := AsFieldError(err)
	if !ok {
		t.Fatalf("expected FieldError, got %T: %v", err, err)
	}
	if fieldErr.Field != "theme.name" {
		t.Fatalf("expected field %q, got %q", "theme.name", fieldErr.Field)
	}
	if fieldErr.Line == 0 {
		t.Fatal("expected non-zero line number")
	}
}
