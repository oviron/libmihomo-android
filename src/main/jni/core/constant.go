//go:build android && cgo

package main

import (
	"net/netip"
	"time"

	P "github.com/metacubex/mihomo/component/process"
	"github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/log"
	"github.com/metacubex/mihomo/tunnel"
)

type InitParams struct {
	HomeDir string `json:"home-dir"`
	Version int    `json:"version"`
}

type SetupParams struct {
	SelectedMap map[string]string `json:"selected-map"`
}

type UpdateParams struct {
	Tun             *tunSchema         `json:"tun"`
	AllowLan        *bool              `json:"allow-lan"`
	MixedPort       *int               `json:"mixed-port"`
	FindProcessMode *P.FindProcessMode `json:"find-process-mode"`
	Mode            *tunnel.TunnelMode `json:"mode"`
	LogLevel        *log.LogLevel      `json:"log-level"`
	IPv6            *bool              `json:"ipv6"`
	Sniffing        *bool              `json:"sniffing"`
	TCPConcurrent   *bool              `json:"tcp-concurrent"`
	Interface       *string            `json:"interface-name"`
	UnifiedDelay    *bool              `json:"unified-delay"`
}

type tunSchema struct {
	Enable       bool               `yaml:"enable" json:"enable"`
	Device       *string            `yaml:"device" json:"device"`
	Stack        *constant.TUNStack `yaml:"stack" json:"stack"`
	DNSHijack    *[]string          `yaml:"dns-hijack" json:"dns-hijack"`
	AutoRoute    *bool              `yaml:"auto-route" json:"auto-route"`
	RouteAddress *[]netip.Prefix    `yaml:"route-address" json:"route-address,omitempty"`
}

type ChangeProxyParams struct {
	GroupName *string `json:"group-name"`
	ProxyName *string `json:"proxy-name"`
}

type TestDelayParams struct {
	ProxyName string `json:"proxy-name"`
	TestUrl   string `json:"test-url"`
	Timeout   int64  `json:"timeout"`
}

// Level is raw int — Dart LogLevel.index / Kotlin .ordinal. Avoids coupling
// to mihomo's LogLevel.UnmarshalText (only fires on JSON strings).
type HostLogParams struct {
	Level   int    `json:"level"`
	Tag     string `json:"tag"`
	Payload string `json:"payload"`
}

type ExternalProvider struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	VehicleType string    `json:"vehicle-type"`
	Count       int       `json:"count"`
	Path        string    `json:"path"`
	UpdateAt    time.Time `json:"update-at"`
}

const (
	messageMethod                  Method = "message"
	initClashMethod                Method = "initClash"
	getIsInitMethod                Method = "getIsInit"
	forceGcMethod                  Method = "forceGc"
	validateConfigMethod           Method = "validateConfig"
	updateConfigMethod             Method = "updateConfig"
	resetTrafficMethod             Method = "resetTraffic"
	resetConnectionsMethod         Method = "resetConnections"
	getCountryCodeMethod           Method = "getCountryCode"
	updateGeoDataMethod            Method = "updateGeoData"
	sideLoadExternalProviderMethod Method = "sideLoadExternalProvider"
	startLogMethod                 Method = "startLog"
	stopLogMethod                  Method = "stopLog"
	startListenerMethod            Method = "startListener"
	stopListenerMethod             Method = "stopListener"
	crashMethod                    Method = "crash"
	setupConfigMethod              Method = "setupConfig"
	getConfigMethod                Method = "getConfig"
	deleteFileMethod               Method = "deleteFile"
	getProxiesMethod               Method = "getProxies"
	changeProxyMethod              Method = "changeProxy"
	testDelayMethod                Method = "testDelay"
	probeCurrentProxyIpMethod      Method = "probeCurrentProxyIp"
	queryExternalProvidersMethod   Method = "queryExternalProviders"
	getExternalProviderMethod      Method = "getExternalProvider"
	updateExternalProviderMethod   Method = "updateExternalProvider"
	getTrafficMethod               Method = "getTraffic"
	getTotalTrafficMethod          Method = "getTotalTraffic"
	getMemoryMethod                Method = "getMemory"
	getConnectionsMethod           Method = "getConnections"
	queryProxyGroupOrderMethod     Method = "queryProxyGroupOrder"
	subscribeConnectionsMethod     Method = "subscribeConnections"
	unsubscribeConnectionsMethod   Method = "unsubscribeConnections"
	closeConnectionMethod          Method = "closeConnection"
	closeAllConnectionsMethod      Method = "closeAllConnections"
	setLogcatLevelMethod           Method = "setLogcatLevel"
	setFileLevelMethod             Method = "setFileLevel"
	setFileEnabledMethod           Method = "setFileEnabled"
	setLogFilePathMethod           Method = "setLogFilePath"
	forwardHostLogMethod           Method = "forwardHostLog"
)

type Method string

type MessageType string

type Message struct {
	Type MessageType `json:"type"`
	Data interface{} `json:"data"`
}

const (
	LogMessage         MessageType = "log"
	ConnectionsMessage MessageType = "connections"
)
