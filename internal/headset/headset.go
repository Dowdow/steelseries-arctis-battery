package headset

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Dowdow/steelseries-arctis-battery/internal/display"
	"github.com/Dowdow/steelseries-arctis-battery/internal/hid"
)

const STEELSERIES_VENDOR_ID = 0x1038

const (
	scanInterval = 5 * time.Second
	pullInterval = 1 * time.Minute
	readTimeout  = 2 * time.Second
)

type Headset struct {
	// Display name on the keyboard screen
	name               string
	vendorId           uint16
	productId          uint16
	inputBufferLength  uint16
	outputBufferLength uint16
	hidCommand         []byte
	batteryBufferIndex int
	// parseLevel converts the device response into a 0-100 battery level, for
	// models that don't report a plain percentage (status bytes, 0-4 scales...).
	// When nil, the byte at batteryBufferIndex is used as-is.
	parseLevel func(response []byte) (int, error)
	// chargingProductId is the product ID of the extra USB HID interface that
	// only appears while the headset is charging over USB (its firmware
	// update/bootloader interface, repurposed here only to check presence,
	// never written to). 0 when the headset has no such tell, or it isn't
	// known yet.
	chargingProductId uint16
}

var headsets = []*Headset{
	// Arctis 7 (2019)
	{
		name:               "Arctis 7",
		vendorId:           STEELSERIES_VENDOR_ID,
		productId:          0x12AD,
		inputBufferLength:  31,
		outputBufferLength: 31,
		hidCommand:         []byte{0x06, 0x18},
		batteryBufferIndex: 2,
		// The battery HID report never reports charging status (confirmed
		// against the wireless dongle, both by observation and by HeadsetControl's
		// own implementation). Charging over USB is detected instead by the
		// separate "Arctis 7 Bootloader" HID interface (product ID 0x12AE)
		// that appears on the bus only while the cable is plugged in.
		chargingProductId: 0x12AE,
	},
}

func Listen(ctx context.Context, wg *sync.WaitGroup, batteryMessageChannel chan display.Message) {
	defer wg.Done()

	var active atomic.Bool

	scan := func() {
		if active.Load() {
			return
		}

		if !display.Send(ctx, batteryMessageChannel, display.Message{Scanning: true}) {
			return
		}

		devices, err := hid.Enumerate()
		if err != nil {
			return
		}

		for _, headset := range headsets {
			for _, device := range devices {
				if !headset.matches(device) {
					continue
				}

				// Probing with a real battery read weeds out the other HID
				// interfaces exposed by the same dongle
				level, err := headset.readBattery(device.Path)
				if err != nil {
					continue
				}

				charging := headset.isCharging(devices)
				if !display.Send(ctx, batteryMessageChannel, display.Message{Name: headset.name, Level: level, Charging: charging}) {
					return
				}

				wg.Add(1)
				go headset.listen(ctx, wg, batteryMessageChannel, &active, device.Path)
				return
			}
		}
	}

	scan()

	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			scan()
		}
	}
}

func (h *Headset) listen(ctx context.Context, wg *sync.WaitGroup, batteryMessageChannel chan display.Message, active *atomic.Bool, path string) {
	defer wg.Done()

	active.Store(true)
	defer active.Store(false)

	ticker := time.NewTicker(pullInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			level, err := h.readBattery(path)
			if err != nil {
				// Headset gone: stop listening, the scan loop takes over again
				return
			}
			devices, _ := hid.Enumerate()
			charging := h.isCharging(devices)
			if !display.Send(ctx, batteryMessageChannel, display.Message{Name: h.name, Level: level, Charging: charging}) {
				return
			}
		}
	}
}

// isCharging reports whether this headset's charging companion interface is
// present among the given devices. Presence is only ever checked, never
// opened or written to — it's a bootloader/firmware-update interface.
func (h *Headset) isCharging(devices []hid.DeviceInfo) bool {
	if h.chargingProductId == 0 {
		return false
	}
	for _, d := range devices {
		if d.VendorID == h.vendorId && d.ProductID == h.chargingProductId {
			return true
		}
	}
	return false
}

func (h *Headset) matches(device hid.DeviceInfo) bool {
	if device.VendorID != h.vendorId || device.ProductID != h.productId {
		return false
	}
	if device.InputReportLength != 0 && device.InputReportLength != h.inputBufferLength {
		return false
	}
	if device.OutputReportLength != 0 && device.OutputReportLength != h.outputBufferLength {
		return false
	}
	return true
}

func (h *Headset) readBattery(path string) (int, error) {
	device, err := hid.Open(path)
	if err != nil {
		return 0, err
	}
	defer device.Close()

	command := make([]byte, h.outputBufferLength)
	copy(command, h.hidCommand)

	if err := device.Write(command); err != nil {
		return 0, err
	}

	response := make([]byte, h.inputBufferLength)
	read, err := device.ReadTimeout(response, readTimeout)
	if err != nil {
		return 0, err
	}
	if read <= h.batteryBufferIndex {
		return 0, fmt.Errorf("response too short: %d bytes", read)
	}

	if h.parseLevel != nil {
		return h.parseLevel(response[:read])
	}

	return int(response[h.batteryBufferIndex]), nil
}
