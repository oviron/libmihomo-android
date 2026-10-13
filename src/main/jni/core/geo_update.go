//go:build android && cgo

package main

import (
	"os"
	"sync"
	"time"

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
		interval := time.Duration(updater.GeoUpdateInterval()) * time.Hour
		if updated, ok := geoUpdatedAt(); ok && updater.GeoAutoUpdate() &&
			interval > 0 && time.Since(updated) >= interval {
			if err := updater.UpdateGeoDatabases(); err != nil {
				log.Warnln("[GEO] auto-update failed: %v", err)
			} else {
				markGeoChecked()
			}
		}
		time.Sleep(time.Hour)
	}
}

func geoPaths() []string {
	return []string{C.Path.GeoIP(), C.Path.MMDB(), C.Path.ASN(), C.Path.GeoSite()}
}

// geoUpdatedAt mirrors mihomo's own check: no database on disk, nothing to refresh.
func geoUpdatedAt() (time.Time, bool) {
	for _, path := range geoPaths() {
		if info, err := os.Stat(path); err == nil {
			return info.ModTime(), true
		}
	}
	return time.Time{}, false
}

// markGeoChecked restarts the interval: an unchanged database is not
// rewritten, and its old mtime would otherwise ask for a check every hour.
func markGeoChecked() {
	now := time.Now()
	for _, path := range geoPaths() {
		_ = os.Chtimes(path, now, now)
	}
}
