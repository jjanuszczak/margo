// Package layoutaudit performs observation-only checks against rendered HTML.
package layoutaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jjanuszczak/margo/internal/diagnostics"
	"github.com/jjanuszczak/margo/internal/output/pdf"
)

const (
	defaultTimeout = 5 * time.Second
	timeoutEnvVar  = "MARGO_LAYOUT_AUDIT_TIMEOUT_SECONDS"
)

// Artifact describes a generated HTML surface to inspect.
type Artifact struct {
	Path                string
	Profile             string
	SlideSelector       string
	SlideLabel          string
	ViewportWidth       int
	ViewportHeight      int
	AllowVerticalScroll bool
	Fit                 *FitPolicy
}

type FitPolicy struct {
	Enabled  bool
	MinScale float64
}

type result struct {
	Findings    []finding    `json:"findings"`
	Adjustments []adjustment `json:"adjustments"`
}

type adjustment struct {
	Index int     `json:"index"`
	Title string  `json:"title"`
	Scale float64 `json:"scale"`
}

type finding struct {
	Index        int     `json:"index"`
	Title        string  `json:"title"`
	Direction    string  `json:"direction"`
	Amount       float64 `json:"amount"`
	Mechanism    string  `json:"mechanism"`
	ScrollWidth  float64 `json:"scrollWidth"`
	ClientWidth  float64 `json:"clientWidth"`
	ScrollHeight float64 `json:"scrollHeight"`
	ClientHeight float64 `json:"clientHeight"`
}

var auditAttribute = regexp.MustCompile(`data-margo-layout-audit="([^"]+)"`)
var scriptTag = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)

// Run audits each supplied artifact. Phase 1 reports findings as warnings and
// never changes the generated artifact or fails the build because of layout.
func Run(artifacts []Artifact) (diagnostics.Report, error) {
	var report diagnostics.Report
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.Path) == "" {
			continue
		}
		if _, err := os.Stat(artifact.Path); err != nil {
			return report, fmt.Errorf("stat layout audit artifact %q: %w", artifact.Path, err)
		}
		parsed, err := auditArtifact(artifact)
		if err != nil {
			if errors.Is(err, errBrowserUnavailable) {
				report.Add(diagnostics.Diagnostic{
					Severity: diagnostics.SeverityWarning,
					Code:     "layout_audit_unavailable",
					Message:  err.Error(),
					Path:     artifact.Path,
				})
				continue
			}
			report.Add(diagnostics.Diagnostic{
				Severity: diagnostics.SeverityWarning,
				Code:     "layout_audit_failed",
				Message:  err.Error(),
				Path:     artifact.Path,
			})
			continue
		}
		for _, item := range parsed.Findings {
			title := item.Title
			if title == "" {
				title = fmt.Sprintf("slide %d", item.Index+1)
			}
			report.Add(diagnostics.Diagnostic{
				Severity: diagnostics.SeverityWarning,
				Code:     "layout_overflow",
				Message: fmt.Sprintf("%s (%s) overflows %s by %s (%s)",
					title,
					artifact.Profile,
					item.Direction,
					formatPixels(item.Amount),
					item.Mechanism),
				Path: artifact.Path,
			})
		}
		for _, item := range parsed.Adjustments {
			title := item.Title
			if title == "" {
				title = fmt.Sprintf("slide %d", item.Index+1)
			}
			report.Add(diagnostics.Diagnostic{
				Severity: diagnostics.SeverityInfo,
				Code:     "layout_fit",
				Message:  fmt.Sprintf("%s (%s) fitted to %.0f%% scale", title, artifact.Profile, item.Scale*100),
				Path:     artifact.Path,
			})
		}
	}
	return report, nil
}

var errBrowserUnavailable = errors.New("layout audit browser unavailable")

