//go:build android && cgo

package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	mihomoHttp "github.com/metacubex/mihomo/component/http"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/tunnel"
)

// Sentinel checked by Dart UI to render a REJECT badge instead of spinning.
const rejectedProbeBody = `{"status":"REJECT"}`

// Public IP/geo probe endpoint. Returns JSON the Dart UI parses as-is.
const probeURL = "https://ipinfo.io/json"

// handleProbeCurrentProxyIp uses WithSpecialProxy so resolveMetadata bypasses
// user rules. Empty modeHint falls back to tunnel.Mode(), which can lag a
// mode switch by one Dart-side debounce.
func handleProbeCurrentProxyIp(modeHint string) string {
	target := determineProbeTarget(modeHint)
	if target == "REJECT" {
		return rejectedProbeBody
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	resp, err := mihomoHttp.HttpRequest(
		ctx, probeURL, http.MethodGet, nil, nil,
		mihomoHttp.WithSpecialProxy(target),
	)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			_ = resp.Body.Close()
		}
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(body)
}

func determineProbeTarget(modeHint string) string {
	switch resolveMode(modeHint) {
	case tunnel.Direct:
		return "DIRECT"
	case tunnel.Global:
		return "GLOBAL"
	}
	// Rule mode: the first top-level MATCH rule is the default route for
	// traffic that didn't hit any narrower rule. A MATCH inside a sub-rule is
	// scoped to that sub-rule and does not count.
	for _, rule := range tunnel.Rules() {
		if rule.RuleType() == C.MATCH {
			return rule.Adapter()
		}
	}
	// No MATCH at all: mihomo drops unspecified traffic. Surface REJECT
	// honestly instead of lying with GLOBAL[selected].
	return "REJECT"
}

func resolveMode(modeHint string) tunnel.TunnelMode {
	if m, ok := tunnel.ModeMapping[strings.ToLower(modeHint)]; ok {
		return m
	}
	return tunnel.Mode()
}
