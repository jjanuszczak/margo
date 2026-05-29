package deck

import (
	"fmt"
	"strings"
)

func ApplySectionDividers(slides []Slide) []Slide {
	if len(slides) == 0 {
		return nil
	}

	result := make([]Slide, 0, len(slides)+len(slides)/2)
	prevSection := ""
	sectionCounts := map[string]int{}

	for _, slide := range slides {
		sectionTitle := strings.TrimSpace(slide.Section)
		if shouldInsertSectionDivider(prevSection, sectionTitle, slide) {
			sectionCounts[sectionTitle]++
			result = append(result, syntheticSectionSlide(sectionTitle, sectionCounts[sectionTitle]))
		}
		result = append(result, slide)
		if sectionTitle != "" {
			prevSection = sectionTitle
		}
	}

	return result
}

func shouldInsertSectionDivider(prevSection, currentSection string, slide Slide) bool {
	if currentSection == "" {
		return false
	}
	if isExplicitSectionSlide(slide) {
		return false
	}
	return prevSection != currentSection
}

func isExplicitSectionSlide(slide Slide) bool {
	return resolveStructuredLayoutName(slide) == "section"
}

func syntheticSectionSlide(sectionTitle string, ordinal int) Slide {
	sectionID := slugSection(sectionTitle)
	return Slide{
		ID:         fmt.Sprintf("section-%s-%d", sectionID, ordinal),
		Synthetic:  true,
		BundlePath: "",
		FrontMatter: FrontMatter{
			Title:   sectionTitle,
			Section: sectionTitle,
			Layout:  "section",
			Type:    "section",
		},
		BodyMarkdown: "# " + sectionTitle,
	}
}

func resolveStructuredLayoutName(slide Slide) string {
	if strings.TrimSpace(slide.Layout) != "" {
		return strings.TrimSpace(slide.Layout)
	}
	if strings.TrimSpace(slide.Type) != "" {
		return strings.TrimSpace(slide.Type)
	}
	return "default"
}
