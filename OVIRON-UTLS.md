# oviron-utls release line

Branch `oviron-utls` builds this library against
[oviron/mihomo](https://github.com/oviron/mihomo) and
[oviron/utls](https://github.com/oviron/utls) in place of `metacubex/mihomo`
and `metacubex/utls`: the same mihomo release plus the `chrome155`
fingerprint name, and the MetaCubeX `v1.9.0-mod-meta` branch plus Firefox 158
and Chrome 155 parrots. The branch sits on a main-line
release tag, and its commits change only two `replace` lines in
`src/main/jni/core/go.mod`, `go.sum` and this file. A
replace in a dependency does not reach this module, so the uTLS replace is
repeated here rather than inherited from oviron/mihomo.

Everything else is the main-line release underneath: the bundled mihomo core,
the JNI/facade surface and `bridgeABI`. The main line (`main`, plain `vX.Y.Z`
tags, `CHANGELOG.md`, the version matrix in `README.md`) does not change; this
file is the changelog of the line.

## Releases

| Tag | Main-line base | mihomo core | uTLS | bridgeABI |
|---|---|---|---|---|
| `v0.3.14-oviron.2` | `v0.3.14` | `oviron/mihomo` `v1.19.34-oviron.2` | `oviron/utls` `v1.9.0-oviron.3` | 3 |
| `v0.3.14-oviron.1` | `v0.3.14` | `v1.19.34` | `oviron/utls` `v1.9.0-oviron.2` | 3 |
| `v0.3.13-oviron.1` | `v0.3.13` | `v1.19.33` | `oviron/utls` `v1.9.0-oviron.1` | 3 |
| `v0.3.12-oviron.1` | `v0.3.12` | `v1.19.33` | `oviron/utls` `v1.9.0-oviron.1` | 3 |

## What differs from the main line

As of `v0.3.14-oviron.2`, compared with `v0.3.14` (`metacubex/utls`
`v1.9.0-alpha.1`):

### Added
- `client-fingerprint: chrome155` sends the ClientHello of Chrome 155 on macOS
  with an X25519MLKEM768 key share, as captured by refraction-networking/utls;
  the default fingerprint of Xray-core v26.11.9. `chrome` stays Chrome 133.

### Changed
- `client-fingerprint: firefox` sends the ClientHello of current Firefox stable
  (Firefox 158) in normal browsing mode; the main line sends Firefox 148. Both
  carry an X25519MLKEM768 key share. Firefox 158 also sends a 240-byte GREASE
  ECH payload, as Firefox 155+ does. Over REALITY, mihomo sends the
  post-quantum key share only with `reality-opts.support-x25519mlkem768:
  true`, as on the main line.
- `chrome` (Chrome 133), `safari` (Safari 26.3) and `firefox120` send the same
  ClientHello as on the main line. `random`, which picks chrome, safari, ios
  or firefox once per process, picks up the newer `firefox`.

### Fixed
- Fingerprints that offer X25519 and P-256 key shares, such as `firefox`,
  `firefox120`, finish the handshake when the server picks P-256 instead of
  failing (refraction-networking/utls#433).

## Tags and releases

A line release is tagged `v<main-line version>-oviron.<n>`. `<n>` counts line
releases on the same base and starts again at 1 on a new one.

`release.yml` is the main-line workflow, unchanged. The hyphen in the tag makes
it publish a pre-release, so GitHub never marks a line release Latest and
`releases/latest` keeps pointing at the main line. The assets are named after
the tag as usual (`libmihomo-android-v0.3.13-oviron.1.aar` with its `.sha256`,
`.asc` and `.metadata.json`) and are signed with the same key. Under SemVer the
suffix ranks below the base version, so a tool that follows SemVer never moves
a consumer from `v0.3.13` to `v0.3.13-oviron.1`: to use the line, pin the exact
tag.

The workflow fills the release body with generated notes. Replace them with the
"What differs from the main line" section above:

```sh
gh release edit vX.Y.Z-oviron.N --notes-file notes.md
```

## Following a new main-line release

The line stays one commit. Move it onto the new tag, rebuild, and tag
`-oviron.1`:

```sh
git fetch origin --tags
git switch oviron-utls
git rebase --onto vNEW vOLD
(cd src/main/jni/core && go mod tidy)
```

If `go.sum` conflicts, take the main-line file
(`git checkout vNEW -- src/main/jni/core/go.sum`), run `go mod tidy` to put the
oviron/utls lines back, then `git add` and `git rebase --continue`.

Then, as for a main-line release (`MAINTENANCE.md`): build all three ABIs, check
the 11 JNI exports, run the host tests. Also check that the built library
carries the fork:

```sh
go version -m src/main/jniLibs/arm64-v8a/libclash.so | grep utls
# Expected: => github.com/oviron/utls v1.9.0-oviron.3, and => github.com/oviron/mihomo
```

Update the table and the comparison above, amend the commit, and publish.
`ci.yml` runs on `main` and on pull requests only, so start it on the branch by
hand and tag once it is green:

```sh
git commit --amend
git push --force-with-lease origin oviron-utls
gh workflow run ci.yml --ref oviron-utls
git tag vNEW-oviron.1
git push origin vNEW-oviron.1
```

Earlier tags keep the commits they point at. A new oviron/utls or oviron/mihomo
release on the same base is a new commit on the branch: change the `replace`,
`go mod tidy`, rebuild, commit, push, and tag the next `-oviron.<n>` once CI is
green.

When the main line moves to a `metacubex/utls` with a Firefox 158 parrot of its
own, the line is no longer needed.
