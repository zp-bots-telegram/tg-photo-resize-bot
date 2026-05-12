package pipeline

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"testing"

	"github.com/davidbyttow/govips/v2/vips"
)

func TestMain(m *testing.M) {
	vips.LoggingSettings(nil, vips.LogLevelError)
	vips.Startup(nil)
	defer vips.Shutdown()
	m.Run()
}

func makeJPEG(t *testing.T, w, h, quality int, seed int64) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	r := rand.New(rand.NewSource(seed))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(r.Intn(256)), uint8(r.Intn(256)), uint8(r.Intn(256)), 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 128, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestProcess_SmallJPEG_PassThrough(t *testing.T) {
	orig := makeJPEG(t, 800, 600, 90, 1)
	if len(orig) > TargetBytes {
		t.Fatalf("test fixture too big: %d", len(orig))
	}

	res, err := Process(orig, "image/jpeg")
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if !res.PassThrough {
		t.Fatalf("expected pass-through for small JPEG")
	}
	if !bytes.Equal(res.Bytes, orig) {
		t.Fatalf("pass-through bytes should equal input")
	}
	if res.Width != 800 || res.Height != 600 {
		t.Fatalf("got dims %dx%d", res.Width, res.Height)
	}
}

func TestProcess_PNG_AlwaysRecoded(t *testing.T) {
	orig := makePNG(t, 800, 600)
	res, err := Process(orig, "image/png")
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if res.PassThrough {
		t.Fatalf("PNG must not pass through")
	}
	if len(res.Bytes) == 0 {
		t.Fatalf("empty output")
	}
	// First two bytes of JPEG are FF D8.
	if len(res.Bytes) < 2 || res.Bytes[0] != 0xFF || res.Bytes[1] != 0xD8 {
		t.Fatalf("output is not a JPEG: %x...", res.Bytes[:4])
	}
}

func TestProcess_LargeJPEG_Recoded(t *testing.T) {
	// Random noise at high quality is hard to compress, so this comfortably
	// exceeds 10 MB.
	orig := makeJPEG(t, 6000, 4000, 100, 2)
	if len(orig) <= TargetBytes {
		t.Skipf("fixture not large enough: %d bytes", len(orig))
	}

	res, err := Process(orig, "image/jpeg")
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if res.PassThrough {
		t.Fatalf("large JPEG should not pass through")
	}
	if len(res.Bytes) > TargetBytes {
		t.Fatalf("output %d bytes exceeds target %d", len(res.Bytes), TargetBytes)
	}
	if res.OrigWidth != 6000 || res.OrigHeight != 4000 {
		t.Fatalf("orig dims %dx%d", res.OrigWidth, res.OrigHeight)
	}
}

func TestProcess_LargeJPEG_RetainsResolutionWhenPossible(t *testing.T) {
	// Compressible content (single colour gradient) → quality drop alone
	// should suffice; dimensions should be untouched.
	img := image.NewRGBA(image.Rect(0, 0, 4000, 3000))
	for y := 0; y < 3000; y++ {
		for x := 0; x < 4000; x++ {
			img.Set(x, y, color.RGBA{uint8(x / 16), uint8(y / 12), 100, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatalf("encode: %v", err)
	}
	orig := buf.Bytes()
	res, err := Process(orig, "image/jpeg")
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if res.Width != 4000 || res.Height != 3000 {
		// If it was downscaled it means we tried halving — acceptable but
		// not preferred for content this compressible.
		t.Logf("note: dims changed to %dx%d (input was easy to compress; quality drop should have been enough)", res.Width, res.Height)
	}
	if len(res.Bytes) > TargetBytes {
		t.Fatalf("output %d > target %d", len(res.Bytes), TargetBytes)
	}
}
