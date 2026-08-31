package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	ThemesDirName          = "themes"
	ThemeMetadataFile      = "theme.yaml"
	DefaultLayoutRelPath   = "layouts/default.html"
	DeckLayoutRelPath      = "layouts/deck.html"
	PrintDeckLayoutRelPath = "layouts/print-deck.html"
	SlideDefaultRelPath    = "layouts/slide-default.html"
)

func Load(projectRoot, themeName string) (Metadata, error) {
	if themeName == "" {
		return Metadata{}, &Error{
			Message: "theme name is required",
		}
	}

	rootDir := filepath.Join(projectRoot, ThemesDirName, themeName)
	return loadFromRootDir(rootDir, themeName)
}

func loadFromRootDir(rootDir, themeName string) (Metadata, error) {
	metadataPath := filepath.Join(rootDir, ThemeMetadataFile)
	layoutPath := filepath.Join(rootDir, DefaultLayoutRelPath)
	deckLayoutPath := filepath.Join(rootDir, DeckLayoutRelPath)
	printDeckLayoutPath := filepath.Join(rootDir, PrintDeckLayoutRelPath)
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
	if meta.PPTX == nil {
		contractPath := filepath.Join(rootDir, "pptx", "theme.yaml")
		if contractRaw, contractErr := os.ReadFile(contractPath); contractErr == nil {
			var contract PPTXMetadata
			if err := yaml.Unmarshal(contractRaw, &contract); err != nil {
				return Metadata{}, &Error{Path: contractPath, Message: fmt.Sprintf("parse PPTX theme contract: %v", err)}
			}
			if err := validatePPTXMetadata(&contract, contractPath); err != nil {
				return Metadata{}, err
			}
			meta.PPTX = &contract
		} else if !os.IsNotExist(contractErr) {
			return Metadata{}, &Error{Path: contractPath, Message: fmt.Sprintf("read PPTX theme contract: %v", contractErr)}
		}
	}
	meta.RootDir = rootDir
	meta.SlideLayouts = map[string]string{}
	meta.Partials = map[string]string{}

	deckLayoutExists := false
	if _, err := os.Stat(deckLayoutPath); err == nil {
		deckLayoutExists = true
		meta.DeckLayout = deckLayoutPath
		meta.RequiredLayout = append(meta.RequiredLayout, DeckLayoutRelPath)
	}
	if _, err := os.Stat(printDeckLayoutPath); err == nil {
		meta.PrintDeckLayout = printDeckLayoutPath
	}
	slideDefaultExists := false
	if _, err := os.Stat(slideDefaultPath); err == nil {
		slideDefaultExists = true
		meta.DefaultLayout = slideDefaultPath
		meta.RequiredLayout = append(meta.RequiredLayout, SlideDefaultRelPath)
	}
	if deckLayoutExists != slideDefaultExists {
		missingPath := slideDefaultPath
		missingRel := SlideDefaultRelPath
		if !deckLayoutExists {
			missingPath = deckLayoutPath
			missingRel = DeckLayoutRelPath
		}
		return Metadata{}, &Error{
			Path:    missingPath,
			Message: fmt.Sprintf("incomplete theme layout contract: %q requires %q", filepath.Base(missingPath), missingRel),
		}
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
	meta.Partials, err = discoverPartials(filepath.Join(rootDir, "partials"))
	if err != nil {
		return Metadata{}, &Error{
			Path:    filepath.Join(rootDir, "partials"),
			Message: fmt.Sprintf("read theme partials: %v", err),
		}
	}

	return meta, nil
}

func discoverPartials(root string) (map[string]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	partials := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".html" {
			continue
		}
		partialName := strings.TrimSuffix(name, ".html")
		partials[partialName] = filepath.Join(root, name)
	}
	return partials, nil
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
	if err := validateMetadata(meta, path); err != nil {
		return Metadata{}, err
	}
	return meta, nil
}

func validateMetadata(meta Metadata, path string) error {
	if err := validatePPTXMetadata(meta.PPTX, path); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, option := range meta.ConfigOptions {
		name := strings.TrimSpace(option.Name)
		if name == "" {
			return &Error{
				Path:    path,
				Message: "theme config option name is required",
			}
		}
		if _, ok := seen[name]; ok {
			return &Error{
				Path:    path,
				Message: fmt.Sprintf("duplicate theme config option %q", name),
			}
		}
		seen[name] = struct{}{}

		optionType := option.Type
		if optionType == "" {
			optionType = "string"
		}
		switch optionType {
		case "string", "bool", "number":
		default:
			return &Error{
				Path:    path,
				Message: fmt.Sprintf("theme config option %q has unsupported type %q", name, option.Type),
			}
		}
		if len(option.Values) > 0 && optionType != "string" {
			return &Error{
				Path:    path,
				Message: fmt.Sprintf("theme config option %q values are only supported for string options", name),
			}
		}
		valueSeen := map[string]struct{}{}
		for _, value := range option.Values {
			if strings.TrimSpace(value) == "" {
				return &Error{
					Path:    path,
					Message: fmt.Sprintf("theme config option %q contains an empty allowed value", name),
				}
			}
			if _, ok := valueSeen[value]; ok {
				return &Error{
					Path:    path,
					Message: fmt.Sprintf("theme config option %q contains duplicate allowed value %q", name, value),
				}
			}
			valueSeen[value] = struct{}{}
		}
		if option.Default != nil {
			if _, err := normalizeOptionValue(option, option.Default); err != nil {
				return &Error{
					Path:    path,
					Message: fmt.Sprintf("theme config option %q default: %v", name, err),
				}
			}
		}
	}
	return nil
}

func validatePPTXMetadata(pptx *PPTXMetadata, path string) error {
	if pptx == nil {
		return nil
	}
	switch strings.TrimSpace(pptx.SlideSize) {
	case "", "widescreen", "standard":
	default:
		return &Error{Path: path, Message: fmt.Sprintf("theme PPTX slide_size %q is unsupported; use widescreen or standard", pptx.SlideSize)}
	}
	for name, value := range pptx.Colors {
		value = strings.TrimPrefix(strings.TrimSpace(value), "#")
		if len(value) != 6 {
			return &Error{Path: path, Message: fmt.Sprintf("theme PPTX color %q must be a six-digit hex value", name)}
		}
		for _, char := range value {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
				return &Error{Path: path, Message: fmt.Sprintf("theme PPTX color %q must be a six-digit hex value", name)}
			}
		}
	}
	for name := range pptx.Layouts {
		if strings.TrimSpace(name) == "" {
			return &Error{Path: path, Message: "theme PPTX layout names are required"}
		}
	}
	return nil
}
