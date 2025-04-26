package sse

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	GAME_NAME         = "ARCTIS_BATTERY"
	GAME_DISPLAY_NAME = "Arctis Battery Indicator"
	GAME_DEVELOPER    = "Dowdow"
	EVENT_NAME        = "BATTERY_UPDATE"
)

type Game struct {
	Game string `json:"game"`
}

type GameRegister struct {
	Game            string `json:"game"`
	GameDisplayName string `json:"game_display_name"`
	Developer       string `json:"developer,omitempty"`
}

type EventRegister struct {
	Game          string `json:"game"`
	Event         string `json:"event"`
	MinValue      int    `json:"min_value"`
	MaxValue      int    `json:"max_value"`
	IconId        int    `json:"icon_id,omitempty"`
	ValueOptional bool   `json:"value_optional,omitempty"`
}

type EventUnregister struct {
	Game  string `json:"game"`
	Event string `json:"event"`
}

type EventBinding struct {
	Game     string    `json:"game"`
	Event    string    `json:"event"`
	Handlers []Handler `json:"handlers"`
}

type Handler struct {
	DeviceType string       `json:"device-type"`
	Zone       string       `json:"zone,omitempty"`
	Mode       string       `json:"mode"`
	Datas      []ScreenData `json:"datas"`
}

type ScreenData struct {
	Lines []LineData `json:"lines"`
}

type LineData struct {
	HasText         bool   `json:"has-text,omitempty"`
	HasProgressBar  bool   `json:"has-progress-bar,omitempty"`
	ContextFrameKey string `json:"context-frame-key,omitempty"`
	Prefix          string `json:"prefix,omitempty"`
}

type EventBatteryUpdate struct {
	Game  string      `json:"game"`
	Event string      `json:"event"`
	Data  BatteryData `json:"data"`
}

type BatteryData struct {
	Frame FrameData `json:"frame"`
	Value int       `json:"value,omitempty"`
}

type FrameData struct {
	Text string `json:"text"`
}

type SSEBatteryMessage struct {
	Text  string
	Value int
}

func RegisterGame() error {
	return request("game_metadata", GameRegister{
		Game:            GAME_NAME,
		GameDisplayName: GAME_DISPLAY_NAME,
		Developer:       GAME_DEVELOPER,
	})
}

func UnregisterGame() error {
	return request("remove_game", Game{
		Game: GAME_NAME,
	})
}

func SendHeartbeat() error {
	return request("game_heartbeat", Game{
		Game: GAME_NAME,
	})
}

func RegisterEvent() error {
	return request("register_game_event", EventRegister{
		Game:          GAME_NAME,
		Event:         EVENT_NAME,
		MinValue:      0,
		MaxValue:      100,
		IconId:        1,
		ValueOptional: false,
	})
}

func UnregisterEvent() error {
	return request("remove_game_event", EventUnregister{
		Game:  GAME_NAME,
		Event: EVENT_NAME,
	})
}

func BindEvent() error {
	return request("bind_game_event", EventBinding{
		Game:  GAME_NAME,
		Event: EVENT_NAME,
		Handlers: []Handler{
			{
				DeviceType: "screened",
				Zone:       "one",
				Mode:       "screen",
				Datas: []ScreenData{
					{
						Lines: []LineData{
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

func SendEvent(text string, percent int) error {
	return request("game_event", EventBatteryUpdate{
		Game:  GAME_NAME,
		Event: EVENT_NAME,
		Data: BatteryData{
			Frame: FrameData{
				Text: text,
			},
			Value: percent,
		},
	})
}

func Listen(ctx context.Context, wg *sync.WaitGroup, batteryMessageChannel chan SSEBatteryMessage) {
	defer wg.Done()

	err := RegisterGame()
	if err != nil {
		fmt.Printf("error while registering game: %v", err)
		return
	}

	err = RegisterEvent()
	if err != nil {
		fmt.Printf("error while registering event: %v", err)
		return
	}

	err = BindEvent()
	if err != nil {
		fmt.Printf("error while binding event: %v", err)
		return
	}

	// Keepalive
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := SendHeartbeat()
				if err != nil {
					fmt.Printf("error while sending hearbeat: %v", err)
					return
				}
			}
		}
	}()

	// Listen updates from the channel and send battery events
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case message := <-batteryMessageChannel:
				err := SendEvent(message.Text, message.Value)
				if err != nil {
					fmt.Printf("error while sending battery event: %v", err)
					return
				}
			}
		}
	}()
}
