package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jjanuszczak/margo/internal/deck"
	"gopkg.in/yaml.v3"
)

const DefaultFilename = "margo.yaml"

type RawConfig struct {
	Path  string
	Bytes []byte
}

type ParseResult struct {
	Config deck.ProjectConfig
}

type FieldError struct {
	Path    string
	Field   string
	Line    int
	Message string
}

func (e *FieldError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d: %s", e.Path, e.Line, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
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

// SetThemeName updates only theme.name and writes the config atomically. It
// preserves all other fields, including theme options.
func SetThemeName(path, name string) error {
	if name == "" {
		return errors.New("theme name is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %q: %w", path, err)
	}

	var document yaml.Node
	if err := yaml.Unmarshal(raw, &document); err != nil {
		return fmt.Errorf("parse yaml %q: %w", path, err)
	}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("config %q must contain a YAML mapping", path)
	}
	root := document.Content[0]
	themeNode := findMapValue(root, "theme")
	if themeNode == nil || themeNode.Kind != yaml.MappingNode {
		return fmt.Errorf("config %q is missing a theme mapping", path)
	}
	nameNode := findMapValue(themeNode, "name")
	if nameNode == nil {
		themeNode.Content = append(themeNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "name"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name},
		)
	} else {
		nameNode.Kind = yaml.ScalarNode
		nameNode.Tag = "!!str"
		nameNode.Value = name
		nameNode.Content = nil
	}

	updated, err := yaml.Marshal(&document)
	if err != nil {
		return fmt.Errorf("marshal config %q: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat config %q: %w", path, err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".margo-config-*")
	if err != nil {
		return fmt.Errorf("create config update: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(info.Mode()); err != nil {
		temporary.Close()
		return fmt.Errorf("set config permissions: %w", err)
	}
	if _, err := temporary.Write(updated); err != nil {
		temporary.Close()
		return fmt.Errorf("write config update: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close config update: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace config %q: %w", path, err)
	}
	return nil
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
			PNG  bool `yaml:"png"`
			PPTX bool `yaml:"pptx"`
		} `yaml:"outputs"`
		Presentation struct {
			Navigation struct {
				Notes bool `yaml:"notes"`
			} `yaml:"navigation"`
		} `yaml:"presentation"`
		Snippets struct {
			Head    string `yaml:"head"`
			BodyEnd string `yaml:"body_end"`
		} `yaml:"snippets"`
	}

	var parsed fileConfig
	if err := yaml.Unmarshal(raw.Bytes, &parsed); err != nil {
		return ParseResult{}, fmt.Errorf("parse yaml %q: %w", raw.Path, err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(raw.Bytes, &root); err != nil {
		return ParseResult{}, fmt.Errorf("parse yaml node %q: %w", raw.Path, err)
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
				PNG:  parsed.Outputs.PNG,
				PPTX: parsed.Outputs.PPTX,
			},
			Presentation: deck.PresentationSettings{
				Navigation: deck.NavigationSettings{Notes: parsed.Presentation.Navigation.Notes},
			},
			Snippets: deck.SnippetSettings{
				Head:    parsed.Snippets.Head,
				BodyEnd: parsed.Snippets.BodyEnd,
			},
		},
	}

	if result.Config.Version == "" {
		result.Config.Version = "1"
	}
	if !parsed.Outputs.HTML && !parsed.Outputs.PDF && !parsed.Outputs.PNG && !parsed.Outputs.PPTX {
		result.Config.Outputs.HTML = true
	}

	if result.Config.Deck.Title == "" {
		return ParseResult{}, &FieldError{
			Path:    raw.Path,
			Field:   "deck.title",
			Line:    findFieldLine(&root, "deck", "title"),
			Message: "deck.title is required",
		}
	}
	if result.Config.Theme.Name == "" {
		return ParseResult{}, &FieldError{
			Path:    raw.Path,
			Field:   "theme.name",
			Line:    findFieldLine(&root, "theme", "name"),
			Message: "theme.name is required",
		}
	}

	return result, nil
}

func findFieldLine(root *yaml.Node, path ...string) int {
	if root == nil {
		return 0
	}

	node := root
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	for _, segment := range path {
		if node.Kind != yaml.MappingNode {
			return node.Line
		}
		next := findMapValue(node, segment)
		if next == nil {
			keyNode := findMapKey(node, segment)
			if keyNode != nil {
				return keyNode.Line
			}
			return node.Line
		}
		node = next
	}
	if node.Line > 0 {
		return node.Line
	}
	return 0
}

func findMapKey(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i]
		}
	}
	return nil
}

func findMapValue(node *yaml.Node, key string) *yaml.Node {
	keyNode := findMapKey(node, key)
	if keyNode == nil {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i] == keyNode {
			return node.Content[i+1]
		}
	}
	return nil
}

func AsFieldError(err error) (*FieldError, bool) {
	var fieldErr *FieldError
	if errors.As(err, &fieldErr) {
		return fieldErr, true
	}
	return nil, false
}
