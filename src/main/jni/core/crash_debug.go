//go:build android && cgo && debug

package main

// handleCrash deliberately panics for crash-reporter integration tests.
// Only available in debug builds; release builds use the no-op stub.
func handleCrash() {
	panic("handle invoke crash")
}
