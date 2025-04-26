package headset

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Dowdow/steelseries-arctis-battery/hid"
	"github.com/Dowdow/steelseries-arctis-battery/setupapi"
)

const STEELSERIES_VENDOR_ID = 0x1038

type Headset struct {
	name               string
	vendorId           uint16
	productId          uint16
	inputBufferLength  uint16
	outputBufferLength uint16
	hidCommand         []byte
	batteryBufferIndex int
}

type HeadsetBatteryMessage struct {
	Name     string
	Level    int
	Scanning bool
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
	},
}

func Listen(ctx context.Context, wg *sync.WaitGroup, batteryMessageChannel chan HeadsetBatteryMessage) {
	defer wg.Done()

	var active atomic.Bool

	scan := func() {
		if active.Load() {
			return
		}

		batteryMessageChannel <- HeadsetBatteryMessage{
			Scanning: true,
		}

		paths, err := setupapi.ScanHIDDevices()
		if err != nil {
			return
		}

		for _, headset := range headsets {
			for _, path := range paths {
				supported, err := hid.IsDeviceSupported(path, headset.vendorId, headset.productId, headset.inputBufferLength, headset.outputBufferLength)
				if err != nil {
					continue
				}

				if !supported {
					continue
				}

				wg.Add(1)
				go headset.listen(ctx, wg, batteryMessageChannel, &active, path)
				return
			}
		}
	}

	scan()

	ticker := time.NewTicker(5 * time.Second)
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

func (h *Headset) listen(ctx context.Context, wg *sync.WaitGroup, batteryMessageChannel chan HeadsetBatteryMessage, active *atomic.Bool, path string) {
	defer wg.Done()

	active.Store(true)
	defer active.Store(false)

	pull := func() error {
		batteryLevel, err := hid.GetBatteryLevel(path, h.inputBufferLength, h.outputBufferLength, h.hidCommand, h.batteryBufferIndex)
		if err != nil {
			return err
		}

		batteryMessageChannel <- HeadsetBatteryMessage{
			Name:     h.name,
			Level:    batteryLevel,
			Scanning: false,
		}

		return nil
	}

	pull()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := pull()
			if err != nil {
				return
			}
		}
	}
}
