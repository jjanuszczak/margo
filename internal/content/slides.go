package content

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
	"margo/internal/deck"
)

const slidesDirName = "slides"

func DiscoverSlides(projectRoot string) ([]deck.Slide, error) {
	slidesRoot := filepath.Join(projectRoot, slidesDirName)
	entries, err := os.ReadDir(slidesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read slides directory %q: %w", slidesRoot, err)
	}

	var slides []deck.Slide
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		bundlePath := filepath.Join(slidesRoot, entry.Name())
		indexPath := filepath.Join(bundlePath, "index.md")
		data, err := os.ReadFile(indexPath)
		if err != nil {
			return nil, fmt.Errorf("read slide bundle %q: %w", indexPath, err)
		}

		slide, err := parseSlide(projectRoot, indexPath, bundlePath, string(data))
		if err != nil {
			return nil, err
		}

		slides = append(slides, slide)
	}

	sort.SliceStable(slides, func(i, j int) bool {
		if slides[i].Order == slides[j].Order {
			return slides[i].BundlePath < slides[j].BundlePath
		}
		return slides[i].Order < slides[j].Order
	})

	return slides, nil
}

func parseSlide(projectRoot, indexPath, bundlePath, source string) (deck.Slide, error) {
	slide := deck.Slide{
		ID:           filepath.Base(bundlePath),
		BundlePath:   bundlePath,
		BodyMarkdown: source,
	}

	if !strings.HasPrefix(source, "---\n") {
		slide.Title = humanize(filepath.Base(bundlePath))
		return slide, nil
	}

	parts := strings.SplitN(source, "\n---\n", 2)
	if len(parts) != 2 {
		return deck.Slide{}, fmt.Errorf("%s: unclosed front matter", indexPath)
	}

	frontMatter := strings.TrimPrefix(parts[0], "---\n")
	body := parts[1]
	slide.BodyMarkdown = strings.TrimSpace(body)

	if err := yaml.Unmarshal([]byte(frontMatter), &slide.FrontMatter); err != nil {
		return deck.Slide{}, fmt.Errorf("%s: parse slide front matter: %w", indexPath, err)
	}

	slide.Notes = append(slide.Notes, normalizeNotes(slide.FrontMatter.Notes)...)
	slide.BodyMarkdown, slide.Notes = extractBodyNotes(slide.BodyMarkdown, slide.Notes)
	resolvedBody, err := resolveIncludes(projectRoot, indexPath, slide.BodyMarkdown)
	if err != nil {
		return deck.Slide{}, err
	}
	slide.BodyMarkdown = strings.TrimSpace(resolvedBody)

	if slide.Title == "" {
		slide.Title = humanize(filepath.Base(bundlePath))
	}

	return slide, nil
}

func humanize(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func normalizeNotes(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return nil
		}
		return []string{trimmed}
	case []any:
		notes := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				trimmed := strings.TrimSpace(s)
				if trimmed != "" {
					notes = append(notes, trimmed)
				}
			}
		}
		return notes
	default:
		return nil
	}
}

func extractBodyNotes(body string, notes []string) (string, []string) {
	lines := strings.Split(body, "\n")
	start := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "## Notes" || trimmed == "### Notes" || trimmed == "## Speaker Notes" || trimmed == "### Speaker Notes" {
			start = i
			break
		}
	}
	if start == -1 {
		return strings.TrimSpace(body), notes
	}

	bodyLines := strings.TrimSpace(strings.Join(lines[:start], "\n"))
	noteLines := lines[start+1:]
	if len(noteLines) == 0 {
		return bodyLines, notes
	}

	noteText := strings.TrimSpace(strings.Join(noteLines, "\n"))
	if noteText != "" {
		notes = append(notes, noteText)
	}

	return bodyLines, notes
}
