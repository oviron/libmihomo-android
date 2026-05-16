//go:build android && cgo

package main

import (
	"net"

	"github.com/metacubex/mihomo/adapter/provider"
	"github.com/metacubex/mihomo/component/mmdb"
	"github.com/metacubex/mihomo/component/updater"
	cp "github.com/metacubex/mihomo/constant/provider"
	"github.com/metacubex/mihomo/tunnel"
)

// These handlers run inside the per-action goroutine spawned by invokeAction
// so they do not need their own goroutine wrappers.

func handleSideLoadExternalProvider(providerName string, data []byte, fn func(value string)) {
	runLock.Lock()
	defer runLock.Unlock()
	p, ok := tunnel.Providers()[providerName]
	if !ok || p.VehicleType() == cp.Compatible {
		fn("external provider is not exist")
		return
	}
	psp, ok := p.(*provider.ProxySetProvider)
	if !ok {
		fn("not a proxy provider")
		return
	}
	if _, _, err := psp.SideUpdate(data); err != nil {
		fn(err.Error())
		return
	}
	fn("")
}

func handleUpdateGeoData(geoType string, fn func(value string)) {
	var err error
	switch geoType {
	case "MMDB":
		err = updater.UpdateMMDB()
	case "ASN":
		err = updater.UpdateASN()
	case "GEOIP":
		err = updater.UpdateGeoIp()
	case "GEOSITE":
		err = updater.UpdateGeoSite()
	}
	if err != nil {
		fn(err.Error())
		return
	}
	fn("")
}

// handleGetCountryCode does a read-only MMDB lookup; mihomo's mmdb package
// uses atomic.Pointer for hot-swap, so we don't need runLock.
func handleGetCountryCode(ip string, fn func(value string)) {
	codes := mmdb.IPInstance().LookupCode(net.ParseIP(ip))
	if len(codes) == 0 {
		fn("")
		return
	}
	fn(codes[0])
}
