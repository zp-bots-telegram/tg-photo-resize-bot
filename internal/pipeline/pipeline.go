// Package pipeline prepares an input image for sendPhoto, preserving
// the original bytes when they already fit Telegram's upload limits
// and re-encoding only when necessary.
package pipeline

import (
	"errors"
	"fmt"

	"github.com/davidbyttow/govips/v2/vips"
)

const (
	MaxInputBytes = 20 * 1024 * 1024
	// TargetBytes is the sendPhoto multipart upload cap.
	TargetBytes = 10 * 1024 * 1024
	// MaxTotalDims is the rejection threshold: sendPhoto returns
	// PHOTO_INVALID_DIMENSIONS when width + height exceeds this.
	// Anything up to but not exceeding this sum is accepted; Telegram
	// stores a downsampled `w` PhotoSize (long edge 2560) but its own
	// downsampler has access to all the pixels we upload, so passing
	// through the full resolution typically yields a better final
	// preview than pre-resizing on our side.
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

// Process decides whether the input can be sent as-is to sendPhoto
// and otherwise re-encodes it to fit MaxTotalDims and TargetBytes.
//
// Pass-through eligibility: the format is one sendPhoto accepts
// (JPEG or PNG), the upload fits TargetBytes, and the dimension sum
// fits MaxTotalDims. In that case the bytes are forwarded verbatim
// — Telegram's own re-encode and downsample step then determines
// the stored quality, and giving it the full-resolution source
// generally beats pre-resizing ourselves.
func Process(orig []byte, mime string) (*Result, error) {
	img, err := vips.NewImageFromBuffer(orig)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	defer img.Close()

	origW := img.Width()
	origH := img.Height()

	if sendPhotoAccepts(mime) && len(orig) <= TargetBytes && origW+origH <= MaxTotalDims {
		return &Result{
			Bytes:       orig,
			Width:       origW,
			Height:      origH,
			OrigWidth:   origW,
			OrigHeight:  origH,
			PassThrough: true,
		}, nil
	}

	// Need to encode: HEIC source, oversized dimensions, or oversized
	// upload. Seed with the smallest scale needed to fit MaxTotalDims;
	// the loop drops quality (and ultimately scale) further if the
	// encoded bytes still exceed TargetBytes.
	initialScale := scaleToFitTotalDims(origW, origH, MaxTotalDims)
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

func sendPhotoAccepts(mime string) bool {
	switch mime {
	case "image/jpeg", "image/jpg", "image/png":
		return true
	}
	return false
}

func scaleToFitTotalDims(w, h, maxSum int) float64 {
	sum := w + h
	if sum <= maxSum {
		return 1.0
	}
	return float64(maxSum) / float64(sum)
}
