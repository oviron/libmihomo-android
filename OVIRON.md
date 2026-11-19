# The line `oviron-full`

Branch `oviron-full` builds this library on the layer `full` of
[oviron/mihomo](https://github.com/oviron/mihomo) in place of
`metacubex/mihomo`: the line for a server that has one of those features turned on. The other line, `oviron-compat`, does the same
with the layer `compat`.

| Line | mihomo | Adds to the main line |
|---|---|---|
| `oviron-compat` | layer `compat` | the TLS fingerprints of [oviron/utls](https://github.com/oviron/utls) and fixes of compatibility with Xray-core servers |
| `oviron-full` | layer `full` | that, and the client side of everything else an Xray-core 26.11.9 server offers |

A line is a main-line release tag plus one commit: two `replace` lines in
`src/main/jni/core/go.mod`, `go.sum`, one line of `release.yml` and this file.
A replace in a dependency does not reach this module, so the uTLS replace is
repeated here rather than inherited from oviron/mihomo. Everything else is the
main-line release underneath: the JNI/facade surface and `bridgeABI`. The main
line (`main`, plain `vX.Y.Z` tags, `CHANGELOG.md`, the version matrix in
`README.md`) does not change; this file is the changelog of the line.

## Releases

| Tag | Main-line base | mihomo core | uTLS | bridgeABI |
|---|---|---|---|---|
| `v0.3.14-full.1` | `v0.3.14` | `oviron/mihomo` `v1.19.34-full.1` | `oviron/utls` `v1.9.0-oviron.3` | 3 |

Releases before the two lines were tagged `v<main line>-oviron.<n>` on one
branch: `v0.3.14-oviron.4` carried the layer `compat`, `v0.3.14-oviron.3` the
layer `full`, and the ones before them only the fingerprints.

## What differs from the main line

As of `v0.3.14-full.1`, compared with `v0.3.14` (`metacubex/mihomo`
`v1.19.34`, `metacubex/utls` `v1.9.0-alpha.1`):

Everything the line `oviron-compat` differs by (its `OVIRON.md`), and:

### Added
- The client side of what an Xray-core 26.11.9 server offers, new proxy keys
  in the YAML config: `finalmask` (`fragment`, `noise`, `sudoku`, `xmc`,
  `header-custom`, `salamander`, `udphop`, `xdns`, `xicmp`, `realm`,
  `mkcp-legacy`, `quic-params`), `mux` (Mux.Cool), MASQUE over HTTP/3 and
  HTTP/2, `reverse` for VLESS, `network: xdrive`, `network: kcp` and
  `network: hysteria` for VLESS and Trojan, XHTTP for VMess and Trojan,
  `stream-settings` for Shadowsocks, SOCKS5 and HTTP, `sockopt`,
  `reality-opts.mldsa65-verify`. Every key is in
  [FORK-FULL.md](https://github.com/oviron/mihomo/blob/full/FORK-FULL.md) of
  oviron/mihomo.
- On Android `xicmp` has only its `dgram` mode (no root); it was run as an
  ordinary user on an Android 15 emulator, not inside a VPN app on a device.

### Changed
- The core is 2.0 MB larger per ABI than on `oviron-compat` (arm64-v8a: 45.8 MB
  against 43.8).

## Tags and releases

A line release is tagged `v<main-line version>-full.<n>`. `<n>` counts the
releases of the line on the same main-line base and starts again at 1 on a new
one.

`release.yml` is the main-line workflow with one line changed: the core version
in `metadata.json` is the tag of oviron/mihomo, not the upstream version it
replaces, so that a consumer can tell the builds apart. The hyphen in the tag
makes the workflow publish a pre-release, so GitHub never marks a line release
Latest and `releases/latest` keeps pointing at the main line. The assets are
named after the tag as usual (`libmihomo-android-v0.3.14-full.1.aar` with
its `.sha256`, `.asc` and `.metadata.json`) and are signed with the same key.
Under SemVer the suffix ranks below the base version, so a tool that follows
SemVer never moves a consumer from `v0.3.14` to a line release: to use a line,
pin the exact tag.

The workflow fills the release body with generated notes. Replace them with the
"What differs from the main line" section above:

```sh
gh release edit vX.Y.Z-full.N --notes-file notes.md
```

## A new release of the line

The line stays one commit. For a new main-line release move it onto the new
tag; for a new oviron/mihomo or oviron/utls release change the `replace`:

```sh
git fetch origin --tags
git switch oviron-full
git rebase --onto vNEW vOLD        # only for a new main-line release
(cd src/main/jni/core && go mod tidy)
```

If `go.sum` conflicts, take the main-line file
(`git checkout vNEW -- src/main/jni/core/go.sum`), run `go mod tidy` to put the
oviron lines back, then `git add` and `git rebase --continue`.

Then, as for a main-line release (`MAINTENANCE.md`): build all three ABIs, check
the 11 JNI exports, run the host tests. Also check that the built library
carries the fork:

```sh
go version -m src/main/jniLibs/arm64-v8a/libclash.so | grep oviron
# Expected: => github.com/oviron/mihomo v1.19.34-full.1 and => github.com/oviron/utls v1.9.0-oviron.3
```

Update the table and the comparison above, amend the commit, and publish.
`ci.yml` runs on `main` and on pull requests only, so start it on the branch by
hand and tag once it is green:

```sh
git commit --amend
git push --force-with-lease origin oviron-full
gh workflow run ci.yml --ref oviron-full
git tag vX.Y.Z-full.N
git push origin vX.Y.Z-full.N
```

Earlier tags keep the commits they point at.
