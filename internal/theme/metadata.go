package theme

type Metadata struct {
	Name            string         `yaml:"name"`
	Version         string         `yaml:"version"`
	Description     string         `yaml:"description"`
	ConfigOptions   []ConfigOption `yaml:"config_options"`
	PPTX            *PPTXMetadata  `yaml:"pptx,omitempty"`
	Source          *Source        `yaml:"source,omitempty"`
	RequiredLayout  []string
	RootDir         string
	DefaultLayout   string
	DeckLayout      string
	PrintDeckLayout string
	SlideLayouts    map[string]string
	Partials        map[string]string
}

type PPTXMetadata struct {
	SlideSize string                `yaml:"slide_size"`
	Fonts     PPTXFonts             `yaml:"fonts"`
	Colors    map[string]string     `yaml:"colors"`
	Assets    map[string]string     `yaml:"assets"`
	Layouts   map[string]PPTXLayout `yaml:"layouts"`
}

type PPTXFonts struct {
	Heading string `yaml:"heading"`
	Body    string `yaml:"body"`
}

type PPTXLayout struct {
	Name          string  `yaml:"name"`
	ImagePosition string  `yaml:"image_position"`
	BodyX         float64 `yaml:"body_x"`
	BodyY         float64 `yaml:"body_y"`
	BodyWidth     float64 `yaml:"body_width"`
	BodyHeight    float64 `yaml:"body_height"`
	ImageWidth    float64 `yaml:"image_width"`
	ImageHeight   float64 `yaml:"image_height"`
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
