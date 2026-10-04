package theme

type Metadata struct {
	Name            string              `yaml:"name"`
	Version         string              `yaml:"version"`
	Description     string              `yaml:"description"`
	ConfigOptions   []ConfigOption      `yaml:"config_options"`
	PPTX            *PPTXMetadata       `yaml:"pptx,omitempty"`
	Contract        *ContractMetadata   `yaml:"contract,omitempty"`
	LayoutContract  *LayoutContract     `yaml:"layout_contract,omitempty"`
	Responsive      *ResponsiveContract `yaml:"responsive,omitempty"`
	Source          *Source             `yaml:"source,omitempty"`
	RequiredLayout  []string
	RootDir         string
	DefaultLayout   string
	DeckLayout      string
	PrintDeckLayout string
	SlideLayouts    map[string]string
	Partials        map[string]string
}

type ContractMetadata struct {
	Source      string `yaml:"source,omitempty"`
	Status      string `yaml:"status,omitempty"`
	GeneratedBy string `yaml:"generated_by,omitempty"`
}

// LayoutContract describes the theme's canonical slide geometry and the
// semantic regions that future fitting and splitting phases may use.
type LayoutContract struct {
	Slide   LayoutSlideGeometry             `yaml:"slide"`
	Layouts map[string]LayoutContractLayout `yaml:"layouts"`
}

type LayoutSlideGeometry struct {
	Width  int `yaml:"width"`
	Height int `yaml:"height"`
}

type LayoutContractLayout struct {
	Reserved LayoutReservedSpace `yaml:"reserved,omitempty"`
	Regions  []LayoutRegion      `yaml:"regions,omitempty"`
}

type LayoutReservedSpace struct {
	Top    int `yaml:"top,omitempty"`
	Right  int `yaml:"right,omitempty"`
	Bottom int `yaml:"bottom,omitempty"`
	Left   int `yaml:"left,omitempty"`
}

type LayoutRegion struct {
	Name           string  `yaml:"name"`
	Role           string  `yaml:"role"`
	MaxLines       int     `yaml:"max_lines,omitempty"`
	MinFontSize    float64 `yaml:"min_font_size,omitempty"`
	MinScale       float64 `yaml:"min_scale,omitempty"`
	OverflowPolicy string  `yaml:"overflow_policy,omitempty"`
	Flexible       bool    `yaml:"flexible,omitempty"`
	AllowSplit     bool    `yaml:"allow_split,omitempty"`
}

// ResponsiveContract declares the viewport profiles a theme promises to
// render. Profiles are used by the observation-only layout audit in Phase 2.
type ResponsiveContract struct {
	Profiles []ResponsiveProfile `yaml:"profiles"`
}

type ResponsiveProfile struct {
	Name                string `yaml:"name"`
	Width               int    `yaml:"width"`
	Height              int    `yaml:"height"`
	Mode                string `yaml:"mode"`
	AllowVerticalScroll bool   `yaml:"allow_vertical_scroll,omitempty"`
}

// ResponsiveProfiles returns the declared audit profiles, preserving the
// Phase 1 desktop fallback for themes that predate the responsive contract.
func (m Metadata) ResponsiveProfiles() []ResponsiveProfile {
	if m.Responsive == nil || len(m.Responsive.Profiles) == 0 {
		return []ResponsiveProfile{{Name: "desktop", Width: 1920, Height: 1080, Mode: "fixed_canvas"}}
	}
	return append([]ResponsiveProfile(nil), m.Responsive.Profiles...)
}

// FitSettings returns the most conservative fitting bound declared by any
// theme region that explicitly opts into automatic fitting.
func (m Metadata) FitSettings() (enabled bool, minScale float64) {
	minScale = 1
	if m.LayoutContract == nil {
		return false, minScale
	}
	for _, layout := range m.LayoutContract.Layouts {
		for _, region := range layout.Regions {
			if region.OverflowPolicy != "fit" {
				continue
			}
			enabled = true
			scale := region.MinScale
			if scale <= 0 {
				scale = 0.8
			}
			if scale < minScale {
				minScale = scale
			}
		}
	}
	return enabled, minScale
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
	Type                 string `yaml:"type,omitempty"`
	Repo                 string `yaml:"repo,omitempty"`
	Ref                  string `yaml:"ref,omitempty"`
	ResolvedRef          string `yaml:"resolved_ref,omitempty"`
	ArchiveFormat        string `yaml:"archive_format,omitempty"`
	ArchiveSHA256        string `yaml:"archive_sha256,omitempty"`
	ImportedThemeName    string `yaml:"imported_theme_name,omitempty"`
	ImportedThemeVersion string `yaml:"imported_theme_version,omitempty"`
}

type ConfigOption struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"`
	Description string   `yaml:"description"`
	Required    bool     `yaml:"required"`
	Default     any      `yaml:"default"`
	Values      []string `yaml:"values"`
}
