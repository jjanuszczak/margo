package theme

type Metadata struct {
	Name           string
	Version        string
	Description    string
	ConfigOptions  []ConfigOption
	RequiredLayout []string
	RootDir        string
	DefaultLayout  string
	DeckLayout     string
	SlideLayouts   map[string]string
}

type ConfigOption struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Default     any
}
