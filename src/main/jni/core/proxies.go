//go:build android && cgo

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/metacubex/mihomo/adapter/outboundgroup"
	"github.com/metacubex/mihomo/common/utils"
	"github.com/metacubex/mihomo/component/profile/cachefile"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/tunnel"
)

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

func handleChangeProxy(group, name string) error {
	proxy, ok := tunnel.Proxies()[group]
	if !ok {
		return fmt.Errorf("group %q not found", group)
	}
	selector, ok := proxy.Adapter().(outboundgroup.SelectAble)
	if !ok {
		return fmt.Errorf("group %q is not selectable", group)
	}
	if err := selector.Set(name); err != nil {
		return err
	}
	cachefile.Cache().SetSelected(proxy.Name(), name)
	return nil
}

func handleAsyncTestDelay(name, url string, timeoutMs int) int {
	proxy, ok := proxyByName(name)
	if !ok {
		return -1
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	delay, err := proxy.URLTest(ctx, url, utils.IntRanges[uint16](nil))
	if err != nil || ctx.Err() != nil || delay == 0 {
		return -1
	}
	return int(delay)
}

func proxyByName(name string) (C.Proxy, bool) {
	p, ok := allProxies()[name]
	return p, ok
}
