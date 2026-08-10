package deck

type ProjectConfig struct {
	Version      string
	Deck         DeckMetadata
	Theme        ThemeSelection
	Outputs      OutputSettings
	Presentation PresentationSettings
	Snippets     SnippetSettings
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

type PresentationSettings struct {
	Navigation NavigationSettings
}

type NavigationSettings struct {
	Notes bool
}

type SnippetSettings struct {
	Head    string
	BodyEnd string
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
	Notes        []Note
	Assets       []string
}

// Note is Markdown material associated with a slide but separate from its
// visible slide content. Name is derived from its bundle filename when a note
// file does not declare front matter, while the legacy inline note field uses
// the stable "Notes" name.
type Note struct {
	ID         string
	Name       string
	Path       string
	Markdown   string
	Order      int
	Visibility string
	Draft      bool
	Kind       string
	Tags       []string
	Language   string
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
