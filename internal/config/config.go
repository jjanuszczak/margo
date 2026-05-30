package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"margo/internal/deck"
)

const DefaultFilename = "margo.yaml"

type RawConfig struct {
	Path  string
	Bytes []byte
}

type ParseResult struct {
	Config deck.ProjectConfig
}

func LoadRaw(path string) (RawConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RawConfig{}, fmt.Errorf("read config %q: %w", path, err)
	}

	return RawConfig{
		Path:  path,
		Bytes: data,
	}, nil
}

func Parse(raw RawConfig) (ParseResult, error) {
	type fileConfig struct {
		Version string `yaml:"version"`
		Deck    struct {
			Title        string `yaml:"title"`
			Subtitle     string `yaml:"subtitle"`
			Author       string `yaml:"author"`
			Date         string `yaml:"date"`
			Description  string `yaml:"description"`
			Language     string `yaml:"language"`
			Organization string `yaml:"organization"`
			Copyright    string `yaml:"copyright"`
			Logo         string `yaml:"logo"`
			Footer       string `yaml:"footer"`
		} `yaml:"deck"`
		Theme struct {
			Name    string         `yaml:"name"`
			Options map[string]any `yaml:",inline"`
		} `yaml:"theme"`
		Outputs struct {
			HTML bool `yaml:"html"`
			PDF  bool `yaml:"pdf"`
			PPTX bool `yaml:"pptx"`
		} `yaml:"outputs"`
	}

	var parsed fileConfig
	if err := yaml.Unmarshal(raw.Bytes, &parsed); err != nil {
		return ParseResult{}, fmt.Errorf("parse yaml %q: %w", raw.Path, err)
	}

	result := ParseResult{
		Config: deck.ProjectConfig{
			Version: parsed.Version,
			Deck: deck.DeckMetadata{
				Title:        parsed.Deck.Title,
				Subtitle:     parsed.Deck.Subtitle,
				Author:       parsed.Deck.Author,
				Date:         parsed.Deck.Date,
				Description:  parsed.Deck.Description,
				Language:     parsed.Deck.Language,
				Organization: parsed.Deck.Organization,
				Copyright:    parsed.Deck.Copyright,
				Logo:         parsed.Deck.Logo,
				Footer:       parsed.Deck.Footer,
			},
			Theme: deck.ThemeSelection{
				Name:    parsed.Theme.Name,
				Options: parsed.Theme.Options,
			},
			Outputs: deck.OutputSettings{
				HTML: parsed.Outputs.HTML,
				PDF:  parsed.Outputs.PDF,
				PPTX: parsed.Outputs.PPTX,
			},
		},
	}

	if result.Config.Version == "" {
		result.Config.Version = "1"
	}
	if !parsed.Outputs.HTML && !parsed.Outputs.PDF && !parsed.Outputs.PPTX {
		result.Config.Outputs.HTML = true
	}

	if result.Config.Deck.Title == "" {
		return ParseResult{}, fmt.Errorf("%s: deck.title is required", raw.Path)
	}
	if result.Config.Theme.Name == "" {
		return ParseResult{}, fmt.Errorf("%s: theme.name is required", raw.Path)
	}

	return result, nil
}
