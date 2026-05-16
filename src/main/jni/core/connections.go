//go:build android && cgo

package main

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/metacubex/mihomo/tunnel/statistic"
)

const connectionsTickInterval = time.Second

var (
	connectionsMu     sync.Mutex
	connectionsCancel context.CancelFunc
	connectionsDone   chan struct{}
)

func handleGetConnections() string {
	data, _ := json.Marshal(statistic.DefaultManager.Snapshot())
	return string(data)
}

func handleSubscribeConnections() {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	// Wait for the previous ticker goroutine to fully exit before starting a
	// new one; otherwise two tickers can both emit on the same tick during a
	// re-subscribe race.
	if connectionsCancel != nil {
		connectionsCancel()
		<-connectionsDone
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	connectionsCancel = cancel
	connectionsDone = done
	go func() {
		defer close(done)
		ticker := time.NewTicker(connectionsTickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, _ := json.Marshal(statistic.DefaultManager.Snapshot())
				sendMessage(Message{
					Type: ConnectionsMessage,
					Data: json.RawMessage(data),
				})
			}
		}
	}()
}

func handleUnsubscribeConnections() {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()
	if connectionsCancel != nil {
		connectionsCancel()
		<-connectionsDone
		connectionsCancel = nil
		connectionsDone = nil
	}
}

func handleCloseConnection(id string) bool {
	tracker := statistic.DefaultManager.Get(id)
	if tracker == nil {
		return false
	}
	_ = tracker.Close()
	return true
}

func handleCloseAllConnections() bool {
	statistic.DefaultManager.Range(func(t statistic.Tracker) bool {
		_ = t.Close()
		return true
	})
	return true
}
