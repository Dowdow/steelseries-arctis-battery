package headset

import (
	"context"
	"sync"
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

func Listen(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			paths, err := setupapi.ScanHIDDevices()
			if err != nil {
				// Envoyer le message quelquepart ?
				return
			}

			// Optimiser la recherche de headset. Pour ne pas ouvrir et fermer le périphérique pour chaque casques mais ouvrir
			// une seule fois le périphérique et tester tous les casques possible et s'arrêter au premier trouvé
			for _, headset := range headsets {
				for _, path := range paths {
					supported, err := hid.IsDeviceSupported(path, headset.vendorId, headset.productId, headset.inputBufferLength, headset.outputBufferLength)
					if err != nil {
						// Envoyer le message quelquepart ?
						continue
					}

					if !supported {
						continue
					}

					go headset.listen(ctx, wg, path)
				}
			}
		}
	}
}

func (h *Headset) listen(ctx context.Context, wg *sync.WaitGroup, path string) {
	defer wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			batteryLevel, err := hid.GetBatteryLevel(path, h.inputBufferLength, h.outputBufferLength, h.hidCommand, h.batteryBufferIndex)
			if err != nil {
				return
			}

			// Envoyer le niveau de batterie dans un channel avec un objet spécial qui contient le nom et le % de batterie
		}
	}
}
