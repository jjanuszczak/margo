package content

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jjanuszczak/margo/internal/deck"
	"github.com/jjanuszczak/margo/internal/ignore"
	"gopkg.in/yaml.v3"
)

const slidesDirName = "slides"
const notesDirName = "notes"

type noteFrontMatter struct {
	ID         string   `yaml:"id"`
	Title      string   `yaml:"title"`
	Order      int      `yaml:"order"`
	Visibility string   `yaml:"visibility"`
	Draft      bool     `yaml:"draft"`
	Kind       string   `yaml:"kind"`
	Tags       []string `yaml:"tags"`
	Language   string   `yaml:"language"`
}

func DiscoverSlides(projectRoot string) ([]deck.Slide, error) {
	slidesRoot := filepath.Join(projectRoot, slidesDirName)
	ignored, err := ignore.Load(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("load ignore file: %w", err)
	}
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
		bundleRel := filepath.ToSlash(filepath.Join(slidesDirName, entry.Name()))
		if ignored.ShouldIgnore(bundleRel, true) {
			continue
		}

		bundlePath := filepath.Join(slidesRoot, entry.Name())
		indexPath := filepath.Join(bundlePath, "index.md")
		data, err := os.ReadFile(indexPath)
		if err != nil {
			return nil, fmt.Errorf("read slide bundle %q: %w", indexPath, err)
		}

		slide, err := parseSlide(projectRoot, indexPath, bundlePath, string(data), ignored)
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

func parseSlide(projectRoot, indexPath, bundlePath, source string, ignored ignore.Matcher) (deck.Slide, error) {
	slide := deck.Slide{
		ID:           filepath.Base(bundlePath),
		BundlePath:   bundlePath,
		BodyMarkdown: source,
	}
	assets, err := discoverBundleAssets(projectRoot, bundlePath, ignored)
	if err != nil {
		return deck.Slide{}, fmt.Errorf("%s: discover bundle assets: %w", indexPath, err)
	}
	slide.Assets = assets

	if !strings.HasPrefix(source, "---\n") {
		slide.Title = humanize(filepath.Base(bundlePath))
		fileNotes, err := discoverBundleNotes(projectRoot, bundlePath, ignored)
		if err != nil {
			return deck.Slide{}, fmt.Errorf("%s: discover slide notes: %w", indexPath, err)
		}
		slide.Notes = append(slide.Notes, fileNotes...)
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
	fileNotes, err := discoverBundleNotes(projectRoot, bundlePath, ignored)
	if err != nil {
		return deck.Slide{}, fmt.Errorf("%s: discover slide notes: %w", indexPath, err)
	}
	slide.Notes = append(slide.Notes, fileNotes...)

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

func normalizeNotes(value any) []deck.Note {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return nil
		}
		return []deck.Note{{Name: "Notes", Markdown: trimmed}}
	case []any:
		values := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				trimmed := strings.TrimSpace(s)
				if trimmed != "" {
					values = append(values, trimmed)
				}
			}
		}
		if len(values) == 0 {
			return nil
		}
		return []deck.Note{{Name: "Notes", Markdown: strings.Join(values, "\n\n")}}
	default:
		return nil
	}
}

func extractBodyNotes(body string, notes []deck.Note) (string, []deck.Note) {
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
		for i := range notes {
			if notes[i].Name == "Notes" {
				notes[i].Markdown = strings.TrimSpace(notes[i].Markdown + "\n\n" + noteText)
				return bodyLines, notes
			}
		}
		notes = append(notes, deck.Note{Name: "Notes", Markdown: noteText})
	}

	return bodyLines, notes
}

func discoverBundleNotes(projectRoot, bundlePath string, ignored ignore.Matcher) ([]deck.Note, error) {
	notesPath := filepath.Join(bundlePath, notesDirName)
	entries, err := os.ReadDir(notesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read notes directory %q: %w", notesPath, err)
	}

	notes := make([]deck.Note, 0, len(entries))
	noteIDs := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		path := filepath.Join(notesPath, entry.Name())
		relToProject, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return nil, err
		}
		if ignored.ShouldIgnore(relToProject, false) {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read note file %q: %w", path, err)
		}
		note, err := parseBundleNote(path, entry.Name(), string(data))
		if err != nil {
			return nil, err
		}
		markdown, err := resolveIncludes(projectRoot, path, note.Markdown)
		if err != nil {
			return nil, err
		}
		markdown = strings.TrimSpace(markdown)
		if markdown == "" {
			continue
		}
		note.Markdown = markdown
		if existingPath, exists := noteIDs[note.ID]; exists {
			return nil, fmt.Errorf("note file %q duplicates note id %q already used by %q", path, note.ID, existingPath)
		}
		noteIDs[note.ID] = path
		notes = append(notes, note)
	}
	sort.SliceStable(notes, func(i, j int) bool {
		if notes[i].Order == notes[j].Order {
			return notes[i].Path < notes[j].Path
		}
		return notes[i].Order < notes[j].Order
	})
	return notes, nil
}

func parseBundleNote(path, filename, source string) (deck.Note, error) {
	stem := strings.TrimSuffix(filename, filepath.Ext(filename))
	note := deck.Note{
		ID:   stem,
		Name: humanize(stem),
		Path: filepath.ToSlash(filepath.Join(notesDirName, filename)),
	}

	if !strings.HasPrefix(source, "---\n") {
		note.Markdown = source
		return note, nil
	}

	parts := strings.SplitN(source, "\n---\n", 2)
	if len(parts) != 2 {
		return deck.Note{}, fmt.Errorf("note file %q: unclosed front matter", path)
	}

	var frontMatter noteFrontMatter
	if err := yaml.Unmarshal([]byte(strings.TrimPrefix(parts[0], "---\n")), &frontMatter); err != nil {
		return deck.Note{}, fmt.Errorf("note file %q: parse front matter: %w", path, err)
	}
	if strings.TrimSpace(frontMatter.ID) != "" {
		note.ID = strings.TrimSpace(frontMatter.ID)
	}
	if strings.TrimSpace(frontMatter.Title) != "" {
		note.Name = strings.TrimSpace(frontMatter.Title)
	}
	note.Order = frontMatter.Order
	note.Visibility = strings.ToLower(strings.TrimSpace(frontMatter.Visibility))
	note.Draft = frontMatter.Draft
	note.Kind = strings.TrimSpace(frontMatter.Kind)
	note.Tags = frontMatter.Tags
	note.Language = strings.TrimSpace(frontMatter.Language)
	note.Markdown = parts[1]

	if note.Visibility != "" && note.Visibility != "visible" && note.Visibility != "hidden" {
		return deck.Note{}, fmt.Errorf("note file %q: invalid visibility %q; use visible or hidden", path, frontMatter.Visibility)
	}
	return note, nil
}

func discoverBundleAssets(projectRoot string, bundlePath string, ignored ignore.Matcher) ([]string, error) {
	var assets []string

	err := filepath.WalkDir(bundlePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == filepath.Join(bundlePath, notesDirName) {
				return filepath.SkipDir
			}
			relToProject, relErr := filepath.Rel(projectRoot, path)
			if relErr == nil && relToProject != "." && ignored.ShouldIgnore(relToProject, true) {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(bundlePath, path)
		if err != nil {
			return err
		}
		relToProject, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}
		if ignored.ShouldIgnore(relToProject, false) {
			return nil
		}
		if relPath == "index.md" {
			return nil
		}

		assets = append(assets, filepath.ToSlash(relPath))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(assets)
	return assets, nil
}
