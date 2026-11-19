# The line `oviron-compat`

Branch `oviron-compat` builds this library on the layer `compat` of
[oviron/mihomo](https://github.com/oviron/mihomo) in place of
`metacubex/mihomo`: the line for daily use. The other line, `oviron-full`, does the same
with the layer `full`.

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
| `v0.3.14-compat.1` | `v0.3.14` | `oviron/mihomo` `v1.19.34-compat.1` | `oviron/utls` `v1.9.0-oviron.3` | 3 |

Releases before the two lines were tagged `v<main line>-oviron.<n>` on one
branch: `v0.3.14-oviron.4` carried the layer `compat`, `v0.3.14-oviron.3` the
layer `full`, and the ones before them only the fingerprints.

## What differs from the main line

As of `v0.3.14-compat.1`, compared with `v0.3.14` (`metacubex/mihomo`
`v1.19.34`, `metacubex/utls` `v1.9.0-alpha.1`):

### Added
- `client-fingerprint: chrome155` sends the ClientHello of Chrome 155 on macOS
  with an X25519MLKEM768 key share, as captured by refraction-networking/utls;
  the default fingerprint of Xray-core v26.11.9. `chrome` stays Chrome 133.
- `testseed` for VLESS with `flow: xtls-rprx-vision` pads the way Xray-core
  does: without it blocks of 900 bytes and more, such as a post-quantum inner
  ClientHello, go out unpadded. The fix is off until the key is set, for
  example `testseed: [900, 500, 900, 256]`.
- `reality-opts.client-version` and `grpc-opts.authority`.

### Changed
- `client-fingerprint: firefox` sends the ClientHello of current Firefox stable
  (Firefox 158) in normal browsing mode; the main line sends Firefox 148. Both
  carry an X25519MLKEM768 key share. Over REALITY, mihomo sends the
  post-quantum key share only with `reality-opts.support-x25519mlkem768:
  true`, as on the main line.
- `random` over REALITY with the post-quantum key share no longer picks `ios`,
  which has none.
- XHTTP reads the defaults of its padding obfuscation keys and a `?query` in
  the path the way Xray-core does; `?ed=N` in an HTTPUpgrade path turns on
  fast open.

### Fixed
- Fingerprints that offer X25519 and P-256 key shares, such as `firefox` and
  `firefox120`, finish the handshake when the server picks P-256 instead of
  failing (refraction-networking/utls#433).
- The SOCKS5 outbound fails a request the server rejected instead of going on
  with a garbage address. VLESS `encryption` bounds a record by 16640 bytes
  like Xray-core, not by 17000.

The details are in
[FORK.md](https://github.com/oviron/mihomo/blob/compat/FORK.md) and
[FORK-COMPAT.md](https://github.com/oviron/mihomo/blob/compat/FORK-COMPAT.md)
of oviron/mihomo.

## Tags and releases

A line release is tagged `v<main-line version>-compat.<n>`. `<n>` counts the
releases of the line on the same main-line base and starts again at 1 on a new
one.

`release.yml` is the main-line workflow with one line changed: the core version
in `metadata.json` is the tag of oviron/mihomo, not the upstream version it
replaces, so that a consumer can tell the builds apart. The hyphen in the tag
makes the workflow publish a pre-release, so GitHub never marks a line release
Latest and `releases/latest` keeps pointing at the main line. The assets are
named after the tag as usual (`libmihomo-android-v0.3.14-compat.1.aar` with
its `.sha256`, `.asc` and `.metadata.json`) and are signed with the same key.
Under SemVer the suffix ranks below the base version, so a tool that follows
SemVer never moves a consumer from `v0.3.14` to a line release: to use a line,
pin the exact tag.

The workflow fills the release body with generated notes. Replace them with the
"What differs from the main line" section above:

```sh
gh release edit vX.Y.Z-compat.N --notes-file notes.md
```

## A new release of the line

The line stays one commit. For a new main-line release move it onto the new
tag; for a new oviron/mihomo or oviron/utls release change the `replace`:

```sh
git fetch origin --tags
git switch oviron-compat
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
# Expected: => github.com/oviron/mihomo v1.19.34-compat.1 and => github.com/oviron/utls v1.9.0-oviron.3
```

Update the table and the comparison above, amend the commit, and publish.
`ci.yml` runs on `main` and on pull requests only, so start it on the branch by
hand and tag once it is green:

```sh
git commit --amend
git push --force-with-lease origin oviron-compat
gh workflow run ci.yml --ref oviron-compat
git tag vX.Y.Z-compat.N
git push origin vX.Y.Z-compat.N
```

Earlier tags keep the commits they point at.
