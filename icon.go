package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
)

// diskIcon draws a two-pixel template outline at Retina resolution.
func diskIcon() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	ink := image.NewUniform(color.Black)
	for _, r := range []image.Rectangle{
		image.Rect(5, 7, 27, 9), image.Rect(3, 9, 5, 25),
		image.Rect(27, 9, 29, 25), image.Rect(5, 25, 27, 27),
		image.Rect(5, 18, 27, 20), image.Rect(21, 22, 24, 24),
	} {
		draw.Draw(img, r, ink, image.Point{}, draw.Src)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}
