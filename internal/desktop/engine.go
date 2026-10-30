package desktop

import (
	"bufio"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jjanuszczak/margo/internal/config"
	"github.com/jjanuszczak/margo/internal/content"
	"github.com/jjanuszczak/margo/internal/deck"
	"github.com/jjanuszczak/margo/internal/diagnostics"
	"github.com/jjanuszczak/margo/internal/layoutaudit"
	"github.com/jjanuszczak/margo/internal/manifest"
	htmloutput "github.com/jjanuszczak/margo/internal/output/html"
	"github.com/jjanuszczak/margo/internal/output/pdf"
	"github.com/jjanuszczak/margo/internal/scaffold"
	"github.com/jjanuszczak/margo/internal/theme"
)

//go:embed preview-shell.html
var previewShell []byte

type Engine struct {
	root, temp, origin string
	generation         int
	callbacks          Callbacks
	thumbnails         *thumbnailJob
}

func Run(input io.Reader, output io.Writer, callbacks Callbacks) error {
	temp, err := os.MkdirTemp("", "margo-desktop-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	token := make([]byte, 24)
	if _, err := rand.Read(token); err != nil {
		listener.Close()
		return err
	}
	prefix := "/" + hex.EncodeToString(token) + "/"
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != listener.Addr().String() {
			http.Error(w, "invalid host", 403)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "read only", 405)
			return
		}
		if !strings.HasPrefix(r.URL.Path, prefix) {
			http.NotFound(w, r)
			return
		}
		relative := strings.TrimPrefix(r.URL.Path, prefix)
		path, err := confined(temp, relative)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		file, err := os.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	}), ReadHeaderTimeout: 5 * time.Second}
	defer server.Close()
	go server.Serve(listener)
	engine := Engine{temp: temp, origin: "http://" + listener.Addr().String() + prefix, callbacks: callbacks}
	defer engine.stopThumbnails()
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 16*1024*1024)
	encoder := json.NewEncoder(output)
	for scanner.Scan() {
		var request Request
		var response Response
		if err := json.Unmarshal(scanner.Bytes(), &request); err != nil {
			response.Error = err.Error()
		} else {
			response = engine.Handle(request)
		}
		response.Protocol = Protocol
		if response.Diagnostics == nil {
			response.Diagnostics = []diagnostics.Diagnostic{}
		}
		if err := encoder.Encode(response); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (e *Engine) Handle(r Request) Response {
	response, err := e.handle(r)
	if err != nil {
		response.Error = err.Error()
		d := diagnostics.Diagnostic{Severity: diagnostics.SeverityError, Code: "desktop_" + r.Op, Message: err.Error()}
		if fe, ok := config.AsFieldError(err); ok {
			d.Path = fe.Path
			d.Line = fe.Line
			d.Message = fe.Message
		}
		if te, ok := theme.AsError(err); ok {
			d.Path = te.Path
			d.Line = te.Line
			d.Message = te.Message
		}
		// Content errors prefix the exact source path. Preserve it for source navigation.
		if d.Path == "" {
			if i := strings.Index(err.Error(), ".md:"); i >= 0 {
				d.Path = err.Error()[:i+3]
			}
		}
		response.Diagnostics = append(response.Diagnostics, d)
	}
	return response
}

func (e *Engine) handle(r Request) (Response, error) {
	if r.Protocol != Protocol {
		return Response{}, fmt.Errorf("desktop protocol mismatch: engine %d, app %d", Protocol, r.Protocol)
	}
	if r.Op == "hello" {
		return Response{}, nil
	}
	if r.Op == "create-project" {
		if strings.TrimSpace(r.Name) == "" || strings.TrimSpace(r.Root) == "" {
			return Response{}, fmt.Errorf("project name and destination are required")
		}
		root, err := filepath.Abs(r.Root)
		if err != nil {
			return Response{}, err
		}
		if err := scaffold.CreateDeck(scaffold.DeckOptions{Name: r.Name, TargetDir: root}); err != nil {
			return Response{}, err
		}
		if strings.TrimSpace(r.ThemeRepo) != "" {
			installed, err := theme.Install(theme.InstallOptions{ProjectRoot: root, Repo: r.ThemeRepo, Ref: r.ThemeRef, Name: r.ThemeName})
			if err != nil {
				return Response{}, fmt.Errorf("deck created with the default theme, but importing the selected theme failed: %w", err)
			}
			if err := config.SetThemeName(filepath.Join(root, "margo.yaml"), installed.Name); err != nil {
				return Response{}, fmt.Errorf("deck and theme %q were created, but selecting the theme failed: %w", installed.Name, err)
			}
		}
		e.root = root
		snapshot, err := e.snapshot()
		return Response{Snapshot: snapshot}, err
	}
	if r.Op == "close" {
		e.stopThumbnails()
		e.root = ""
		return Response{}, nil
	}
	if r.Op == "open" {
		root, err := filepath.Abs(r.Root)
		if err != nil {
			return Response{}, err
		}
		if filepath.Base(root) == "margo.yaml" {
			root = filepath.Dir(root)
		}
		root, err = filepath.EvalSymlinks(root)
		if err != nil {
			return Response{}, err
		}
		if _, err = os.Stat(filepath.Join(root, "margo.yaml")); err != nil {
			return Response{}, fmt.Errorf("select a folder containing margo.yaml: %w", err)
		}
		e.stopThumbnails()
		previous := e.root
		e.root = root
		snapshot, err := e.snapshot()
		if err != nil {
			e.root = previous
			return Response{}, err
		}
		return Response{Snapshot: snapshot}, nil
	}
	if e.root == "" {
		return Response{}, fmt.Errorf("open a project first")
	}
	switch r.Op {
	case "thumbnails":
		return e.startThumbnails(r.Revision)
	case "thumbnail-status":
		return e.thumbnailStatus(r.Position), nil
	case "snapshot":
		snapshot, err := e.snapshot()
		return Response{Snapshot: snapshot}, err
	case "skills":
		return Response{Skills: e.projectSkills()}, nil
	case "read":
		file, err := readFile(e.root, r.Path)
		return Response{File: file}, err
	case "save":
		file, err := readFile(e.root, r.Path)
		if err != nil {
			return Response{Conflict: true}, err
		}
		if file.Revision != r.Revision {
			return Response{Conflict: true, File: file}, fmt.Errorf("file changed on disk; reload or compare before saving")
		}
		path, err := sourcePath(e.root, r.Path)
		if err != nil {
			return Response{}, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return Response{}, err
		}
		// Recheck immediately before the atomic replacement. There is no force-save API.
		latest, err := os.ReadFile(path)
		if err != nil {
			return Response{}, err
		}
		if hash(latest) != r.Revision {
			return Response{Conflict: true}, fmt.Errorf("file changed during save")
		}
		if err := atomicWrite(path, []byte(r.Content), info.Mode()); err != nil {
			return Response{}, err
		}
		file, err = readFile(e.root, r.Path)
		return Response{File: file}, err
	case "preview":
		return e.preview(r)
	case "create", "duplicate", "move", "delete", "note", "media":
		revision, err := projectRevision(e.root)
		if err != nil {
			return Response{}, err
		}
		if revision != r.Revision {
			return Response{Conflict: true}, fmt.Errorf("project changed; refresh before modifying its structure")
		}
		if r.Op == "media" {
			return e.importMedia(r)
		}
		if r.Op == "note" {
			if _, err := confined(e.root, filepath.Join("slides", r.Slide, "index.md")); err != nil {
				return Response{}, err
			}
			if filepath.Base(r.Slide) != r.Slide || r.Slide == "." || r.Slide == ".." {
				return Response{}, fmt.Errorf("invalid slide")
			}
			if _, err := confined(e.root, filepath.Join("slides", r.Slide, "notes")); err != nil {
				return Response{}, err
			}
			path, err := scaffold.CreateNote(scaffold.NoteOptions{ProjectRoot: e.root, Slide: r.Slide, Name: r.Name})
			if err != nil {
				return Response{}, err
			}
			rel, _ := filepath.Rel(e.root, path)
			file, err := readFile(e.root, filepath.ToSlash(rel))
			return Response{File: file}, err
		}
		if e.callbacks.Mutate == nil {
			return Response{}, fmt.Errorf("slide operations unavailable")
		}
		selected, err := e.callbacks.Mutate(e.root, r)
		if err != nil {
			return Response{}, err
		}
		snapshot, err := e.snapshot()
		return Response{Snapshot: snapshot, Selected: selected}, err
	case "build":
		return e.build(r.Format)
	default:
		return Response{}, fmt.Errorf("unknown desktop operation %q", r.Op)
	}
}

func (e *Engine) load() (deck.Model, theme.Metadata, error) {
	raw, err := config.LoadRaw(filepath.Join(e.root, "margo.yaml"))
	if err != nil {
		return deck.Model{}, theme.Metadata{}, err
	}
	parsed, err := config.Parse(raw)
	if err != nil {
		return deck.Model{}, theme.Metadata{}, err
	}
	slides, err := content.DiscoverSlides(e.root)
	if err != nil {
		return deck.Model{}, theme.Metadata{}, err
	}
	mf, has, err := manifest.Load(e.root)
	if err != nil {
		return deck.Model{}, theme.Metadata{}, err
	}
	if has {
		slides, err = manifest.Apply(slides, mf)
		if err != nil {
			return deck.Model{}, theme.Metadata{}, err
		}
	}
	active, err := theme.Load(e.root, parsed.Config.Theme.Name)
	if err != nil {
		return deck.Model{}, active, err
	}
	parsed.Config.Theme.Options, err = theme.ResolveOptions(active, parsed.Config.Theme.Options)
	return deck.Model{Config: parsed.Config, Slides: slides, Sections: deck.BuildSections(slides)}, active, err
}

func (e *Engine) snapshot() (*Snapshot, error) {
	revision, err := projectRevision(e.root)
	if err != nil {
		return nil, err
	}
	model, active, err := e.load()
	if err != nil {
		return nil, err
	}
	snapshot := &Snapshot{Root: e.root, Title: model.Config.Deck.Title, Theme: active.Name, Revision: revision, Slides: []Slide{}, Formats: []string{"html", "pptx"}, SupportsTiles: e.themeSupportsTiles(active)}
	if _, err := pdf.DetectBrowser(); err == nil {
		snapshot.Formats = append(snapshot.Formats, "pdf")
	}
	for _, s := range model.Slides {
		notes := []Note{}
		for _, n := range s.Notes {
			notes = append(notes, Note{Name: n.Name, Path: projectNotePath(e.root, s, n), Content: n.Markdown})
		}
		snapshot.Slides = append(snapshot.Slides, Slide{ID: s.ID, Title: s.Title, Path: filepath.ToSlash(filepath.Join("slides", s.ID, "index.md")), Draft: s.Draft, Visibility: s.Visibility, Layout: s.Layout, Notes: notes, Assets: append([]string{}, s.Assets...)})
	}
	return snapshot, nil
}

func (e *Engine) themeSupportsTiles(active theme.Metadata) bool {
	for _, name := range []string{"tiles.html", "tile.html"} {
		path := filepath.Join(e.root, "shortcodes", name)
		if _, err := os.Stat(path); err != nil {
			path = filepath.Join(active.RootDir, "shortcodes", name)
		}
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	foundStyles := false
	_ = filepath.WalkDir(filepath.Join(active.RootDir, "assets"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".css") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(data), ".shortcode-tiles") && strings.Contains(string(data), ".shortcode-tile") {
			foundStyles = true
			return filepath.SkipAll
		}
		return nil
	})
	return foundStyles
}

