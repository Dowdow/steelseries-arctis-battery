package oled

import (
	"testing"

	"github.com/Dowdow/steelseries-arctis-battery/internal/display"
)

func countPixels(frame []byte) int {
	count := 0
	for _, b := range frame {
		for ; b != 0; b &= b - 1 {
			count++
		}
	}
	return count
}

func TestRenderFrameSize(t *testing.T) {
	for _, message := range []display.Message{
		{Scanning: true},
		{Name: "Arctis 7", Level: 0},
		{Name: "Arctis 7", Level: 100},
		{Name: "Arctis 7", Level: -5},
		{Name: "Arctis 7", Level: 250},
	} {
		frame := render(message)
		if len(frame) != frameBytes {
			t.Errorf("render(%+v): frame is %d bytes, want %d", message, len(frame), frameBytes)
		}
		if countPixels(frame) == 0 {
			t.Errorf("render(%+v): frame is empty", message)
		}
	}
}

func TestRenderFillFollowsLevel(t *testing.T) {
	empty := countPixels(render(display.Message{Name: "Arctis 7", Level: 0}))
	half := countPixels(render(display.Message{Name: "Arctis 7", Level: 50}))
	full := countPixels(render(display.Message{Name: "Arctis 7", Level: 100}))
	if !(empty < half && half < full) {
		t.Errorf("progress bar fill does not grow with level: %d, %d, %d", empty, half, full)
	}
}

func TestRenderChargingDrawsMorePixels(t *testing.T) {
	for _, level := range []int{0, 50, 100} {
		notCharging := countPixels(render(display.Message{Name: "Arctis 7", Level: level, Charging: false}))
		charging := countPixels(render(display.Message{Name: "Arctis 7", Level: level, Charging: true}))
		if charging <= notCharging {
			t.Errorf("level %d: charging frame (%d px) should have more pixels than non-charging (%d px)", level, charging, notCharging)
		}
	}
}
