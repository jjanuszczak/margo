package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	cssSlideBlock   = regexp.MustCompile(`(?s)\.slide\s*\{([^}]*)\}`)
	cssPadding      = regexp.MustCompile(`(?m)\bpadding\s*:\s*(\d+)px`)
	cssMedia        = regexp.MustCompile(`(?m)@media\s*\([^)]*max-width\s*:\s*(\d+)px`)
	cssAspectRatio  = regexp.MustCompile(`(?m)aspect-ratio\s*:\s*(\d+)\s*/\s*(\d+)`)
	cssBodyFontSize = regexp.MustCompile(`(?m)\bbody\s*\{[^}]*font-size\s*:\s*(\d+(?:\.\d+)?)px`)
)

// GenerateContract adds a provisional layout and responsive contract to a
// theme metadata file. It never overwrites an existing contract.
func GenerateContract(rootDir string, meta Metadata) (bool, error) {
	if meta.Contract != nil || meta.LayoutContract != nil || meta.Responsive != nil {
		return false, nil
	}
	metadataPath := filepath.Join(rootDir, ThemeMetadataFile)
	source, err := os.ReadFile(metadataPath)
	if err != nil {
		return false, fmt.Errorf("read theme metadata: %w", err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(source, &document); err != nil {
		return false, fmt.Errorf("parse theme metadata: %w", err)
	}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return false, fmt.Errorf("theme metadata must contain a YAML mapping")
	}

	css := readThemeCSS(rootDir)
	layout := inferLayoutContract(meta, css)
	responsive := inferResponsiveContract(meta, css)
	contract := ContractMetadata{
		Source:      "inferred",
		Status:      "review_required",
		GeneratedBy: "margo theme contract init",
	}
	if err := appendYAMLField(document.Content[0], "contract", contract); err != nil {
		return false, err
	}
	if err := appendYAMLField(document.Content[0], "layout_contract", layout); err != nil {
		return false, err
	}
	if err := appendYAMLField(document.Content[0], "responsive", responsive); err != nil {
		return false, err
	}
	output, err := yaml.Marshal(&document)
	if err != nil {
		return false, fmt.Errorf("serialize generated theme contract: %w", err)
	}
	if err := os.WriteFile(metadataPath, output, 0o644); err != nil {
		return false, fmt.Errorf("write generated theme contract: %w", err)
	}
	return true, nil
}

func inferLayoutContract(meta Metadata, css string) LayoutContract {
	width, height := inferSlideSize(meta, css)
	reserved := inferReservedSpace(css)
	fontSize := 16.0
	if match := cssBodyFontSize.FindStringSubmatch(css); len(match) == 2 {
		if parsed, err := strconv.ParseFloat(match[1], 64); err == nil {
			fontSize = parsed
		}
	}
	layoutNames := []string{"default"}
	for name := range meta.SlideLayouts {
		if name != "default" {
			layoutNames = append(layoutNames, name)
		}
	}
	sort.Strings(layoutNames[1:])
	layouts := make(map[string]LayoutContractLayout, len(layoutNames))
	for _, name := range layoutNames {
		layouts[name] = LayoutContractLayout{
			Reserved: reserved,
			Regions: []LayoutRegion{{
				Name:           "content",
				Role:           "body",
				MinFontSize:    fontSize,
				MinScale:       0.8,
				OverflowPolicy: "warn",
				Flexible:       true,
			}},
		}
	}
	return LayoutContract{Slide: LayoutSlideGeometry{Width: width, Height: height}, Layouts: layouts}
}

func inferResponsiveContract(meta Metadata, css string) ResponsiveContract {
	width, height := inferSlideSize(meta, css)
	profiles := []ResponsiveProfile{{Name: "desktop", Width: width, Height: height, Mode: "fixed_canvas"}}
	if cssMedia.MatchString(css) {
		profiles = append(profiles, ResponsiveProfile{
			Name:                "mobile",
			Width:               390,
			Height:              844,
			Mode:                "reflow",
			AllowVerticalScroll: inferVerticalScroll(css),
		})
	}
	return ResponsiveContract{Profiles: profiles}
}

func inferSlideSize(meta Metadata, css string) (int, int) {
	if meta.PPTX != nil && strings.EqualFold(meta.PPTX.SlideSize, "standard") {
		return 1280, 960
	}
	if match := cssAspectRatio.FindStringSubmatch(css); len(match) == 3 {
		width, widthErr := strconv.Atoi(match[1])
		height, heightErr := strconv.Atoi(match[2])
		if widthErr == nil && heightErr == nil && width > 0 && height > 0 {
			if width*3 == height*4 {
				return 1280, 960
			}
		}
	}
	return 1920, 1080
}

func inferReservedSpace(css string) LayoutReservedSpace {
	match := cssSlideBlock.FindStringSubmatch(css)
	if len(match) != 2 {
		return LayoutReservedSpace{}
	}
	padding := cssPadding.FindStringSubmatch(match[1])
	if len(padding) != 2 {
		return LayoutReservedSpace{}
	}
	value, err := strconv.Atoi(padding[1])
	if err != nil {
		return LayoutReservedSpace{}
	}
	return LayoutReservedSpace{Top: value, Right: value, Bottom: value, Left: value}
}

func inferVerticalScroll(css string) bool {
	lower := strings.ToLower(css)
	return strings.Contains(lower, "overflow-y: auto") ||
		strings.Contains(lower, "overflow-y: scroll") ||
		strings.Contains(lower, "min-height") ||
		strings.Contains(lower, "height: auto")
}

func readThemeCSS(rootDir string) string {
	var builder strings.Builder
	_ = filepath.WalkDir(filepath.Join(rootDir, "assets"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".css" {
			return nil
		}
		contents, readErr := os.ReadFile(path)
		if readErr == nil {
			builder.Write(contents)
			builder.WriteByte('\n')
		}
		return nil
	})
	return builder.String()
}

func appendYAMLField(root *yaml.Node, key string, value any) error {
	for index := 0; index+1 < len(root.Content); index += 2 {
		if root.Content[index].Value == key {
			return fmt.Errorf("theme metadata already contains %q", key)
		}
	}
	encoded, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode generated %q: %w", key, err)
	}
	var valueDocument yaml.Node
	if err := yaml.Unmarshal(encoded, &valueDocument); err != nil {
		return fmt.Errorf("parse generated %q: %w", key, err)
	}
	root.Content = append(root.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		valueDocument.Content[0],
	)
	return nil
}
