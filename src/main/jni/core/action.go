//go:build android && cgo

package main

import (
	"encoding/json"
	"unsafe"

	"github.com/metacubex/mihomo/log"
)

type Action struct {
	Id     string      `json:"id"`
	Method Method      `json:"method"`
	Data   interface{} `json:"data"`
}

type ActionResult struct {
	Id       string      `json:"id"`
	Method   Method      `json:"method"`
	Data     interface{} `json:"data"`
	Code     int         `json:"code"`
	callback unsafe.Pointer
}

func (result *ActionResult) Json() ([]byte, error) {
	data, err := json.Marshal(result)
	return data, err
}

func (result *ActionResult) success(data interface{}) {
	result.Code = 0
	result.Data = data
	result.send()
}

func (result *ActionResult) error(data interface{}) {
	result.Code = -1
	result.Data = data
	result.send()
}

// parseStringData type-asserts action.Data to string and routes an
// "invalid data type" error to the caller on mismatch. Returns (s, ok)
// where ok=false means the caller should `return` after this call.
func parseStringData(data any, result *ActionResult) (string, bool) {
	s, ok := data.(string)
	if !ok {
		result.error("invalid data type")
		return "", false
	}
	return s, true
}

func parseLogLevelData(data any, result *ActionResult) (log.LogLevel, bool) {
	switch v := data.(type) {
	case float64:
		return log.LogLevel(int(v)), true
	case int:
		return log.LogLevel(v), true
	case string:
		l, ok := log.LogLevelMapping[v]
		if !ok {
			result.error("invalid log level: " + v)
			return log.INFO, false
		}
		return l, true
	default:
		result.error("invalid data type")
		return log.INFO, false
	}
}

// Wraps "" = success, non-empty = error contract used by geo_data.go.
// Data is always the string the handler produced — empty on success, the
// error message otherwise. Consumers parse `data` as String regardless.
func actionCallback(result *ActionResult) func(string) {
	return func(value string) {
		if value == "" {
			result.success("")
		} else {
			result.error(value)
		}
	}
}

