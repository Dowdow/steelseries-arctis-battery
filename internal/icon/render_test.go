package icon

import (
	"image"
	"testing"
)

func sampleRGB(img *image.NRGBA, x, y int) (r, g, b uint8) {
	i := img.PixOffset(x, y)
	return img.Pix[i], img.Pix[i+1], img.Pix[i+2]
}

func TestComposeMatchesSilhouetteSize(t *testing.T) {
	img := compose(86, false)
	if img.Bounds() != silhouette.Bounds() {
		t.Errorf("compose() size = %v, want %v", img.Bounds(), silhouette.Bounds())
	}
}

func TestComposeKeepsHeadphoneBlack(t *testing.T) {
	// Sample a pixel on the headband, away from the badges and text: it
	// should be unchanged from the base black silhouette regardless of level.
	green := compose(86, false)
	red := compose(8, false)

	gr, gg, gb := sampleRGB(green, 60, 30)
	rr, rg, rb := sampleRGB(red, 60, 30)

	if gr != rr || gg != rg || gb != rb {
		t.Errorf("headphone color changed with level: green=(%d,%d,%d) red=(%d,%d,%d), want identical (black stays black)", gr, gg, gb, rr, rg, rb)
	}
}

func TestComposeChargingAddsCircle(t *testing.T) {
	notCharging := compose(50, false)
	charging := compose(50, true)

	x, y := int(chargeCenterX), int(badgeCenterY)
	nr, ng, nb := sampleRGB(notCharging, x, y)
	cr, cg, cb := sampleRGB(charging, x, y)

	if nr == colorLightBlue.R && ng == colorLightBlue.G && nb == colorLightBlue.B {
		t.Error("charge badge center is already light blue when not charging")
	}
	if cr != colorLightBlue.R || cg != colorLightBlue.G || cb != colorLightBlue.B {
		t.Errorf("charge badge center = (%d,%d,%d) while charging, want light blue (%d,%d,%d)", cr, cg, cb, colorLightBlue.R, colorLightBlue.G, colorLightBlue.B)
	}
}

func TestComposeThresholdsPickDifferentBadgeColors(t *testing.T) {
	green := compose(86, false)
	orange := compose(35, false)
	red := compose(8, false)

	x, y := int(levelCenterX), int(badgeCenterY)
	gr, gg, gb := sampleRGB(green, x, y)
	or, og, ob := sampleRGB(orange, x, y)
	rr, rg, rb := sampleRGB(red, x, y)

	if gr == or && gg == og && gb == ob {
		t.Error("green and orange thresholds produced the same badge color")
	}
	if gr == rr && gg == rg && gb == rb {
		t.Error("green and red thresholds produced the same badge color")
	}
}
