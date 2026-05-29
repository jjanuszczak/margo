package archetype

type Metadata struct {
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	DefaultLayout string `yaml:"default_layout"`
	DefaultType   string `yaml:"default_type"`
}
