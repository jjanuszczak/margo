package content

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var includePattern = regexp.MustCompile(`{{<\s*include\s+"([^"]+)"\s*>}}`)

func resolveIncludes(projectRoot, sourcePath, markdown string) (string, error) {
	return resolveIncludesWithStack(projectRoot, sourcePath, markdown, []string{sourcePath})
}

func resolveIncludesWithStack(projectRoot, sourcePath, markdown string, stack []string) (string, error) {
	lines := strings.Split(markdown, "\n")
	for i, line := range lines {
		matches := includePattern.FindAllStringSubmatch(line, -1)
		if len(matches) == 0 {
			continue
		}

		expanded := line
		for _, match := range matches {
			includeRef := match[1]
			includePath, err := resolveIncludePath(projectRoot, includeRef)
			if err != nil {
				return "", fmt.Errorf("%s:%d: %w", sourcePath, i+1, err)
			}

			includeSource, err := os.ReadFile(includePath)
			if err != nil {
				return "", fmt.Errorf("%s:%d: read include %q: %w", sourcePath, i+1, includeRef, err)
			}
			if cycle := includeCycle(stack, includePath); cycle != "" {
				return "", fmt.Errorf("%s:%d: include cycle detected: %s", sourcePath, i+1, cycle)
			}

			resolved, err := resolveIncludesWithStack(projectRoot, includePath, string(includeSource), append(stack, includePath))
			if err != nil {
				return "", err
			}

			expanded = strings.Replace(expanded, match[0], resolved, 1)
		}
		lines[i] = expanded
	}

	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}

func resolveIncludePath(projectRoot, includeRef string) (string, error) {
	if strings.TrimSpace(includeRef) == "" {
		return "", fmt.Errorf("include path must not be empty")
	}

	cleanRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}

	candidate := filepath.Clean(filepath.Join(cleanRoot, includeRef))
	rel, err := filepath.Rel(cleanRoot, candidate)
	if err != nil {
		return "", fmt.Errorf("resolve include path %q: %w", includeRef, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("include path %q escapes the project root", includeRef)
	}

	return candidate, nil
}

func includeCycle(stack []string, includePath string) string {
	for i, path := range stack {
		if path != includePath {
			continue
		}

		paths := append(append([]string{}, stack[i:]...), includePath)
		for j, path := range paths {
			paths[j] = filepath.Base(path)
		}
		return strings.Join(paths, " -> ")
	}
	return ""
}
