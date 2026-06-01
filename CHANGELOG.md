# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Until v1.0 the public API is considered unstable; breaking changes bump
`bridgeABI` and ship in a minor or patch release.

## [Unreleased]

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