func (e *Engine) projectSkills() []Skill {
	files := scaffold.AgentFiles()
	contents := make(map[string]string)
	for path, content := range files {
		rel := filepath.ToSlash(path)
		if strings.HasPrefix(rel, ".agents/skills/") && strings.HasSuffix(strings.ToLower(rel), ".md") {
			contents[rel] = content
		}
	}
	root, err := confined(e.root, filepath.Join(".agents", "skills"))
	if err == nil {
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
				return nil
			}
			rel, err := filepath.Rel(e.root, path)
			if err != nil {
				return nil
			}
			data, err := os.ReadFile(path)
			if err == nil {
				contents[filepath.ToSlash(rel)] = string(data)
			}
			return nil
		})
	}
	skills := make([]Skill, 0, len(contents))
	for rel, content := range contents {
		if !strings.HasSuffix(filepath.ToSlash(rel), "/SKILL.md") {
			continue
		}
		name, description := filepath.Base(filepath.Dir(filepath.FromSlash(rel))), "Project skill"
		if strings.HasPrefix(content, "---") {
			if end := strings.Index(content[3:], "---"); end >= 0 {
				for _, line := range strings.Split(content[3:3+end], "\n") {
					if strings.HasPrefix(line, "name:") {
						name = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "name:")), "\"'")
					}
					if strings.HasPrefix(line, "description:") {
						description = strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "description:")), "\"'")
					}
				}
			}
		}
		skill := Skill{Name: name, Description: description, Path: rel, Content: content}
		referencePrefix := strings.TrimSuffix(filepath.ToSlash(rel), "/SKILL.md") + "/references/"
		for refPath, refContent := range contents {
			if strings.HasPrefix(filepath.ToSlash(refPath), referencePrefix) && strings.HasSuffix(strings.ToLower(refPath), ".md") {
				skill.References = append(skill.References, SkillReference{Path: refPath, Content: refContent})
			}
		}
		sort.Slice(skill.References, func(i, j int) bool { return skill.References[i].Path < skill.References[j].Path })
		skills = append(skills, skill)
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills
}

