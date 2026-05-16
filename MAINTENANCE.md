# Maintenance procedure

Bus-factor mitigation document. If the primary maintainer (oviron) goes silent
for 60+ days, fork this repo and continue using these instructions. Everything
needed is in the repo — no out-of-band secrets except the GPG private key,
which any successor can rotate.

## Build environment

- **Go**: 1.20+ (release pipeline uses the latest stable Go available at run time)
- **Android Gradle Plugin**: 8.12.2 (pinned in `build.gradle.kts`)
- **Kotlin**: 2.2.10
- **Gradle**: 8.13 (via wrapper)
- **JDK**: 17+ (Temurin in CI)
- **Android NDK**: 28.0.13004108 (pinned in CI workflow)
- **Android SDK**: compileSdk 36, minSdk 21
- **OS**: ubuntu-latest in CI; locally Linux/macOS both fine

## Source layout

```
build-native.sh              # Driver — invokes `go build -buildmode=c-shared`
                             #   per ABI, drops libclash.so into src/main/jniLibs/<abi>/
build.gradle.kts             # AGP library project — packages the .aar from
                             #   the pre-built .so files plus the Kotlin facade
settings.gradle.kts
gradle.properties
gradle/, gradlew, gradlew.bat
src/main/AndroidManifest.xml # Empty library manifest
src/main/kotlin/             # Kotlin facade (Clash.kt)
src/main/jni/core/           # Go bridge sources — module
                             #   github.com/oviron/libmihomo-android, depends
                             #   on metacubex/mihomo via go.mod
src/main/jniLibs/<abi>/      # Build artifacts — populated by build-native.sh,
                             #   gitignored
consumer-rules.pro           # R8/proguard rules shipped with the .aar
oviron-signing.pub.asc       # Maintainer GPG public key
```

The Go module path is `github.com/oviron/libmihomo-android`; internal packages
(`platform`, `tun`) live as subdirectories of `src/main/jni/core/`.

## How a release is made

Tags matching `v*` trigger the `release.yml` workflow:

1. Checks out the repo (no submodules — mihomo is pulled via `go.mod`).
2. Installs Go, JDK, NDK (with cache).
3. Runs `./build-native.sh` — produces `libclash.so` for all three ABIs.
4. Runs `./gradlew :assembleRelease` — packages the `.aar` (bundled stub + 3 `.so`).
5. Renames the artifact to `libmihomo-android-<tag>.aar`, generates SHA-256.
6. GPG-signs the `.aar` (detached, armored) with the maintainer key — public key in `oviron-signing.pub.asc`.
7. Creates a GitHub Release with three files attached (`.aar`, `.aar.sha256`, `.aar.asc`).

Cut a release manually:

```sh
# Make sure CI is green on main:
gh run list --limit 3

# Update CHANGELOG.md (mandatory):
$EDITOR CHANGELOG.md

# Tag and push:
git tag v0.X.Y
git push origin v0.X.Y

# Watch CI:
gh run watch
```

## How upstream bumps are handled

When mihomo cuts a new tag:

1. Update `go.mod` and rebuild:

   ```sh
   cd src/main/jni/core
   go get github.com/metacubex/mihomo@vNEW
   go mod tidy
   cd ../../../..
   ANDROID_NDK=$HOME/Library/Android/sdk/ndk/28.0.13004108 ./build-native.sh
   ```

2. If `go build` fails — mihomo broke our bridge. Read the compile error,
   patch `src/main/jni/core/lib.go` (most cases are renamed mihomo internals
   under `dialer`, `tunnel`, `listener/sing_tun`). Bump `bridgeABI` in
   `lib.go` if the JNI surface itself changed.

3. Smoke-test the produced `.so`:

   ```sh
   nm -D src/main/jniLibs/arm64-v8a/libclash.so | grep -E ' T (invokeAction|startTUN|quickSetup|stopTun|setEventListener|getTraffic|getTotalTraffic|suspend|forceGC|updateDns|bridgeABI)$' | wc -l
   # Expected: 11
   ```

4. Commit, update `CHANGELOG.md`, tag, push.

mihomo cuts ≈1.5 tags per month; ~80% are additive (no work), ~15% need only a
`go.mod` bump, ~5% break the bridge and require a one-or-two line patch.

## GPG signing key

The maintainer's GPG key is RSA 4096 `4A94DA488A4C5033`, fingerprint
`1139C91B6525883E6783DCF04A94DA488A4C5033`. Public key committed at repo root
as `oviron-signing.pub.asc` (same key as
[oviron/libbyedpi-android](https://github.com/oviron/libbyedpi-android) — one
trust anchor for both libraries).

CI uses the key via GitHub Actions secrets `GPG_PRIVATE_KEY` (armored secret
key) and `GPG_PASSPHRASE`. To rotate:

1. Generate new GPG key: `gpg --full-generate-key`.
2. `gh secret set GPG_PRIVATE_KEY --repo oviron/libmihomo-android < secret.asc`
3. `gh secret set GPG_PASSPHRASE --repo oviron/libmihomo-android`
4. Replace `oviron-signing.pub.asc` with the new public key.
5. Announce in the next release notes, ideally with a cross-signature from the
   old key.

After cutting any release, push the public key to a keyserver so consumers can
fetch it independently of the repo:

```sh
gpg --keyserver keys.openpgp.org --send-keys 4A94DA488A4C5033
```

## Verifying releases at build time

Recommended consumer-side pattern (`build.gradle.kts`):

```kotlin
val mihomoVersion = "0.1.0"
val mihomoSha = "<paste from libmihomo-android-vX.Y.Z.aar.sha256>"
val mihomoAar = layout.buildDirectory.file("libs/libmihomo-android-v$mihomoVersion.aar")

val downloadMihomo = tasks.register("downloadMihomo") {
    outputs.file(mihomoAar)
    doLast {
        val target = mihomoAar.get().asFile
        target.parentFile.mkdirs()
        val url = "https://github.com/oviron/libmihomo-android/releases/download/v$mihomoVersion/libmihomo-android-v$mihomoVersion.aar"
        target.outputStream().use { out ->
            uri(url).toURL().openStream().use { it.copyTo(out) }
        }
        val actual = java.security.MessageDigest.getInstance("SHA-256")
            .digest(target.readBytes())
            .joinToString("") { "%02x".format(it) }
        require(actual.equals(mihomoSha, ignoreCase = true)) {
            "libmihomo SHA-256 mismatch: expected=$mihomoSha actual=$actual"
        }
    }
}
```

## Reproducibility

A given source tree + the same pinned AGP/Kotlin/Go/NDK toolchain produces
byte-identical artifacts. Go is compiled with `-ldflags="-w -s"` (no debug
info, no build IDs derived from time). AGP applies zip epoch normalization
to the `.aar`.

Stronger reproducibility (across NDK patch revisions or different host
platforms) is not claimed.

## Mirror

GitHub Releases is currently the only distribution channel. Mirroring to a
non-GitHub host (Codeberg / IPFS / etc.) is **deferred until concrete need**.
If GitHub deletes this repo: consumers downloading via fixed URL lose access;
the build is reproducible from any local clone per the section above. Any
third party can fork to alternative hosting and continue releases.

## Contact

- Primary maintainer: oviron (@oviron on GitHub)
- Security disclosures: see `SECURITY.md`
- Issues / questions: GitHub Issues
