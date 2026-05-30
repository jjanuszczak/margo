package theme

import (
	"fmt"
	"sort"
	"strings"
)

// ResolveOptions merges deck-provided theme options with theme metadata defaults
// and validates that provided option names and basic value types are allowed.
func ResolveOptions(meta Metadata, selected map[string]any) (map[string]any, error) {
	resolved := map[string]any{}
	declared := map[string]ConfigOption{}

	for _, option := range meta.ConfigOptions {
		declared[option.Name] = option
		if option.Default != nil {
			value, err := normalizeOptionValue(option, option.Default)
			if err != nil {
				return nil, fmt.Errorf("theme option %q default: %w", option.Name, err)
			}
			resolved[option.Name] = value
		}
	}

	for key, raw := range selected {
		if key == "name" {
			continue
		}
		option, ok := declared[key]
		if !ok {
			return nil, fmt.Errorf("unknown theme option %q for theme %q (supported: %s)", key, meta.Name, strings.Join(sortedOptionNames(declared), ", "))
		}
		value, err := normalizeOptionValue(option, raw)
		if err != nil {
			return nil, fmt.Errorf("theme option %q: %w", key, err)
		}
		resolved[key] = value
	}

	for _, option := range meta.ConfigOptions {
		if !option.Required {
			continue
		}
		raw, ok := resolved[option.Name]
		if !ok {
			return nil, fmt.Errorf("missing required theme option %q", option.Name)
		}
		if value, ok := raw.(string); ok && strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("missing required theme option %q", option.Name)
		}
	}

	return resolved, nil
}

func normalizeOptionValue(option ConfigOption, raw any) (any, error) {
	switch option.Type {
	case "", "string":
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", raw)
		}
		if len(option.Values) > 0 && !containsString(option.Values, value) {
			return nil, fmt.Errorf("invalid value %q (allowed: %s)", value, strings.Join(option.Values, ", "))
		}
		return value, nil
	case "bool":
		value, ok := raw.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool, got %T", raw)
		}
		return value, nil
	case "number":
		switch value := raw.(type) {
		case int:
			return float64(value), nil
		case int64:
			return float64(value), nil
		case float64:
			return value, nil
		default:
			return nil, fmt.Errorf("expected number, got %T", raw)
		}
	default:
		return raw, nil
	}
}

func sortedOptionNames(declared map[string]ConfigOption) []string {
	names := make([]string, 0, len(declared))
	for name := range declared {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
