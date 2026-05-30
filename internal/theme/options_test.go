package theme

import (
	"strings"
	"testing"
)

func TestResolveOptionsAppliesDefaultsAndOverrides(t *testing.T) {
	meta := Metadata{
		Name: "default",
		ConfigOptions: []ConfigOption{
			{Name: "color_mode", Type: "string", Default: "light", Values: []string{"light", "dark"}},
			{Name: "typography", Type: "string", Default: "editorial", Values: []string{"editorial", "executive"}},
			{Name: "accent_color", Type: "string", Default: "#8f6f33"},
		},
	}

	resolved, err := ResolveOptions(meta, map[string]any{
		"color_mode":   "dark",
		"accent_color": "#4db6ac",
	})
	if err != nil {
		t.Fatalf("ResolveOptions returned error: %v", err)
	}

	if got := resolved["color_mode"]; got != "dark" {
		t.Fatalf("color_mode = %v, want dark", got)
	}
	if got := resolved["typography"]; got != "editorial" {
		t.Fatalf("typography = %v, want editorial default", got)
	}
	if got := resolved["accent_color"]; got != "#4db6ac" {
		t.Fatalf("accent_color = %v, want #4db6ac", got)
	}
}

func TestResolveOptionsRejectsUnknownOption(t *testing.T) {
	meta := Metadata{
		Name: "default",
		ConfigOptions: []ConfigOption{
			{Name: "color_mode", Type: "string", Default: "light"},
		},
	}

	_, err := ResolveOptions(meta, map[string]any{
		"surprise": "value",
	})
	if err == nil {
		t.Fatal("ResolveOptions returned nil error, want unknown option error")
	}
	if !strings.Contains(err.Error(), `unknown theme option "surprise"`) || !strings.Contains(err.Error(), "supported: color_mode") {
		t.Fatalf("ResolveOptions error = %v, want unknown option message", err)
	}
}

func TestResolveOptionsRejectsWrongType(t *testing.T) {
	meta := Metadata{
		Name: "default",
		ConfigOptions: []ConfigOption{
			{Name: "color_mode", Type: "string", Default: "light"},
		},
	}

	_, err := ResolveOptions(meta, map[string]any{
		"color_mode": true,
	})
	if err == nil {
		t.Fatal("ResolveOptions returned nil error, want wrong type error")
	}
	if !strings.Contains(err.Error(), "expected string") {
		t.Fatalf("ResolveOptions error = %v, want expected string", err)
	}
}

func TestResolveOptionsRejectsInvalidEnumeratedValue(t *testing.T) {
	meta := Metadata{
		Name: "default",
		ConfigOptions: []ConfigOption{
			{Name: "color_mode", Type: "string", Default: "light", Values: []string{"light", "dark"}},
		},
	}

	_, err := ResolveOptions(meta, map[string]any{
		"color_mode": "sepia",
	})
	if err == nil {
		t.Fatal("ResolveOptions returned nil error, want invalid value error")
	}
	if !strings.Contains(err.Error(), `invalid value "sepia"`) || !strings.Contains(err.Error(), "allowed: light, dark") {
		t.Fatalf("ResolveOptions error = %v, want allowed values message", err)
	}
}
