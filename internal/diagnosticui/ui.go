package diagnosticui

import (
	"encoding/json"
	"fmt"
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
		if item.Code == "layout_overflow" || item.Code == "layout_fit" {
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
	bodyPosition := strings.LastIndex(string(document), "</body>")
	if bodyPosition < 0 {
		return fmt.Errorf("interactive diagnostics artifact has no closing body tag")
	}
	payload, err := json.Marshal(struct {
		Items []diagnostics.Diagnostic `json:"items"`
	}{Items: report.Items})
	if err != nil {
		return fmt.Errorf("encode interactive diagnostics: %w", err)
	}
	panel := fmt.Sprintf(`<style data-margo-diagnostics>
.margo-layout-diagnostics-button { position: relative; }
.margo-layout-diagnostics-count { display: inline-flex; min-width: 1.2em; justify-content: center; margin-left: .25em; padding: .05em .35em; border-radius: 999px; background: #b42318; color: #fff; font-size: .8em; }
.margo-layout-diagnostics-panel { position: fixed; z-index: 1000; right: 1rem; bottom: 1rem; width: min(34rem, calc(100vw - 2rem)); max-height: min(70vh, 42rem); overflow: auto; padding: 1rem; border: 1px solid #ccd; border-radius: .75rem; background: Canvas; color: CanvasText; box-shadow: 0 1rem 3rem rgb(0 0 0 / 20%%); }
.margo-layout-diagnostics-panel[hidden] { display: none; }
.margo-layout-diagnostics-header { display: flex; align-items: center; justify-content: space-between; gap: 1rem; margin-bottom: .75rem; }
.margo-layout-diagnostics-header h2 { margin: 0; font-size: 1rem; }
.margo-layout-diagnostics-close { border: 0; background: transparent; color: inherit; cursor: pointer; font-size: 1.25rem; }
.margo-layout-diagnostic { margin: .6rem 0; padding: .65rem; border-left: .25rem solid #b42318; background: #fff1f0; }
.margo-layout-diagnostic[data-code="layout_fit"] { border-left-color: #067647; background: #ecfdf3; }
.margo-layout-diagnostic button { border: 0; padding: 0; background: transparent; color: inherit; cursor: pointer; text-align: left; }
.margo-layout-diagnostic strong, .margo-layout-diagnostic span, .margo-layout-diagnostic small { display: block; }
.margo-layout-diagnostic small { margin-top: .25rem; opacity: .75; }
</style>
<script type="application/json" data-margo-diagnostics>%s</script>
<script data-margo-diagnostics-ui>
(() => {
  const payload = JSON.parse(document.querySelector('[data-margo-diagnostics]')?.textContent || '{"items":[]}');
  const relevant = (payload.items || []).filter(item => item.code === 'layout_overflow' || item.code === 'layout_fit');
  if (!relevant.length) return;
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
  const titleFor = item => item.meta?.title || 'Slide ' + ((Number(item.meta?.slide_index) || 0) + 1);
  const suggestionFor = item => item.code === 'layout_fit' ? 'Automatic fitting was applied within the theme limit.' : 'Review the content, theme contract, or a future split proposal.';
  relevant.forEach(item => {
    const row = document.createElement('div');
    row.className = 'margo-layout-diagnostic';
    row.dataset.code = item.code || '';
    const jump = document.createElement('button');
    jump.type = 'button';
    jump.innerHTML = '<strong>' + titleFor(item) + '</strong><span>' + (item.message || '') + '</span><small>' + suggestionFor(item) + '</small>';
    jump.addEventListener('click', () => {
      const slide = slides[Number(item.meta?.slide_index)];
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
`, string(payload))
	output := make([]byte, 0, len(document)+len(panel))
	output = append(output, document[:bodyPosition]...)
	output = append(output, panel...)
	output = append(output, document[bodyPosition:]...)
	if err := os.WriteFile(path, output, 0o644); err != nil {
		return fmt.Errorf("write interactive diagnostics artifact: %w", err)
	}
	return nil
}
