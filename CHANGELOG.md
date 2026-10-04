# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Until v1.0 the public API is considered unstable; breaking changes bump
`bridgeABI` and ship in a minor or patch release.

## [Unreleased]

## [0.3.7] — 2026-10-04

### Added
- DNS query history. The core keeps the last 500 queries that reach mihomo's
  `resolver.DefaultService`: TUN DNS hijack and the `dns` outbound. Each entry
  has domain, query type, answers, rcode or error, latency in ms, and time.
  The service is wrapped after every config apply, so no mihomo fork is
  needed. Two new actions for `invokeAction`: `getDnsQueries` returns a JSON
  array, newest first; `clearDnsQueries` empties it. Not recorded: the
  `dns.listen` server (it keeps its own service reference) and lookups the core
  makes for itself (proxy server names, rule matching). JNI surface and
  `bridgeABI` (`3`) unchanged.

### Changed
- Build toolchain: Kotlin `2.2.10` → `2.4.20`, AGP `8.12.2` → `8.13.2`. The
  facade compiles with language and API version 2.2 (`mv=[2,2,0]` in class
  metadata), so hosts on Kotlin 2.2 keep reading it. Kotlin 2.4 warns that
  Gradle 8.x is deprecated; moving to Gradle 9 needs AGP 9 and is left for a
  separate change.
- CI: `actions/setup-java` `v4` → `v6.0.1`, `gradle/actions/setup-gradle`
  `v4.3.1` → `v5.0.2`, both pinned by commit SHA. `setup-gradle` v6 is not
  taken: its caching moved to a proprietary component under separate terms.
- CI runs `go test -race` on the `dnsquery` package.

## [0.3.6] — 2026-10-04

