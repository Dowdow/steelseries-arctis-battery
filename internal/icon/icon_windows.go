//go:build windows

package icon

import (
	"bytes"
	"encoding/binary"
	"image/png"
)

// Default is the plain black headphone, used while scanning/idle. The
// Windows system tray expects ICO data.
var Default = iconICO

// Render draws the colorized, percentage-stamped headphone icon as a
// single-image ICO (the same PNG-in-ICO container format used by icon.ico).
func Render(level int, charging bool) []byte {
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, compose(level, charging)); err != nil {
		return Default
	}
	return wrapICO(pngBuf.Bytes())
}

func wrapICO(pngData []byte) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0, 0, 1, 0, 1, 0}) // ICONDIR: reserved, type=icon, count=1
	entry := make([]byte, 16)
	// width=0, height=0 mean 256px; colorCount/reserved left zero
	binary.LittleEndian.PutUint16(entry[4:6], 1)  // planes
	binary.LittleEndian.PutUint16(entry[6:8], 32) // bit count
	binary.LittleEndian.PutUint32(entry[8:12], uint32(len(pngData)))
	binary.LittleEndian.PutUint32(entry[12:16], 22) // offset: right after header+entry
	buf.Write(entry)
	buf.Write(pngData)
	return buf.Bytes()
}
