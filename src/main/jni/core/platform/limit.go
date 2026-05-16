//go:build linux

// Package platform implements Android/Linux-specific helpers used by the
// mihomo bridge: FD-pressure guards (this file) and /proc/net UID lookup
// (procfs.go). Adapted from MetaCubeX/ClashMetaForAndroid (GPL-3.0).
package platform

import (
	"syscall"

	"github.com/metacubex/mihomo/log"
)

// fdSafetyDenom defines the soft reserve: blocked above 3/4 of process RLIMIT_NOFILE.
const fdSafetyDenom = 4
const fdRlimitFallback = 1024

var (
	nullFd     = -1
	maxFdCount int
)

func init() {
	fd, err := syscall.Open("/dev/null", syscall.O_WRONLY, 0644)
	if err != nil {
		// Degrade instead of crashing: as a library we cannot panic at load.
		log.Warnln("platform.limit: cannot open /dev/null (%v); FD-pressure guard disabled", err)
		return
	}
	nullFd = fd

	var limit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
		maxFdCount = fdRlimitFallback
	} else {
		// #nosec G115 -- RLIMIT_NOFILE on Android is bounded by kernel (<<2^31).
		maxFdCount = int(limit.Cur)
	}

	maxFdCount = maxFdCount * (fdSafetyDenom - 1) / fdSafetyDenom
}

// ShouldBlockConnection returns true when the process is near its FD limit,
// so the dialer hook can refuse new sockets before exhausting the table.
// If the /dev/null guard could not be opened at init, this is a no-op.
func ShouldBlockConnection() bool {
	if nullFd < 0 {
		return false
	}
	fd, err := syscall.Dup(nullFd)
	if err != nil {
		return true
	}
	_ = syscall.Close(fd)
	return fd > maxFdCount
}
