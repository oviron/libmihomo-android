//go:build android && cgo

package main

import (
	"os"
	"sync"
	"time"

	"github.com/metacubex/mihomo/component/geodata"
	"github.com/metacubex/mihomo/component/updater"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/log"
)

var geoUpdaterOnce sync.Once

// startGeoUpdater runs mihomo's geo-auto-update schedule, which only its CLI
// main starts. The loop re-reads the settings, so a later config moves or stops it.
func startGeoUpdater() {
	if !updater.GeoAutoUpdate() {
		return
	}
	geoUpdaterOnce.Do(func() { go geoUpdateLoop() })
}

func geoUpdateLoop() {
	log.Infoln("[GEO] auto-update started")
	for {
		if paths, due := geoUpdateDue(); due {
			if err := updater.UpdateGeoDatabases(); err != nil {
				log.Warnln("[GEO] auto-update failed: %v", err)
			} else {
				markGeoChecked(paths)
			}
		}
		time.Sleep(time.Hour)
	}
}

// geoUpdateDue reads the settings under runLock, which a config apply writes
// them under, and is due once the oldest database in use is a full interval old.
func geoUpdateDue() ([]string, bool) {
	runLock.Lock()
	auto := updater.GeoAutoUpdate()
	interval := time.Duration(updater.GeoUpdateInterval()) * time.Hour
	paths := geoPathsInUse()
	runLock.Unlock()
	if !auto || interval <= 0 {
		return nil, false
	}
	var oldest time.Time
	var onDisk []string
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			onDisk = append(onDisk, path)
			if oldest.IsZero() || info.ModTime().Before(oldest) {
				oldest = info.ModTime()
			}
		}
	}
	// Like mihomo: no database on disk yet, nothing to refresh.
	return onDisk, len(onDisk) > 0 && time.Since(oldest) >= interval
}

// geoPathsInUse are the databases UpdateGeoDatabases checks.
func geoPathsInUse() []string {
	var paths []string
	if geodata.GeoIpEnable() {
		if geodata.GeodataMode() {
			paths = append(paths, C.Path.GeoIP())
		} else {
			paths = append(paths, C.Path.MMDB())
		}
	}
	if geodata.ASNEnable() {
		paths = append(paths, C.Path.ASN())
	}
	if geodata.GeoSiteEnable() {
		paths = append(paths, C.Path.GeoSite())
	}
	return paths
}

// markGeoChecked restarts the interval: an unchanged database is not
// rewritten, and its old mtime would otherwise ask for a check every hour.
func markGeoChecked(paths []string) {
	now := time.Now()
	for _, path := range paths {
		_ = os.Chtimes(path, now, now)
	}
}