func handleAction(action *Action, result ActionResult) {
	switch action.Method {
	case initClashMethod:
		paramsString, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		result.success(handleInitClash(paramsString))
		return
	case getIsInitMethod:
		result.success(handleGetIsInit())
		return
	case forceGcMethod:
		handleForceGC()
		result.success(true)
		return
	case validateConfigMethod:
		path, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		result.success(handleValidateConfig(path))
		return
	case updateConfigMethod:
		s, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		result.success(handleUpdateConfig([]byte(s)))
		return
	case setupConfigMethod:
		s, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		result.success(handleSetupConfig([]byte(s)))
		return
	case getConfigMethod:
		path, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		config, err := handleGetConfig(path)
		if err != nil {
			result.error(err.Error())
			return
		}
		result.success(config)
		return
	case resetTrafficMethod:
		handleResetTraffic()
		result.success(true)
		return
	case resetConnectionsMethod:
		result.success(handleResetConnections())
		return
	case sideLoadExternalProviderMethod:
		paramsString, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		var params = map[string]string{}
		if err := json.Unmarshal([]byte(paramsString), &params); err != nil {
			result.error(err.Error())
			return
		}
		handleSideLoadExternalProvider(
			params["providerName"],
			[]byte(params["data"]),
			actionCallback(&result),
		)
		return
	case updateGeoDataMethod:
		paramsString, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		var params = map[string]string{}
		if err := json.Unmarshal([]byte(paramsString), &params); err != nil {
			result.error(err.Error())
			return
		}
		handleUpdateGeoData(params["geo-type"], actionCallback(&result))
		return
	case startLogMethod:
		handleStartLog()
		result.success(true)
		return
	case stopLogMethod:
		handleStopLog()
		result.success(true)
		return
	case startListenerMethod:
		result.success(handleStartListener())
		return
	case stopListenerMethod:
		result.success(handleStopListener())
		return
	case getCountryCodeMethod:
		ip, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		handleGetCountryCode(ip, func(value string) {
			result.success(value)
		})
		return
	case crashMethod:
		result.success(true)
		handleCrash()
	case deleteFileMethod:
		path, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		handleDelFile(path, result)
		return
	case getProxiesMethod:
		result.success(handleGetProxies())
		return
	case changeProxyMethod:
		s, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		var params ChangeProxyParams
		if err := json.Unmarshal([]byte(s), &params); err != nil {
			result.error(err.Error())
			return
		}
		if params.GroupName == nil || params.ProxyName == nil {
			result.error("group-name and proxy-name are required")
			return
		}
		if err := handleChangeProxy(*params.GroupName, *params.ProxyName); err != nil {
			result.error(err.Error())
			return
		}
		result.success("")
		return
	case testDelayMethod:
		s, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		var params TestDelayParams
		if err := json.Unmarshal([]byte(s), &params); err != nil {
			result.error(err.Error())
			return
		}
		result.success(handleAsyncTestDelay(params.ProxyName, params.TestUrl, int(params.Timeout)))
		return
	case probeCurrentProxyIpMethod:
		modeHint, _ := action.Data.(string)
		result.success(handleProbeCurrentProxyIp(modeHint))
		return
	case queryExternalProvidersMethod:
		result.success(queryExternalProviders())
		return
	case getExternalProviderMethod:
		s, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		var params struct {
			Type string `json:"type"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(s), &params); err != nil {
			result.error(err.Error())
			return
		}
		result.success(getExternalProvider(params.Type, params.Name))
		return
	case updateExternalProviderMethod:
		s, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		var params struct {
			Type string `json:"type"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(s), &params); err != nil {
			result.error(err.Error())
			return
		}
		if errMsg := updateExternalProvider(params.Type, params.Name); errMsg != "" {
			result.error(errMsg)
		} else {
			result.success("")
		}
		return
	case getTrafficMethod:
		result.success(handleGetTraffic())
		return
	case getTotalTrafficMethod:
		result.success(handleGetTotalTraffic())
		return
	case getMemoryMethod:
		result.success(handleGetMemory())
		return
	case getConnectionsMethod:
		result.success(handleGetConnections())
		return
	case queryProxyGroupOrderMethod:
		result.success(queryProxyGroupOrder())
		return
	case subscribeConnectionsMethod:
		handleSubscribeConnections()
		result.success(true)
		return
	case unsubscribeConnectionsMethod:
		handleUnsubscribeConnections()
		result.success(true)
		return
	case closeConnectionMethod:
		id, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		result.success(handleCloseConnection(id))
		return
	case closeAllConnectionsMethod:
		result.success(handleCloseAllConnections())
		return
	case setLogcatLevelMethod:
		level, ok := parseLogLevelData(action.Data, &result)
		if !ok {
			return
		}
		setLogcatLevelImpl(level)
		result.success(true)
		return
	case setFileLevelMethod:
		level, ok := parseLogLevelData(action.Data, &result)
		if !ok {
			return
		}
		setFileLevelImpl(level)
		result.success(true)
		return
	case setFileEnabledMethod:
		enabled, ok := action.Data.(bool)
		if !ok {
			result.error("invalid data type")
			return
		}
		setFileEnabledImpl(enabled)
		result.success(true)
		return
	case setLogFilePathMethod:
		path, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		if err := setLogFilePathImpl(path); err != nil {
			result.error(err.Error())
			return
		}
		result.success(true)
		return
	case forwardHostLogMethod:
		s, ok := parseStringData(action.Data, &result)
		if !ok {
			return
		}
		var params HostLogParams
		if err := json.Unmarshal([]byte(s), &params); err != nil {
			result.error(err.Error())
			return
		}
		hostLog(log.LogLevel(params.Level), params.Tag, params.Payload)
		result.success(true)
		return
	default:
		result.error("unknown method: " + string(action.Method))
	}
}
