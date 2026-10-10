package oled

import (
	"fmt"
	"image"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/Dowdow/steelseries-arctis-battery/internal/display"
)

const (
	screenWidth  = 128
	screenHeight = 40
	frameBytes   = screenWidth * screenHeight / 8
)

// Progress bar geometry
const (
	barTop    = 24
	barBottom = 38
)

// boltGlyph is a small lightning bolt, drawn next to the percentage while the
// headset is charging.
var boltGlyph = []string{
	"....XX.",
	"...XX..",
	"..XX...",
	".XXXXXX",
	"XXXXXX.",
	"...XX..",
	"..XX...",
	".XX....",
}

const (
	boltWidth = 7
	boltGap   = 3
)

// render draws a battery status message into a 1-bit frame, row-major,
// 8 horizontal pixels per byte, most significant bit first.
func render(message display.Message) []byte {
	img := image.NewGray(image.Rect(0, 0, screenWidth, screenHeight))
	drawer := font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: basicfont.Face7x13,
	}

	if message.Scanning {
		text := "Scanning..."
		width := drawer.MeasureString(text).Ceil()
		drawer.Dot = fixed.P((screenWidth-width)/2, 24)
		drawer.DrawString(text)
		return pack(img)
	}

	// Headset name on the left, battery percentage (with an optional
	// charging bolt) right-aligned
	drawer.Dot = fixed.P(0, 14)
	drawer.DrawString(message.Name)

	percent := fmt.Sprintf("%d%%", message.Level)
	percentWidth := drawer.MeasureString(percent).Ceil()

	x := screenWidth - percentWidth
	if message.Charging {
		x -= boltWidth + boltGap
		drawBolt(img, x, 6)
		x += boltWidth + boltGap
	}
	drawer.Dot = fixed.P(x, 14)
	drawer.DrawString(percent)

	drawBar(img, message.Level)

	return pack(img)
}

func drawBar(img *image.Gray, level int) {
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}

	white := image.White.C

	// Outline
	for x := range screenWidth {
		img.Set(x, barTop, white)
		img.Set(x, barBottom, white)
	}
	for y := barTop; y <= barBottom; y++ {
		img.Set(0, y, white)
		img.Set(screenWidth-1, y, white)
	}

	// Fill
	fill := (screenWidth - 4) * level / 100
	for y := barTop + 2; y <= barBottom-2; y++ {
		for x := 2; x < 2+fill; x++ {
			img.Set(x, y, white)
		}
	}
}

func drawBolt(img *image.Gray, originX, originY int) {
	white := image.White.C
	for row, line := range boltGlyph {
		for col, ch := range line {
			if ch == 'X' {
				img.Set(originX+col, originY+row, white)
			}
		}
	}
}

func pack(img *image.Gray) []byte {
	frame := make([]byte, frameBytes)
	for y := range screenHeight {
		for x := range screenWidth {
			if img.GrayAt(x, y).Y > 127 {
				frame[y*screenWidth/8+x/8] |= 0x80 >> (x % 8)
			}
		}
	}
	return frame
}
