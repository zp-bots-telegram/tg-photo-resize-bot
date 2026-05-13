// Package exifx extracts a small set of common EXIF tags from JPEG,
// PNG, and HEIC bytes. Parsing failures degrade to empty Tags rather
// than returning errors — the reply should still go through.
package exifx

import (
	"errors"
	"fmt"
	"strings"

	exif "github.com/dsoprea/go-exif/v3"
	heicexif "github.com/dsoprea/go-heic-exif-extractor/v2"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

type Tags struct {
	Make         string
	Model        string
	ISO          int
	FocalLength  float64 // mm
	FNumber      float64 // aperture
	ExposureTime string  // formatted, e.g. "1/250" or "0.5"
}

func (t Tags) Empty() bool {
	return t.Make == "" && t.Model == "" && t.ISO == 0 &&
		t.FocalLength == 0 && t.FNumber == 0 && t.ExposureTime == ""
}

// Extract returns the supported EXIF tags from data. Unrecognised
// formats and parse failures return zero Tags, nil — callers should
// render the caption without an EXIF block in that case.
func Extract(data []byte, mime string) (Tags, error) {
	raw, err := extractRaw(data, mime)
	if err != nil {
		return Tags{}, err
	}
	if raw == nil {
		return Tags{}, nil
	}
	return parseTags(raw), nil
}

func extractRaw(data []byte, mime string) ([]byte, error) {
	switch normalize(mime) {
	case "image/heif", "image/heic":
		mp := heicexif.NewHeicExifMediaParser()
		intfc, err := mp.ParseBytes(data)
		if err != nil {
			// Fall through to generic scan — some HEIC files store
			// EXIF in a way the parser doesn't recognise but bytes are
			// still findable by scanning.
			return searchAndExtract(data)
		}
		_, raw, err := intfc.Exif()
		if err != nil {
			if errors.Is(err, exif.ErrNoExif) {
				return nil, nil
			}
			return searchAndExtract(data)
		}
		return raw, nil
	default:
		return searchAndExtract(data)
	}
}

func searchAndExtract(data []byte) ([]byte, error) {
	raw, err := exif.SearchAndExtractExif(data)
	if err != nil {
		if errors.Is(err, exif.ErrNoExif) {
			return nil, nil
		}
		return nil, err
	}
	return raw, nil
}

func parseTags(raw []byte) Tags {
	var t Tags
	entries, _, err := exif.GetFlatExifData(raw, nil)
	if err != nil {
		return t
	}
	for _, e := range entries {
		switch e.TagName {
		case "Make":
			if t.Make == "" {
				t.Make = asString(e.Value)
			}
		case "Model":
			if t.Model == "" {
				t.Model = asString(e.Value)
			}
		case "ISOSpeedRatings", "PhotographicSensitivity":
			if t.ISO == 0 {
				t.ISO = asInt(e.Value)
			}
		case "FocalLength":
			if t.FocalLength == 0 {
				t.FocalLength = asRational(e.Value)
			}
		case "FNumber":
			if t.FNumber == 0 {
				t.FNumber = asRational(e.Value)
			}
		case "ExposureTime":
			if t.ExposureTime == "" {
				t.ExposureTime = asExposure(e.Value)
			}
		}
	}
	return t
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimRight(strings.TrimSpace(s), "\x00")
	}
	return ""
}

func asInt(v any) int {
	switch x := v.(type) {
	case []uint16:
		if len(x) > 0 {
			return int(x[0])
		}
	case []uint32:
		if len(x) > 0 {
			return int(x[0])
		}
	case uint16:
		return int(x)
	case uint32:
		return int(x)
	case int:
		return x
	}
	return 0
}

func asRational(v any) float64 {
	switch x := v.(type) {
	case []exifcommon.Rational:
		if len(x) > 0 && x[0].Denominator != 0 {
			return float64(x[0].Numerator) / float64(x[0].Denominator)
		}
	case []exifcommon.SignedRational:
		if len(x) > 0 && x[0].Denominator != 0 {
			return float64(x[0].Numerator) / float64(x[0].Denominator)
		}
	}
	return 0
}

func asExposure(v any) string {
	switch x := v.(type) {
	case []exifcommon.Rational:
		if len(x) == 0 || x[0].Denominator == 0 {
			return ""
		}
		num := x[0].Numerator
		den := x[0].Denominator
		if num == 0 {
			return "0"
		}
		// Reduce so 10/2500 → 1/250.
		g := gcd(num, den)
		num /= g
		den /= g
		if den == 1 {
			return fmt.Sprintf("%d", num)
		}
		if num > den {
			// Long exposure stored non-reduced, e.g. 5/2 → "2.5".
			return fmt.Sprintf("%g", float64(num)/float64(den))
		}
		return fmt.Sprintf("%d/%d", num, den)
	}
	return ""
}

func gcd(a, b uint32) uint32 {
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return 1
	}
	return a
}

func normalize(mime string) string {
	m := strings.ToLower(strings.TrimSpace(mime))
	if m == "image/jpg" {
		return "image/jpeg"
	}
	return m
}
