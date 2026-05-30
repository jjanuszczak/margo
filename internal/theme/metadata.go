package theme

type Metadata struct {
	Name           string         `yaml:"name"`
	Version        string         `yaml:"version"`
	Description    string         `yaml:"description"`
	ConfigOptions  []ConfigOption `yaml:"config_options"`
	RequiredLayout []string
	RootDir        string
	DefaultLayout  string
	DeckLayout     string
	SlideLayouts   map[string]string
}

type ConfigOption struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Default     any    `yaml:"default"`
	Values      []string `yaml:"values"`
}
