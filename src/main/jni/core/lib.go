//go:build android && cgo

package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/metacubex/mihomo/component/dialer"
	"github.com/metacubex/mihomo/component/process"
	"github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/dns"
	"github.com/metacubex/mihomo/log"
	"github.com/oviron/libmihomo-android/platform"
	t "github.com/oviron/libmihomo-android/tun"
	"golang.org/x/sync/semaphore"
)

// Semaphore weight=full = exclusive critical section; weight=1 = shared hot path.
const (
	tunSemCapacity = 4
	tunSemFullLock = tunSemCapacity
)

func main() {}

var (
	eventListener   unsafe.Pointer
	eventListenerMu sync.RWMutex
)

// TunHandler owns one TUN listener and one JNI global ref (callback).
// Concurrency fence: clear() acquires weight=full while hot paths take
// weight=1, so a `closed` check after Acquire is race-free. Always close().
type TunHandler struct {
	listener io.Closer
	callback unsafe.Pointer
	closed   bool

	limit *semaphore.Weighted
}

func (th *TunHandler) start(fd int, device, stack, address, dns string) {
	runLock.Lock()
	defer runLock.Unlock()
	_ = th.limit.Acquire(context.Background(), tunSemFullLock)
	defer th.limit.Release(tunSemFullLock)
	th.initHook()
	listener, err := t.Start(fd, device, stack, address, dns)
	if err != nil {
		log.Errorln("TUN start failed: %v", err)
		th.clear()
		return
	}
	th.listener = listener
}

func (th *TunHandler) close() {
	_ = th.limit.Acquire(context.Background(), tunSemFullLock)
	defer th.limit.Release(tunSemFullLock)
	th.clear()
}

func (th *TunHandler) clear() {
	th.closed = true
	th.removeHook()
	if th.listener != nil {
		_ = th.listener.Close()
		th.listener = nil
	}
	if th.callback != nil {
		releaseObject(th.callback)
		th.callback = nil
	}
}

func (th *TunHandler) handleProtect(fd int) {
	_ = th.limit.Acquire(context.Background(), 1)
	defer th.limit.Release(1)
	if th.closed {
		return
	}
	protect(th.callback, fd)
}

func (th *TunHandler) handleResolveProcess(source, target net.Addr) string {
	_ = th.limit.Acquire(context.Background(), 1)
	defer th.limit.Release(1)
	if th.closed {
		return ""
	}
	var protocol int
	uid := -1
	switch source.Network() {
	case "udp", "udp4", "udp6":
		protocol = syscall.IPPROTO_UDP
	case "tcp", "tcp4", "tcp6":
		protocol = syscall.IPPROTO_TCP
	}
	if version.Load() < 29 {
		uid = platform.QuerySocketUidFromProcFs(source, target)
	}
	return resolveProcess(th.callback, protocol, source.String(), target.String(), uid)
}

func (th *TunHandler) initHook() {
	dialer.DefaultSocketHook = func(network, address string, conn syscall.RawConn) error {
		if platform.ShouldBlockConnection() {
			return errBlocked
		}
		return conn.Control(func(fd uintptr) {
			tunHandler.handleProtect(int(fd))
		})
	}
	process.DefaultPackageNameResolver = func(metadata *constant.Metadata) (string, error) {
		src, dst := metadata.RawSrcAddr, metadata.RawDstAddr
		if src == nil || dst == nil {
			return "", process.ErrInvalidNetwork
		}
		return tunHandler.handleResolveProcess(src, dst), nil
	}
}

func (th *TunHandler) removeHook() {
	dialer.DefaultSocketHook = nil
	process.DefaultPackageNameResolver = nil
}

var (
	tunLock    sync.Mutex
	errBlocked = errors.New("blocked")
	tunHandler *TunHandler
)

// stopTunLocked tears down the active TunHandler; caller must hold tunLock.
func stopTunLocked() {
	if tunHandler != nil {
		tunHandler.close()
	}
}

func handleStopTun() {
	tunLock.Lock()
	defer tunLock.Unlock()
	stopTunLocked()
}

func handleStartTun(callback unsafe.Pointer, fd int, device, stack, address, dns string) {
	tunLock.Lock()
	defer tunLock.Unlock()
	stopTunLocked()
	if fd == 0 {
		// JNI glue always wraps the callback with new_global before invoking
		// us; release here so the global ref does not leak when the caller
		// passes fd=0 (no TUN to start).
		releaseObject(callback)
		return
	}
	tunHandler = &TunHandler{
		callback: callback,
		limit:    semaphore.NewWeighted(tunSemCapacity),
	}
	tunHandler.start(fd, device, stack, address, dns)
}

