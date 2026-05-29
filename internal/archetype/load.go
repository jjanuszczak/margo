package archetype

import (
	"fmt"
	"os"
	"path/filepath"

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
