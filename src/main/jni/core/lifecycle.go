//go:build android && cgo

package main

import (
	"encoding/json"
	"runtime"
	"runtime/debug"

	"github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/log"
)

func handleInitClash(paramsString string) bool {
	runLock.Lock()
	defer runLock.Unlock()
	var params = InitParams{}
	if err := json.Unmarshal([]byte(paramsString), &params); err != nil {
		return false
	}
	// #nosec G115 -- Android Build.VERSION.SDK_INT is single-digit-decade int.
	version.Store(int32(params.Version))
	constant.SetHomeDir(params.HomeDir)
	isInit.Store(true)
	return true
}

func handleGetIsInit() bool {
	return isInit.Load()
}

func handleForceGC() {
	log.Infoln("[APP] request force GC")
	runtime.GC()
	debug.FreeOSMemory()
}
