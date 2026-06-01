package render

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"margo/internal/deck"
	"margo/internal/diagnostics"
	"margo/internal/ignore"
)

var markdownAssetRefPattern = regexp.MustCompile(`\]\(([^)\s]+)([^)]*)\)`)

func StageAssets(projectRoot string, outputDir string, slides []deck.Slide) error {
	ignored, err := ignore.Load(projectRoot)
	if err != nil {
		return fmt.Errorf("load ignore file: %w", err)
	}
	if err := copyDeckAssets(projectRoot, outputDir, ignored); err != nil {
		return err
	}
	for _, slide := range slides {
		if slide.Synthetic || slide.BundlePath == "" {
			continue
		}
		for _, asset := range slide.Assets {
			srcPath := filepath.Join(slide.BundlePath, filepath.FromSlash(asset))
			dstPath := filepath.Join(projectRoot, outputDir, "slides", slide.ID, filepath.FromSlash(asset))
			if err := copyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("copy slide asset %q: %w", srcPath, err)
			}
		}
	}
	return nil
}

func copyDeckAssets(projectRoot string, outputDir string, ignored ignore.Matcher) error {
	srcRoot := filepath.Join(projectRoot, "assets")
	info, err := os.Stat(srcRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat deck assets: %w", err)
	}
	if !info.IsDir() {
		return nil
	}

	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		projectRel := filepath.ToSlash(filepath.Join("assets", relPath))
		if d.IsDir() {
			if ignored.ShouldIgnore(projectRel, true) {
				return filepath.SkipDir
			}
		} else if ignored.ShouldIgnore(projectRel, false) {
			return nil
		}

		dstPath := filepath.Join(projectRoot, outputDir, "assets", relPath)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		return copyFile(path, dstPath)
	})
}

func copyFile(srcPath string, dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return nil
}

func RewriteMarkdownAssetRefs(projectRoot string, slide deck.Slide, source string) (string, diagnostics.Report) {
	var report diagnostics.Report
	rewritten := markdownAssetRefPattern.ReplaceAllStringFunc(source, func(match string) string {
		parts := markdownAssetRefPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		rewritten, warning := ResolveAssetReference(projectRoot, slide, parts[1], "slide markdown asset")
		if warning != nil {
			report.Add(*warning)
		}
		return "](" + rewritten + parts[2] + ")"
	})
	return rewritten, report
}

func ResolveAssetReference(projectRoot string, slide deck.Slide, ref string, context string) (string, *diagnostics.Diagnostic) {
	if ref == "" || isExternalOrSpecialURL(ref) {
		return ref, nil
	}

	baseRef, suffix := splitURLSuffix(ref)
	if deckRef, ok := resolveDeckAssetReference(projectRoot, baseRef); ok {
		return deckRef + suffix, nil
	}

	candidate := filepath.Clean(filepath.Join(slide.BundlePath, filepath.FromSlash(baseRef)))
	if withinRoot(candidate, slide.BundlePath) {
		if _, err := os.Stat(candidate); err == nil {
			relPath, err := filepath.Rel(slide.BundlePath, candidate)
			if err == nil {
				return filepath.ToSlash(filepath.Join("slides", slide.ID, relPath)) + suffix, nil
			}
		}
	}

	return ref, &diagnostics.Diagnostic{
		Severity: diagnostics.SeverityWarning,
		Code:     "asset_missing",
		Message:  fmt.Sprintf("%s %q could not be resolved", context, ref),
		Path:     filepath.Join(slide.BundlePath, "index.md"),
	}
}

func resolveDeckAssetReference(projectRoot string, ref string) (string, bool) {
	assetsRoot := filepath.Join(projectRoot, "assets")

	candidateRefs := []string{ref}
	if trimmed := strings.TrimPrefix(filepath.ToSlash(ref), "assets/"); trimmed != ref {
		candidateRefs = append(candidateRefs, trimmed)
	}

	for _, candidateRef := range candidateRefs {
		candidate := filepath.Clean(filepath.Join(assetsRoot, filepath.FromSlash(candidateRef)))
		if !withinRoot(candidate, assetsRoot) {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			relPath, err := filepath.Rel(assetsRoot, candidate)
			if err == nil {
				return filepath.ToSlash(filepath.Join("assets", relPath)), true
			}
		}
	}

	return "", false
}

func splitURLSuffix(ref string) (string, string) {
	index := strings.IndexAny(ref, "?#")
	if index == -1 {
		return ref, ""
	}
	return ref[:index], ref[index:]
}

func isExternalOrSpecialURL(ref string) bool {
	lower := strings.ToLower(ref)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "data:") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "#") ||
		strings.HasPrefix(lower, "/")
}

func withinRoot(path string, root string) bool {
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relPath != ".." && !strings.HasPrefix(relPath, ".."+string(filepath.Separator))
}
