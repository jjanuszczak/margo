package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"margo/internal/deck"
)

const Filename = "manifest.yaml"

type File struct {
	Slides []string `yaml:"slides"`
}

func Load(projectRoot string) (File, bool, error) {
	path := filepath.Join(projectRoot, Filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, false, nil
		}
		return File{}, false, fmt.Errorf("read manifest %q: %w", path, err)
	}

	var parsed File
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return File{}, false, fmt.Errorf("parse manifest %q: %w", path, err)
	}
	if len(parsed.Slides) == 0 {
		return File{}, false, fmt.Errorf("%s: slides must contain at least one slide bundle id", path)
	}

	return parsed, true, nil
}

func Apply(slides []deck.Slide, file File) ([]deck.Slide, error) {
	byID := make(map[string]deck.Slide, len(slides))
	for _, slide := range slides {
		byID[slide.ID] = slide
	}

	seen := make(map[string]bool, len(file.Slides))
	ordered := make([]deck.Slide, 0, len(slides))
	for _, id := range file.Slides {
		if seen[id] {
			return nil, fmt.Errorf("manifest slide order contains duplicate slide %q", id)
		}
		slide, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("manifest references unknown slide %q", id)
		}
		seen[id] = true
		ordered = append(ordered, slide)
	}

	for _, slide := range slides {
		if !seen[slide.ID] {
			return nil, fmt.Errorf("manifest is missing slide %q", slide.ID)
		}
	}

	return ordered, nil
}
