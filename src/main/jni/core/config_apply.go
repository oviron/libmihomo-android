//go:build android && cgo

package main

import (
	"encoding/json"

	"github.com/metacubex/mihomo/config"
	"github.com/metacubex/mihomo/log"
)

func handleValidateConfig(path string) string {
	buf, err := readFile(path)
	if err != nil {
		return err.Error()
	}
	if _, err := config.UnmarshalRawConfig(buf); err != nil {
		return err.Error()
	}
	return ""
}

func handleGetConfig(path string) (*config.RawConfig, error) {
	bytes, err := readFile(path)
	if err != nil {
		return nil, err
	}
	prof, err := config.UnmarshalRawConfig(bytes)
	if err != nil {
		return nil, err
	}
	return prof, nil
}

func handleUpdateConfig(bytes []byte) string {
	if !isInit.Load() {
		return "not initialized"
	}
	if currentConfig == nil {
		return "config not loaded"
	}
	var params = &UpdateParams{}
	if err := json.Unmarshal(bytes, params); err != nil {
		return err.Error()
	}
	updateConfig(params)
	return ""
}

func handleSetupConfig(bytes []byte) string {
	if !isInit.Load() {
		return "not initialized"
	}
	var params = defaultSetupParams()
	if err := UnmarshalJson(bytes, params); err != nil {
		log.Errorln("setupConfig: unmarshal error: %v", err)
		return err.Error()
	}
	if err := applyConfig(params); err != nil {
		return err.Error()
	}
	return ""
}