func (e *Engine) preview(r Request) (Response, error) {
	model, active, err := e.load()
	if err != nil {
		return Response{}, err
	}
	if r.Presentation && !r.Notes {
		model.Slides = deck.ApplySectionDividers(deck.FilterSlides(model.Slides, deck.FilterOptions{IncludeDrafts: true}))
	} else {
		found := false
		for _, s := range model.Slides {
			if s.ID == r.Slide {
				model.Slides = []deck.Slide{s}
				found = true
				break
			}
		}
		if !found {
			return Response{}, fmt.Errorf("slide no longer exists")
		}
	}
	e.generation++
	directory := filepath.Join(e.temp, strconv.Itoa(e.generation))
	if r.Notes {
		if err := writeNotesPreview(e.root, directory, model, r.Path); err != nil {
			os.RemoveAll(directory)
			return Response{}, err
		}
		if e.generation > 8 {
			os.RemoveAll(filepath.Join(e.temp, strconv.Itoa(e.generation-8)))
		}
		return Response{Preview: e.origin + strconv.Itoa(e.generation) + "/notes.html"}, nil
	}
	relative, err := filepath.Rel(e.root, directory)
	if err != nil {
		return Response{}, err
	}
	report, err := htmloutput.WriteTo(e.root, relative, model, active)
	if err != nil {
		os.RemoveAll(directory)
		return Response{}, err
	}
	enabled, minScale := active.FitSettings()
	if err := layoutaudit.ApplyStructuralRemediation(filepath.Join(directory, "index.html"), active.StructuralRemediationPolicies()); err != nil {
		return Response{}, err
	}
	if err := layoutaudit.ApplyFit(filepath.Join(directory, "index.html"), layoutaudit.FitPolicy{Enabled: enabled, MinScale: minScale}); err != nil {
		return Response{}, err
	}
	if err := os.WriteFile(filepath.Join(directory, "preview.html"), previewShell, 0o644); err != nil {
		return Response{}, err
	}
	// Keep only a bounded number of preview generations while allowing in-flight asset loads.
	if e.generation > 8 {
		os.RemoveAll(filepath.Join(e.temp, strconv.Itoa(e.generation-8)))
	}
	return Response{Preview: e.origin + strconv.Itoa(e.generation) + "/preview.html", Diagnostics: report.Items}, nil
}

