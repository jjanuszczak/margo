package desktop

import (
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/jjanuszczak/margo/internal/deck"
	"github.com/jjanuszczak/margo/internal/output/render"
)

//go:embed notes-preview.html
var notesPreviewTemplate string

// Notes remain authored bundle files. This inspection view uses the canonical
// Markdown renderer and the same unprivileged preview webview as slide output.
func writeNotesPreview(root, directory string, model deck.Model, selectedPath string) error {
	slide := model.Slides[0]
	type noteView struct {
		deck.Note
		Body template.HTML
	}
	notes := []noteView{}
	for _, note := range slide.Notes {
		path := projectNotePath(root, slide, note)
		if selectedPath != "" && filepath.ToSlash(path) != selectedPath {
			continue
		}
		notes = append(notes, noteView{note, render.MarkdownToHTML(note.Markdown)})
	}
	if selectedPath != "" && len(notes) == 0 {
		return fmt.Errorf("note no longer exists in this slide")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	relative, err := filepath.Rel(root, directory)
	if err != nil {
		return err
	}
	if err := render.StageAssets(root, relative, model.Slides); err != nil {
		return err
	}
	tmpl, err := template.New("notes").Parse(notesPreviewTemplate)
	if err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(directory, "notes.html"))
	if err != nil {
		return err
	}
	defer file.Close()
	title := strings.TrimSpace(slide.Title)
	if title == "" {
		title = slide.ID
	}
	return tmpl.Execute(file, struct {
		Title        string
		NotesEnabled bool
		Notes        []noteView
	}{title, model.Config.Presentation.Navigation.Notes, notes})
}

// The content model stores named-note paths relative to the slide bundle.
// Desktop source reads and saves use project-relative paths.
func projectNotePath(root string, slide deck.Slide, note deck.Note) string {
	if note.Path == "" {
		return ""
	}
	path := filepath.FromSlash(note.Path)
	if !filepath.IsAbs(path) {
		path = filepath.Join(slide.BundlePath, path)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(relative)
}
