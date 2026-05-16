//go:build android && cgo && !debug

package main

import "github.com/metacubex/mihomo/log"

// handleCrash is a no-op in release builds. The crashMethod action surface
// is preserved so existing Dart callers don't get "unknown method", but the
// process is not killed.
func handleCrash() {
	log.Warnln("crashMethod invoked but not available in release build")
}
