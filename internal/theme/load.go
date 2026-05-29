package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	ThemesDirName       = "themes"
	ThemeMetadataFile   = "theme.yaml"
	DefaultLayoutRelPath = "layouts/default.html"
	DeckLayoutRelPath    = "layouts/deck.html"
	SlideDefaultRelPath  = "layouts/slide-default.html"
)

func Load(projectRoot, themeName string) (Metadata, error) {
	if themeName == "" {
		return Metadata{}, fmt.Errorf("theme name is required")
	}

	rootDir := filepath.Join(projectRoot, ThemesDirName, themeName)
	metadataPath := filepath.Join(rootDir, ThemeMetadataFile)
	layoutPath := filepath.Join(rootDir, DefaultLayoutRelPath)
	deckLayoutPath := filepath.Join(rootDir, DeckLayoutRelPath)
	slideDefaultPath := filepath.Join(rootDir, SlideDefaultRelPath)

	raw, err := os.ReadFile(metadataPath)
	if err != nil {
		return Metadata{}, fmt.Errorf("read theme metadata %q: %w", metadataPath, err)
	}
	meta, err := parseMetadata(string(raw), metadataPath)
	if err != nil {
		return Metadata{}, err
	}
	meta.RootDir = rootDir
	meta.SlideLayouts = map[string]string{}

	if _, err := os.Stat(deckLayoutPath); err == nil {
		meta.DeckLayout = deckLayoutPath
		meta.RequiredLayout = append(meta.RequiredLayout, DeckLayoutRelPath)
	}
	if _, err := os.Stat(slideDefaultPath); err == nil {
		meta.DefaultLayout = slideDefaultPath
		meta.RequiredLayout = append(meta.RequiredLayout, SlideDefaultRelPath)
	}
	if meta.DeckLayout == "" || meta.DefaultLayout == "" {
		if _, err := os.Stat(layoutPath); err != nil {
			return Metadata{}, fmt.Errorf("missing theme layout; expected %q or the deck/slide layout pair: %w", layoutPath, err)
		}
		meta.DefaultLayout = layoutPath
		meta.RequiredLayout = append(meta.RequiredLayout, DefaultLayoutRelPath)
	} else {
		layoutEntries, err := os.ReadDir(filepath.Join(rootDir, "layouts"))
		if err != nil {
			return Metadata{}, fmt.Errorf("read theme layouts: %w", err)
		}
		for _, entry := range layoutEntries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasPrefix(name, "slide-") || !strings.HasSuffix(name, ".html") || name == "slide-default.html" {
				continue
			}
			layoutName := strings.TrimSuffix(strings.TrimPrefix(name, "slide-"), ".html")
			meta.SlideLayouts[layoutName] = filepath.Join(rootDir, "layouts", name)
		}
	}

	if meta.Name == "" {
		meta.Name = themeName
	}

	return meta, nil
}

func parseMetadata(source, path string) (Metadata, error) {
	var meta Metadata
	if err := yaml.Unmarshal([]byte(source), &meta); err != nil {
		return Metadata{}, fmt.Errorf("parse theme metadata %q: %w", path, err)
	}

	if meta.Version == "" {
		meta.Version = "0.1.0"
	}
	return meta, nil
}
