// Package pipeline turns an input image into a JPEG that fits within
// Telegram's sendPhoto limits, preserving the original bytes when
// possible to avoid generational quality loss.
package pipeline

import (
	"errors"
	"fmt"

	"github.com/davidbyttow/govips/v2/vips"
)

const (
	MaxInputBytes = 20 * 1024 * 1024
	TargetBytes   = 10 * 1024 * 1024
	// MaxStoredEdge is the long-edge size Telegram's bot pipeline will
	// actually keep for a photo: empirically 2560 (the `w` PhotoSize).
	// Anything larger gets silently downsampled by Telegram, so we do
	// the resize ourselves to control the resampler and save bandwidth.
	MaxStoredEdge = 2560
	// MaxTotalDims is the hard reject threshold: sendPhoto returns
	// PHOTO_INVALID_DIMENSIONS when width + height exceeds this.
	MaxTotalDims = 10000
	MaxHalvings  = 4
)

// ErrUncompressible is returned when the encode loop cannot bring the
// output under TargetBytes within MaxHalvings dimension reductions.
var ErrUncompressible = errors.New("image cannot be compressed under target size within shrink limit")

type Result struct {
	Bytes       []byte
	Width       int
	Height      int
	OrigWidth   int
	OrigHeight  int
	PassThrough bool
}

// Process decodes the input, decides whether it can be returned
// verbatim, and otherwise re-encodes as JPEG fitting MaxStoredEdge
// and TargetBytes. mime is the declared MIME type and is used to
// gate the pass-through path.
func Process(orig []byte, mime string) (*Result, error) {
	img, err := vips.NewImageFromBuffer(orig)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	defer img.Close()

	origW := img.Width()
	origH := img.Height()

	if isJPEG(mime) && len(orig) <= TargetBytes && longEdge(origW, origH) <= MaxStoredEdge {
		return &Result{
			Bytes:       orig,
			Width:       origW,
			Height:      origH,
			OrigWidth:   origW,
			OrigHeight:  origH,
			PassThrough: true,
		}, nil
	}

	initialScale := scaleToFit(origW, origH, MaxStoredEdge)
	out, w, h, err := encodeLoop(orig, initialScale)
	if err != nil {
		return nil, err
	}
	return &Result{
		Bytes:      out,
		Width:      w,
		Height:     h,
		OrigWidth:  origW,
		OrigHeight: origH,
	}, nil
}

func encodeLoop(orig []byte, startScale float64) ([]byte, int, int, error) {
	// Start at Q100 so Telegram's server-side re-encoder gets the
	// cleanest possible source. Empirically the PSNR delta between
	// Q80→TG and Q100→TG on a synthetic mixed-content 2560×1920 image
	// was 0.03 dB (well below the JND), but going high costs only our
	// upload bandwidth and avoids stacking generation loss on photos
	// where the input quality might matter more.
	scale := startScale
	for halvings := 0; halvings <= MaxHalvings; halvings++ {
		for q := 100; q >= 80; q -= 5 {
			out, w, h, err := encodeOnce(orig, scale, q)
			if err != nil {
				return nil, 0, 0, err
			}
			if len(out) <= TargetBytes && w+h <= MaxTotalDims {
				return out, w, h, nil
			}
		}
		scale *= 0.5
	}
	return nil, 0, 0, ErrUncompressible
}

func encodeOnce(orig []byte, scale float64, quality int) ([]byte, int, int, error) {
	img, err := vips.NewImageFromBuffer(orig)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decode: %w", err)
	}
	defer img.Close()

	if err := img.AutoRotate(); err != nil {
		return nil, 0, 0, fmt.Errorf("autorotate: %w", err)
	}

	if scale != 1.0 {
		if err := img.Resize(scale, vips.KernelLanczos3); err != nil {
			return nil, 0, 0, fmt.Errorf("resize: %w", err)
		}
	}

	params := vips.NewJpegExportParams()
	params.Quality = quality
	params.OptimizeCoding = true
	params.StripMetadata = false

	out, _, err := img.ExportJpeg(params)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("export: %w", err)
	}
	return out, img.Width(), img.Height(), nil
}

func isJPEG(mime string) bool {
	return mime == "image/jpeg" || mime == "image/jpg"
}

func longEdge(w, h int) int {
	if w > h {
		return w
	}
	return h
}

func scaleToFit(w, h, maxEdge int) float64 {
	long := longEdge(w, h)
	if long <= maxEdge {
		return 1.0
	}
	return float64(maxEdge) / float64(long)
}