### Changed
- Bumped bundled [metacubex/mihomo](https://github.com/MetaCubeX/mihomo)
  `v1.19.31` → `v1.19.32` (16 upstream commits). No CVE fixes this time. The
  client-path fixes: a nil dereference when the context is cancelled while an
  h2 connection is being set up, half-close in sing-mux, the effective MSS now
  accounts for TCP options, and a race in anytls idle-session cleanup. The
  converter passes xhttp `extra.headers` through. Additive: `hash-key` for
  load-balance groups and a `congestion-controller` option for tun. Upstream
  changed the default tun stack to `mips`; `startTUN` always takes the stack
  from its `stack` argument, so callers keep the stack they pass. Dependencies
  that come along: `sing-tun` `0.4.24` → `0.4.27`, `sing` `0.5.7` → `0.5.8`,
  `sing-mux` `0.3.10` → `0.3.12`, `utls` `1.8.7` → `1.8.8`, `mieru` `3.37.0` →
  `3.38.0`, plus `gvisor`, `mipstack`, `metacubex/http`, `metacubex/cpu` and
  `netipx`. `golang.org/x/*` stay at the 0.3.5 versions; a source-mode
  `govulncheck` of the android/arm64 build reports nothing reachable.
  JNI/facade surface and `bridgeABI` (`3`) unchanged; all three ABIs rebuild
  clean and export the expected 11 symbols.

## [0.3.5] — 2026-09-27

Same mihomo `v1.19.31` as 0.3.4; toolchain, dependency and packaging fixes.

### Security
- `golang.org/x/net` `0.35.0` → `0.59.0` and `golang.org/x/text` `0.22.0` →
  `0.42.0` (dragging `x/crypto` to `0.57.0` and `x/sys`, `x/sync`, `x/term`
  along). A source-mode `govulncheck` of the android/arm64 build reached four
  advisories through our call graph: GO-2026-5026, GO-2026-4918 (HTTP/2
  transport infinite loop on a bad `SETTINGS_MAX_FRAME_SIZE`), GO-2025-3503
  (proxy bypass via IPv6 zone IDs) and GO-2026-5970 (`x/text` infinite loop).
  It now reports none reachable. These modules sat at mihomo's go1.20-era
  pins in every earlier release.

### Changed
- Built with Go `1.27` (was `1.25`, which left support when 1.27 shipped).
  `go.mod` now declares `go 1.26.0`, the floor the new `golang.org/x/*`
  require, and pins `godebug default=go1.20` so runtime GODEBUG defaults
  (TLS, x509, net/http) stay at the level upstream mihomo builds with.
- The EasyTier outbound is compiled out (`no_easytier`), same as Tailscale
  (`no_tailscale`): 0.3.4 grew `libclash.so` by ~11 MB per ABI for a
  mesh-VPN outbound nothing on our side uses, most of it an embedded
  6.9 MB WebAssembly core plus the wazero runtime. An `easytier` proxy in a
  config is now rejected with a "disabled by build tag" error, as a
  `tailscale` one already was.
- `build-native.sh` defaults to the release tag set
  (`with_gvisor,cmfa,no_tailscale,no_easytier`), so CI and local builds
  compile what ships. Until now only the release workflow dropped Tailscale.

### Fixed
- The CI `go-vuln` job scanned nothing. Every Go file is gated by
  `//go:build android && cgo`, and the job ran on the host GOOS, so
  `govulncheck ./...` loaded zero packages and passed vacuously. It now scans
  the android/arm64 build with the release tags, with `govulncheck` pinned
  to `v1.8.0` so an upstream release cannot again demand a newer toolchain
  mid-cycle. golangci-lint `v2.7.2` → `v2.14.0` for the new `go` line.

## [0.3.4] — 2026-09-27

### Changed
- Bumped bundled [metacubex/mihomo](https://github.com/MetaCubeX/mihomo)
  `v1.19.30` → `v1.19.31` (51 upstream commits). No CVE fixes this time; the
  client-path fixes are hysteria v1 UDP handling restored and hysteria2 UDP
  sessions closed with their connection, a VLESS decryption cleanup panic, a
  nil dereference in WireGuard init, split-DNS over a tailnet peer failing
  silently in tsnet mode, `DomainSet` wildcard matching with overlapping
  rules, IPv6 URLs in xhttp, and the OpenVPN tls-auth HMAC digest derived from
  `auth` instead of hard-coded SHA-1. A sweep of "close connection after error
  handling" fixes across doq, mkcp, snell, kcptun, tuic and sudoku, plus
  lower gVisor/mipstack memory use. Additive: EasyTier outbound, `stack: mips`
  for tun (reachable through the existing `stack` string of `startTUN`),
  ZeroTier `identity-secret`. New indirect deps `easytier-go` and
  `metacubex/wazero` ride in with EasyTier; the YAML library moves to
  `go.yaml.in/yaml/v3`; `sing-tun` `0.4.22` → `0.4.24`, `mieru` `3.35.0` →
  `3.37.0`, `tailscale` to v1.102.3, `gvisor`, `sing-quic`,
  `sing-shadowsocks{,2}` and `amneziawg-go` come along. JNI/facade surface and
  `bridgeABI` (`3`) unchanged; all three ABIs rebuild clean and export the
  expected 11 symbols.

### Fixed
- A core crash no longer erases its own traceback. `captureStdFd` redirects fd
  2 into a pipe drained by a goroutine in the same process, so a fatal signal
  killed the reader before the runtime's traceback could be forwarded — every
  segfault reached logcat as nothing but the Zygote's `signal 11` line, with no
  tombstone (the Go runtime handles the signal itself, so debuggerd never sees
  it). `debug.SetCrashOutput` now points the runtime at `crash.log`, written
  next to the file-log sink and therefore in the consumer's external files dir,
  where it survives the process and can be pulled with `adb`.

## [0.3.3] — 2026-08-17

### Changed
- Bumped bundled [metacubex/mihomo](https://github.com/MetaCubeX/mihomo)
  `v1.19.29` → `v1.19.30` (83 upstream commits). Security-relevant:
  **CVE-2026-56862** in `crypto/tls` (via `metacubex/tls` `0.1.7` → `0.1.8`).
  Two fixes land directly on our client path — the sing_tun UDP DNS-hijack
  path sent zero-filled or stale packets whenever a reply's uncompressed size
  exceeded `SafeDnsPacketSize` (`PackBuffer` reallocating out from under the
  send buffer), and Tailscale no longer fails to recover after the network
  comes back. The rest is DNS correctness (echo EDNS0 opt, truncate UDP
  replies to the client's advertised buffer, DNS initialized before NTP), a
  large sniffer rework (H2C/QUICv2 sniffing, cross-record ClientHello
  assembly, coalesced QUIC packets, no more waits on partial reads,
  connections stay open after a failed sniff), and additive outbounds
  (ZeroTier, AmneziaWG 3.0/3.1, anytls `client-metadata`, hysteria2
  `handshake-timeout`, an `ip-stack` option for wireguard/masque/openvpn).
  New indirect deps `metacubex/mipstack` and `metacubex/zerotier-go` ride in
  with ZeroTier; `quic-go` `0.59` → `0.61`, `sing-tun` `0.4.21` → `0.4.22`,
  `tailscale` to v1.102.2, `gvisor`, `mieru` `3.35.0`, `restls-client-go`
  `0.1.9` and `bart` `0.29.0` come along. JNI/facade surface and `bridgeABI`
  (`3`) unchanged; all three ABIs rebuild clean and export the expected 11
  symbols.
- CI and the release pipeline now pin Go `1.25` (was `1.23`); `govulncheck`
  requires it since `x/vuln` v1.6.0. `go.mod` still declares `go 1.20`.

## [0.3.2] — 2026-07-19

### Changed
- Bumped bundled [metacubex/mihomo](https://github.com/MetaCubeX/mihomo)
  `v1.19.28` → `v1.19.29`. Upstream is predominantly additive protocol work
  (JLS and restls for vmess/vless/trojan/shadowsocks/anytls/snell, shadowquic
  outbound and listener, OpenVPN TLS rekey and `tls-crypt-v2`, anytls 0.0.13),
  which pulls in two new indirect deps — `metacubex/jls-tls` and
  `metacubex/jls-quic-go` — and drops `metacubex/sing-shadowtls` after upstream
  reimplemented shadowtls in-tree. `metacubex/tailscale` rides along to
  `20260711142031`. JNI/facade surface and `bridgeABI` (`3`) unchanged; all
  three ABIs rebuild clean and export the expected 11 symbols.

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
