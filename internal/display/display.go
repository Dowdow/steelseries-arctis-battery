// Package display defines the battery status message and the interface that
// every output backend (GameSense, direct OLED, ...) implements.
package display

import (
	"context"
	"sync"
)

// Message is either a battery status update produced by the headset scanner,
// or an Off signal telling a backend to clear whatever it is displaying
// because it was just disabled (the headset scanner keeps running either way).
type Message struct {
	Name     string
	Level    int
	Scanning bool
	Charging bool
	Off      bool
}

// Display renders battery status messages somewhere.
type Display interface {
	// Listen consumes messages until ctx is cancelled. Implementations must
	// keep draining the channel even when their backend is unavailable, so
	// producers can never block on a dead display.
	Listen(ctx context.Context, wg *sync.WaitGroup, messages <-chan Message)
}

// Send delivers a message unless the context is cancelled, so a shutdown can
// never deadlock on an abandoned channel.
func Send(ctx context.Context, ch chan<- Message, message Message) bool {
	select {
	case <-ctx.Done():
		return false
	case ch <- message:
		return true
	}
}
