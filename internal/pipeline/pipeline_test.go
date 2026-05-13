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

func TestProcess_SmallPNG_PassThrough(t *testing.T) {
	// sendPhoto accepts PNG too, and Telegram does its own JPEG re-encode
	// on the server side. Pass the bytes through and let it.
	orig := makePNG(t, 800, 600)
	res, err := Process(orig, "image/png")
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if !res.PassThrough {
		t.Fatalf("expected pass-through for small PNG")
	}
	if !bytes.Equal(res.Bytes, orig) {
		t.Fatalf("pass-through bytes should equal input")
	}
}

func TestProcess_OversizePNG_Recoded(t *testing.T) {
	// PNG above the MaxTotalDims threshold can't pass through (sendPhoto
	// would reject), so we re-encode as JPEG inside the dim cap.
	orig := makePNG(t, 7000, 4000) // sum 11000, exceeds MaxTotalDims=10000
	res, err := Process(orig, "image/png")
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if res.PassThrough {
		t.Fatalf("oversize PNG must not pass through")
	}
	if res.Bytes[0] != 0xFF || res.Bytes[1] != 0xD8 {
		t.Fatalf("output is not a JPEG: %x...", res.Bytes[:4])
	}
	if res.Width+res.Height > MaxTotalDims {
		t.Fatalf("output dim sum %d exceeds MaxTotalDims=%d", res.Width+res.Height, MaxTotalDims)
	}
}

func TestProcess_HEIC_NeverPassesThrough(t *testing.T) {
	// We can't actually generate a HEIC in stdlib, but the eligibility
	// check is mime-driven: treat a JPEG-shaped fixture as if it were
	// HEIC to verify it's routed through the encode path.
	orig := makeJPEG(t, 800, 600, 95, 4)
	res, err := Process(orig, "image/heic")
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if res.PassThrough {
		t.Fatalf("HEIC mime must never pass through (sendPhoto won't accept it)")
	}
}

func TestProcess_LargeJPEG_OversizeUpload_Recoded(t *testing.T) {
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
		t.Fatalf("oversize JPEG should not pass through")
	}
	if len(res.Bytes) > TargetBytes {
		t.Fatalf("output %d bytes exceeds target %d", len(res.Bytes), TargetBytes)
	}
	if res.OrigWidth != 6000 || res.OrigHeight != 4000 {
		t.Fatalf("orig dims %dx%d", res.OrigWidth, res.OrigHeight)
	}
}

func TestProcess_OversizeDimsJPEG_Recoded(t *testing.T) {
	// 5500+4600 = 10100 > MaxTotalDims. Must be downsampled regardless
	// of file size.
	orig := makeJPEG(t, 5500, 4600, 70, 5)
	res, err := Process(orig, "image/jpeg")
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if res.PassThrough {
		t.Fatalf("oversize-dim JPEG must not pass through")
	}
	if res.Width+res.Height > MaxTotalDims {
		t.Fatalf("output dim sum %d exceeds MaxTotalDims=%d", res.Width+res.Height, MaxTotalDims)
	}
	if res.OrigWidth != 5500 || res.OrigHeight != 4600 {
		t.Fatalf("orig dims %dx%d", res.OrigWidth, res.OrigHeight)
	}
}

func TestProcess_AboveStoredEdgeButUnderLimits_PassesThrough(t *testing.T) {
	// 4000x3000 is above Telegram's 2560 stored long-edge cap but well
	// under MaxTotalDims and TargetBytes — pass the bytes through and
	// let Telegram do the downsample with all the pixels we have.
	orig := makeJPEG(t, 4000, 3000, 90, 6)
	if len(orig) > TargetBytes {
		t.Skipf("fixture too big to test pass-through: %d", len(orig))
	}
	res, err := Process(orig, "image/jpeg")
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if !res.PassThrough {
		t.Fatalf("expected pass-through, got re-encoded")
	}
	if !bytes.Equal(res.Bytes, orig) {
		t.Fatalf("pass-through bytes should equal input")
	}
}
