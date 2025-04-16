package icon

import (
	_ "embed"
)

//go:embed icon.ico
var Icon []byte

//go:embed icon-green.ico
var IconGreen []byte

//go:embed icon-orange.ico
var IconOrange []byte

//go:embed icon-red.ico
var IconRed []byte

//go:embed icon.png
var IconPng []byte

//go:embed icon-green.png
var IconPngGreen []byte

//go:embed icon-orange.png
var IconPngOrange []byte

//go:embed icon-red.png
var IconPngRed []byte
