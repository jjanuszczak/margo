package desktop

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// confined rejects symlinks in every component, including internal symlinks.
// Editing through aliases can otherwise replace or leak unexpected source files.
func confined(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) {
		return "", fmt.Errorf("expected a project-relative path")
	}
	relative = filepath.Clean(filepath.FromSlash(relative))
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes project")
	}
	current := root
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlinks are not supported by desktop operations: %s", relative)
		}
	}
	return current, nil
}

func sourcePath(root, relative string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(relative)))
	if clean != "margo.yaml" && clean != "manifest.yaml" && !(strings.HasPrefix(clean, "slides/") && strings.HasSuffix(clean, ".md")) {
		return "", fmt.Errorf("only project config and slide Markdown sources can be edited")
	}
	return confined(root, relative)
}

func readFile(root, relative string) (*File, error) {
	path, err := sourcePath(root, relative)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &File{Path: relative, Content: string(data), Revision: hash(data)}, nil
}

func atomicWrite(path string, data []byte, mode fs.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".margo-save-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err = f.Chmod(mode); err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// projectRevision includes source and all engine inputs, including includes,
// partials, shortcodes and manifests. Generated output and tool state are excluded.
func projectRevision(root string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		first := strings.Split(filepath.ToSlash(rel), "/")[0]
		if d.IsDir() && (strings.HasPrefix(first, ".") || first == "dist" || first == "node_modules" || first == "target") {
			return filepath.SkipDir
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("desktop projects cannot contain symlinks: %s", rel)
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".margo-save-") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00%d\x00%s\x00", rel, len(data), hash(data))
		return nil
	})
	return hex.EncodeToString(h.Sum(nil)), err
}
