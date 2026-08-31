package pptx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"margo/internal/deck"
	"margo/internal/diagnostics"
	"margo/internal/theme"
)

const (
	OutputDir  = "dist/pptx"
	OutputFile = "dist/pptx/deck.pptx"

	slideWidth  int64 = 12192000
	slideHeight int64 = 6858000
	emuPerInch  int64 = 914400
)

var imageLinePattern = regexp.MustCompile(`^!\[([^]]*)\]\(([^)\s]+)(?:\s+[^)]*)?\)\s*$`)
var linkPattern = regexp.MustCompile(`\[([^]]+)\]\(([^)]+)\)`)
var shortcodePattern = regexp.MustCompile(`\{\{<[^>]+>\}\}`)

// Write creates an editable Open XML PowerPoint package. It deliberately does
// not consume rendered HTML, so adding this output cannot change HTML or PDF.
func Write(projectRoot string, model deck.Model, activeTheme theme.Metadata) (diagnostics.Report, error) {
	var report diagnostics.Report
	outputPath := filepath.Join(projectRoot, OutputFile)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return report, fmt.Errorf("create pptx output directory: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return report, fmt.Errorf("create pptx output: %w", err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	parts := packageParts(projectRoot, model, activeTheme, &report)
	for path, content := range parts {
		if err := writeZipPart(writer, path, content); err != nil {
			_ = writer.Close()
			return report, fmt.Errorf("write pptx part %q: %w", path, err)
		}
	}
	if err := writer.Close(); err != nil {
		return report, fmt.Errorf("close pptx output: %w", err)
	}
	return report, nil
}

func packageParts(projectRoot string, model deck.Model, activeTheme theme.Metadata, report *diagnostics.Report) map[string][]byte {
	parts := map[string][]byte{
		"[Content_Types].xml":                          []byte(contentTypes(len(model.Slides), modelHasNotes(model))),
		"_rels/.rels":                                  []byte(rootRels),
		"ppt/presentation.xml":                         []byte(presentationXML(len(model.Slides))),
		"ppt/_rels/presentation.xml.rels":              []byte(presentationRels(len(model.Slides), modelHasNotes(model))),
		"ppt/theme/theme1.xml":                         []byte(themeXML()),
		"ppt/slideMasters/slideMaster1.xml":            []byte(slideMasterXML()),
		"ppt/slideMasters/_rels/slideMaster1.xml.rels": []byte(slideMasterRels),
		"ppt/slideLayouts/slideLayout1.xml":            []byte(slideLayoutXML()),
		"ppt/slideLayouts/_rels/slideLayout1.xml.rels": []byte(slideLayoutRels),
	}
	if modelHasNotes(model) {
		parts["ppt/notesMasters/notesMaster1.xml"] = []byte(notesMasterXML())
		parts["ppt/notesMasters/_rels/notesMaster1.xml.rels"] = []byte(notesMasterRels)
	}

	for i, slide := range model.Slides {
		slideNumber := i + 1
		images := collectImages(projectRoot, slide, activeTheme, report)
		links := collectLinks(slide)
		for i := range links {
			links[i].Rel = fmt.Sprintf("rId%d", len(images)+2+i)
		}
		parts[fmt.Sprintf("ppt/slides/slide%d.xml", slideNumber)] = []byte(slideXML(model, slide, activeTheme, images, links, report))
		rels := slideRels(slideNumber, images, links, modelHasNotes(model))
		parts[fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", slideNumber)] = []byte(rels)
		for _, image := range images {
			parts[filepath.ToSlash(filepath.Join("ppt/media", image.Name))] = image.Data
		}
		if modelHasNotes(model) {
			parts[fmt.Sprintf("ppt/notesSlides/notesSlide%d.xml", slideNumber)] = []byte(notesSlideXML(slideNumber, slide))
			parts[fmt.Sprintf("ppt/notesSlides/_rels/notesSlide%d.xml.rels", slideNumber)] = []byte(notesSlideRels(slideNumber))
		}
	}
	return parts
}

type imagePart struct {
	Name       string
	Rel        string
	Data       []byte
	Ext        string
	Background bool
}

type linkPart struct {
	URL string
	Rel string
}

func collectImages(projectRoot string, slide deck.Slide, activeTheme theme.Metadata, report *diagnostics.Report) []imagePart {
	seen := map[string]bool{}
	var result []imagePart
	if activeTheme.PPTX != nil {
		if ref := activeTheme.PPTX.Assets["background"]; strings.TrimSpace(ref) != "" {
			if path := themeAssetPath(activeTheme, projectRoot, ref); path != "" {
				if data, err := os.ReadFile(path); err == nil {
					ext := normalizedImageExt(path)
					if ext != "" {
						result = append(result, imagePart{Name: "theme-background" + ext, Rel: "rId2", Data: data, Ext: ext, Background: true})
						seen[path] = true
					}
				}
			}
		}
	}
	for _, line := range strings.Split(slide.BodyMarkdown, "\n") {
		match := imageLinePattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(match) != 3 {
			continue
		}
		ref := strings.TrimSpace(match[2])
		if strings.Contains(ref, "://") || strings.HasPrefix(ref, "data:") {
			report.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityWarning, Code: "pptx_asset_fallback", Message: fmt.Sprintf("remote image %q is not embedded in PPTX", ref), Path: slide.BundlePath})
			continue
		}
		baseRef := ref
		if idx := strings.IndexAny(baseRef, "?#"); idx >= 0 {
			baseRef = baseRef[:idx]
		}
		path := filepath.Clean(filepath.Join(slide.BundlePath, filepath.FromSlash(baseRef)))
		if strings.HasPrefix(filepath.ToSlash(baseRef), "assets/") {
			path = filepath.Join(projectRoot, filepath.FromSlash(baseRef))
		}
		if seen[path] {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			report.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityWarning, Code: "asset_missing", Message: fmt.Sprintf("PPTX image asset %q could not be read", ref), Path: slide.BundlePath})
			continue
		}
		ext := normalizedImageExt(path)
		if ext != ".png" && ext != ".jpeg" && ext != ".gif" && ext != ".svg" {
			report.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityWarning, Code: "pptx_asset_fallback", Message: fmt.Sprintf("image type %q is not supported as a PPTX media part", ext), Path: slide.BundlePath})
			continue
		}
		name := fmt.Sprintf("slide-%s-%d%s", safeName(slide.ID), len(result)+1, ext)
		result = append(result, imagePart{Name: name, Rel: fmt.Sprintf("rId%d", len(result)+2), Data: data, Ext: ext})
		seen[path] = true
	}
	return result
}

func normalizedImageExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".jpg" {
		return ".jpeg"
	}
	if ext == ".png" || ext == ".jpeg" || ext == ".gif" || ext == ".svg" {
		return ext
	}
	return ""
}

func themeAssetPath(activeTheme theme.Metadata, projectRoot, ref string) string {
	ref = filepath.FromSlash(strings.TrimSpace(ref))
	if filepath.IsAbs(ref) {
		return ref
	}
	if strings.HasPrefix(filepath.ToSlash(ref), "assets/") {
		return filepath.Join(projectRoot, ref)
	}
	return filepath.Join(activeTheme.RootDir, ref)
}

func slideXML(model deck.Model, slide deck.Slide, activeTheme theme.Metadata, images []imagePart, links []linkPart, report *diagnostics.Report) string {
	background := "FFFFFF"
	foreground := "1F2937"
	accent := "8F6F33"
	if active := model.Config.Theme.Options; active != nil {
		accent = strings.TrimPrefix(option(active, "accent_color", "#"+accent), "#")
	}
	if activeTheme.PPTX != nil {
		background = pptxColor(activeTheme.PPTX.Colors, "background", background)
		foreground = pptxColor(activeTheme.PPTX.Colors, "foreground", foreground)
		accent = pptxColor(activeTheme.PPTX.Colors, "accent", accent)
	}
	if strings.EqualFold(option(model.Config.Theme.Options, "color_mode", "light"), "dark") {
		background, foreground = "111827", "F9FAFB"
	}
	if color := strings.TrimPrefix(strings.TrimSpace(slide.Background.Color), "#"); len(color) == 6 {
		background = strings.ToUpper(color)
	}
	layout := strings.ToLower(slide.Layout)
	if layout == "" {
		layout = strings.ToLower(slide.Type)
	}
	geometry := resolveLayoutGeometry(activeTheme, layout)
	var shapes strings.Builder
	shapes.WriteString(groupProperties())
	if len(images) > 0 && images[0].Background {
		shapes.WriteString(imageShape(images[0], 0, 0, slideWidth, slideHeight))
	}
	shapes.WriteString(textBox(2, geometry.titleX, geometry.titleY, geometry.titleWidth, geometry.titleHeight, slide.Title, 3000, accent, true, false, ""))

	bodyY := geometry.bodyY
	imageIndex := 0
	if len(images) > 0 && images[0].Background {
		imageIndex = 1
	}
	blocks := parseBlocks(slide.BodyMarkdown, slide, report)
	linkRels := make(map[string]string, len(links))
	for _, link := range links {
		linkRels[link.URL] = link.Rel
	}
	for _, block := range blocks {
		if block.kind == blockTable {
			shapes.WriteString(tableShape(block.rows, 700000, bodyY, 11000000, 1200000))
			bodyY += int64(len(block.rows))*420000 + 100000
			continue
		}
		if block.kind == blockImage {
			if imageIndex < len(images) {
				x, w := geometry.imageX, geometry.imageWidth
				position := strings.ToLower(hint(slide.ImageHints, "position"))
				if position == "left" {
					x = 700000
				} else if position == "center" {
					x = 3900000
				}
				if geometry.imageHeight == 0 {
					geometry.imageHeight = 2800000
				}
				shapes.WriteString(imageShape(images[imageIndex], x, bodyY, w, geometry.imageHeight))
				if caption := hint(slide.ImageHints, "caption"); caption != "" {
					shapes.WriteString(textBox(900+imageIndex, x, bodyY+geometry.imageHeight+50000, w, 300000, caption, 1100, foreground, false, false, ""))
				}
				imageIndex++
				if layout != "media-left" && layout != "media-right" {
					bodyY += geometry.imageHeight + 100000
				}
			}
			continue
		}
		if block.kind == blockHeading {
			if block.text == slide.Title {
				continue
			}
			shapes.WriteString(textBox(10+block.level, geometry.bodyX, bodyY, geometry.bodyWidth, 500000, block.text, 2200, foreground, block.level <= 2, false, ""))
			bodyY += 560000
			continue
		}
		if block.text == "" {
			continue
		}
		shapes.WriteString(textBox(20+int(bodyY), geometry.bodyX, bodyY, geometry.bodyWidth, 430000, block.text, 1700, foreground, false, block.bullet, linkRels[block.link]))
		bodyY += 470000
		if bodyY > 6200000 {
			break
		}
	}

	footer := slide.FooterText
	if strings.TrimSpace(footer) == "" {
		footer = model.Config.Deck.Footer
	}
	if !slide.HideFooter && strings.TrimSpace(footer) != "" {
		shapes.WriteString(textBox(100, geometry.bodyX, 6450000, geometry.bodyWidth, 300000, footer, 1100, foreground, false, false, ""))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:bg><p:bgPr><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:effectLst/></p:bgPr></p:bg><p:spTree>%s</p:spTree></p:cSld><p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr></p:sld>`, background, shapes.String())
}

type layoutGeometry struct {
	titleX, titleY, titleWidth, titleHeight int64
	bodyX, bodyY, bodyWidth, bodyHeight     int64
	imageX, imageWidth, imageHeight         int64
}

func resolveLayoutGeometry(activeTheme theme.Metadata, layout string) layoutGeometry {
	geometry := layoutGeometry{
		titleX: 500000, titleY: 300000, titleWidth: 11200000, titleHeight: 800000,
		bodyX: 700000, bodyY: 1250000, bodyWidth: 11000000, bodyHeight: 5000000,
		imageX: 7200000, imageWidth: 4300000, imageHeight: 2800000,
	}
	if strings.Contains(layout, "media-left") {
		geometry.imageX, geometry.imageWidth = 700000, 5200000
	}
	if layout == "image" {
		geometry.imageX, geometry.imageWidth, geometry.imageHeight = 700000, 11000000, 3600000
	}
	if activeTheme.PPTX == nil {
		return geometry
	}
	recipe, ok := activeTheme.PPTX.Layouts[layout]
	if !ok {
		return geometry
	}
	if recipe.BodyX > 0 {
		geometry.bodyX = inches(recipe.BodyX)
	}
	if recipe.BodyY > 0 {
		geometry.bodyY = inches(recipe.BodyY)
	}
	if recipe.BodyWidth > 0 {
		geometry.bodyWidth = inches(recipe.BodyWidth)
	}
	if recipe.BodyHeight > 0 {
		geometry.bodyHeight = inches(recipe.BodyHeight)
	}
	if recipe.ImageWidth > 0 {
		geometry.imageWidth = inches(recipe.ImageWidth)
	}
	if recipe.ImageHeight > 0 {
		geometry.imageHeight = inches(recipe.ImageHeight)
	}
	switch strings.ToLower(recipe.ImagePosition) {
	case "left":
		geometry.imageX = 700000
	case "center":
		geometry.imageX = (slideWidth - geometry.imageWidth) / 2
	case "right":
		geometry.imageX = slideWidth - geometry.imageWidth - 700000
	}
	return geometry
}

func inches(value float64) int64 {
	return int64(value*float64(emuPerInch) + 0.5)
}

const (
	blockText = iota
	blockHeading
	blockImage
	blockTable
)

type block struct {
	kind   int
	text   string
	link   string
	level  int
	bullet bool
	rows   [][]string
}

func parseBlocks(source string, slide deck.Slide, report *diagnostics.Report) []block {
	var blocks []block
	lines := strings.Split(strings.TrimSpace(source), "\n")
	for i := 0; i < len(lines); i++ {
		raw := lines[i]
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "---") {
			continue
		}
		if strings.HasPrefix(line, "{{") {
			if shortcodePattern.MatchString(line) || strings.HasPrefix(line, "{{") {
				report.Add(diagnostics.Diagnostic{Severity: diagnostics.SeverityWarning, Code: "pptx_unsupported_shortcode", Message: fmt.Sprintf("shortcode %q is omitted from PPTX; HTML/PDF remain unchanged", line), Path: slide.BundlePath})
			}
			continue
		}
		if isTableLine(line) {
			var rows [][]string
			for i < len(lines) && isTableLine(strings.TrimSpace(lines[i])) {
				row := splitTableRow(strings.TrimSpace(lines[i]))
				if len(row) > 0 && !isTableSeparator(row) {
					rows = append(rows, row)
				}
				i++
			}
			i--
			if len(rows) > 0 {
				blocks = append(blocks, block{kind: blockTable, rows: rows})
			}
			continue
		}
		if imageLinePattern.MatchString(line) {
			blocks = append(blocks, block{kind: blockImage})
			continue
		}
		if strings.HasPrefix(line, "#") {
			level := len(line) - len(strings.TrimLeft(line, "#"))
			blocks = append(blocks, block{kind: blockHeading, level: level, text: stripMarkdown(strings.TrimSpace(strings.TrimLeft(line, "#")))})
			continue
		}
		link := linkTarget(line)
		bullet := strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ")
		if bullet {
			line = strings.TrimSpace(line[2:])
		}
		line = stripMarkdown(line)
		if line != "" {
			blocks = append(blocks, block{kind: blockText, text: line, link: link, bullet: bullet})
		}
	}
	return blocks
}

func isTableLine(line string) bool {
	return strings.HasPrefix(line, "|") && strings.Count(line, "|") >= 2
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(line, "|"), "|"))
	if line == "" {
		return nil
	}
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = stripMarkdown(strings.TrimSpace(parts[i]))
	}
	return parts
}

func isTableSeparator(row []string) bool {
	if len(row) == 0 {
		return true
	}
	for _, cell := range row {
		if strings.Trim(cell, "-: ") != "" {
			return false
		}
	}
	return true
}

func collectLinks(slide deck.Slide) []linkPart {
	seen := map[string]bool{}
	var links []linkPart
	for _, line := range strings.Split(slide.BodyMarkdown, "\n") {
		url := linkTarget(line)
		if url == "" || seen[url] {
			continue
		}
		seen[url] = true
		links = append(links, linkPart{URL: url})
	}
	return links
}

func textBox(id int, x, y, w, h int64, text string, size int, color string, bold, bullet bool, hyperlink string) string {
	text = html.EscapeString(strings.TrimSpace(text))
	_ = hyperlink
	bulletXML := ""
	if bullet {
		bulletXML = "<a:buChar char=" + strconv.Quote("•") + "/>"
	}
	boldXML := ""
	if bold {
		boldXML = " b=" + strconv.Quote("1")
	}
	hyperlinkXML := ""
	if hyperlink != "" {
		hyperlinkXML = fmt.Sprintf(`<a:hlinkClick r:id="%s"/>`, html.EscapeString(hyperlink))
	}
	return fmt.Sprintf(`<p:sp><p:nvSpPr><p:cNvPr id="%d" name="Text %d"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr><p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:noFill/></p:spPr><p:txBody><a:bodyPr wrap="square"/><a:lstStyle/><a:p><a:pPr marL="0" indent="0">%s</a:pPr><a:r><a:rPr lang="en-US" sz="%d"%s><a:solidFill><a:srgbClr val="%s"/></a:solidFill>%s</a:rPr><a:t>%s</a:t></a:r><a:endParaRPr lang="en-US" sz="%d"/></a:p></p:txBody></p:sp>`, id, id, x, y, w, h, bulletXML, size, boldXML, color, hyperlinkXML, text, size)
}

func imageShape(image imagePart, x, y, w, h int64) string {
	return fmt.Sprintf(`<p:pic><p:nvPicPr><p:cNvPr id="%d" name="%s"/><p:cNvPicPr/><p:nvPr/></p:nvPicPr><p:blipFill><a:blip r:embed="%s"/><a:stretch><a:fillRect/></a:stretch></p:blipFill><p:spPr><a:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></p:spPr></p:pic>`, 1000, html.EscapeString(image.Name), image.Rel, x, y, w, h)
}

func tableShape(rows [][]string, x, y, w, h int64) string {
	if len(rows) == 0 {
		return ""
	}
	columns := 0
	for _, row := range rows {
		if len(row) > columns {
			columns = len(row)
		}
	}
	if columns == 0 {
		return ""
	}
	var grid strings.Builder
	for i := 0; i < columns; i++ {
		grid.WriteString(fmt.Sprintf(`<a:gridCol w="%d"/>`, w/int64(columns)))
	}
	var body strings.Builder
	rowHeight := h / int64(len(rows))
	for rowIndex, row := range rows {
		body.WriteString(fmt.Sprintf(`<a:tr h="%d">`, rowHeight))
		for column := 0; column < columns; column++ {
			cell := ""
			if column < len(row) {
				cell = html.EscapeString(row[column])
			}
			bold := ""
			if rowIndex == 0 {
				bold = ` b="1"`
			}
			body.WriteString(fmt.Sprintf(`<a:tc><a:txBody><a:bodyPr/><a:lstStyle/><a:p><a:r><a:rPr lang="en-US" sz="1300"%s/><a:t>%s</a:t></a:r></a:p></a:txBody><a:tcPr/></a:tc>`, bold, cell))
		}
		body.WriteString(`</a:tr>`)
	}
	return fmt.Sprintf(`<p:graphicFrame><p:nvGraphicFramePr><p:cNvPr id="500" name="Table"/><p:cNvGraphicFramePr/><p:nvPr/></p:nvGraphicFramePr><p:xfrm><a:off x="%d" y="%d"/><a:ext cx="%d" cy="%d"/></p:xfrm><a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/table"><a:tbl><a:tblPr firstRow="1" bandRow="1"><a:tableStyleId>{5C22544A-7EE6-4342-B048-85BDC9FD1C3A}</a:tableStyleId></a:tblPr><a:tblGrid>%s</a:tblGrid>%s</a:tbl></a:graphicData></a:graphic></p:graphicFrame>`, x, y, w, h, grid.String(), body.String())
}

func groupProperties() string {
	return `<p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/><a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm></p:grpSpPr>`
}

func slideRels(number int, images []imagePart, links []linkPart, notes bool) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>`)
	for _, image := range images {
		b.WriteString(fmt.Sprintf(`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/%s"/>`, image.Rel, html.EscapeString(image.Name)))
	}
	for i, link := range links {
		rel := fmt.Sprintf("rId%d", len(images)+2+i)
		b.WriteString(fmt.Sprintf(`<Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="%s" TargetMode="External"/>`, rel, html.EscapeString(link.URL)))
	}
	if notes {
		b.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesSlide" Target="../notesSlides/notesSlide%d.xml"/>`, len(images)+len(links)+2, number))
	}
	b.WriteString(`</Relationships>`)
	return b.String()
}

