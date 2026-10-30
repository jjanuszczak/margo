package png

import (
	"context"
	"fmt"
	"os"

	"github.com/jjanuszczak/margo/internal/output/pdf"
)

// CaptureHTML captures a rendered interactive or print document with the existing
// browser screenshot pipeline. Callers control cancellation and artifact lifetime.
func CaptureHTML(ctx context.Context, htmlPath, outputPath string) error {
	browser, err := pdf.DetectBrowser()
	if err != nil {
		return err
	}
	profile, err := os.MkdirTemp("", "margo-capture-*")
	if err != nil {
		return fmt.Errorf("create capture profile: %w", err)
	}
	defer os.RemoveAll(profile)
	return runBrowser(ctx, browser, buildScreenshotArgs(htmlPath, outputPath, profile), outputPath)
}