func (e *Engine) importMedia(r Request) (Response, error) {
	if filepath.Base(r.Slide) != r.Slide || r.Slide == "." || r.Slide == ".." {
		return Response{}, fmt.Errorf("invalid slide")
	}
	if _, err := readFile(e.root, filepath.ToSlash(filepath.Join("slides", r.Slide, "index.md"))); err != nil {
		return Response{}, err
	}
	data, err := os.ReadFile(r.Source)
	if err != nil {
		return Response{}, err
	}
	ext := strings.ToLower(filepath.Ext(r.Source))
	allowed := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".svg": true, ".mp4": true, ".webm": true}
	if !allowed[ext] {
		return Response{}, fmt.Errorf("choose a PNG, JPEG, GIF, WebP, SVG, MP4 or WebM file")
	}
	name := "media-" + hash(data)[:12] + ext
	path, err := confined(e.root, filepath.Join("slides", r.Slide, name))
	if err != nil {
		return Response{}, err
	}
	if _, err = os.Stat(path); os.IsNotExist(err) {
		if err = atomicWrite(path, data, 0o644); err != nil {
			return Response{}, err
		}
	}
	markdown := "![" + name + "](" + url.PathEscape(name) + ")"
	if ext == ".mp4" || ext == ".webm" {
		markdown = "{{< video src=\"" + name + "\" />}}"
	}
	return Response{Markdown: markdown}, nil
}