func notesSlideXML(number int, slide deck.Slide) string {
	var notes strings.Builder
	for _, note := range slide.Notes {
		if strings.TrimSpace(note.Markdown) != "" {
			notes.WriteString(textBox(20+number, 0, 0, 0, 0, stripMarkdown(note.Markdown), 1200, "000000", false, false, ""))
		}
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><p:notes xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree>%s</p:spTree></p:cSld><p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr></p:notes>`, notes.String())
}

func notesSlideRels(number int) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesMaster" Target="../notesMasters/notesMaster1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="../slides/slide%d.xml"/></Relationships>`, number)
}

func modelHasNotes(model deck.Model) bool {
	for _, slide := range model.Slides {
		if len(slide.Notes) > 0 {
			return true
		}
	}
	return false
}

func option(options map[string]any, key, fallback string) string {
	if value, ok := options[key].(string); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func hint(hints map[string]any, key string) string {
	if value, ok := hints[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func pptxColor(colors map[string]string, key, fallback string) string {
	value := strings.TrimPrefix(strings.TrimSpace(colors[key]), "#")
	if len(value) == 6 {
		return strings.ToUpper(value)
	}
	return strings.TrimPrefix(fallback, "#")
}

func stripMarkdown(value string) string {
	value = linkPattern.ReplaceAllString(value, "$1")
	value = strings.ReplaceAll(value, "`", "")
	value = strings.ReplaceAll(value, "**", "")
	value = strings.ReplaceAll(value, "__", "")
	return strings.TrimSpace(value)
}

func linkTarget(value string) string {
	match := linkPattern.FindStringSubmatch(value)
	if len(match) == 3 && (strings.HasPrefix(match[2], "https://") || strings.HasPrefix(match[2], "http://")) {
		return match[2]
	}
	return ""
}

func safeName(value string) string {
	value = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '-'
	}, value)
	if value == "" {
		return "slide"
	}
	return value
}

func writeZipPart(writer *zip.Writer, path string, content []byte) error {
	entry, err := writer.Create(filepath.ToSlash(path))
	if err != nil {
		return err
	}
	_, err = io.Copy(entry, bytes.NewReader(content))
	return err
}

func contentTypes(slides int, notes bool) string {
	var overrides strings.Builder
	overrides.WriteString(`<Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/><Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/><Override PartName="/ppt/slideLayouts/slideLayout1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/><Override PartName="/ppt/theme/theme1.xml" ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/>`)
	for i := 1; i <= slides; i++ {
		overrides.WriteString(fmt.Sprintf(`<Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, i))
		if notes {
			overrides.WriteString(fmt.Sprintf(`<Override PartName="/ppt/notesSlides/notesSlide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.notesSlide+xml"/>`, i))
		}
	}
	if notes {
		overrides.WriteString(`<Override PartName="/ppt/notesMasters/notesMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.notesMaster+xml"/>`)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Default Extension="png" ContentType="image/png"/><Default Extension="jpeg" ContentType="image/jpeg"/><Default Extension="jpg" ContentType="image/jpeg"/><Default Extension="gif" ContentType="image/gif"/><Default Extension="svg" ContentType="image/svg+xml"/>%s</Types>`, overrides.String())
}

const rootRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/></Relationships>`
const slideMasterRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/></Relationships>`
const slideLayoutRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/></Relationships>`
const notesMasterRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/></Relationships>`

func presentationXML(slides int) string {
	var ids strings.Builder
	for i := 1; i <= slides; i++ {
		ids.WriteString(fmt.Sprintf(`<p:sldId id="%d" r:id="rId%d"/>`, 255+i, i+1))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:sldMasterIdLst><p:sldMasterId id="2147483648" r:id="rId1"/></p:sldMasterIdLst><p:sldIdLst>%s</p:sldIdLst><p:sldSz cx="%d" cy="%d" type="screen16x9"/><p:notesSz cx="6858000" cy="9144000"/></p:presentation>`, ids.String(), slideWidth, slideHeight)
}

func presentationRels(slides int, notes bool) string {
	var rels strings.Builder
	rels.WriteString(`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="slideMasters/slideMaster1.xml"/>`)
	for i := 1; i <= slides; i++ {
		rels.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>`, i+1, i))
	}
	if notes {
		rels.WriteString(`<Relationship Id="rId999" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesMaster" Target="notesMasters/notesMaster1.xml"/>`)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">%s</Relationships>`, rels.String())
}

func slideMasterXML() string {
	return `<p:sldMaster xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree>` + groupProperties() + `</p:spTree></p:cSld><p:sldLayoutIdLst><p:sldLayoutId id="1" r:id="rId1"/></p:sldLayoutIdLst><p:clrMap accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" bg1="lt1" bg2="lt2" folHlink="folHlink" hlink="hlink" tx1="dk1" tx2="dk2"/></p:sldMaster>`
}

func slideLayoutXML() string {
	return `<p:sldLayout xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" type="blank" preserve="1"><p:cSld name="Blank"><p:spTree>` + groupProperties() + `</p:spTree></p:cSld><p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr></p:sldLayout>`
}

func notesMasterXML() string {
	return `<p:notesMaster xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:cSld><p:spTree>` + groupProperties() + `</p:spTree></p:cSld><p:clrMap accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" bg1="lt1" bg2="lt2" folHlink="folHlink" hlink="hlink" tx1="dk1" tx2="dk2"/></p:notesMaster>`
}

func themeXML() string {
	return `<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="Margo"><a:themeElements><a:clrScheme name="Margo"><a:dk1><a:sysClr val="windowText" lastClr="000000"/></a:dk1><a:lt1><a:sysClr val="window" lastClr="FFFFFF"/></a:lt1><a:dk2><a:srgbClr val="1F2937"/></a:dk2><a:lt2><a:srgbClr val="FFFFFF"/></a:lt2><a:accent1><a:srgbClr val="8F6F33"/></a:accent1><a:accent2><a:srgbClr val="4DB6AC"/></a:accent2><a:accent3><a:srgbClr val="2563EB"/></a:accent3><a:accent4><a:srgbClr val="DC2626"/></a:accent4><a:accent5><a:srgbClr val="7C3AED"/></a:accent5><a:accent6><a:srgbClr val="059669"/></a:accent6><a:hlink><a:srgbClr val="0563C1"/></a:hlink><a:folHlink><a:srgbClr val="954F72"/></a:folHlink></a:clrScheme><a:fontScheme name="Margo"><a:majorFont><a:latin typeface="Aptos Display"/></a:majorFont><a:minorFont><a:latin typeface="Aptos"/></a:minorFont></a:fontScheme><a:fmtScheme name="Margo"><a:fillStyleLst/><a:lnStyleLst/><a:effectStyleLst/><a:bgFillStyleLst/></a:fmtScheme></a:themeElements></a:theme>`
}
