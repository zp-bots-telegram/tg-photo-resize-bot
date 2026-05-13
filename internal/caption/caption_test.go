package caption

import (
	"strings"
	"testing"

	"github.com/zackpollard/tg-photo-resize-bot/internal/exifx"
)

func TestRender_ResolutionAlwaysPresent(t *testing.T) {
	got := Render(4000, 3000, exifx.Tags{})
	if !strings.HasPrefix(got, "<pre>") || !strings.HasSuffix(got, "</pre>") {
		t.Errorf("expected <pre>...</pre> envelope: %q", got)
	}
	if !strings.Contains(got, "Resolution") {
		t.Errorf("missing Resolution label: %q", got)
	}
	if !strings.Contains(got, "4000×3000") {
		t.Errorf("missing dims: %q", got)
	}
}

func TestRender_FullMetadataBlock(t *testing.T) {
	got := Render(6000, 4000, exifx.Tags{
		Make:         "SONY",
		Model:        "ILCE-7M3",
		ISO:          400,
		FocalLength:  85,
		FNumber:      1.8,
		ExposureTime: "1/250",
	})

	if !strings.HasPrefix(got, "<pre>") {
		t.Errorf("missing <pre> start: %q", got)
	}
	if !strings.HasSuffix(got, "</pre>") {
		t.Errorf("missing </pre>: %q", got)
	}
	for _, want := range []string{
		"Resolution",
		"6000×4000",
		"Camera",
		"SONY ILCE-7M3",
		"ISO",
		"400",
		"Lens",
		"85mm",
		"f/1.8",
		"Shutter",
		"1/250s",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected substring %q in %q", want, got)
		}
	}
}

func TestRender_OmitsMissingFields(t *testing.T) {
	got := Render(800, 600, exifx.Tags{ISO: 100})
	if strings.Contains(got, "Camera") {
		t.Errorf("Camera line should be omitted: %q", got)
	}
	if strings.Contains(got, "Lens") {
		t.Errorf("Lens line should be omitted: %q", got)
	}
	if strings.Contains(got, "Shutter") {
		t.Errorf("Shutter line should be omitted: %q", got)
	}
	if !strings.Contains(got, "ISO") {
		t.Errorf("ISO line should be present: %q", got)
	}
	if !strings.Contains(got, "Resolution") {
		t.Errorf("Resolution line should always be present: %q", got)
	}
}

func TestRender_EscapesHTML(t *testing.T) {
	got := Render(1, 1, exifx.Tags{Model: "<script>"})
	if strings.Contains(got, "<script>") {
		t.Errorf("raw HTML leaked: %q", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("expected escaped form: %q", got)
	}
}

func TestRender_LensOnlyFocal(t *testing.T) {
	got := Render(1, 1, exifx.Tags{FocalLength: 50})
	if !strings.Contains(got, "50mm") {
		t.Errorf("missing focal: %q", got)
	}
	if strings.Contains(got, "f/") {
		t.Errorf("should not have f-number when missing: %q", got)
	}
}
