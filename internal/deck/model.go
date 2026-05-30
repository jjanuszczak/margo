package deck

type ProjectConfig struct {
	Version string
	Deck    DeckMetadata
	Theme   ThemeSelection
	Outputs OutputSettings
}

type DeckMetadata struct {
	Title        string
	Subtitle     string
	Author       string
	Date         string
	Description  string
	Language     string
	Organization string
	Copyright    string
	Logo         string
	Footer       string
}

type ThemeSelection struct {
	Name    string         `yaml:"name"`
	Options map[string]any `yaml:",inline"`
}

type OutputSettings struct {
	HTML bool
	PDF  bool
	PPTX bool
}

type Model struct {
	Config   ProjectConfig
	Sections []Section
	Slides   []Slide
}

type Section struct {
	ID       string
	Title    string
	Metadata map[string]any
}

type Slide struct {
	ID         string
	BundlePath string
	Synthetic  bool
	FrontMatter
	BodyMarkdown string
	Notes        []string
	Assets       []string
}

type FrontMatter struct {
	Title          string         `yaml:"title"`
	Order          int            `yaml:"order"`
	Section        string         `yaml:"section"`
	Layout         string         `yaml:"layout"`
	Type           string         `yaml:"type"`
	Notes          any            `yaml:"notes"`
	Draft          bool           `yaml:"draft"`
	Visibility     string         `yaml:"visibility"`
	HideLogo       bool           `yaml:"hide_logo"`
	HideFooter     bool           `yaml:"hide_footer"`
	FooterText     string         `yaml:"footer_text"`
	Background     Background     `yaml:"background"`
	ImageHints     map[string]any `yaml:"image_hints"`
	ThemeOverrides map[string]any `yaml:"theme_overrides"`
}

type Background struct {
	Color   string  `yaml:"color"`
	Image   string  `yaml:"image"`
	Overlay string  `yaml:"overlay"`
	Opacity float64 `yaml:"opacity"`
}
