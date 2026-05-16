//go:build android && cgo

package main

/*
#cgo LDFLAGS: -llog
#include <android/log.h>
#include <stdlib.h>
*/
import "C"

import (
	"bufio"
	"fmt"
	"os"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/metacubex/mihomo/log"
	"golang.org/x/sys/unix"
)

const (
	tagLibclash       = "libclash"
	tagLibclashStderr = "libclash-stderr"
	tagDart           = "dart"
	tagKotlin         = "kotlin"
)

// Per-sink levels are decoupled from mihomo's global log.Level() (which only
// gates logrus stdout).
var (
	logcatLevel atomic.Int32
	fileLevel   atomic.Int32
	fileEnabled atomic.Bool
)

type dispatchEvent struct {
	when     time.Time
	level    log.LogLevel
	tag      string
	payload  string
	fromHost bool // skip logcat write — Kotlin/Dart already wrote
}

const dispatchCapacity = 1024

var (
	logDispatchCh = make(chan dispatchEvent, dispatchCapacity)
	dropCount     atomic.Uint64
)

func init() {
	logcatLevel.Store(int32(log.DEBUG))
	fileLevel.Store(int32(log.WARNING))

	probeLogcat()
	go subscribeMihomo()
	captureStdFd(1)
	captureStdFd(2)
	go dispatchLoop()
}

func probeLogcat() {
	tag := C.CString(tagLibclash)
	msg := C.CString("log bridge init")
	C.__android_log_write(C.ANDROID_LOG_INFO, tag, msg)
	C.free(unsafe.Pointer(msg))
	C.free(unsafe.Pointer(tag))
}

func subscribeMihomo() {
	sub := log.Subscribe()
	if sub == nil {
		return
	}
	for event := range sub {
		dispatch(dispatchEvent{
			when:    time.Now(),
			level:   event.LogLevel,
			tag:     tagLibclash,
			payload: event.Payload,
		})
	}
}

// unix.Dup3 — syscall.Dup2 is missing on arm64. On scanner error fd is rebound
// to /dev/null so writes never block on a pipe with no reader.
func captureStdFd(targetFd int) {
	r, w, err := os.Pipe()
	if err != nil {
		return
	}
	if err := unix.Dup3(int(w.Fd()), targetFd, 0); err != nil {
		_ = r.Close()
		_ = w.Close()
		return
	}
	_ = w.Close()
	go func() {
		defer func() { _ = r.Close() }()
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			dispatch(dispatchEvent{
				when:    time.Now(),
				level:   log.INFO,
				tag:     tagLibclashStderr,
				payload: scanner.Text(),
			})
		}
		if err := scanner.Err(); err != nil {
			emergencyLogcat(log.WARNING, fmt.Sprintf("stdout/stderr scanner failed (%v); fd %d → /dev/null", err, targetFd))
		}
		if devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0); err == nil {
			_ = unix.Dup3(int(devNull.Fd()), targetFd, 0)
			_ = devNull.Close()
		}
	}()
}

// Non-blocking: drop oldest on overflow so mihomo's event loop never stalls.
func dispatch(e dispatchEvent) {
	select {
	case logDispatchCh <- e:
	default:
		select {
		case <-logDispatchCh:
			dropCount.Add(1)
		default:
		}
		select {
		case logDispatchCh <- e:
		default:
			dropCount.Add(1)
		}
	}
}

// Serial drain — also why writeFileSink takes no lock against itself.
// Recovers and respawns on panic so a sink bug can't silence the pipeline.
func dispatchLoop() {
	defer func() {
		if r := recover(); r != nil {
			emergencyLogcat(log.ERROR, fmt.Sprintf("log dispatcher panic: %v — restarting", r))
			go dispatchLoop()
		}
	}()
	flush := time.NewTicker(500 * time.Millisecond)
	defer flush.Stop()
	for {
		select {
		case e := <-logDispatchCh:
			route(e)
		case <-flush.C:
			flushFileSink()
			if d := dropCount.Swap(0); d > 0 {
				route(dispatchEvent{
					when:    time.Now(),
					level:   log.WARNING,
					tag:     tagLibclash,
					payload: fmt.Sprintf("log bus overflow: dropped %d events under burst", d),
				})
			}
		}
	}
}

// Bypasses dispatcher — used from panic recovery where dispatcher is broken.
func emergencyLogcat(level log.LogLevel, msg string) {
	tag := C.CString(tagLibclash)
	cmsg := C.CString(msg)
	C.__android_log_write(androidLogPriority(level), tag, cmsg)
	C.free(unsafe.Pointer(cmsg))
	C.free(unsafe.Pointer(tag))
}

func route(e dispatchEvent) {
	if !e.fromHost && e.level >= log.LogLevel(logcatLevel.Load()) {
		writeLogcat(e)
	}
	if fileEnabled.Load() && e.level >= log.LogLevel(fileLevel.Load()) {
		writeFileSink(e)
	}
}

func writeLogcat(e dispatchEvent) {
	tag := C.CString(e.tag)
	msg := C.CString(e.payload)
	C.__android_log_write(androidLogPriority(e.level), tag, msg)
	C.free(unsafe.Pointer(msg))
	C.free(unsafe.Pointer(tag))
}

func androidLogPriority(l log.LogLevel) C.int {
	switch l {
	case log.DEBUG:
		return C.ANDROID_LOG_DEBUG
	case log.INFO:
		return C.ANDROID_LOG_INFO
	case log.WARNING:
		return C.ANDROID_LOG_WARN
	case log.ERROR:
		return C.ANDROID_LOG_ERROR
	case log.SILENT:
		return C.ANDROID_LOG_SILENT
	default:
		return C.ANDROID_LOG_INFO
	}
}

func hostLog(level log.LogLevel, tag, payload string) {
	if tag == "" {
		tag = "host"
	}
	dispatch(dispatchEvent{
		when:     time.Now(),
		level:    level,
		tag:      tag,
		payload:  payload,
		fromHost: true,
	})
}

func setLogcatLevelImpl(level log.LogLevel) {
	logcatLevel.Store(int32(level))
}

func setFileLevelImpl(level log.LogLevel) {
	fileLevel.Store(int32(level))
}

// Gate, plus best-effort reopen if the sink was closed by a prior write error.
func setFileEnabledImpl(enabled bool) {
	if !enabled {
		fileEnabled.Store(false)
		flushFileSink()
		return
	}
	fileSinkMu.Lock()
	if fileSinkWr == nil && fileSinkPath != "" {
		if err := openFileSinkLocked(fileSinkPath); err != nil {
			emergencyLogcat(log.ERROR, fmt.Sprintf("reopen file sink failed: %v", err))
		}
	}
	fileSinkMu.Unlock()
	fileEnabled.Store(true)
}
