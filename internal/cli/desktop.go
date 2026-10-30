package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jjanuszczak/margo/internal/desktop"
	"github.com/jjanuszczak/margo/internal/scaffold"
)

func runDesktopEngine() error {
	return desktop.Run(os.Stdin, os.Stdout, desktop.Callbacks{Mutate: desktopMutate})
}

// desktopMutate shares the exact sequence/rename/trash primitives used by the CLI.
// It returns structured selection identity rather than scraping console messages.
func desktopMutate(root string, r desktop.Request) (string, error) {
	slides, hasManifest, err := loadResolvedSlides(root)
	if err != nil {
		return "", err
	}
	ids := make([]string, len(slides))
	at := -1
	for i, s := range slides {
		ids[i] = s.ID
		if s.ID == r.Slide {
			at = i
		}
	}
	if r.Op != "create" && at < 0 {
		return "", fmt.Errorf("slide no longer exists")
	}
	switch r.Op {
	case "create":
		path, err := scaffold.CreateSlide(scaffold.SlideOptions{ProjectRoot: root, Name: r.Name, Archetype: "default"})
		if err != nil {
			return "", err
		}
		newID := filepath.Base(filepath.Dir(path))
		ids = append(ids, newID)
		at = len(ids) - 1
	case "duplicate":
		if strings.TrimSpace(r.Name) == "" {
			return "", fmt.Errorf("name the duplicate slide")
		}
		path, err := scaffold.CreateSlide(scaffold.SlideOptions{ProjectRoot: root, Name: r.Name, Archetype: "default"})
		if err != nil {
			return "", err
		}
		destination := filepath.Dir(path)
		cleanup := true
		defer func() {
			if cleanup {
				os.RemoveAll(destination)
			}
		}()
		data, err := os.ReadFile(filepath.Join(slides[at].BundlePath, "index.md"))
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return "", err
		}
		if err := copyBundleExtras(slides[at].BundlePath, destination, nil); err != nil {
			return "", err
		}
		newID := filepath.Base(destination)
		at++
		ids = append(ids, "")
		copy(ids[at+1:], ids[at:])
		ids[at] = newID
		cleanup = false
	case "move":
		if r.Position < 1 || r.Position > len(ids) {
			return "", fmt.Errorf("position must be between 1 and %d", len(ids))
		}
		moving := ids[at]
		ids = append(ids[:at], ids[at+1:]...)
		at = r.Position - 1
		ids = append(ids, "")
		copy(ids[at+1:], ids[at:])
		ids[at] = moving
	case "delete":
		if len(ids) <= 1 {
			return "", fmt.Errorf("cannot delete the last slide")
		}
		if _, err := moveSlideToTrash(root, r.Slide); err != nil {
			return "", err
		}
		ids = append(ids[:at], ids[at+1:]...)
		if at >= len(ids) {
			at = len(ids) - 1
		}
	default:
		return "", fmt.Errorf("unsupported slide operation")
	}
	final := finalSlideIDs(ids, hasPositionalBundleNames(slides))
	if err := applySlideSequence(root, ids, final, hasManifest); err != nil {
		return "", err
	}
	return final[at], nil
}
