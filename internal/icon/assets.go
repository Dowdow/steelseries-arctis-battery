package icon

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
)

//go:embed icon.ico
var iconICO []byte

//go:embed icon.png
var iconPNG []byte

// silhouette is the base headphone artwork, decoded once and reused as the
// source for Render's colorization (see render.go).
var silhouette = func() image.Image {
	img, err := png.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		panic(err)
	}
	return img
}()
