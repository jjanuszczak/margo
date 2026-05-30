package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	ThemesDirName        = "themes"
	ThemeMetadataFile    = "theme.yaml"
	DefaultLayoutRelPath = "layouts/default.html"
	DeckLayoutRelPath    = "layouts/deck.html"
	SlideDefaultRelPath  = "layouts/slide-default.html"
)

func Load(projectRoot, themeName string) (Metadata, error) {
	if themeName == "" {
		return Metadata{}, &Error{
			Message: "theme name is required",
		}
	}

	rootDir := filepath.Join(projectRoot, ThemesDirName, themeName)
	metadataPath := filepath.Join(rootDir, ThemeMetadataFile)
	layoutPath := filepath.Join(rootDir, DefaultLayoutRelPath)
	deckLayoutPath := filepath.Join(rootDir, DeckLayoutRelPath)
	slideDefaultPath := filepath.Join(rootDir, SlideDefaultRelPath)

	raw, err := os.ReadFile(metadataPath)
	if err != nil {
		return Metadata{}, &Error{
			Path:    metadataPath,
			Message: fmt.Sprintf("read theme metadata: %v", err),
		}
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
			return Metadata{}, &Error{
				Path:    filepath.Join(rootDir, "layouts"),
				Message: fmt.Sprintf("missing theme layout; expected %q or the deck/slide layout pair", layoutPath),
			}
		}
		meta.DefaultLayout = layoutPath
		meta.RequiredLayout = append(meta.RequiredLayout, DefaultLayoutRelPath)
	} else {
		layoutEntries, err := os.ReadDir(filepath.Join(rootDir, "layouts"))
		if err != nil {
			return Metadata{}, &Error{
				Path:    filepath.Join(rootDir, "layouts"),
				Message: fmt.Sprintf("read theme layouts: %v", err),
			}
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
		return Metadata{}, &Error{
			Path:    path,
			Message: fmt.Sprintf("parse theme metadata: %v", err),
		}
	}

	if meta.Version == "" {
		meta.Version = "0.1.0"
	}
	return meta, nil
}
