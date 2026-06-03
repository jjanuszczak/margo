package printhtml

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureProjectRoot(tb testing.TB, deckName string) string {
	tb.Helper()

	sourceRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "examples", deckName))
	if err != nil {
		tb.Fatalf("resolve fixture project root: %v", err)
	}

	projectRoot := filepath.Join(tb.TempDir(), deckName)
	if err := copyFixtureDeck(sourceRoot, projectRoot, allowedFixturePrefixes(deckName)); err != nil {
		tb.Fatalf("copy fixture project root: %v", err)
	}
	return projectRoot
}

func copyFixtureDeck(srcRoot string, dstRoot string, allowed []string) error {
	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return os.MkdirAll(dstRoot, 0o755)
		}
		if relPath == "dist" && d.IsDir() {
			return filepath.SkipDir
		}
		if strings.HasPrefix(relPath, "dist"+string(filepath.Separator)) {
			return nil
		}
		if !shouldCopyFixturePath(relPath, allowed) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		targetPath := filepath.Join(dstRoot, relPath)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return err
		}
		return nil
	})
}

func shouldCopyFixturePath(relPath string, allowed []string) bool {
	for _, prefix := range allowed {
		if relPath == prefix {
			return true
		}
		if strings.HasPrefix(relPath, prefix+string(filepath.Separator)) {
			return true
		}
		if strings.HasPrefix(prefix, relPath+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func allowedFixturePrefixes(deckName string) []string {
	switch deckName {
	case "arca-investor-memo":
		return []string{
			"margo.yaml",
			"slides",
			"themes",
			"archetypes",
			"assets",
			"shortcodes",
			"layouts",
		}
	case "reference-deck":
		return []string{
			"archetypes/agenda",
			"archetypes/closing",
			"archetypes/default",
			"archetypes/image",
			"archetypes/media-left",
			"archetypes/media-right",
			"archetypes/metric",
			"archetypes/quote",
			"archetypes/section",
			"archetypes/title",
			"archetypes/two-column",
			"assets/shared-grid.svg",
			"assets/video-poster.svg",
			"shared/authoring-pillars.md",
			"shared/output-pillars.md",
			"shortcodes/eyebrow.html",
			"slides/01-title",
			"slides/02-why",
			"slides/03-draft",
			"slides/04-hidden",
			"slides/05-customer-story",
			"slides/06-architecture-view",
			"slides/07-mermaid-qa",
			"themes/default",
			"margo.yaml",
		}
	default:
		return []string{"margo.yaml", "slides", "themes", "archetypes", "assets"}
	}
}
