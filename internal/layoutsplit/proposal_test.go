package layoutsplit

import (
	"testing"

	"github.com/jjanuszczak/margo/internal/deck"
)

func TestProposeRanksExplicitAndRecommendedBoundaries(t *testing.T) {
	proposals := Propose(deck.Slide{BodyMarkdown: "# Interview map\n\nIntro\n\n## Executive and commercial\n\n- CEO\n\n<!-- margo: split -->\n\n## Control and field execution\n\n- Risk"})
	if len(proposals) != 3 {
		t.Fatalf("expected three proposals, got %d", len(proposals))
	}
	if proposals[0].Confidence != "explicit" || proposals[0].Boundary != "explicit split marker" {
		t.Fatalf("expected explicit proposal first, got %#v", proposals[0])
	}
}

func TestProposeSkipsUnsafeSlides(t *testing.T) {
	if got := Propose(deck.Slide{BodyMarkdown: "# One\n\nOnly one section"}); got != nil {
		t.Fatalf("expected no proposals, got %#v", got)
	}
}

func TestProposeSkipsBoundariesInsideFencedCode(t *testing.T) {
	slide := deck.Slide{BodyMarkdown: "# Example\n\n```md\n## Not a section\n<!-- margo: split -->\n```\n\n## Real section\n\nContent"}
	proposals := Propose(slide)
	if len(proposals) != 1 || proposals[0].Boundary != "before Real section" {
		t.Fatalf("expected only the real heading boundary, got %#v", proposals)
	}
}

func TestProposeSkipsNestedAndUnseparatedHeadings(t *testing.T) {
	slide := deck.Slide{BodyMarkdown: "# Example\n\n- ## Nested heading\n\n## Real section\n\nContent\n## Unsafe heading\n\nMore content"}
	proposals := Propose(slide)
	if len(proposals) != 1 || proposals[0].Boundary != "before Real section" {
		t.Fatalf("expected only the separated top-level heading, got %#v", proposals)
	}
}
