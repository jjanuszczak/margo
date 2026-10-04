package diagnosticui

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"github.com/jjanuszczak/margo/internal/diagnostics"
)

const reportPath = "dist/margo-diagnostics.json"

func WriteReport(projectRoot string, report diagnostics.Report) error {
	if err := os.MkdirAll(filepath.Join(projectRoot, "dist"), 0o755); err != nil {
		return fmt.Errorf("create diagnostics directory: %w", err)
	}
	payload, err := json.MarshalIndent(struct {
		Items []diagnostics.Diagnostic `json:"items"`
	}{Items: report.Items}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode diagnostics report: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Join(projectRoot, "dist"), ".margo-diagnostics-*")
	if err != nil {
		return fmt.Errorf("create diagnostics report: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(append(payload, '\n')); err != nil {
		temporary.Close()
		return fmt.Errorf("write diagnostics report: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close diagnostics report: %w", err)
	}
	if err := os.Rename(temporaryPath, filepath.Join(projectRoot, reportPath)); err != nil {
		return fmt.Errorf("publish diagnostics report: %w", err)
	}
	return nil
}

func InjectPanel(path string, report diagnostics.Report) error {
	hasLayoutDiagnostic := false
	for _, item := range report.Items {
		if item.Code == "layout_overflow" || item.Code == "layout_fit" || item.Code == "layout_structure" {
			hasLayoutDiagnostic = true
			break
		}
	}
	if !hasLayoutDiagnostic {
		return nil
	}
	document, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read interactive diagnostics artifact: %w", err)
	}
	documentText := string(document)
	insertionPosition := strings.LastIndex(documentText, "</main>")
	if insertionPosition < 0 {
		insertionPosition = strings.LastIndex(documentText, "</body>")
	}
	if insertionPosition < 0 {
		return fmt.Errorf("interactive diagnostics artifact has no closing main or body tag")
	}
	payload, err := json.Marshal(struct {
		Items []diagnostics.Diagnostic `json:"items"`
	}{Items: report.Items})
	if err != nil {
		return fmt.Errorf("encode interactive diagnostics: %w", err)
	}
	staticRows := strings.Builder{}
	layoutWarningCount := 0
	for _, item := range report.Items {
		if item.Code != "layout_overflow" && item.Code != "layout_fit" && item.Code != "layout_structure" {
			continue
		}
		if item.Code == "layout_overflow" || item.Code == "layout_structure" {
			layoutWarningCount++
		}
		slideIndex := -1
		title := "Layout diagnostic"
		if item.Meta != nil {
			if value, ok := item.Meta["title"].(string); ok && value != "" {
				title = value
			}
			switch value := item.Meta["slide_index"].(type) {
			case int:
				slideIndex = value
			case float64:
				slideIndex = int(value)
			}
		}
		suggestion := "Review the content or theme contract."
		if item.Code == "layout_fit" {
			suggestion = "Automatic fitting was applied within the theme limit."
		}
		if item.Code == "layout_structure" {
			suggestion = "Use a full-width layout or populate both layout regions before reducing type size."
		}
		if item.Code == "layout_overflow" && item.Meta != nil {
			if proposals, ok := item.Meta["split_proposals"].([]map[string]any); ok && len(proposals) > 0 {
				if boundary, ok := proposals[0]["boundary"].(string); ok {
					suggestion = "Suggested split " + boundary + " (proposal only)."
				}
			}
		}
		staticRows.WriteString(`<button type="button" class="margo-layout-diagnostic" data-code="`)
		staticRows.WriteString(html.EscapeString(item.Code))
		staticRows.WriteString(`" data-margo-slide-index="`)
		staticRows.WriteString(fmt.Sprintf("%d", slideIndex))
		staticRows.WriteString(`"><strong>`)
		staticRows.WriteString(html.EscapeString(title))
		staticRows.WriteString(`</strong><span>`)
		staticRows.WriteString(html.EscapeString(item.Message))
		staticRows.WriteString(`</span><small>`)
		staticRows.WriteString(html.EscapeString(suggestion))
		staticRows.WriteString(`</small></button>`)
	}
	panel := fmt.Sprintf(`<style data-margo-diagnostics>
.margo-layout-diagnostics-button { position: relative; }
.margo-layout-diagnostics-count { display: inline-flex; min-width: 1.2em; justify-content: center; margin-left: .25em; padding: .05em .35em; border-radius: 999px; background: #b42318; color: #fff; font-size: .8em; }
.margo-layout-diagnostics-fallback { position: fixed; z-index: 1000; right: 1rem; bottom: 1rem; width: min(34rem, calc(100vw - 2rem)); max-height: min(70vh, 42rem); overflow: auto; padding: .5rem; border: 1px solid #ccd; border-radius: .75rem; background: Canvas; color: CanvasText; box-shadow: 0 1rem 3rem rgb(0 0 0 / 20%%); }
.margo-layout-diagnostics-fallback summary { cursor: pointer; padding: .5rem; font-weight: 600; }
.margo-layout-diagnostics-panel { position: fixed; z-index: 1000; right: 1rem; bottom: 1rem; width: min(34rem, calc(100vw - 2rem)); max-height: min(70vh, 42rem); overflow: auto; padding: 1rem; border: 1px solid #ccd; border-radius: .75rem; background: Canvas; color: CanvasText; box-shadow: 0 1rem 3rem rgb(0 0 0 / 20%%); }
.margo-layout-diagnostics-panel[hidden] { display: none; }
.margo-layout-diagnostics-header { display: flex; align-items: center; justify-content: space-between; gap: 1rem; margin-bottom: .75rem; }
.margo-layout-diagnostics-header h2 { margin: 0; font-size: 1rem; }
.margo-layout-diagnostics-close { border: 0; background: transparent; color: inherit; cursor: pointer; font-size: 1.25rem; }
.margo-layout-diagnostic { display: block; width: 100%%; margin: .6rem 0; padding: .65rem; border: 0; border-left: .25rem solid #b42318; background: #fff1f0; color: inherit; cursor: pointer; text-align: left; }
.margo-layout-diagnostic[data-code="layout_fit"] { border-left-color: #067647; background: #ecfdf3; }
.margo-layout-diagnostic button { border: 0; padding: 0; background: transparent; color: inherit; cursor: pointer; text-align: left; }
.margo-layout-diagnostic strong, .margo-layout-diagnostic span, .margo-layout-diagnostic small { display: block; }
.margo-layout-diagnostic small { margin-top: .25rem; opacity: .75; }
</style>
<details class="margo-layout-diagnostics-fallback" data-margo-layout-fallback>
  <summary class="slide-nav margo-layout-diagnostics-button">Layout<span class="margo-layout-diagnostics-count">%d</span></summary>
  %s
</details>
<script type="application/json" data-margo-diagnostics>%s</script>
<script data-margo-diagnostics-ui>
(() => {
  const payloadNode = document.querySelector('script[type="application/json"][data-margo-diagnostics]');
  const payload = JSON.parse((payloadNode && payloadNode.textContent) || '{"items":[]}');
	const relevant = (payload.items || []).filter(item => item.code === 'layout_overflow' || item.code === 'layout_fit' || item.code === 'layout_structure');
  if (!relevant.length) return;
  const fallback = document.querySelector('[data-margo-layout-fallback]');
  if (fallback) fallback.remove();
  const controls = document.querySelector('.controls') || document.querySelector('.deck');
  if (!controls) return;
  const button = document.createElement('button');
  button.type = 'button';
  button.className = 'slide-nav margo-layout-diagnostics-button';
  button.setAttribute('aria-expanded', 'false');
  button.textContent = 'Layout';
  const count = document.createElement('span');
  count.className = 'margo-layout-diagnostics-count';
  count.textContent = String(relevant.filter(item => item.code === 'layout_overflow').length);
  button.append(count);
  controls.append(button);
  const panel = document.createElement('aside');
  panel.className = 'margo-layout-diagnostics-panel';
  panel.hidden = true;
  panel.setAttribute('aria-label', 'Layout diagnostics');
  panel.innerHTML = '<div class="margo-layout-diagnostics-header"><h2>Layout diagnostics</h2><button class="margo-layout-diagnostics-close" type="button" aria-label="Close layout diagnostics">×</button></div><div data-margo-diagnostics-items></div>';
  document.body.append(panel);
  const list = panel.querySelector('[data-margo-diagnostics-items]');
  const slides = Array.from(document.querySelectorAll('.slide'));
  const metaFor = item => item.meta || {};
  const titleFor = item => metaFor(item).title || 'Slide ' + ((Number(metaFor(item).slide_index) || 0) + 1);
	const suggestionFor = item => {
		if (item.code === 'layout_fit') return 'Automatic fitting was applied within the theme limit.';
		if (item.code === 'layout_structure') return 'Use a full-width layout or populate both layout regions before reducing type size.';
    const proposals = item.meta && item.meta.split_proposals;
    if (proposals && proposals.length) return 'Suggested split ' + proposals[0].boundary + ' (proposal only).';
    return 'Review the content or theme contract.';
  };
  relevant.forEach(item => {
    const row = document.createElement('div');
    row.className = 'margo-layout-diagnostic';
    row.dataset.code = item.code || '';
    const jump = document.createElement('button');
    jump.type = 'button';
    jump.innerHTML = '<strong>' + titleFor(item) + '</strong><span>' + (item.message || '') + '</span><small>' + suggestionFor(item) + '</small>';
    jump.addEventListener('click', () => {
      const slide = slides[Number(metaFor(item).slide_index)];
      if (slide) {
        slides.forEach(candidate => candidate.classList.remove('active'));
        slide.classList.add('active');
        slide.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      }
    });
    row.append(jump);
    list.append(row);
  });
  const close = panel.querySelector('.margo-layout-diagnostics-close');
  button.addEventListener('click', () => { panel.hidden = !panel.hidden; button.setAttribute('aria-expanded', String(!panel.hidden)); });
  close.addEventListener('click', () => { panel.hidden = true; button.setAttribute('aria-expanded', 'false'); });
})();
</script>
`, layoutWarningCount, staticRows.String(), string(payload))
	output := make([]byte, 0, len(document)+len(panel))
	output = append(output, document[:insertionPosition]...)
	output = append(output, panel...)
	output = append(output, document[insertionPosition:]...)
	if err := os.WriteFile(path, output, 0o644); err != nil {
		return fmt.Errorf("write interactive diagnostics artifact: %w", err)
	}
	return nil
}
