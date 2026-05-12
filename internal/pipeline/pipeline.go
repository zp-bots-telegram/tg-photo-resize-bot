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
	MaxEdge       = 10000
	MaxHalvings   = 4
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
// verbatim, and otherwise re-encodes as JPEG fitting TargetBytes /
// MaxEdge. mime is the declared MIME type and is used to gate the
// pass-through path.
func Process(orig []byte, mime string) (*Result, error) {
	img, err := vips.NewImageFromBuffer(orig)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	defer img.Close()

	origW := img.Width()
	origH := img.Height()

	if isJPEG(mime) && len(orig) <= TargetBytes && maxEdge(origW, origH) <= MaxEdge {
		return &Result{
			Bytes:       orig,
			Width:       origW,
			Height:      origH,
			OrigWidth:   origW,
			OrigHeight: origH,
			PassThrough: true,
		}, nil
	}

	out, w, h, err := encodeLoop(orig, origW, origH)
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

func encodeLoop(orig []byte, origW, origH int) ([]byte, int, int, error) {
	scale := 1.0
	for halvings := 0; halvings <= MaxHalvings; halvings++ {
		startQ := 95
		if halvings > 0 {
			startQ = 90
		}
		for q := startQ; q >= 80; q -= 5 {
			out, w, h, err := encodeOnce(orig, scale, q)
			if err != nil {
				return nil, 0, 0, err
			}
			if len(out) <= TargetBytes && maxEdge(w, h) <= MaxEdge {
				return out, w, h, nil
			}
		}
		scale *= 0.5
	}
	_ = origW
	_ = origH
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

func maxEdge(w, h int) int {
	if w > h {
		return w
	}
	return h
}
