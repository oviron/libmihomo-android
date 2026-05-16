//go:build android && cgo

package main

import (
	"sync"

	"github.com/metacubex/mihomo/common/observable"
	"github.com/metacubex/mihomo/log"
)

// logSubscriber is guarded by logSubMu; logDone signals the fan-out goroutine
// has fully drained, so a fresh subscribe never overlaps the previous one.
var (
	logSubscriber observable.Subscription[log.Event]
	logSubMu      sync.Mutex
	logDone       chan struct{}
)

func handleStartLog() {
	logSubMu.Lock()
	defer logSubMu.Unlock()
	stopLogLocked()

	sub := log.Subscribe()
	if sub == nil {
		return
	}
	logSubscriber = sub
	done := make(chan struct{})
	logDone = done
	go func() {
		defer close(done)
		for logData := range sub {
			if logData.LogLevel < log.Level() {
				continue
			}
			sendMessage(Message{
				Type: LogMessage,
				Data: logData,
			})
		}
	}()
}

func handleStopLog() {
	logSubMu.Lock()
	defer logSubMu.Unlock()
	stopLogLocked()
}

// stopLogLocked unsubscribes and waits for the fan-out goroutine; caller must
// hold logSubMu. UnSubscribe closes the channel so `range sub` terminates.
func stopLogLocked() {
	if logSubscriber == nil {
		return
	}
	log.UnSubscribe(logSubscriber)
	logSubscriber = nil
	if logDone != nil {
		<-logDone
		logDone = nil
	}
}
