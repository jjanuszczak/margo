package watch

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Snapshot struct {
	Entries []Entry
}

type Entry struct {
	Path    string
	Size    int64
	ModTime time.Time
}

func Poll(projectRoot string, interval time.Duration, onChange func() error) error {
	current, err := SnapshotProject(projectRoot)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		next, err := SnapshotProject(projectRoot)
		if err != nil {
			return err
		}

		if equalSnapshots(current, next) {
			continue
		}

		if err := onChange(); err != nil {
			return err
		}
		current = next
	}

	return nil
}

func SnapshotProject(projectRoot string) (Snapshot, error) {
	var entries []Entry
	err := filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		if d.IsDir() {
			if shouldSkipDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}

		if !shouldWatchFile(rel) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		entries = append(entries, Entry{
			Path:    rel,
			Size:    info.Size(),
			ModTime: info.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		return Snapshot{}, fmt.Errorf("snapshot project %q: %w", projectRoot, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	return Snapshot{Entries: entries}, nil
}

func shouldSkipDir(rel string) bool {
	first := strings.Split(rel, string(filepath.Separator))[0]
	switch first {
	case "dist", ".git", ".gocache":
		return true
	default:
		return false
	}
}

func shouldWatchFile(rel string) bool {
	switch {
	case rel == "margo.yaml":
		return true
	case strings.HasPrefix(rel, "slides"+string(filepath.Separator)):
		return true
	case strings.HasPrefix(rel, "themes"+string(filepath.Separator)):
		return true
	case strings.HasPrefix(rel, "archetypes"+string(filepath.Separator)):
		return true
	case strings.HasPrefix(rel, "assets"+string(filepath.Separator)):
		return true
	default:
		return false
	}
}

func equalSnapshots(a, b Snapshot) bool {
	if len(a.Entries) != len(b.Entries) {
		return false
	}

	for i := range a.Entries {
		if a.Entries[i].Path != b.Entries[i].Path {
			return false
		}
		if a.Entries[i].Size != b.Entries[i].Size {
			return false
		}
		if !a.Entries[i].ModTime.Equal(b.Entries[i].ModTime) {
			return false
		}
	}

	return true
}
