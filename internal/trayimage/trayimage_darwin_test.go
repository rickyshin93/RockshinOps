package trayimage

import (
	"bytes"
	"image/png"
	"testing"
)

func TestRenderDrawsRetinaStatusItemHeight(t *testing.T) {
	data, err := Render([]Item{{"internaldrive", "412.3 GB"}, {"cpu", "23%"}, {"battery.75", "86%"}})
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dy() != 36 || b.Dx() < 200 {
		t.Fatalf("bounds %v", b)
	}
	if _, err := Render([]Item{{"not.a.symbol", "1"}}); err == nil {
		t.Fatal("unknown symbol rendered")
	}
}