func auditArtifact(artifact Artifact) (result, error) {
	browser, err := pdf.DetectBrowser()
	if err != nil {
		return result{}, fmt.Errorf("%w: %v; install Chrome/Chromium or set MARGO_CHROME_PATH", errBrowserUnavailable, err)
	}

	document, err := os.ReadFile(artifact.Path)
	if err != nil {
		return result{}, fmt.Errorf("read layout audit artifact: %w", err)
	}
	auditDocument, err := appendAuditScript(document, artifact)
	if err != nil {
		return result{}, err
	}
	auditPath := filepath.Join(filepath.Dir(artifact.Path), fmt.Sprintf(".margo-layout-audit-%d.html", time.Now().UnixNano()))
	if err := os.WriteFile(auditPath, auditDocument, 0o644); err != nil {
		return result{}, fmt.Errorf("write layout audit document: %w", err)
	}
	defer os.Remove(auditPath)

	profile, err := os.MkdirTemp("", "margo-layout-audit-profile-*")
	if err != nil {
		return result{}, fmt.Errorf("create layout audit browser profile: %w", err)
	}
	defer os.RemoveAll(profile)

	timeout := auditTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	args := []string{
		"--headless=new",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--disable-background-networking",
		"--allow-file-access-from-files",
		"--no-first-run",
		"--no-default-browser-check",
		"--hide-scrollbars",
		"--run-all-compositor-stages-before-draw",
		"--virtual-time-budget=2000",
		"--user-data-dir=" + profile,
		"--window-size=" + strconv.Itoa(artifact.ViewportWidth) + "," + strconv.Itoa(artifact.ViewportHeight),
		"--dump-dom",
		fileURL(auditPath),
	}
	cmd := exec.CommandContext(ctx, browser.Path, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return result{}, fmt.Errorf("capture layout audit output: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return result{}, fmt.Errorf("start layout audit with %q: %w", browser.Path, err)
	}

	chunks := make(chan []byte, 16)
	go func() {
		defer close(chunks)
		buffer := make([]byte, 32*1024)
		for {
			count, readErr := stdout.Read(buffer)
			if count > 0 {
				chunk := append([]byte(nil), buffer[:count]...)
				chunks <- chunk
			}
			if readErr != nil {
				return
			}
		}
	}()
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	var output []byte
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				chunks = nil
				continue
			}
			output = append(output, chunk...)
			if auditAttribute.Match(output) {
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
				for chunk := range chunks {
					output = append(output, chunk...)
				}
				<-waitCh
				return parseResults(output)
			}
		case err := <-waitCh:
			if chunks != nil {
				for chunk := range chunks {
					output = append(output, chunk...)
				}
			}
			if auditAttribute.Match(output) {
				return parseResults(output)
			}
			if err != nil {
				return result{}, fmt.Errorf("run layout audit with %q: %w: %s", browser.Path, err, strings.TrimSpace(stderr.String()))
			}
			return result{}, errors.New("layout audit browser output did not contain results")
		case <-timer.C:
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			if chunks != nil {
				for range chunks {
				}
			}
			<-waitCh
			return result{}, fmt.Errorf("layout audit timed out after %s using %s", timeout, browser.Path)
		case <-ctx.Done():
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			if chunks != nil {
				for range chunks {
				}
			}
			<-waitCh
			return result{}, ctx.Err()
		}
	}
}

