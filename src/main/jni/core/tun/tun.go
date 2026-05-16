//go:build android && cgo

// Package tun builds a sing-tun listener over the file descriptor supplied by
// Android's VpnService.Builder. Adapted from MetaCubeX/ClashMetaForAndroid
// (GPL-3.0). AutoRoute is forced false because the actual route is configured
// by VpnService.Builder, not mihomo.
package tun

import (
	"io"
	"net"
	"net/netip"
	"strings"

	"github.com/metacubex/mihomo/constant"
	LC "github.com/metacubex/mihomo/listener/config"
	"github.com/metacubex/mihomo/listener/sing_tun"
	"github.com/metacubex/mihomo/log"
	"github.com/metacubex/mihomo/tunnel"
)

// MTU matches CMfA. Android VpnService.Builder caps at 65535.
const tunMTU = 9000

// Start builds a sing_tun listener from the address/dns/stack inputs supplied
// by the Android VpnService.Builder side. Caller chooses the device name —
// it surfaces as the interface label inside mihomo logs and metrics. Returns
// an io.Closer so consumers do not have to import sing_tun directly.
func Start(fd int, device, stack, address, dns string) (io.Closer, error) {
	var prefix4 []netip.Prefix
	var prefix6 []netip.Prefix
	tunStack, ok := constant.StackTypeMapping[strings.ToLower(stack)]
	if !ok {
		tunStack = constant.TunSystem
	}
	for _, a := range strings.Split(address, ",") {
		a = strings.TrimSpace(a)
		if len(a) == 0 {
			continue
		}
		prefix, err := netip.ParsePrefix(a)
		if err != nil {
			return nil, err
		}
		if prefix.Addr().Is4() {
			prefix4 = append(prefix4, prefix)
		} else {
			prefix6 = append(prefix6, prefix)
		}
	}

	var dnsHijack []string
	for _, d := range strings.Split(dns, ",") {
		d = strings.TrimSpace(d)
		if len(d) == 0 {
			continue
		}
		dnsHijack = append(dnsHijack, net.JoinHostPort(d, "53"))
	}

	options := LC.Tun{
		Enable:              true,
		Device:              device,
		Stack:               tunStack,
		DNSHijack:           dnsHijack,
		AutoRoute:           false,
		AutoDetectInterface: false,
		Inet4Address:        prefix4,
		Inet6Address:        prefix6,
		MTU:                 tunMTU,
		FileDescriptor:      fd,
	}

	listener, err := sing_tun.New(options, tunnel.Tunnel)
	if err != nil {
		return nil, err
	}
	log.Infoln("TUN started: device=%s addresses=%s stack=%s", device, address, tunStack)
	return listener, nil
}
