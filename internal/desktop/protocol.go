// Package desktop implements the private, versioned desktop engine protocol.
// The transport is stdin/stdout. Only generated preview artifacts are served over HTTP.
package desktop

import "github.com/jjanuszczak/margo/internal/diagnostics"

const Protocol = 1

type Request struct {
	Protocol     int    `json:"protocol"`
	Op           string `json:"op"`
	Root         string `json:"root,omitempty"`
	Path         string `json:"path,omitempty"`
	Content      string `json:"content,omitempty"`
	Revision     string `json:"revision,omitempty"`
	Slide        string `json:"slide,omitempty"`
	Name         string `json:"name,omitempty"`
	Position     int    `json:"position,omitempty"`
	Source       string `json:"source,omitempty"`
	Format       string `json:"format,omitempty"`
	Notes        bool   `json:"notes,omitempty"`
	Presentation bool   `json:"presentation,omitempty"`
}

type Response struct {
	Thumbnails     []Thumbnail              `json:"thumbnails,omitempty"`
	ThumbnailsDone bool                     `json:"thumbnails_done,omitempty"`
	ThumbnailError string                   `json:"thumbnail_error,omitempty"`
	Revision       string                   `json:"revision,omitempty"`
	Protocol       int                      `json:"protocol"`
	Error          string                   `json:"error,omitempty"`
	Conflict       bool                     `json:"conflict,omitempty"`
	Snapshot       *Snapshot                `json:"snapshot,omitempty"`
	File           *File                    `json:"file,omitempty"`
	Preview        string                   `json:"preview,omitempty"`
	Selected       string                   `json:"selected,omitempty"`
	Artifact       string                   `json:"artifact,omitempty"`
	Markdown       string                   `json:"markdown,omitempty"`
	Log            string                   `json:"log,omitempty"`
	Diagnostics    []diagnostics.Diagnostic `json:"diagnostics"`
}

type Snapshot struct {
	Root     string   `json:"root"`
	Title    string   `json:"title"`
	Theme    string   `json:"theme"`
	Revision string   `json:"revision"`
	Slides   []Slide  `json:"slides"`
	Formats  []string `json:"formats"`
}

type Slide struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Path       string   `json:"path"`
	Draft      bool     `json:"draft"`
	Visibility string   `json:"visibility"`
	Layout     string   `json:"layout"`
	Notes      []Note   `json:"notes"`
	Assets     []string `json:"assets"`
}

type Note struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
}

type File struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Revision string `json:"revision"`
}

// Callbacks reuse the CLI's canonical slide sequencing and build workflows.
// Calls are serialized and always scoped to the explicitly selected root.
type Callbacks struct {
	Mutate func(root string, request Request) (selected string, err error)
}
