package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type ThemeOptions struct {
	ProjectRoot string
	Name        string
	Blank       bool
}

func CreateTheme(opts ThemeOptions) (string, error) {
	if opts.ProjectRoot == "" {
		return "", errors.New("project root is required")
	}
	if opts.Name == "" {
		return "", errors.New("theme name is required")
	}

	themeDir := filepath.Join(opts.ProjectRoot, "themes", opts.Name)
	if _, err := os.Stat(themeDir); err == nil {
		return "", fmt.Errorf("theme already exists: %s", themeDir)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat theme directory %q: %w", themeDir, err)
	}

	files := map[string]string{}
	if opts.Blank {
		files[filepath.Join("themes", opts.Name, "theme.yaml")] = defaultThemeMetadataWithName(opts.Name)
		files[filepath.Join("themes", opts.Name, "layouts", "deck.html")] = blankThemeDeckLayout()
		files[filepath.Join("themes", opts.Name, "layouts", "slide-default.html")] = blankThemeSlideLayout()
	} else {
		for path, content := range ThemeFiles(opts.Name, true) {
			files[path] = content
		}
	}

	for relativePath, content := range files {
		fullPath := filepath.Join(opts.ProjectRoot, relativePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return "", fmt.Errorf("create theme parent directory %q: %w", fullPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return "", fmt.Errorf("write theme file %q: %w", fullPath, err)
		}
	}

	return themeDir, nil
}

func blankThemeDeckLayout() string {
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Deck.Title }}</title>
</head>
<body>
  {{ range .Slides }}
  <section>{{ .Body }}</section>
  {{ end }}
</body>
</html>
`
}

func blankThemeSlideLayout() string {
	return `{{ .Body }}
`
}
