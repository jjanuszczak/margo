package deck

import "strings"

type FilterOptions struct {
	IncludeDrafts bool
}

func FilterSlides(slides []Slide, opts FilterOptions) []Slide {
	filtered := make([]Slide, 0, len(slides))
	for _, slide := range slides {
		if isExcludedVisibility(slide.Visibility) {
			continue
		}
		if slide.Draft && !opts.IncludeDrafts {
			continue
		}
		filtered = append(filtered, slide)
	}
	return filtered
}

func isExcludedVisibility(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "hidden", "hide", "skip", "skipped":
		return true
	default:
		return false
	}
}
