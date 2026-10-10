package sse

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// ErrUnsupportedOS is returned when the SteelSeries GG app (and therefore the
// GameSense server) does not exist on the current platform.
var ErrUnsupportedOS = errors.New("GameSense is not available on this OS")

var httpClient = &http.Client{Timeout: 5 * time.Second}

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
		return "", fmt.Errorf("%w: %s", ErrUnsupportedOS, runtime.GOOS)
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

func request(endpoint string, payload any) error {
	serverAddress, err := getServerAddress()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s/%s", serverAddress, endpoint)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error while marshalling JSON: %v", err)
	}

	response, err := httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
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