func appendAuditScript(document []byte, artifact Artifact) ([]byte, error) {
	// Audit a static copy. Preview-only runtime modules and live-server polling
	// are not layout inputs, and can otherwise keep headless Chrome alive while
	// the audit is waiting for a DOM snapshot.
	document = scriptTag.ReplaceAll(document, nil)
	document = bytes.ReplaceAll(document,
		[]byte("https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs"),
		[]byte("data:text/javascript,export default {};"))
	headPosition := bytes.LastIndex(document, []byte("</head>"))
	if headPosition < 0 {
		return nil, errors.New("layout audit artifact does not contain a closing head tag")
	}
	bodyPosition := bytes.LastIndex(document, []byte("</body>"))
	if bodyPosition < 0 {
		return nil, errors.New("layout audit artifact does not contain a closing body tag")
	}
	selector, err := json.Marshal(artifact.SlideSelector)
	if err != nil {
		return nil, fmt.Errorf("encode layout audit selector: %w", err)
	}
	allowVerticalScroll, err := json.Marshal(artifact.AllowVerticalScroll)
	if err != nil {
		return nil, fmt.Errorf("encode layout audit scroll policy: %w", err)
	}
	fitEnabled := false
	fitMinScale := 1.0
	if artifact.Fit != nil && artifact.Fit.Enabled {
		fitEnabled = true
		fitMinScale = artifact.Fit.MinScale
		if fitMinScale <= 0 || fitMinScale > 1 {
			fitMinScale = 0.8
		}
	}
	script := fmt.Sprintf(`<script>
(() => {
  const selector = %s;
  const allowVerticalScroll = %s;
  const fitEnabled = %t;
  const fitMinScale = %f;
  const slides = Array.from(document.querySelectorAll(selector));
  const findings = [];
  const visible = element => {
    const style = window.getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    return style.display !== 'none' && style.visibility !== 'hidden' && rect.width > 0 && rect.height > 0;
  };
  const titleFor = slide => {
    const heading = slide.querySelector('h1,h2,h3,[data-slide-title]');
    return (heading?.textContent || '').trim().replace(/\\s+/g, ' ');
  };
  const measure = slide => {
    const bounds = slide.getBoundingClientRect();
    let left = 0, right = 0, top = 0, bottom = 0;
    slide.querySelectorAll('*').forEach(element => {
      if (!visible(element)) return;
      const rect = element.getBoundingClientRect();
      left = Math.max(left, bounds.left - rect.left);
      right = Math.max(right, rect.right - bounds.right);
      top = Math.max(top, bounds.top - rect.top);
      bottom = Math.max(bottom, rect.bottom - bounds.bottom);
    });
    const checks = [
      ['left', left, 'descendant'],
      ['right', right, 'descendant'],
      ['top', top, 'descendant'],
    ];
    if (!allowVerticalScroll) checks.push(['bottom', bottom, 'descendant']);
    if (slide.scrollWidth > slide.clientWidth + 1) checks.push(['right', slide.scrollWidth - slide.clientWidth, 'scroll container']);
    if (!allowVerticalScroll && slide.scrollHeight > slide.clientHeight + 1) checks.push(['bottom', slide.scrollHeight - slide.clientHeight, 'scroll container']);
    const largest = new Map();
    checks.forEach(([direction, amount, mechanism]) => {
      if (amount > 1 && (!largest.has(direction) || largest.get(direction).amount < amount)) {
        largest.set(direction, { amount, mechanism });
      }
    });
    return largest;
  };
  const fit = (slide, index) => {
    if (!fitEnabled) return null;
    const region = slide.querySelector('.slide-body, .print-slide-body');
    if (!region) return null;
    let scale = 1;
    for (let attempt = 0; attempt < 16; attempt++) {
      if (measure(slide).size === 0) break;
      scale = Math.max(fitMinScale, scale * 0.95);
      region.style.zoom = String(scale);
      if (scale <= fitMinScale) break;
    }
    return scale < 0.999 ? { index, title: titleFor(slide), scale } : null;
  };
  const inspect = () => {
    const adjustments = [];
    slides.forEach((slide, index) => {
      const before = slides.map(item => item.getAttribute('style'));
      if (slide.matches('.slide')) {
        slides.forEach(item => { item.style.display = 'none'; });
        slide.style.display = 'block';
      }
      const adjustment = fit(slide, index);
      if (adjustment) adjustments.push(adjustment);
      const largest = measure(slide);
      largest.forEach(({ amount, mechanism }, direction) => {
        findings.push({ index, title: titleFor(slide), direction, amount, mechanism, scrollWidth: slide.scrollWidth, clientWidth: slide.clientWidth, scrollHeight: slide.scrollHeight, clientHeight: slide.clientHeight });
      });
      slides.forEach((item, itemIndex) => {
        const style = before[itemIndex];
        if (style === null) item.removeAttribute('style'); else item.setAttribute('style', style);
      });
    });
    document.documentElement.setAttribute('data-margo-layout-audit', encodeURIComponent(JSON.stringify({ findings, adjustments })));
  };
  const waitForAssets = async () => {
    if (document.fonts?.ready) await document.fonts.ready;
    await Promise.all(Array.from(document.images).map(image => image.complete ? Promise.resolve() : new Promise(resolve => { image.addEventListener('load', resolve, { once: true }); image.addEventListener('error', resolve, { once: true }); })));
    setTimeout(inspect, 50);
  };
  waitForAssets();
})();
</script>`, string(selector), string(allowVerticalScroll), fitEnabled, fitMinScale)
	prelude := []byte(`<script>window.setInterval = () => 0; window.fetch = () => Promise.reject(new Error('layout audit'));</script>`)
	withPrelude := make([]byte, 0, len(document)+len(prelude))
	withPrelude = append(withPrelude, document[:headPosition]...)
	withPrelude = append(withPrelude, prelude...)
	withPrelude = append(withPrelude, document[headPosition:]...)
	adjustedBodyPosition := bodyPosition + len(prelude)
	result := make([]byte, 0, len(withPrelude)+len(script))
	result = append(result, withPrelude[:adjustedBodyPosition]...)
	result = append(result, script...)
	result = append(result, withPrelude[adjustedBodyPosition:]...)
	return result, nil
}

