# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Until v1.0 the public API is considered unstable; breaking changes bump
`bridgeABI` and ship in a minor or patch release.

## [Unreleased]

## [0.3.1] — 2026-07-09

### Changed
- Bumped bundled [metacubex/mihomo](https://github.com/MetaCubeX/mihomo)
  `v1.19.27` → `v1.19.28`. Picks up the crypto/tls fix for **CVE-2026-42505**
  (via `metacubex/tls` `0.1.6` → `0.1.7`) plus several remote-triggerable panic
  fixes — AmneziaWG receive-path slice-bounds OOB, masque dial panic, `rand.IntN`
  on a nil slice, and a `sing-mux` UDP-write bug. Rides along `sing-tun`
  `0.4.20` → `0.4.21`, `utls` `1.8.4` → `1.8.7`, `sing-mux` `0.3.10`,
  `restls-client-go` `0.1.8`, and `mieru` `3.34.0` dep bumps. The rest of
  upstream is additive outbound/listener features (`rematch` outbound, masque
  h3, openvpn peer-info, snell shadow-tls) outside our client path. The
  `dualStackDialContext` fallback-connection-leak fix touches our outbound
  dialing. JNI/facade surface and `bridgeABI` (`3`) unchanged.

## [0.3.0] — 2026-07-08

### Fixed
- resolveProcess JNI upcall no longer aborts the `:remote` process. On Android
  11+ package-visibility filtering the consumer's `getPackagesForUid()` can
  return an empty array, whose `.first()` throws across the JNI boundary; the
  native shim now null/exception-guards the `packageName` upcall
  (`native-lib.cpp`) and `jni_get_string` (`jni_helper.cpp`), returning an empty
  string instead of a `JNI DETECTED ERROR ... obj == null` SIGABRT crash-loop.

### Added
- `resolveProcess` may now return `"<uid>\n<package>"`; the Go resolver parses
  the leading uid into `metadata.Uid`, reviving mihomo UID-based rule matching. A
  plain package string (no newline) still works, so the parse is backward
  compatible.

### Changed
- **Breaking:** `bridgeABI` `2` → `3`. The `resolveProcess` return-string
  protocol gained the optional `uid\npackage` convention; a consumer emitting it
  against an ABI-2 `.so` would have its package silently misparsed, so the ABI
  gate refuses the mismatched pairing. Rebuild against the new facade. Bundled
  mihomo core unchanged (`v1.19.27`).

## [0.2.0] — 2026-07-07

### Added
- `startTUN` now takes an `mtu` parameter (Go export, C header, JNI, and Kotlin
  facade), so the caller sets the tun interface MTU instead of the hardcoded
  `9000`. `mtu <= 0` falls back to the previous `9000` default. Lets a consumer
  tune the MTU down for mobile/encapsulated paths where the jumbo default
  silently blackholes oversized packets.

### Changed
- **Breaking:** `bridgeABI` `1` → `2`. The `startTUN` native-boundary signature
  gained the `mtu` argument, so a consumer built against ABI 1 cannot load this
  `.so` (the facade/`.so` ABI check refuses the mismatch). Rebuild against the
  new facade. Bundled mihomo core is unchanged from `v0.1.5` (`v1.19.27`).

## [0.1.5] — 2026-06-17

### Changed
- Bumped bundled [metacubex/mihomo](https://github.com/MetaCubeX/mihomo)
  `v1.19.26` → `v1.19.27`. Security release: fixes several remote-triggerable
  core crashes — QUIC sniffer out-of-bounds read (crash via a single UDP
  packet), Vision TLS filter OOB via a crafted `session_id`, Trojan UDP relay
  panic, and a socks4 unbounded allocation. The QUIC sniffer fix is the one
  relevant to our client path. Pulls in `age`/`sevenzip`/`brotli` indirect
  deps behind upstream's new `path-in-bundle` rule-providers and
  `age-secret-key` features. Upstream removed `global-client-fingerprint`
  (set `client-fingerprint` on the proxy instead); not used by our consumers.
  JNI/facade surface and `bridgeABI` unchanged.

## [0.1.4] — 2026-06-02

### Changed
- Bumped bundled [metacubex/mihomo](https://github.com/MetaCubeX/mihomo)
  `v1.19.25` → `v1.19.26`. No CVE in this range; substance is
  OpenVPN/Snell/mieru/Tailscale work outside our outbound path. Rides along
  sing-tun `0.4.20`, quic-go, and metacubex/tls `0.1.6` dep bumps. JNI/facade
  surface and `bridgeABI` unchanged.
- Added Go build tag `no_tailscale` to the release build
  (`with_gvisor,cmfa,no_tailscale`). Drops the unused Tailscale mesh-VPN
  outbound stack, shrinking `libclash.so` by ~12 MB/ABI (~36 MB across the
  `.aar`).

### Added
- `metadata.json` release asset: machine-readable manifest declaring the
  bundled core (`mihomo vX.Y.Z`), `bridgeABI`, ABIs, and AAR SHA-256. The
  bundled core version is now also in the release title and the README
  version matrix, so consumers (and the planned in-app version picker) can
  see which core a wrapper release ships before downloading.

## [0.1.3] — 2026-05-29

### Changed
- Bumped bundled [metacubex/mihomo](https://github.com/MetaCubeX/mihomo)
  `v1.19.24` → `v1.19.25`. Picks up the `net/http` CVE-2026-39825 fix and
  the vless xhttp h3 quic-dial panic fix; the JNI/facade surface and
  `bridgeABI` are unchanged.

## [0.1.0] — 2026-05-17

Initial public release.

### Added
- JNI bridge to [metacubex/mihomo](https://github.com/MetaCubeX/mihomo)
  `v1.19.24`, statically linked into `libclash.so` per ABI (arm64-v8a,
  armeabi-v7a, x86_64), built with Go tags `with_gvisor,cmfa`.
- `libmihomo-jni.so` per ABI, built via Gradle CMake from
  `src/main/cpp/{native-lib.cpp, jni_helper.cpp}`. Holds `JNI_OnLoad`
  that wires up the bridge callback function pointers.
- Kotlin facade `io.github.oviron.libmihomo.Clash` with 11 entry points
  (`invokeAction`, `quickSetup`, `startTUN`, `stopTun`,
  `setEventListener`, `getTraffic`, `getTotalTraffic`, `suspended`,
  `forceGC`, `updateDNS`, `bridgeABI`) and lambda-friendly overloads.
- Typed callback interfaces `TunInterface` (`protect`,
  `resolverProcess`) and `InvokeInterface` (`onResult`).
- `Clash.load(nativeLibDir: String)` explicit load step via
  `System.load(absolutePath)`. The path argument is the seam for
  runtime-pluggable versions.
- `Clash.isLoaded()` / `Clash.assertReady()` for explicit load-state
  introspection. Every facade method invokes `assertReady()`.
- `bridgeABI() = 1` runtime constant for facade ↔ `.so` compat check.
- `consumer-rules.pro` covers `Clash` + `TunInterface` +
  `InvokeInterface` + native-method wildcard so consumer R8 cannot
  strip JNI-referenced classes.
- `scripts/validate-jni-keep.sh` diffs JNI lookups in C/C++ sources
  against `-keep` rules. Wired into Gradle `preBuild`, so any build
  with mismatched coverage fails before a tag is cut.
- `assembleRelease` packages all three ABIs plus the Kotlin facade
  into a single `.aar`. Release artifacts: `.aar` + `.aar.sha256` +
  `.aar.asc` (detached GPG signature, key fingerprint
  `1139 C91B 6525 883E 6783 DCF0 4A94 DA48 8A4C 5033`).

### Notes
- Compiled binary is GPL-3.0 (mihomo static link). Bridge source in
  this repo is also GPL-3.0.
- Pinned toolchain: Go 1.20+ runtime, NDK `28.0.13004108`, AGP `8.12.2`,
  Kotlin `2.2.10`.
