package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"fyne.io/systray"
	"github.com/Dowdow/steelseries-arctis-battery/headset"
	"github.com/Dowdow/steelseries-arctis-battery/icon"
	"github.com/Dowdow/steelseries-arctis-battery/sse"
)

var (
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
)

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			<-sigChan
			systray.Quit()
		}
	}()

	systray.Run(onReady, func() { stop() })
}

func onReady() {
	systray.SetTitle("Steelseries Arctis Battery")
	systray.SetTooltip("Steelseries Arctis Battery")
	systray.SetIcon(icon.Icon)

	mQuit := systray.AddMenuItem("Quit", "Quit")
	go func() {
		for range mQuit.ClickedCh {
			systray.Quit()
		}
	}()

	systray.AddSeparator()

	started := true
	mToggleStartStop := systray.AddMenuItem("Stop", "Stop the process")
	go func() {
		for range mToggleStartStop.ClickedCh {
			if started {
				stop()
				mToggleStartStop.SetTitle("Start")
				mToggleStartStop.SetTooltip("Start the process")
				systray.SetIcon(icon.Icon)
			} else {
				start()
				mToggleStartStop.SetTitle("Stop")
				mToggleStartStop.SetTooltip("Stop the process")
			}
			started = !started
		}
	}()

	start()
}

func start() {
	ctx, cancel = context.WithCancel(context.Background())

	sseBatteryMessageChannel := make(chan sse.SSEBatteryMessage)
	headsetBatteryMessageChannel := make(chan headset.HeadsetBatteryMessage)

	wg.Add(1)
	go sse.Listen(ctx, &wg, sseBatteryMessageChannel)

	wg.Add(1)
	go headset.Listen(ctx, &wg, headsetBatteryMessageChannel)

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case message := <-headsetBatteryMessageChannel:
				if message.Scanning {
					systray.SetIcon(icon.Icon)
					sseBatteryMessageChannel <- sse.SSEBatteryMessage{
						Text:  "Scanning...",
						Value: 0,
					}
				} else {
					if message.Level >= 50 {
						systray.SetIcon(icon.IconGreen)
					} else if message.Level >= 20 {
						systray.SetIcon(icon.IconOrange)
					} else {
						systray.SetIcon(icon.IconRed)
					}

					sseBatteryMessageChannel <- sse.SSEBatteryMessage{
						Text:  fmt.Sprintf("%s - %d%%", message.Name, message.Level),
						Value: message.Level,
					}
				}
			}
		}
	}()
}

func stop() {
	if cancel != nil {
		cancel()
	}
	wg.Wait()
}
