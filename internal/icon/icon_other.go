//go:build !windows

package icon

import (
	"bytes"
	"image/png"
)

// Default is the plain black headphone, used while scanning/idle. Linux
// (StatusNotifier) and macOS expect PNG data.
var Default = iconPNG

// Render draws the colorized, percentage-stamped headphone icon as PNG.
func Render(level int, charging bool) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, compose(level, charging)); err != nil {
		return Default
	}
	return buf.Bytes()
}
