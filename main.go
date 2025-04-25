package main

import (
	"context"
	"sync"

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
	systray.Run(onReady, onExit)
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
				mToggleStartStop.SetTitle("Start")
				mToggleStartStop.SetTooltip("Start the process")
			} else {
				mToggleStartStop.SetTitle("Stop")
				mToggleStartStop.SetTooltip("Stop the process")
			}
			started = !started
		}
	}()

	ctx, cancel = context.WithCancel(context.Background())

	wg.Add(1)
	go sse.Listen(ctx, &wg)

	wg.Add(1)
	go headset.Listen(ctx, &wg)
}

func onExit() {
	if cancel != nil {
		cancel()
	}
	wg.Wait()
}
