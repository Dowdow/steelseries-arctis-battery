package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const (
	GAME_NAME         = "ARCTIS_BATTERY"
	GAME_DISPLAY_NAME = "Arctis Battery Indicator"
	GAME_DEVELOPER    = "Dowdow"
	EVENT_NAME        = "BATTERY_UPDATE"
)

type GameRegister struct {
	Game            string `json:"game"`
	GameDisplayName string `json:"game_display_name"`
	Developer       string `json:"developer,omitempty"`
}

type GameUnregister struct {
	Game string `json:"game"`
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
	Percent string `json:"percent"`
}

// Server address caching
var (
	serverAddressCache    string
	serverAddressCacheExp time.Time
	cacheMutex            sync.RWMutex
	cacheValidDuration    = 1 * time.Hour
)

func getServerAddress() (string, error) {
	cacheMutex.RLock()
	if serverAddressCache != "" && time.Now().Before(serverAddressCacheExp) {
		addr := serverAddressCache
		cacheMutex.RUnlock()
		return addr, nil
	}
	cacheMutex.RUnlock()

	var corePropsPath string

	switch runtime.GOOS {
	case "windows":
		corePropsPath = filepath.Join(os.Getenv("PROGRAMDATA"), "SteelSeries", "SteelSeries Engine 3", "coreProps.json")
	case "darwin":
		corePropsPath = "/Library/Application Support/SteelSeries Engine 3/coreProps.json"
	default:
		return "", fmt.Errorf("OS is not supported: %s", runtime.GOOS)
	}

	data, err := os.ReadFile(corePropsPath)
	if err != nil {
		return "", fmt.Errorf("error while reading coreProps.json: %v", err)
	}

	var coreProps struct {
		Address string `json:"address"`
	}

	if err := json.Unmarshal(data, &coreProps); err != nil {
		return "", fmt.Errorf("error while unmarshaling coreProps.json: %v", err)
	}

	cacheMutex.Lock()
	serverAddressCache = coreProps.Address
	serverAddressCacheExp = time.Now().Add(cacheValidDuration)
	cacheMutex.Unlock()

	return coreProps.Address, nil
}

func request(endpoint string, payload interface{}) error {
	serverAddress, err := getServerAddress()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s/%s", serverAddress, endpoint)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error while marshalling JSON: %v", err)
	}

	response, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error while sending request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("error response from server: %d %s", response.StatusCode, string(body))
	}

	return nil
}

func registerGame() error {
	return request("game_metadata", GameRegister{
		Game:            GAME_NAME,
		GameDisplayName: GAME_DISPLAY_NAME,
		Developer:       GAME_DEVELOPER,
	})
}

func unregisterGame() error {
	return request("remove_game", GameUnregister{
		Game: GAME_NAME,
	})
}

func registerEvent() error {
	return request("register_game_event", EventRegister{
		Game:          GAME_NAME,
		Event:         EVENT_NAME,
		MinValue:      0,
		MaxValue:      100,
		IconId:        1,
		ValueOptional: false,
	})
}

func unregisterEvent() error {
	return request("remove_game_event", EventUnregister{
		Game:  GAME_NAME,
		Event: EVENT_NAME,
	})
}

func bindEvent() error {
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
								Prefix:          "Arctis 7 - ",
								ContextFrameKey: "percent",
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

func sendEvent(percent int) error {
	return request("game_event", EventBatteryUpdate{
		Game:  GAME_NAME,
		Event: EVENT_NAME,
		Data: BatteryData{
			Frame: FrameData{
				Percent: fmt.Sprintf("%d%%", percent),
			},
			Value: percent,
		},
	})
}
