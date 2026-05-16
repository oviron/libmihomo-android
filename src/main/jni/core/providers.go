//go:build android && cgo

package main

import (
	"encoding/json"
	"fmt"
	"time"

	P "github.com/metacubex/mihomo/constant/provider"
	"github.com/metacubex/mihomo/tunnel"
)

type updatableProvider interface {
	UpdatedAt() time.Time
}

type vehicleProvider interface {
	Vehicle() P.Vehicle
}

// externalProvider is the common surface shared by ProxyProvider and
// RuleProvider for our purposes: both implement Provider + Count.
type externalProvider interface {
	P.Provider
	Count() int
}

// lookupExternalProvider returns the Proxy/Rule provider matching name,
// skipping the implicit Compatible (global) provider. ok=false signals that
// nothing usable exists under that (pType, name) pair. Compared against
// mihomo's canonical ProviderType.String() so a future upstream rename
// surfaces as a build break, not a silent miss.
func lookupExternalProvider(pType, name string) (externalProvider, bool) {
	var p externalProvider
	switch pType {
	case P.Proxy.String():
		if v, ok := tunnel.Providers()[name]; ok {
			p = v
		}
	case P.Rule.String():
		if v, ok := tunnel.RuleProviders()[name]; ok {
			p = v
		}
	}
	if p == nil || p.VehicleType() == P.Compatible {
		return nil, false
	}
	return p, true
}

func queryExternalProviders() string {
	list := make([]ExternalProvider, 0)
	for _, p := range tunnel.Providers() {
		if p.VehicleType() == P.Compatible {
			continue
		}
		list = append(list, providerView(p, p.Count()))
	}
	for _, p := range tunnel.RuleProviders() {
		if p.VehicleType() == P.Compatible {
			continue
		}
		list = append(list, providerView(p, p.Count()))
	}
	data, _ := json.Marshal(list)
	return string(data)
}

func getExternalProvider(pType, name string) string {
	p, ok := lookupExternalProvider(pType, name)
	if !ok {
		return ""
	}
	data, _ := json.Marshal(providerView(p, p.Count()))
	return string(data)
}

func updateExternalProvider(pType, name string) string {
	p, ok := lookupExternalProvider(pType, name)
	if !ok {
		return fmt.Sprintf("%s provider %q not found", pType, name)
	}
	if err := p.Update(); err != nil {
		return err.Error()
	}
	return ""
}

func providerView(p P.Provider, count int) ExternalProvider {
	view := ExternalProvider{
		Name:        p.Name(),
		Type:        p.Type().String(),
		VehicleType: p.VehicleType().String(),
		Count:       count,
	}
	if v, ok := p.(vehicleProvider); ok {
		view.Path = v.Vehicle().Path()
	}
	if u, ok := p.(updatableProvider); ok {
		view.UpdateAt = u.UpdatedAt()
	}
	return view
}
