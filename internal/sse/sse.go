package sse

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Dowdow/steelseries-arctis-battery/internal/display"
)

const (
	GAME_NAME         = "ARCTIS_BATTERY"
	GAME_DISPLAY_NAME = "Arctis Battery Indicator"
	GAME_DEVELOPER    = "Dowdow"
	EVENT_NAME        = "BATTERY_UPDATE"
)

const (
	heartbeatInterval     = 10 * time.Second
	registerRetryInterval = 30 * time.Second
)

type game struct {
	Game string `json:"game"`
}

type gameRegister struct {
	Game            string `json:"game"`
	GameDisplayName string `json:"game_display_name"`
	Developer       string `json:"developer,omitempty"`
}

type eventRegister struct {
	Game          string `json:"game"`
	Event         string `json:"event"`
	MinValue      int    `json:"min_value"`
	MaxValue      int    `json:"max_value"`
	IconId        int    `json:"icon_id,omitempty"`
	ValueOptional bool   `json:"value_optional,omitempty"`
}

type eventBinding struct {
	Game     string    `json:"game"`
	Event    string    `json:"event"`
	Handlers []handler `json:"handlers"`
}

type handler struct {
	DeviceType string       `json:"device-type"`
	Zone       string       `json:"zone,omitempty"`
	Mode       string       `json:"mode"`
	Datas      []screenData `json:"datas"`
}

type screenData struct {
	Lines []lineData `json:"lines"`
}

type lineData struct {
	HasText         bool   `json:"has-text,omitempty"`
	HasProgressBar  bool   `json:"has-progress-bar,omitempty"`
	ContextFrameKey string `json:"context-frame-key,omitempty"`
	Prefix          string `json:"prefix,omitempty"`
}

type eventBatteryUpdate struct {
	Game  string      `json:"game"`
	Event string      `json:"event"`
	Data  batteryData `json:"data"`
}

type batteryData struct {
	Frame frameData `json:"frame"`
	Value int       `json:"value,omitempty"`
}

type frameData struct {
	Text string `json:"text"`
}

// Display sends battery levels to the local GameSense server, which renders
// them on SteelSeries devices with a screen through the SteelSeriesGG app.
type Display struct{}

var _ display.Display = (*Display)(nil)

func New() *Display {
	return &Display{}
}

func registerGame() error {
	return request("game_metadata", gameRegister{
		Game:            GAME_NAME,
		GameDisplayName: GAME_DISPLAY_NAME,
		Developer:       GAME_DEVELOPER,
	})
}

func sendHeartbeat() error {
	return request("game_heartbeat", game{
		Game: GAME_NAME,
	})
}

func registerEvent() error {
	return request("register_game_event", eventRegister{
		Game:          GAME_NAME,
		Event:         EVENT_NAME,
		MinValue:      0,
		MaxValue:      100,
		IconId:        1,
		ValueOptional: false,
	})
}

func bindEvent() error {
	return request("bind_game_event", eventBinding{
		Game:  GAME_NAME,
		Event: EVENT_NAME,
		Handlers: []handler{
			{
				DeviceType: "screened",
				Zone:       "one",
				Mode:       "screen",
				Datas: []screenData{
					{
						Lines: []lineData{
							{
								HasText:         true,
								ContextFrameKey: "text",
							},
							{
								HasProgressBar: true,
							},
						},
					},
				},
			},
		},
	})
}

func sendEvent(text string, percent int) error {
	return request("game_event", eventBatteryUpdate{
		Game:  GAME_NAME,
		Event: EVENT_NAME,
		Data: batteryData{
			Frame: frameData{
				Text: text,
			},
			Value: percent,
		},
	})
}

func registerAll() error {
	if err := registerGame(); err != nil {
		return err
	}
	if err := registerEvent(); err != nil {
		return err
	}
	return bindEvent()
}

// Listen consumes battery messages and forwards them to the GameSense server.
// It always drains the channel: when the server is unreachable (GG not started
// yet) or GameSense is not available on this OS, messages are dropped and
// registration is retried periodically.
func (s *Display) Listen(ctx context.Context, wg *sync.WaitGroup, messages <-chan display.Message) {
	defer wg.Done()

	registered := false
	unsupported := false
	var lastAttempt time.Time

	ensureRegistered := func() bool {
		if registered || unsupported {
			return registered
		}
		if time.Since(lastAttempt) < registerRetryInterval {
			return false
		}
		lastAttempt = time.Now()

		if err := registerAll(); err != nil {
			if errors.Is(err, ErrUnsupportedOS) {
				unsupported = true
				log.Printf("GameSense disabled: %v", err)
			} else {
				log.Printf("error while registering to GameSense: %v", err)
			}
			return false
		}

		registered = true
		return true
	}

	ensureRegistered()

	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if registered {
				if err := sendHeartbeat(); err != nil {
					log.Printf("error while sending heartbeat: %v", err)
					registered = false
				}
			} else {
				ensureRegistered()
			}
		case message := <-messages:
			// Nothing to actively clear: GameSense reverts the screen to its
			// default on its own once heartbeats/events stop coming in.
			if message.Off {
				continue
			}

			if !ensureRegistered() {
				continue
			}

			text, value := "Scanning...", 0
			if !message.Scanning {
				charge := ""
				if message.Charging {
					charge = "+"
				}
				text = fmt.Sprintf("%-11.11s %d%%%s", message.Name, message.Level, charge)
				value = message.Level
			}

			if err := sendEvent(text, value); err != nil {
				log.Printf("error while sending battery event: %v", err)
				registered = false
			}
		}
	}
}
