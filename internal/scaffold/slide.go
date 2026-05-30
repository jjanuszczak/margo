package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"margo/internal/archetype"
	"margo/internal/content"
)

type SlideOptions struct {
	ProjectRoot string
	Name        string
	Archetype   string
}

func CreateSlide(opts SlideOptions) (string, error) {
	if opts.ProjectRoot == "" {
		return "", errors.New("project root is required")
	}
	if strings.TrimSpace(opts.Name) == "" {
		return "", errors.New("slide name is required")
	}

	meta, err := archetype.Load(opts.ProjectRoot, opts.Archetype)
	if err != nil {
		return "", err
	}

	slides, err := content.DiscoverSlides(opts.ProjectRoot)
	if err != nil {
		return "", fmt.Errorf("discover existing slides: %w", err)
	}

	nextOrder := 1
	for _, slide := range slides {
		if slide.Order >= nextOrder {
			nextOrder = slide.Order + 1
		}
	}

	slug := slugify(opts.Name)
	bundleDir := filepath.Join(opts.ProjectRoot, "slides", slug)
	if _, err := os.Stat(bundleDir); err == nil {
		return "", fmt.Errorf("slide bundle already exists: %s", bundleDir)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat slide bundle %q: %w", bundleDir, err)
	}

	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return "", fmt.Errorf("create slide bundle %q: %w", bundleDir, err)
	}

	indexPath := filepath.Join(bundleDir, "index.md")
	body := slideTemplate(opts.Name, nextOrder, meta)
	if err := os.WriteFile(indexPath, []byte(body), 0o644); err != nil {
		return "", fmt.Errorf("write slide file %q: %w", indexPath, err)
	}

	return indexPath, nil
}

func slideTemplate(name string, order int, meta archetype.Metadata) string {
	title := humanize(name)
	layout := meta.DefaultLayout
	if layout == "" {
		layout = "content"
	}
	slideType := meta.DefaultType
	if slideType == "" {
		slideType = "basic"
	}

	if layout == "section" || slideType == "section" {
		return fmt.Sprintf(`---
title: %s
order: %d
layout: section
type: section
section: %s
---

# %s
`, yamlString(title), order, yamlString(title), yamlString(title))
	}

	if layout == "title" || slideType == "title" {
		return fmt.Sprintf(`---
title: %s
order: %d
layout: title
type: title
---

# %s

Add a subtitle or opening statement.
`, yamlString(title), order, yamlString(title))
	}

	if layout == "agenda" || slideType == "agenda" {
		return fmt.Sprintf(`---
title: %s
order: %d
layout: agenda
type: agenda
---

## %s

1. First topic
2. Second topic
3. Third topic
`, yamlString(title), order, yamlString(title))
	}

	if layout == "image" || slideType == "image" {
		return fmt.Sprintf(`---
title: %s
order: %d
layout: image
type: image
image_hints:
  fit: cover
  position: center
  caption: Replace with a visual caption
---

## %s

![Describe the visual](image.png)

Short supporting statement for the image.
`, yamlString(title), order, yamlString(title))
	}

	if layout == "two-column" || slideType == "two-column" {
		return fmt.Sprintf(`---
title: %s
order: %d
layout: two-column
type: two-column
---

## %s

Left column content goes here.

<!-- column-break -->

Right column content goes here.
`, yamlString(title), order, yamlString(title))
	}

	if layout == "media-right" || slideType == "media-right" {
		return fmt.Sprintf(`---
title: %s
order: %d
layout: media-right
type: media-right
image_hints:
  fit: cover
  position: center
---

## %s

![Describe the visual](image.png)

- Primary point
- Supporting point
- Outcome or implication
`, yamlString(title), order, yamlString(title))
	}

	if layout == "media-left" || slideType == "media-left" {
		return fmt.Sprintf(`---
title: %s
order: %d
layout: media-left
type: media-left
image_hints:
  fit: cover
  position: center
---

## %s

![Describe the visual](image.png)

- Primary point
- Supporting point
- Outcome or implication
`, yamlString(title), order, yamlString(title))
	}

	if layout == "quote" || slideType == "quote" {
		return fmt.Sprintf(`---
title: %s
order: %d
layout: quote
type: quote
---

> Replace with a strong quote.

Attribution
`, yamlString(title), order)
	}

	if layout == "metric" || slideType == "metric" {
		return fmt.Sprintf(`---
title: %s
order: %d
layout: metric
type: metric
---

# 42%%

Primary metric label

Supporting context goes here.
`, yamlString(title), order)
	}

	if layout == "closing" || slideType == "closing" {
		return fmt.Sprintf(`---
title: %s
order: %d
layout: closing
type: closing
---

# Thank you

Closing thought, CTA, or contact information.
`, yamlString(title), order)
	}

	return fmt.Sprintf(`---
title: %s
order: %d
layout: %s
type: %s
---

## %s

Replace this content.
`, yamlString(title), order, yamlString(layout), yamlString(slideType), yamlString(title))
}

func slugify(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", "\\", "-", ".", "-")
	value = replacer.Replace(value)
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	value = strings.Trim(value, "-")
	if value == "" {
		return "new-slide"
	}
	return value
}

func humanize(value string) string {
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.TrimSpace(value)
	if value == "" {
		return "Untitled Slide"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
