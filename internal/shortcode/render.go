package shortcode

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
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

	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"assetRef": func(ref string) string {
			return resolveAssetReference(ctx, ref)
		},
		"assetRefStrict": func(ref string) (string, error) {
			return resolveAssetReferenceStrict(ctx, ref)
		},
		"requiredParam": func(params map[string]string, key string) (string, error) {
			value := strings.TrimSpace(params[key])
			if value == "" {
				return "", fmt.Errorf("shortcode %q requires parameter %q", name, key)
			}
			return value, nil
		},
		"optionalParam": func(params map[string]string, key string) string {
			return strings.TrimSpace(params[key])
		},
		"optionalParamOneOf": func(params map[string]string, key string, allowed ...string) (string, error) {
			value := strings.TrimSpace(params[key])
			if value == "" {
				return "", nil
			}
			for _, candidate := range allowed {
				if value == candidate {
					return value, nil
				}
			}
			return "", fmt.Errorf("shortcode %q parameter %q must be one of %q", name, key, strings.Join(allowed, ", "))
		},
		"validateParams": func(shortcodeName string, params map[string]string, allowed ...string) (string, error) {
			allowedSet := make(map[string]struct{}, len(allowed))
			for _, key := range allowed {
				allowedSet[key] = struct{}{}
			}
			for key := range params {
				if _, ok := allowedSet[key]; !ok {
					return "", fmt.Errorf("shortcode %q does not support parameter %q", shortcodeName, key)
				}
			}
			return "", nil
		},
		"validateNoInner": func(shortcodeName, inner string) (string, error) {
			if strings.TrimSpace(inner) != "" {
				return "", fmt.Errorf("shortcode %q does not support inner content", shortcodeName)
			}
			return "", nil
		},
		"requiredInner": func(shortcodeName, inner string) (string, error) {
			if strings.TrimSpace(inner) == "" {
				return "", fmt.Errorf("shortcode %q requires inner content", shortcodeName)
			}
			return "", nil
		},
		"mustMatch": func(shortcodeName, value, label, pattern string) (string, error) {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return "", fmt.Errorf("shortcode %q invalid pattern for %s: %w", shortcodeName, label, err)
			}
			if !re.MatchString(strings.TrimSpace(value)) {
				return "", fmt.Errorf("shortcode %q %s must match %q", shortcodeName, label, pattern)
			}
			return value, nil
		},
		"chartConfigJSON": func(shortcodeName, inner string) (string, error) {
			return chartConfigJSON(shortcodeName, inner)
		},
		"chartID": func(shortcodeName string, params map[string]string, inner string) (string, error) {
			if id := strings.TrimSpace(params["id"]); id != "" {
				return id, nil
			}
			return generatedChartID(ctx.Slide.ID, params, inner, shortcodeName), nil
		},
	}).ParseFiles(path)
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

func resolveAssetReference(ctx Context, ref string) string {
	resolved, ok := resolveAssetReferenceWithStatus(ctx, ref)
	if ok {
		return resolved
	}
	return ref
}

func resolveAssetReferenceStrict(ctx Context, ref string) (string, error) {
	resolved, ok := resolveAssetReferenceWithStatus(ctx, ref)
	if ok {
		return resolved, nil
	}
	return "", fmt.Errorf("asset %q not found in slide bundle or deck assets", ref)
}

func resolveAssetReferenceWithStatus(ctx Context, ref string) (string, bool) {
	if ref == "" || isExternalOrSpecialURL(ref) {
		return ref, true
	}

	baseRef, suffix := splitURLSuffix(ref)
	if deckRef, ok := resolveDeckAssetReference(ctx.ProjectRoot, baseRef); ok {
		return deckRef + suffix, true
	}

	if ctx.Slide.BundlePath != "" {
		candidate := filepath.Clean(filepath.Join(ctx.Slide.BundlePath, filepath.FromSlash(baseRef)))
		if withinRoot(candidate, ctx.Slide.BundlePath) {
			if _, err := os.Stat(candidate); err == nil {
				relPath, err := filepath.Rel(ctx.Slide.BundlePath, candidate)
				if err == nil {
					return filepath.ToSlash(filepath.Join("slides", ctx.Slide.ID, relPath)) + suffix, true
				}
			}
		}
	}

	return ref, false
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

var supportedChartTypes = map[string]struct{}{
	"bar":      {},
	"line":     {},
	"pie":      {},
	"doughnut": {},
	"radar":    {},
}

func chartConfigJSON(shortcodeName, inner string) (string, error) {
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(strings.TrimSpace(inner)), &raw); err != nil {
		return "", fmt.Errorf("shortcode %q chart config must be valid YAML: %w", shortcodeName, err)
	}
	if len(raw) == 0 {
		return "", fmt.Errorf("shortcode %q chart config must be a non-empty object", shortcodeName)
	}

	chartType, ok := raw["type"].(string)
	if !ok || strings.TrimSpace(chartType) == "" {
		return "", fmt.Errorf("shortcode %q chart config requires string field %q", shortcodeName, "type")
	}
	if _, ok := supportedChartTypes[strings.TrimSpace(chartType)]; !ok {
		return "", fmt.Errorf("shortcode %q unsupported chart type %q", shortcodeName, chartType)
	}

	data, ok := raw["data"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("shortcode %q chart config requires object field %q", shortcodeName, "data")
	}
	datasets, ok := data["datasets"].([]any)
	if !ok || len(datasets) == 0 {
		return "", fmt.Errorf("shortcode %q chart config requires non-empty array field %q", shortcodeName, "data.datasets")
	}

	normalized := normalizeYAMLValue(raw)
	encoded, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return "", fmt.Errorf("shortcode %q encode chart config: %w", shortcodeName, err)
	}
	return string(encoded), nil
}

func normalizeYAMLValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(typed))
		for _, key := range keys {
			out[key] = normalizeYAMLValue(typed[key])
		}
		return out
	case map[any]any:
		keys := make([]string, 0, len(typed))
		tmp := make(map[string]any, len(typed))
		for key, item := range typed {
			textKey := fmt.Sprint(key)
			keys = append(keys, textKey)
			tmp[textKey] = item
		}
		sort.Strings(keys)
		out := make(map[string]any, len(tmp))
		for _, key := range keys {
			out[key] = normalizeYAMLValue(tmp[key])
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, normalizeYAMLValue(item))
		}
		return out
	default:
		return typed
	}
}

func generatedChartID(slideID string, params map[string]string, inner string, shortcodeName string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var material strings.Builder
	material.WriteString(slideID)
	material.WriteString("|")
	material.WriteString(shortcodeName)
	material.WriteString("|")
	material.WriteString(strings.TrimSpace(inner))
	for _, key := range keys {
		material.WriteString("|")
		material.WriteString(key)
		material.WriteString("=")
		material.WriteString(params[key])
	}

	sum := sha1.Sum([]byte(material.String()))
	return "margo-chart-" + hex.EncodeToString(sum[:6])
}
