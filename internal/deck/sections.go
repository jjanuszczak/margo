package deck

import "strings"

func BuildSections(slides []Slide) []Section {
	seen := map[string]bool{}
	sections := make([]Section, 0)
	for _, slide := range slides {
		name := strings.TrimSpace(slide.Section)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		sections = append(sections, Section{
			ID:       slugSection(name),
			Title:    name,
			Metadata: map[string]any{},
		})
	}
	return sections
}

func FindSection(slides []Section, name string) (Section, bool) {
	name = strings.TrimSpace(name)
	for _, section := range slides {
		if section.Title == name {
			return section, true
		}
	}
	return Section{}, false
}

func slugSection(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "_", "-")
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}
