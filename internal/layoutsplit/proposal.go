// Package layoutsplit finds safe, proposal-only semantic boundaries for dense slides.
package layoutsplit

import (
	"regexp"
	"sort"
	"strings"

	"github.com/jjanuszczak/margo/internal/deck"
)

type Proposal struct {
	AfterLine      int    `json:"after_line"`
	Boundary       string `json:"boundary"`
	Confidence     string `json:"confidence"`
	Reason         string `json:"reason"`
	EstimatedParts int    `json:"estimated_parts"`
}

var headingPattern = regexp.MustCompile(`^#{1,3}\s+(.+?)\s*#*\s*$`)

// Propose returns explicit or heading-based split boundaries without changing
// the authored slide. The result is intentionally conservative and limited to
// boundaries that preserve complete Markdown sections.
func Propose(slide deck.Slide) []Proposal {
	lines := strings.Split(strings.ReplaceAll(slide.BodyMarkdown, "\r\n", "\n"), "\n")
	if len(lines) < 3 {
		return nil
	}

	var proposals []Proposal
	for index, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.EqualFold(line, "<!-- margo: split -->") || strings.EqualFold(line, "<!-- margo-split -->") {
			if index > 0 && index < len(lines)-1 {
				proposals = append(proposals, Proposal{
					AfterLine:      index,
					Boundary:       "explicit split marker",
					Confidence:     "explicit",
					Reason:         "The author marked this boundary as safe for splitting.",
					EstimatedParts: 2,
				})
			}
			continue
		}
		match := headingPattern.FindStringSubmatch(line)
		if len(match) != 2 || index < 2 || index >= len(lines)-1 {
			continue
		}
		level := len(line) - len(strings.TrimLeft(line, "#"))
		confidence := "possible"
		if level == 2 {
			confidence = "recommended"
		}
		proposals = append(proposals, Proposal{
			AfterLine:      index,
			Boundary:       "before " + strings.TrimSpace(match[1]),
			Confidence:     confidence,
			Reason:         "Keeps the preceding and following sections intact.",
			EstimatedParts: 2,
		})
	}

	if len(proposals) == 0 {
		return nil
	}
	center := len(lines) / 2
	sort.SliceStable(proposals, func(i, j int) bool {
		rank := func(value string) int {
			switch value {
			case "explicit":
				return 0
			case "recommended":
				return 1
			default:
				return 2
			}
		}
		left, right := rank(proposals[i].Confidence), rank(proposals[j].Confidence)
		if left != right {
			return left < right
		}
		leftDistance := abs(proposals[i].AfterLine - center)
		rightDistance := abs(proposals[j].AfterLine - center)
		return leftDistance < rightDistance
	})
	if len(proposals) > 3 {
		proposals = proposals[:3]
	}
	return proposals
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
