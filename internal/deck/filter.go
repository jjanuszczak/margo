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
		slide.Notes = FilterNotes(slide.Notes, opts)
		filtered = append(filtered, slide)
	}
	return filtered
}

// FilterNotes applies the same draft behavior as slides. Hidden notes are
// always omitted from rendered output, while drafts remain available in serve.
func FilterNotes(notes []Note, opts FilterOptions) []Note {
	filtered := make([]Note, 0, len(notes))
	for _, note := range notes {
		if isExcludedVisibility(note.Visibility) {
			continue
		}
		if note.Draft && !opts.IncludeDrafts {
			continue
		}
		filtered = append(filtered, note)
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
