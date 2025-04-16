package main

import (
	"fmt"
	"time"

	"fyne.io/systray"
	"github.com/Dowdow/steelseries-arctis-battery/icon"
)

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

	// On essaye de voir si un casque compatible est disponible

	err := registerGame()
	if err != nil {
		fmt.Printf("error while registering game: %v", err)
		return
	}

	err = registerEvent()
	if err != nil {
		fmt.Printf("error while registering event: %v", err)
		return
	}

	err = bindEvent()
	if err != nil {
		fmt.Printf("error while binding event: %v", err)
		return
	}

	for {
		battery, _ := getBatteryLevel()
		sendEvent(battery)
		time.Sleep(10 * time.Second)
	}

	// unregisterEvent()
}

func onExit() {

}

func main() {
	systray.Run(onReady, onExit)
}
