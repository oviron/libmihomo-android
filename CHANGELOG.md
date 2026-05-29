# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Until v1.0 the public API is considered unstable; breaking changes bump
`bridgeABI` and ship in a minor or patch release.

## [Unreleased]

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
