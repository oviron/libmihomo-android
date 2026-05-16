//go:build android && cgo

package main

import (
	b "bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/adapter/inbound"
	"github.com/metacubex/mihomo/adapter/outboundgroup"
	"github.com/metacubex/mihomo/component/dialer"
	"github.com/metacubex/mihomo/component/resolver"
	"github.com/metacubex/mihomo/config"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/metacubex/mihomo/listener"
	"github.com/metacubex/mihomo/log"
	"github.com/metacubex/mihomo/tunnel"
)

// Mihomo profile filename inside the consumer-supplied home directory.
const configFileName = "config.yaml"

// Package-level state shared across goroutines: cgo-export entrypoints and
// the action goroutine all read/write these without holding runLock, so they
// must be atomic.
var (
	currentConfig     *config.Config
	version           atomic.Int32
	isRunning         atomic.Bool
	isInit            atomic.Bool
	runLock           sync.Mutex
	proxyGroupOrder   []string
	proxyGroupOrderMu sync.RWMutex
)

func updateListeners() {
	if !isRunning.Load() {
		return
	}
	if currentConfig == nil {
		return
	}
	listeners := currentConfig.Listeners
	general := currentConfig.General
	listener.PatchInboundListeners(listeners, tunnel.Tunnel, true)

	allowLan := general.AllowLan
	listener.SetAllowLan(allowLan)
	inbound.SetSkipAuthPrefixes(general.SkipAuthPrefixes)
	inbound.SetAllowedIPs(general.LanAllowedIPs)
	inbound.SetDisAllowedIPs(general.LanDisAllowedIPs)

	bindAddress := general.BindAddress
	listener.SetBindAddress(bindAddress)
	listener.ReCreateHTTP(general.Port, tunnel.Tunnel)
	listener.ReCreateSocks(general.SocksPort, tunnel.Tunnel)
	listener.ReCreateRedir(general.RedirPort, tunnel.Tunnel)
	listener.ReCreateTProxy(general.TProxyPort, tunnel.Tunnel)
	listener.ReCreateMixed(general.MixedPort, tunnel.Tunnel)
	listener.ReCreateShadowSocks(general.ShadowSocksConfig, tunnel.Tunnel)
	listener.ReCreateVmess(general.VmessConfig, tunnel.Tunnel)
	listener.ReCreateTuic(general.TuicServer, tunnel.Tunnel)
}

func patchSelectGroup(mapping map[string]string) {
	for name, proxy := range allProxies() {
		outbound, ok := proxy.(*adapter.Proxy)
		if !ok {
			continue
		}
		selector, ok := outbound.ProxyAdapter.(outboundgroup.SelectAble)
		if !ok {
			continue
		}
		if selected, exist := mapping[name]; exist {
			selector.ForceSet(selected)
		}
	}
}

func defaultSetupParams() *SetupParams {
	return &SetupParams{
		SelectedMap: map[string]string{},
	}
}

func readFile(path string) ([]byte, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path supplied by consumer, runs in app sandbox
	if err != nil {
		return nil, err
	}

	return data, err
}

func updateConfig(params *UpdateParams) {
	runLock.Lock()
	defer runLock.Unlock()
	general := currentConfig.General
	if params.MixedPort != nil {
		general.MixedPort = *params.MixedPort
	}
	if params.Sniffing != nil {
		general.Sniffing = *params.Sniffing
		tunnel.SetSniffing(general.Sniffing)
	}
	if params.FindProcessMode != nil {
		general.FindProcessMode = *params.FindProcessMode
		tunnel.SetFindProcessMode(general.FindProcessMode)
	}
	if params.TCPConcurrent != nil {
		general.TCPConcurrent = *params.TCPConcurrent
		dialer.SetTcpConcurrent(general.TCPConcurrent)
	}
	if params.Interface != nil {
		general.Interface = *params.Interface
		dialer.DefaultInterface.Store(general.Interface)
	}
	if params.UnifiedDelay != nil {
		general.UnifiedDelay = *params.UnifiedDelay
		adapter.UnifiedDelay.Store(general.UnifiedDelay)
	}
	if params.Mode != nil {
		general.Mode = *params.Mode
		tunnel.SetMode(general.Mode)
	}
	if params.LogLevel != nil {
		general.LogLevel = *params.LogLevel
		log.SetLevel(general.LogLevel)
	}
	if params.IPv6 != nil {
		general.IPv6 = *params.IPv6
		resolver.DisableIPv6 = !general.IPv6
	}
	if params.AllowLan != nil {
		general.AllowLan = *params.AllowLan
		listener.SetAllowLan(general.AllowLan)
	}
	if params.Tun != nil {
		general.Tun.Enable = params.Tun.Enable
		if params.Tun.AutoRoute != nil {
			general.Tun.AutoRoute = *params.Tun.AutoRoute
		}
		if params.Tun.Device != nil {
			general.Tun.Device = *params.Tun.Device
		}
		if params.Tun.RouteAddress != nil {
			general.Tun.RouteAddress = *params.Tun.RouteAddress
		}
		if params.Tun.DNSHijack != nil {
			general.Tun.DNSHijack = *params.Tun.DNSHijack
		}
		if params.Tun.Stack != nil {
			general.Tun.Stack = *params.Tun.Stack
		}
	}

	updateListeners()
}

// applyConfig parses config.yaml once, applies it to mihomo, and falls back
// to the embedded default on any failure so the VPN stays up; the first
// error from read > unmarshal > parse is returned for the caller to surface.
func applyConfig(params *SetupParams) error {
	runLock.Lock()
	defer runLock.Unlock()
	configPath := filepath.Join(C.Path.HomeDir(), configFileName)

	var (
		raw  *config.RawConfig
		uerr error
		perr error
	)
	buf, rerr := readFile(configPath)
	if rerr == nil {
		raw, uerr = config.UnmarshalRawConfig(buf)
	}
	if raw != nil {
		currentConfig, perr = config.ParseRawConfig(raw)
	}
	if currentConfig == nil {
		currentConfig, _ = config.ParseRawConfig(config.DefaultRawConfig())
	}

	captureProxyGroupOrder(raw)
	// executor.ApplyConfig(force=true) runs mihomo's own updateListeners,
	// so our local updateListeners is redundant here (it stays useful for
	// the partial-update path in updateConfig).
	executor.ApplyConfig(currentConfig, true)
	patchSelectGroup(params.SelectedMap)

	switch {
	case rerr != nil:
		return rerr
	case uerr != nil:
		return uerr
	default:
		return perr
	}
}

// captureProxyGroupOrder snapshots YAML-declared group order because mihomo's
// parsed Config drops it. An empty snapshot is published when raw is nil.
func captureProxyGroupOrder(raw *config.RawConfig) {
	order := []string{}
	if raw != nil {
		for _, g := range raw.ProxyGroup {
			if name, ok := g["name"].(string); ok && name != "" {
				order = append(order, name)
			}
		}
	}
	proxyGroupOrderMu.Lock()
	proxyGroupOrder = order
	proxyGroupOrderMu.Unlock()
}

func queryProxyGroupOrder() string {
	proxyGroupOrderMu.RLock()
	defer proxyGroupOrderMu.RUnlock()
	data, _ := json.Marshal(proxyGroupOrder)
	return string(data)
}

// UnmarshalJson decodes with UseNumber so large integers from Dart (timeout
// in ms, ports, etc.) keep precision instead of being converted to float64.
func UnmarshalJson(data []byte, v any) error {
	decoder := json.NewDecoder(b.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(v)
}
