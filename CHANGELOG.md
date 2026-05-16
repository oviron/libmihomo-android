# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Until v1.0 the public API is considered unstable; breaking changes bump
`bridgeABI` and ship in a minor or patch release.

## [Unreleased]

## [0.1.0] — 2026-05-17

Initial public release.

### Added
- JNI bridge to [metacubex/mihomo](https://github.com/MetaCubeX/mihomo)
  `v1.19.24`, statically linked into `libclash.so` per ABI (arm64-v8a,
  armeabi-v7a, x86_64).
- Kotlin facade `io.github.oviron.libmihomo.Clash` with 11 `external fun`
  declarations: `invokeAction`, `quickSetup`, `startTUN`, `stopTun`,
  `setEventListener`, `getTraffic`, `getTotalTraffic`, `suspend`, `forceGC`,
  `updateDns`, `bridgeABI`.
- `bridgeABI() = 1` runtime constant for stub ↔ `.so` compatibility checks.
- `startTUN` takes a caller-supplied `device` name (was hardcoded to
  `"FlClash"` in the in-tree FlClash sources).
- `assembleRelease` packages all three ABIs into a single `.aar` plus the
  Kotlin stub. Release artifacts: `.aar` + `.aar.sha256` + `.aar.asc`
  (detached GPG signature, key fingerprint
  `1139 C91B 6525 883E 6783 DCF0 4A94 DA48 8A4C 5033`).

### Notes
- Compiled binary is GPL-3.0 (mihomo static link). Bridge source code in this
  repo is also GPL-3.0. See `LICENSE` and `README.md` § License.
- Pinned toolchain: Go 1.20+ runtime, NDK `28.0.13004108`, AGP `8.12.2`,
  Kotlin `2.2.10`. Reproducible across runs with the same toolchain.
- Built with Go tags `with_gvisor,cmfa`. Without these tags the gVisor TUN
  stack and ClashMetaForAndroid compatibility code are not compiled in.
