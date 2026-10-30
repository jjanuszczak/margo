package desktop

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	imagepng "image/png"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jjanuszczak/margo/internal/deck"
	"github.com/jjanuszczak/margo/internal/layoutaudit"
	htmloutput "github.com/jjanuszczak/margo/internal/output/html"
	"github.com/jjanuszczak/margo/internal/output/pdf"
	pngoutput "github.com/jjanuszczak/margo/internal/output/png"
	"github.com/jjanuszczak/margo/internal/theme"
)

type Thumbnail struct {
	ID    string `json:"id"`
	Image string `json:"image"`
}
type thumbnailJob struct {
	mu             sync.Mutex
	root, revision string
	images         []Thumbnail
	err            string
	done           bool
	cancel         context.CancelFunc
	stopped        chan struct{}
}

func (e *Engine) stopThumbnails() {
	if e.thumbnails != nil {
		e.thumbnails.cancel()
		<-e.thumbnails.stopped
		e.thumbnails = nil
	}
}
func (e *Engine) startThumbnails(revision string) (Response, error) {
	if e.thumbnails != nil && e.thumbnails.root == e.root && e.thumbnails.revision == revision {
		return e.thumbnailStatus(0), nil
	}
	if _, err := pdf.DetectBrowser(); err != nil {
		return Response{}, fmt.Errorf("slide thumbnails require Chrome/Chromium: %w", err)
	}
	model, active, err := e.load()
	if err != nil {
		return Response{}, err
	}
	current, err := projectRevision(e.root)
	if err != nil {
		return Response{}, err
	}
	if revision != current {
		return Response{Conflict: true}, fmt.Errorf("project changed before thumbnail capture")
	}
	e.stopThumbnails()
	directory, err := os.MkdirTemp(e.temp, "thumbnails-")
	if err != nil {
		return Response{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	job := &thumbnailJob{root: e.root, revision: revision, cancel: cancel, stopped: make(chan struct{})}
	e.thumbnails = job
	go func() {
		defer close(job.stopped)
		defer os.RemoveAll(directory)
		defer func() { job.mu.Lock(); job.done = true; job.mu.Unlock() }()
		for _, slide := range model.Slides {
			if ctx.Err() != nil {
				return
			}
			data, err := captureThumbnail(ctx, job.root, directory, model, active, slide)
			if ctx.Err() != nil {
				return
			}
			job.mu.Lock()
			if err != nil {
				job.err = err.Error()
				job.mu.Unlock()
				return
			}
			job.images = append(job.images, Thumbnail{ID: slide.ID, Image: data})
			job.mu.Unlock()
		}
	}()
	return e.thumbnailStatus(0), nil
}
func (e *Engine) thumbnailStatus(offset int) Response {
	if e.thumbnails == nil {
		return Response{Thumbnails: []Thumbnail{}, ThumbnailsDone: true}
	}
	job := e.thumbnails
	job.mu.Lock()
	defer job.mu.Unlock()
	if offset < 0 || offset > len(job.images) {
		offset = 0
	}
	return Response{Thumbnails: append([]Thumbnail{}, job.images[offset:]...), ThumbnailsDone: job.done, ThumbnailError: job.err, Revision: job.revision}
}
func captureThumbnail(ctx context.Context, root, directory string, model deck.Model, active theme.Metadata, slide deck.Slide) (string, error) {
	model.Slides = []deck.Slide{slide}
	relative, err := filepath.Rel(root, directory)
	if err != nil {
		return "", err
	}
	if _, err := htmloutput.WriteTo(root, relative, model, active); err != nil {
		return "", err
	}
	path := filepath.Join(directory, "index.html")
	if err := layoutaudit.ApplyStructuralRemediation(path, active.StructuralRemediationPolicies()); err != nil {
		return "", err
	}
	enabled, min := active.FitSettings()
	if err := layoutaudit.ApplyFit(path, layoutaudit.FitPolicy{Enabled: enabled, MinScale: min}); err != nil {
		return "", err
	}
	captureCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	output := filepath.Join(directory, "capture.png")
	// Remove the previous image so its completion marker cannot satisfy this capture.
	os.Remove(output)
	if err := pngoutput.CaptureHTML(captureCtx, path, output); err != nil {
		return "", err
	}
	file, err := os.Open(output)
	if err != nil {
		return "", err
	}
	defer file.Close()
	source, err := imagepng.Decode(file)
	if err != nil {
		return "", err
	}
	return thumbnailImage(source)
}

// Area sampling keeps text and thin theme details legible in the 320px filmstrip.
func thumbnailImage(source image.Image) (string, error) {
	const width, height = 320, 180
	target := image.NewNRGBA(image.Rect(0, 0, width, height))
	bounds := source.Bounds()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			left, right := bounds.Min.X+x*bounds.Dx()/width, bounds.Min.X+(x+1)*bounds.Dx()/width
			top, bottom := bounds.Min.Y+y*bounds.Dy()/height, bounds.Min.Y+(y+1)*bounds.Dy()/height
			if right <= left {
				right = left + 1
			}
			if bottom <= top {
				bottom = top + 1
			}
			var r, g, b, a, count uint64
			for sy := top; sy < bottom; sy++ {
				for sx := left; sx < right; sx++ {
					c := color.NRGBAModel.Convert(source.At(sx, sy)).(color.NRGBA)
					r += uint64(c.R)
					g += uint64(c.G)
					b += uint64(c.B)
					a += uint64(c.A)
					count++
				}
			}
			target.SetNRGBA(x, y, color.NRGBA{R: uint8(r / count), G: uint8(g / count), B: uint8(b / count), A: uint8(a / count)})
		}
	}
	var output bytes.Buffer
	if err := imagepng.Encode(&output, target); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(output.Bytes()), nil
}
