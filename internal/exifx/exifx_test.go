package exifx

import (
	"testing"

	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

func TestEmpty(t *testing.T) {
	if !(Tags{}).Empty() {
		t.Fatalf("zero Tags should be Empty")
	}
	if (Tags{ISO: 100}).Empty() {
		t.Fatalf("Tags with ISO is not Empty")
	}
	if (Tags{Make: "Sony"}).Empty() {
		t.Fatalf("Tags with Make is not Empty")
	}
}

func TestAsString_TrimsNullAndSpace(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Sony", "Sony"},
		{"Sony\x00", "Sony"},
		{"  Sony  ", "Sony"},
		{"  Sony\x00\x00", "Sony"},
	}
	for _, c := range cases {
		if got := asString(c.in); got != c.want {
			t.Errorf("asString(%q) = %q want %q", c.in, got, c.want)
		}
	}
	if asString([]byte{1, 2, 3}) != "" {
		t.Errorf("non-string value should yield empty")
	}
}

func TestAsInt(t *testing.T) {
	if asInt([]uint16{400}) != 400 {
		t.Errorf("[]uint16 single")
	}
	if asInt([]uint16{}) != 0 {
		t.Errorf("[]uint16 empty")
	}
	if asInt([]uint32{800}) != 800 {
		t.Errorf("[]uint32")
	}
	if asInt(uint16(200)) != 200 {
		t.Errorf("uint16 scalar")
	}
	if asInt(nil) != 0 {
		t.Errorf("nil")
	}
}

func TestAsRational(t *testing.T) {
	if got := asRational([]exifcommon.Rational{{Numerator: 28, Denominator: 10}}); got != 2.8 {
		t.Errorf("28/10 = %v", got)
	}
	if got := asRational([]exifcommon.Rational{{Numerator: 50, Denominator: 1}}); got != 50.0 {
		t.Errorf("50/1 = %v", got)
	}
	if got := asRational([]exifcommon.Rational{{Numerator: 1, Denominator: 0}}); got != 0 {
		t.Errorf("div by zero must return 0, got %v", got)
	}
}

func TestAsExposure(t *testing.T) {
	cases := []struct {
		n, d uint32
		want string
	}{
		{1, 250, "1/250"},
		{10, 2500, "1/250"}, // reduced
		{1, 1, "1"},
		{2, 1, "2"},
		{0, 100, "0"},
		{1, 2000, "1/2000"}, // fast shutter kept as fraction (photographer-readable)
		{5, 2, "2.5"},       // long exposure non-reduced → decimal
		{30, 1, "30"},
	}
	for _, c := range cases {
		got := asExposure([]exifcommon.Rational{{Numerator: c.n, Denominator: c.d}})
		if got != c.want {
			t.Errorf("asExposure(%d/%d) = %q want %q", c.n, c.d, got, c.want)
		}
	}
	if asExposure(nil) != "" {
		t.Errorf("nil should be empty string")
	}
}

func TestNormalize(t *testing.T) {
	if normalize("Image/JPG") != "image/jpeg" {
		t.Errorf("jpg alias")
	}
	if normalize("  image/heic  ") != "image/heic" {
		t.Errorf("trim")
	}
}

func TestExtract_EmptyInput(t *testing.T) {
	tags, err := Extract([]byte{1, 2, 3, 4}, "image/jpeg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tags.Empty() {
		t.Fatalf("expected empty tags, got %+v", tags)
	}
}
