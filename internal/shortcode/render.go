package shortcode

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"margo/internal/deck"
	"margo/internal/theme"
)

var tagPattern = regexp.MustCompile(`\{\{<\s*(/)?\s*([a-zA-Z0-9_-]+)(.*?)>\}\}`)
var attrPattern = regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_-]*)\s*=\s*"([^"]*)"`)

type Context struct {
	ProjectRoot string
	Theme       theme.Metadata
	Slide       deck.Slide
}

type templateData struct {
	Name   string
	Params map[string]string
	Inner  string
	Slide  deck.Slide
}

func Render(source string, ctx Context) (string, error) {
	rendered, next, closed, err := renderSegment(source, 0, "", ctx)
	if err != nil {
		return "", err
	}
	if closed != "" {
		return "", fmt.Errorf("unexpected shortcode closing tag %q", closed)
	}
	if next != len(source) {
		return "", fmt.Errorf("shortcode parser stopped early at byte %d", next)
	}
	return rendered, nil
}

func renderSegment(source string, start int, expectClose string, ctx Context) (string, int, string, error) {
	var out strings.Builder
	cursor := start

	for cursor < len(source) {
		loc := tagPattern.FindStringSubmatchIndex(source[cursor:])
		if loc == nil {
			out.WriteString(source[cursor:])
			if expectClose != "" {
				return "", 0, "", fmt.Errorf("missing closing shortcode tag for %q", expectClose)
			}
			return out.String(), len(source), "", nil
		}

		abs := make([]int, len(loc))
		for i, v := range loc {
			if v >= 0 {
				abs[i] = cursor + v
			} else {
				abs[i] = -1
			}
		}

		out.WriteString(source[cursor:abs[0]])

		isClosing := abs[2] != -1
		name := source[abs[4]:abs[5]]
		rawAttrs := strings.TrimSpace(source[abs[6]:abs[7]])
		cursor = abs[1]

		if isClosing {
			if expectClose == "" {
				return "", 0, name, nil
			}
			if name != expectClose {
				return "", 0, "", fmt.Errorf("unexpected closing shortcode tag %q; expected %q", name, expectClose)
			}
			return out.String(), cursor, name, nil
		}

		isSelfClosing := strings.HasSuffix(rawAttrs, "/")
		if isSelfClosing {
			rawAttrs = strings.TrimSpace(strings.TrimSuffix(rawAttrs, "/"))
		}

		params, err := parseAttrs(rawAttrs)
		if err != nil {
			return "", 0, "", fmt.Errorf("parse shortcode %q params: %w", name, err)
		}

		inner := ""
		if !isSelfClosing {
			var closed string
			inner, cursor, closed, err = renderSegment(source, cursor, name, ctx)
			if err != nil {
				return "", 0, "", err
			}
			if closed != name {
				return "", 0, "", fmt.Errorf("missing closing shortcode tag for %q", name)
			}
		}

		rendered, err := executeShortcode(name, params, inner, ctx)
		if err != nil {
			return "", 0, "", err
		}
		out.WriteString(rendered)
	}

	if expectClose != "" {
		return "", 0, "", fmt.Errorf("missing closing shortcode tag for %q", expectClose)
	}
	return out.String(), cursor, "", nil
}

func parseAttrs(raw string) (map[string]string, error) {
	params := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return params, nil
	}

	matches := attrPattern.FindAllStringSubmatchIndex(raw, -1)
	consumed := strings.Builder{}
	last := 0
	for _, match := range matches {
		prefix := raw[last:match[0]]
		if strings.TrimSpace(prefix) != "" {
			return nil, fmt.Errorf("unsupported shortcode arguments %q", strings.TrimSpace(raw))
		}
		key := raw[match[2]:match[3]]
		value := raw[match[4]:match[5]]
		params[key] = value
		consumed.WriteString(raw[match[0]:match[1]])
		last = match[1]
	}
	if strings.TrimSpace(raw[last:]) != "" {
		return nil, fmt.Errorf("unsupported shortcode arguments %q", strings.TrimSpace(raw))
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("unsupported shortcode arguments %q", strings.TrimSpace(raw))
	}
	return params, nil
}

func executeShortcode(name string, params map[string]string, inner string, ctx Context) (string, error) {
	path, err := resolveTemplatePath(ctx.ProjectRoot, ctx.Theme, name)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(name).ParseFiles(path)
	if err != nil {
		return "", fmt.Errorf("parse shortcode template %q: %w", path, err)
	}

	data := templateData{
		Name:   name,
		Params: params,
		Inner:  inner,
		Slide:  ctx.Slide,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, filepath.Base(path), data); err != nil {
		return "", fmt.Errorf("render shortcode %q: %w", name, err)
	}
	return buf.String(), nil
}

func resolveTemplatePath(projectRoot string, activeTheme theme.Metadata, name string) (string, error) {
	candidates := []string{
		filepath.Join(projectRoot, "shortcodes", name+".html"),
	}
	if activeTheme.RootDir != "" {
		candidates = append(candidates, filepath.Join(activeTheme.RootDir, "shortcodes", name+".html"))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat shortcode template %q: %w", candidate, err)
		}
	}

	return "", fmt.Errorf("unknown shortcode %q", name)
}