// ApplyFit adds the same bounded fitting behavior to a generated artifact so
// interactive preview and print/PDF rendering receive the approved adjustment.
func ApplyFit(path string, policy FitPolicy) error {
	if !policy.Enabled {
		return nil
	}
	document, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read fit artifact: %w", err)
	}
	bodyPosition := bytes.LastIndex(document, []byte("</body>"))
	if bodyPosition < 0 {
		return errors.New("fit artifact does not contain a closing body tag")
	}
	minScale := policy.MinScale
	if minScale <= 0 || minScale > 1 {
		minScale = 0.8
	}
	script := fmt.Sprintf(`<script data-margo-layout-fit>
(() => {
  const minScale = %f;
  const slides = Array.from(document.querySelectorAll('.slide, .print-slide'));
  const visible = element => {
    const style = window.getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    return style.display !== 'none' && style.visibility !== 'hidden' && rect.width > 0 && rect.height > 0;
  };
  const measure = slide => {
    const bounds = slide.getBoundingClientRect();
    let right = 0, bottom = 0;
    slide.querySelectorAll('*').forEach(element => {
      if (!visible(element)) return;
      const rect = element.getBoundingClientRect();
      right = Math.max(right, rect.right - bounds.right);
      bottom = Math.max(bottom, rect.bottom - bounds.bottom);
    });
    return Math.max(right, bottom, slide.scrollWidth - slide.clientWidth, slide.scrollHeight - slide.clientHeight);
  };
  const fit = () => {
    slides.forEach(slide => {
      const region = slide.querySelector('.slide-body, .print-slide-body');
      if (!region) return;
      let scale = 1;
      for (let attempt = 0; attempt < 16 && measure(slide) > 1 && scale > minScale; attempt++) {
        scale = Math.max(minScale, scale * 0.95);
        region.style.zoom = String(scale);
      }
    });
  };
  const ready = async () => {
    if (document.fonts?.ready) await document.fonts.ready;
    await Promise.all(Array.from(document.images).map(image => image.complete ? Promise.resolve() : new Promise(resolve => { image.addEventListener('load', resolve, { once: true }); image.addEventListener('error', resolve, { once: true }); })));
    requestAnimationFrame(() => requestAnimationFrame(fit));
  };
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', ready, { once: true }); else ready();
})();
</script>`, minScale)
	result := make([]byte, 0, len(document)+len(script))
	result = append(result, document[:bodyPosition]...)
	result = append(result, script...)
	result = append(result, document[bodyPosition:]...)
	if err := os.WriteFile(path, result, 0o644); err != nil {
		return fmt.Errorf("write fit artifact: %w", err)
	}
	return nil
}

func parseResults(output []byte) (result, error) {
	matches := auditAttribute.FindSubmatch(output)
	if len(matches) != 2 {
		return result{}, errors.New("layout audit browser output did not contain results")
	}
	decoded, err := urlQueryUnescape(string(matches[1]))
	if err != nil {
		return result{}, fmt.Errorf("decode layout audit results: %w", err)
	}
	var parsed result
	if err := json.Unmarshal([]byte(decoded), &parsed); err != nil {
		return result{}, fmt.Errorf("parse layout audit results: %w", err)
	}
	return parsed, nil
}

func urlQueryUnescape(value string) (string, error) {
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return "", err
	}
	return html.UnescapeString(decoded), nil
}

func fileURL(path string) string { return "file://" + filepath.ToSlash(path) }

func formatPixels(value float64) string {
	return fmt.Sprintf("%.0fpx", value)
}

func auditTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv(timeoutEnvVar))
	if raw == "" {
		return defaultTimeout
	}
	seconds, err := time.ParseDuration(raw + "s")
	if err != nil || seconds <= 0 {
		return defaultTimeout
	}
	return seconds
}
