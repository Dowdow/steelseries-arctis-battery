// Package oled drives the OLED screen of SteelSeries Apex keyboards directly
// over HID, without going through the GameSense server. The protocol is
// reverse-engineered: a single output report (id 0x61) carrying a full
// 128x40 1-bit frame.
package oled

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Dowdow/steelseries-arctis-battery/internal/display"
	"github.com/Dowdow/steelseries-arctis-battery/internal/hid"
)

const STEELSERIES_VENDOR_ID = 0x1038

const (
	frameReportID     = 0x61
	scanRetryInterval = 30 * time.Second
)

type Keyboard struct {
	name      string
	vendorId  uint16
	productId uint16
}

var keyboards = []*Keyboard{
	{name: "Apex Pro", vendorId: STEELSERIES_VENDOR_ID, productId: 0x1610},
	{name: "Apex 7", vendorId: STEELSERIES_VENDOR_ID, productId: 0x1612},
	{name: "Apex Pro TKL", vendorId: STEELSERIES_VENDOR_ID, productId: 0x1614},
	{name: "Apex 7 TKL", vendorId: STEELSERIES_VENDOR_ID, productId: 0x1618},
}

// Display renders battery status frames on the keyboard OLED screen.
type Display struct{}

var _ display.Display = (*Display)(nil)

func New() *Display {
	return &Display{}
}

// screen is an open handle to the keyboard's vendor HID interface.
type screen struct {
	device   *hid.Device
	writeLen int
}

func (o *Display) Listen(ctx context.Context, wg *sync.WaitGroup, messages <-chan display.Message) {
	defer wg.Done()

	var current *screen
	var lastScan time.Time

	defer func() {
		if current != nil {
			current.clear()
			current.device.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case message := <-messages:
			if message.Off {
				if current != nil {
					current.clear()
				}
				continue
			}

			frame := render(message)

			if current == nil {
				if time.Since(lastScan) < scanRetryInterval {
					continue
				}
				lastScan = time.Now()
				// The probe frame doubles as the first render
				current = openScreen(frame)
				continue
			}

			if err := current.writeFrame(frame); err != nil {
				log.Printf("OLED: lost keyboard screen: %v", err)
				current.device.Close()
				current = nil
			}
		}
	}
}

// openScreen looks for a supported keyboard and returns a handle to its vendor
// interface, validated by writing the given frame to it.
func openScreen(frame []byte) *screen {
	devices, err := hid.Enumerate()
	if err != nil {
		return nil
	}

	for _, keyboard := range keyboards {
		for _, device := range devices {
			if device.VendorID != keyboard.vendorId || device.ProductID != keyboard.productId {
				continue
			}

			// The screen sits on the vendor-defined interface (usage page
			// 0xFFC0), not on the keyboard/media interfaces
			if device.UsagePage != 0 && device.UsagePage != 0xFFC0 {
				continue
			}

			handle, err := hid.Open(device.Path)
			if err != nil {
				continue
			}

			// Some platforms require writes of exactly the output report size
			writeLen := max(int(device.OutputReportLength), 1+frameBytes)

			candidate := &screen{device: handle, writeLen: writeLen}
			if err := candidate.writeFrame(frame); err != nil {
				handle.Close()
				continue
			}

			log.Printf("OLED: using %s screen (%s)", keyboard.name, device.Path)
			return candidate
		}
	}

	return nil
}

func (s *screen) writeFrame(frame []byte) error {
	buffer := make([]byte, s.writeLen)
	buffer[0] = frameReportID
	copy(buffer[1:], frame)
	return s.device.Write(buffer)
}

// clear blanks the screen. There is no known command to hand control back to
// the keyboard firmware's own screen, so an all-off frame is the closest we
// can get to restoring the pre-app state.
func (s *screen) clear() {
	if err := s.writeFrame(make([]byte, frameBytes)); err != nil {
		log.Printf("OLED: failed to clear screen: %v", err)
	}
}
