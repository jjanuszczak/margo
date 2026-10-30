package desktop

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"
)

func TestThumbnailImageProducesSmallSlideAspectPNG(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 1920, 1080))
	for y := 0; y < 1080; y++ {
		for x := 0; x < 1920; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 130, G: 80, B: 40, A: 255})
		}
	}
	uri, err := thumbnailImage(source)
	if err != nil {
		t.Fatal(err)
	}
	encoded := strings.TrimPrefix(uri, "data:image/png;base64,")
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != 320 || decoded.Bounds().Dy() != 180 {
		t.Fatal(decoded.Bounds())
	}
	pixel := color.NRGBAModel.Convert(decoded.At(100, 100)).(color.NRGBA)
	if pixel.R != 130 || pixel.G != 80 || pixel.A != 255 {
		t.Fatal(pixel)
	}
}

func TestThumbnailsCaptureRealMargoOutputInBackground(t *testing.T) {
	if os.Getenv("MARGO_DESKTOP_THUMBNAIL_TEST") != "1" {
		t.Skip("opt-in real Chrome capture")
	}
	e := testEngine(t)
	defer e.stopThumbnails()
	snapshot := call(t, e, Request{Op: "snapshot"}).Snapshot
	call(t, e, Request{Op: "thumbnails", Revision: snapshot.Revision})
	deadline := time.Now().Add(55 * time.Second)
	for time.Now().Before(deadline) {
		// Snapshot remains available while captures run, with no blocking capture call.
		begin := time.Now()
		call(t, e, Request{Op: "snapshot"})
		if time.Since(begin) > 2*time.Second {
			t.Fatal("background capture blocked snapshot")
		}
		status := call(t, e, Request{Op: "thumbnail-status"})
		if status.ThumbnailError != "" {
			t.Fatal(status.ThumbnailError)
		}
		if status.ThumbnailsDone {
			if len(status.Thumbnails) != len(snapshot.Slides) {
				t.Fatalf("got %d thumbnails for %d slides", len(status.Thumbnails), len(snapshot.Slides))
			}
			for _, item := range status.Thumbnails {
				if !strings.HasPrefix(item.Image, "data:image/png;base64,") {
					t.Fatal("missing captured image")
				}
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("thumbnail capture did not complete")
}
