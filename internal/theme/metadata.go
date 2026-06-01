package theme

type Metadata struct {
	Name            string         `yaml:"name"`
	Version         string         `yaml:"version"`
	Description     string         `yaml:"description"`
	ConfigOptions   []ConfigOption `yaml:"config_options"`
	Source          *Source        `yaml:"source,omitempty"`
	RequiredLayout  []string
	RootDir         string
	DefaultLayout   string
	DeckLayout      string
	PrintDeckLayout string
	SlideLayouts    map[string]string
}

type Source struct {
	Type        string `yaml:"type,omitempty"`
	Repo        string `yaml:"repo,omitempty"`
	Ref         string `yaml:"ref,omitempty"`
	ResolvedRef string `yaml:"resolved_ref,omitempty"`
}

type ConfigOption struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"`
	Description string   `yaml:"description"`
	Required    bool     `yaml:"required"`
	Default     any      `yaml:"default"`
	Values      []string `yaml:"values"`
}