func handleUpdateDns(value string) {
	go func() {
		log.Infoln("[DNS] updateDns %s", value)
		dns.UpdateSystemDNS(strings.Split(value, ","))
		dns.FlushCacheWithDefaultResolver()
	}()
}

// send forwards the result to the C callback and releases the JNI global ref
// when this is not the message-listener channel. The release is deferred so
// JSON-marshal failure does not leak the global ref.
func (result *ActionResult) send() {
	if result.Method != messageMethod {
		defer releaseObject(result.callback)
	}
	data, err := result.Json()
	if err != nil {
		return
	}
	invokeResult(result.callback, string(data))
}

//export invokeAction
func invokeAction(callback unsafe.Pointer, paramsChar *C.char) {
	params := takeCString(paramsChar)
	var action = &Action{}
	err := json.Unmarshal([]byte(params), action)
	if err != nil {
		invokeResult(callback, err.Error())
		return
	}
	result := ActionResult{
		Id:       action.Id,
		Method:   action.Method,
		callback: callback,
	}
	go handleAction(action, result)
}

// Starts the TUN listener bound to the given file descriptor from Android
// VpnService.Builder. `device` becomes the interface label inside mihomo logs
// and metrics. `stack` is "system" | "gvisor" | "mixed". `address` is a
// comma-separated CIDR list (IPv4/IPv6). `dns` is a comma-separated host list
// hijacked to port 53. Returns asynchronously through callback.
//
//export startTUN
func startTUN(callback unsafe.Pointer, fd C.int, deviceChar, stackChar, addressChar, dnsChar *C.char) {
	handleStartTun(callback, int(fd), takeCString(deviceChar), takeCString(stackChar), takeCString(addressChar), takeCString(dnsChar))
	if !isRunning.Load() {
		handleStartListener()
	} else {
		handleResetConnections()
	}
}

//export quickSetup
func quickSetup(callback unsafe.Pointer, initParamsChar *C.char, setupParamsChar *C.char) {
	go func() {
		defer releaseObject(callback)
		initParamsString := takeCString(initParamsChar)
		setupParamsString := takeCString(setupParamsChar)
		if !handleInitClash(initParamsString) {
			invokeResult(callback, "init failed")
			return
		}
		// updateListeners inside applyConfig gates on isRunning, so set it
		// before handleSetupConfig and roll back on failure to keep state
		// consistent for the next start.
		isRunning.Store(true)
		message := handleSetupConfig([]byte(setupParamsString))
		if message != "" {
			isRunning.Store(false)
		}
		invokeResult(callback, message)
	}()
}

//export setEventListener
func setEventListener(listener unsafe.Pointer) {
	eventListenerMu.Lock()
	old := eventListener
	eventListener = listener
	eventListenerMu.Unlock()
	if old != nil {
		releaseObject(old)
	}
}

// getTraffic returns a C string the caller must free via free_string.
//
//export getTraffic
func getTraffic() *C.char {
	return C.CString(handleGetTraffic())
}

// getTotalTraffic returns a C string the caller must free via free_string.
//
//export getTotalTraffic
func getTotalTraffic() *C.char {
	return C.CString(handleGetTotalTraffic())
}

func sendMessage(message Message) {
	eventListenerMu.RLock()
	defer eventListenerMu.RUnlock()
	if eventListener == nil {
		return
	}
	result := ActionResult{
		Method:   messageMethod,
		callback: eventListener,
		Data:     message,
	}
	result.send()
}

//export stopTun
func stopTun() {
	handleStopTun()
	if isRunning.Load() {
		handleStopListener()
	}
}

//export suspend
func suspend(suspended bool) {
	handleSuspend(suspended)
}

//export forceGC
func forceGC() {
	handleForceGC()
}

//export updateDns
func updateDns(s *C.char) {
	handleUpdateDns(takeCString(s))
}

// Returns an integer identifying the JNI/bridge API surface. Bumped on
// breaking changes. Consumers can check at runtime to detect mismatched
// libclash.so + Kotlin stub pairings.
//
//export bridgeABI
func bridgeABI() C.int {
	return 1
}
