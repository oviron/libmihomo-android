//go:build android && cgo

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/metacubex/mihomo/adapter/outboundgroup"
	"github.com/metacubex/mihomo/common/utils"
	"github.com/metacubex/mihomo/component/profile/cachefile"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/tunnel"
	"github.com/oviron/libmihomo-android/selection"
	"golang.org/x/sync/semaphore"
)

// Caps in-flight URLTest goroutines, bounding leak from non-ctx-aware adapters.
const urlTestSemCapacity = 256

var urlTestSem = semaphore.NewWeighted(urlTestSemCapacity)

// A test waits this long for a slot rather than being refused one: a group
// test starts every node at once, and a big subscription exceeds the cap.
const urlTestSlotWait = 30 * time.Second

// allProxies returns a name->proxy snapshot that merges tunnel.Proxies() with
// proxies exposed by all providers. Provider-supplied proxies override tunnel
// entries on name collision (mirroring mihomo's REST shape).
func allProxies() map[string]C.Proxy {
	out := make(map[string]C.Proxy)
	for name, proxy := range tunnel.Proxies() {
		out[name] = proxy
	}
	for _, prov := range tunnel.Providers() {
		for _, proxy := range prov.Proxies() {
			out[proxy.Name()] = proxy
		}
	}
	return out
}

// handleGetProxies returns the merged proxy snapshot as JSON, wrapped under
// `{"proxies": {...}}` to match the mihomo REST shape Dart already parses.
func handleGetProxies() string {
	data, err := json.Marshal(map[string]any{"proxies": allProxies()})
	if err != nil {
		return ""
	}
	return string(data)
}

// pickMu orders a user's pick against a group test re-seating the same group.
var pickMu sync.Mutex

func handleChangeProxy(group, name string) error {
	pickMu.Lock()
	defer pickMu.Unlock()
	proxy, ok := tunnel.Proxies()[group]
	if !ok {
		return fmt.Errorf("group %q not found", group)
	}
	selector, ok := proxy.Adapter().(outboundgroup.SelectAble)
	if !ok {
		return fmt.Errorf("group %q is not selectable", group)
	}
	if err := selection.Apply(selector, name); err != nil {
		return err
	}
	cachefile.Cache().SetSelected(proxy.Name(), name)
	return nil
}

// Watchdog: race URLTest against ctx.Done() because some adapters don't
// propagate SetDeadline to the underlying socket, so I/O can stall past ctx.
// Semaphore caps leaked goroutines from such stalls.
func handleAsyncTestDelay(name, url string, timeoutMs int) int {
	proxy, ok := proxyByName(name)
	if !ok {
		return -1
	}
	return testProxy(proxy, url, nil, timeoutMs)
}

// testProxy passes expected through to URLTest, which still returns the delay
// of a reply outside it and only marks the node dead for url.
func testProxy(proxy C.Proxy, url string, expected utils.IntRanges[uint16], timeoutMs int) int {
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	slotCtx, cancelSlot := context.WithTimeout(context.Background(), urlTestSlotWait)
	defer cancelSlot()
	if urlTestSem.Acquire(slotCtx, 1) != nil {
		return -1
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	type result struct {
		delay uint16
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		defer urlTestSem.Release(1)
		d, e := proxy.URLTest(ctx, url, expected)
		ch <- result{delay: d, err: e}
	}()

	select {
	case r := <-ch:
		if r.err != nil || ctx.Err() != nil || r.delay == 0 {
			return -1
		}
		return int(r.delay)
	case <-ctx.Done():
		return -1
	}
}

func proxyByName(name string) (C.Proxy, bool) {
	p, ok := allProxies()[name]
	return p, ok
}
