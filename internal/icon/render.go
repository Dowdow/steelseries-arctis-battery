package icon

import (
	"fmt"
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	canvasSize = 256

	// Badge geometry: a level circle bottom-right, a charging circle
	// bottom-left mirrored at the same position/size, matching the original
	// static badge.
	badgeCenterY  = 204.0
	badgeRadius   = 34.5
	levelCenterX  = 204.0
	chargeCenterX = canvasSize - levelCenterX // mirrored, = 52.0
	superSample   = 4

	textCenterX   = canvasSize / 2
	textCenterY   = 100
	maxTextWidth  = 248
	minGlyphScale = 7
	maxGlyphScale = 13
)

var (
	colorGreen     = color.NRGBA{34, 197, 94, 255}
	colorOrange    = color.NRGBA{249, 115, 22, 255}
	colorRed       = color.NRGBA{239, 68, 68, 255}
	colorWhite     = color.NRGBA{255, 255, 255, 255}
	colorLightBlue = color.NRGBA{147, 197, 253, 255}
)

// compose draws the plain black headphone silhouette with the battery
// percentage stamped in the middle (blocky "digital readout" font, no
// outline), a level-colored circle badge bottom-right, and — only while
// charging — a light blue circle badge bottom-left, the same size as the
// level circle. No shape in this composition is outlined; everything is a
// flat fill, per an explicit design call.
func compose(level int, charging bool) *image.NRGBA {
	img := cloneSilhouette()

	target := colorRed
	switch {
	case level >= 50:
		target = colorGreen
	case level >= 20:
		target = colorOrange
	}
	fillCircle(img, levelCenterX, badgeCenterY, badgeRadius, target)

	drawBlockyText(img, fmt.Sprintf("%d", level), textCenterX, textCenterY, colorWhite)

	if charging {
		fillCircle(img, chargeCenterX, badgeCenterY, badgeRadius, colorLightBlue)
	}

	return img
}

func cloneSilhouette() *image.NRGBA {
	b := silhouette.Bounds()
	dst := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x, y, silhouette.At(x, y))
		}
	}
	return dst
}

// fillCircle draws a flat, anti-aliased, unoutlined disc.
func fillCircle(dst *image.NRGBA, cx, cy, r float64, col color.NRGBA) {
	minX, maxX := int(cx-r)-1, int(cx+r)+1
	minY, maxY := int(cy-r)-1, int(cy+r)+1
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			coverage := 0.0
			for sy := 0; sy < superSample; sy++ {
				for sx := 0; sx < superSample; sx++ {
					px := float64(x) + (float64(sx)+0.5)/superSample
					py := float64(y) + (float64(sy)+0.5)/superSample
					dx, dy := px-cx, py-cy
					if dx*dx+dy*dy <= r*r {
						coverage++
					}
				}
			}
			blendPixel(dst, x, y, col, coverage/(superSample*superSample))
		}
	}
}

func blendPixel(dst *image.NRGBA, x, y int, col color.NRGBA, coverage float64) {
	if coverage <= 0 {
		return
	}
	if !(image.Point{x, y}.In(dst.Bounds())) {
		return
	}
	orig := dst.NRGBAAt(x, y)
	dst.SetNRGBA(x, y, color.NRGBA{
		R: uint8(float64(orig.R)*(1-coverage) + float64(col.R)*coverage),
		G: uint8(float64(orig.G)*(1-coverage) + float64(col.G)*coverage),
		B: uint8(float64(orig.B)*(1-coverage) + float64(col.B)*coverage),
		A: 255,
	})
}

// drawBlockyText stamps text using the tiny basicfont bitmap scaled up with
// nearest-neighbor, for a chunky, serif-free "digital readout" look — same
// font family as the keyboard OLED screen. Scale shrinks automatically so
// 1-3 digit values all fit within maxTextWidth. Flat fill, no outline.
func drawBlockyText(dst *image.NRGBA, text string, cx, cy int, fillCol color.NRGBA) {
	face := basicfont.Face7x13
	measurer := font.Drawer{Face: face}
	smallWidth := measurer.MeasureString(text).Ceil() + 4
	metrics := face.Metrics()
	smallHeight := (metrics.Ascent + metrics.Descent).Ceil() + 4

	scale := maxTextWidth / smallWidth
	if scale > maxGlyphScale {
		scale = maxGlyphScale
	}
	if scale < minGlyphScale {
		scale = minGlyphScale
	}

	mask := image.NewAlpha(image.Rect(0, 0, smallWidth, smallHeight))
	maskDrawer := font.Drawer{
		Dst:  mask,
		Src:  image.Opaque,
		Face: face,
		Dot:  fixed.P(2, metrics.Ascent.Ceil()+2),
	}
	maskDrawer.DrawString(text)

	originX := cx - smallWidth*scale/2
	originY := cy - smallHeight*scale/2
	stampMask(dst, mask, originX, originY, scale, fillCol)
}

func stampMask(dst *image.NRGBA, mask *image.Alpha, originX, originY, scale int, col color.NRGBA) {
	b := mask.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if mask.AlphaAt(x, y).A == 0 {
				continue
			}
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					dst.SetNRGBA(originX+x*scale+sx, originY+y*scale+sy, col)
				}
			}
		}
	}
}
