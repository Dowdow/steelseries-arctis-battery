package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"

	"fyne.io/systray"
	"github.com/Dowdow/steelseries-arctis-battery/internal/display"
	"github.com/Dowdow/steelseries-arctis-battery/internal/headset"
	"github.com/Dowdow/steelseries-arctis-battery/internal/icon"
	"github.com/Dowdow/steelseries-arctis-battery/internal/oled"
	"github.com/Dowdow/steelseries-arctis-battery/internal/sse"
)

var (
	// pipelineMu guards ctx/cancel: start()/stop() write them from the
	// Start/Stop menu handler and the systray exit callback, toggle() reads
	// them from the checkbox handlers — all different goroutines.
	pipelineMu sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup

	headsetChannel = make(chan display.Message)

	// Output backends, each behind its own channel and enabled flag. The
	// channels live for the whole process so a toggle can reach a backend's
	// goroutine (e.g. to clear its screen) whether or not it's currently the
	// one being (re)started by the Start/Stop menu item.
	sseSink  = &backendSink{display: sse.New(), channel: make(chan display.Message)}
	oledSink = &backendSink{display: oled.New(), channel: make(chan display.Message)}
	sinks    = []*backendSink{sseSink, oledSink}
)

type backendSink struct {
	display display.Display
	enabled atomic.Bool
	channel chan display.Message
}

func main() {
	sseSink.enabled.Store(runtime.GOOS == "windows" || runtime.GOOS == "darwin")
	oledSink.enabled.Store(runtime.GOOS == "linux")

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
	systray.SetTitle("SteelSeries Arctis Battery")
	systray.SetTooltip("SteelSeries Arctis Battery")
	systray.SetIcon(icon.Default)

	mQuit := systray.AddMenuItem("Quit", "Quit")
	go func() {
		for range mQuit.ClickedCh {
			systray.Quit()
		}
	}()

	systray.AddSeparator()

	started := true
	mToggleStartStop := systray.AddMenuItem("Stop listening", "Stop the listening process")
	go func() {
		for range mToggleStartStop.ClickedCh {
			if started {
				stop()
				mToggleStartStop.SetTitle("Start listening")
				mToggleStartStop.SetTooltip("Start the listening process")
				systray.SetIcon(icon.Default)
			} else {
				start()
				mToggleStartStop.SetTitle("Stop listening")
				mToggleStartStop.SetTooltip("Stop the listening process")
			}
			started = !started
		}
	}()

	systray.AddSeparator()

	mSSE := systray.AddMenuItemCheckbox("Screen via GameSense", "Send battery level to SteelSeriesGG", sseSink.enabled.Load())
	go func() {
		for range mSSE.ClickedCh {
			toggle(sseSink, mSSE)
		}
	}()

	mOLED := systray.AddMenuItemCheckbox("Screen via direct HID", "Draw directly on the keyboard OLED screen", oledSink.enabled.Load())
	go func() {
		for range mOLED.ClickedCh {
			toggle(oledSink, mOLED)
		}
	}()

	start()
}

func toggle(sink *backendSink, item *systray.MenuItem) {
	if sink.enabled.Load() {
		sink.enabled.Store(false)
		item.Uncheck()

		// Best-effort: if the pipeline isn't running, there's nothing to
		// clear, the backend already did so on its own shutdown
		pipelineMu.Lock()
		runningCtx := ctx
		pipelineMu.Unlock()

		if runningCtx != nil {
			display.Send(runningCtx, sink.channel, display.Message{Off: true})
		}
	} else {
		sink.enabled.Store(true)
		item.Check()
	}
}

func start() {
	pipelineMu.Lock()
	ctx, cancel = context.WithCancel(context.Background())
	runningCtx := ctx
	pipelineMu.Unlock()

	for _, sink := range sinks {
		wg.Add(1)
		go sink.display.Listen(runningCtx, &wg, sink.channel)
	}

	wg.Add(1)
	go headset.Listen(runningCtx, &wg, headsetChannel)

	wg.Go(func() {
		for {
			select {
			case <-runningCtx.Done():
				return
			case message := <-headsetChannel:
				updateTrayIcon(message)
				updateTrayTooltip(message)

				for _, sink := range sinks {
					if !sink.enabled.Load() {
						continue
					}
					if !display.Send(runningCtx, sink.channel, message) {
						return
					}
				}
			}
		}
	})
}

func updateTrayIcon(message display.Message) {
	if message.Scanning {
		systray.SetIcon(icon.Default)
		return
	}
	systray.SetIcon(icon.Render(message.Level, message.Charging))
}

func updateTrayTooltip(message display.Message) {
	if message.Scanning {
		systray.SetTooltip("SteelSeries Arctis Battery — scanning...")
		return
	}

	tooltip := fmt.Sprintf("%s — %d%%", message.Name, message.Level)
	if message.Charging {
		tooltip += " (charging)"
	}
	systray.SetTooltip(tooltip)
}

func stop() {
	pipelineMu.Lock()
	stopFunc := cancel
	pipelineMu.Unlock()

	if stopFunc != nil {
		stopFunc()
	}
	wg.Wait()
}
