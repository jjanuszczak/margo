package theme

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ParseTemplateWithPartials(projectRoot string, activeTheme Metadata, entryPath string, funcs template.FuncMap) (*template.Template, error) {
	baseName := filepath.Base(entryPath)
	allFuncs := template.FuncMap{
		"dict": dict,
	}
	for name, fn := range funcs {
		allFuncs[name] = fn
	}

	tmpl := template.New(baseName).Funcs(allFuncs)

	partialPaths, err := resolvePartialPaths(projectRoot, activeTheme)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(partialPaths))
	for name := range partialPaths {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		source, err := os.ReadFile(partialPaths[name])
		if err != nil {
			return nil, fmt.Errorf("read partial %q: %w", name, err)
		}
		partialSource := string(source)
		if !strings.Contains(partialSource, "{{define") {
			partialSource = `{{define "` + name + `"}}` + partialSource + `{{end}}`
		}
		if _, err := tmpl.Parse(partialSource); err != nil {
			return nil, fmt.Errorf("parse partial %q: %w", name, err)
		}
	}

	if _, err := tmpl.ParseFiles(entryPath); err != nil {
		return nil, err
	}

	return tmpl, nil
}

func resolvePartialPaths(projectRoot string, activeTheme Metadata) (map[string]string, error) {
	partials := map[string]string{}
	for name, path := range activeTheme.Partials {
		partials[name] = path
	}

	deckPartials, err := discoverPartials(filepath.Join(projectRoot, "partials"))
	if err != nil {
		return nil, fmt.Errorf("read deck partials: %w", err)
	}
	for name, path := range deckPartials {
		partials[name] = path
	}
	return partials, nil
}

func dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict requires an even number of arguments")
	}
	result := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings")
		}
		result[key] = values[i+1]
	}
	return result, nil
}
