package archetype

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const (
	ArchetypesDirName = "archetypes"
	MetadataFileName  = "archetype.yaml"
)

func Load(projectRoot, archetypeName string) (Metadata, error) {
	if archetypeName == "" {
		archetypeName = "default"
	}

	path := filepath.Join(projectRoot, ArchetypesDirName, archetypeName, MetadataFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		return Metadata{}, fmt.Errorf("read archetype metadata %q: %w", path, err)
	}

	var meta Metadata
	if err := yaml.Unmarshal(raw, &meta); err != nil {
		return Metadata{}, fmt.Errorf("parse archetype metadata %q: %w", path, err)
	}

	if meta.Name == "" {
		meta.Name = archetypeName
	}
	if meta.DefaultLayout == "" {
		meta.DefaultLayout = "content"
	}
	if meta.DefaultType == "" {
		meta.DefaultType = "basic"
	}

	return meta, nil
}

func List(projectRoot string) ([]Metadata, error) {
	root := filepath.Join(projectRoot, ArchetypesDirName)
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read archetypes directory %q: %w", root, err)
	}

	result := make([]Metadata, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		meta, err := Load(projectRoot, entry.Name())
		if err != nil {
			return nil, err
		}
		result = append(result, meta)
	}

	sort.Slice(result, func(i, j int) bool {
		left := archetypeSortKey(result[i].Name)
		right := archetypeSortKey(result[j].Name)
		if left != right {
			return left < right
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}

func archetypeSortKey(name string) string {
	switch name {
	case "default":
		return "0-default"
	case "title":
		return "1-title"
	case "section":
		return "2-section"
	default:
		return "9-" + name
	}
}
