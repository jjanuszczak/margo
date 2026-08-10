package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type NoteOptions struct {
	ProjectRoot string
	Slide       string
	Name        string
}

// CreateNote creates a named note file inside an existing slide bundle.
func CreateNote(opts NoteOptions) (string, error) {
	if strings.TrimSpace(opts.ProjectRoot) == "" {
		return "", errors.New("project root is required")
	}
	if strings.TrimSpace(opts.Slide) == "" {
		return "", errors.New("slide bundle is required")
	}
	if strings.TrimSpace(opts.Name) == "" {
		return "", errors.New("note name is required")
	}

	slide := strings.TrimSpace(opts.Slide)
	if filepath.Base(slide) != slide || slide == "." || slide == ".." {
		return "", fmt.Errorf("slide bundle %q must be a bundle name", opts.Slide)
	}
	bundleDir := filepath.Join(opts.ProjectRoot, "slides", slide)
	if info, err := os.Stat(bundleDir); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("slide bundle does not exist: %s", bundleDir)
		}
		return "", fmt.Errorf("stat slide bundle %q: %w", bundleDir, err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("slide bundle is not a directory: %s", bundleDir)
	}
	if _, err := os.Stat(filepath.Join(bundleDir, "index.md")); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("slide bundle has no index.md: %s", bundleDir)
		}
		return "", fmt.Errorf("stat slide index in %q: %w", bundleDir, err)
	}

	slug := slugify(opts.Name)
	notesDir := filepath.Join(bundleDir, "notes")
	path := filepath.Join(notesDir, slug+".md")
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("note already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat note file %q: %w", path, err)
	}
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		return "", fmt.Errorf("create notes directory %q: %w", notesDir, err)
	}

	body := fmt.Sprintf(`---
id: %s
title: %s
order: 0
visibility: visible
draft: false
kind: note
---

Add notes here.
`, slug, humanize(slug))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", fmt.Errorf("write note file %q: %w", path, err)
	}
	return path, nil
}
