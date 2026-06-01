# libmihomo-android

Standalone Android library that embeds **mihomo** ([metacubex/mihomo](https://github.com/MetaCubeX/mihomo)), the maintained Clash-Meta successor, as `libclash.so` per ABI plus a Kotlin JNI facade. Drop the `.aar` into any Android app that needs a configurable proxy core (VPN clients, network testing, traffic shaping) without forking ClashMetaForAndroid or maintaining your own cgo build pipeline.

The Kotlin facade exposes a small, typed API around mihomo's Go `//export` boundary plus lambda-friendly overloads for Kotlin code. It does not bake in proxy presets, routing strategies, or any opinionated defaults; those are the consumer's responsibility.

**Why this repo exists.** ClashMetaForAndroid uses an in-tree cgo bridge that only its APK consumes. KaringX's `libclash-vpn-service` is closed-source ([issue #350](https://github.com/KaringX/karing/issues/350)). [AndroidLibV2rayLite](https://github.com/2dust/AndroidLibV2rayLite) is the only comparable OSS prior art and it wraps v2ray, not mihomo. So this is, to our knowledge, the first publicly maintained Android JNI library for mihomo. It started as the bridge inside [oviron/FlClash](https://github.com/oviron/FlClash) and was extracted so other clients can use the same `.aar`.

## What's in each release

Each `v*` tag attaches four files to its GitHub Release:

| File | Description |
|---|---|
| `libmihomo-android-vX.Y.Z.aar` | Android library, all three ABIs (arm64-v8a + armeabi-v7a + x86_64) plus the Kotlin facade |
| `libmihomo-android-vX.Y.Z.aar.sha256` | SHA-256 checksum |
| `libmihomo-android-vX.Y.Z.aar.asc` | GPG detached signature |
| `libmihomo-android-vX.Y.Z.metadata.json` | Machine-readable manifest: bundled mihomo version, `bridgeABI`, ABIs, AAR SHA-256 |

The bundled core version is also in the release title (`vX.Y.Z (mihomo vA.B.C)`) and the matrix below.

Verify before consuming:

```sh
sha256sum -c libmihomo-android-vX.Y.Z.aar.sha256

# One-time: import maintainer public key from this repo
gpg --import oviron-signing.pub.asc
gpg --verify libmihomo-android-vX.Y.Z.aar.asc libmihomo-android-vX.Y.Z.aar
# Expected: Good signature from "oviron <awdonkin@gmail.com>"
```

Public key fingerprint: `1139 C91B 6525 883E 6783 DCF0 4A94 DA48 8A4C 5033`. Cross-check against the maintainer's GitHub profile (https://github.com/oviron) or `keys.openpgp.org` before trusting it.

## Version matrix

Which mihomo core each wrapper release bundles. The wrapper version is an independent SemVer; the bundled core version is surfaced in the release title, this table, and `metadata.json`.

| Wrapper | mihomo core | bridgeABI |
|---|---|---|
| `v0.1.4` | `v1.19.26` | 1 |
| `v0.1.3` | `v1.19.25` | 1 |
| `v0.1.2` | `v1.19.24` | 1 |

## What's inside the `.aar`

Two native libraries per ABI plus a small Kotlin facade:

- `libclash.so`: mihomo Go binary built with `-buildmode=c-shared` (~40 MB per ABI). Exports the 11 `//export` Go functions of the bridge.
- `libmihomo-jni.so`: C++ shim (~14 KB per ABI) translating each `Clash` `external fun` into the matching Go export. Holds `JNI_OnLoad` that wires up the bridge callback function pointers.
- `classes.jar`: `Clash` (singleton facade), `TunInterface` (VPN protect + process resolve hooks), `InvokeInterface` (async result callback).

A consumer calls Kotlin methods on `Clash`; the library handles JNI marshalling, GlobalRef lifetime, and callback dispatch.

## Requirements

- **minSdk 21** (Android 5.0+). The `.so` files are built with `-DANDROID_PLATFORM=android-21`; older devices fail at load time.
- **AGP 8.5.1+** in the host APK so it is itself 16 KB page-aligned for Android 15+. The `.so` files in this library are built 16 KB-aligned by NDK 28.
- **ABIs**: arm64-v8a, armeabi-v7a, x86_64. No 32-bit x86 (`x86`) build is shipped; it's irrelevant for current Android devices.

## Integration

### Gradle (file dependency)

The `.aar` is published on GitHub Releases. Pin a specific version, download it during build, reference it as a file dependency:

```kotlin
// app/build.gradle.kts
val mihomoVersion = "0.1.0"
val mihomoAar = layout.buildDirectory.file("libs/libmihomo-android-v$mihomoVersion.aar")

val downloadMihomo = tasks.register("downloadMihomo") {
    inputs.property("mihomoVersion", mihomoVersion)
    outputs.file(mihomoAar)
    doLast {
        val target = mihomoAar.get().asFile
        target.parentFile.mkdirs()
        val url = "https://github.com/oviron/libmihomo-android/releases/download/v$mihomoVersion/libmihomo-android-v$mihomoVersion.aar"
        target.outputStream().use { out ->
            uri(url).toURL().openStream().use { it.copyTo(out) }
        }
    }
}

dependencies {
    implementation(files(mihomoAar).builtBy(downloadMihomo))
}
```

A safer variant verifies SHA-256 + GPG before trusting the download; see `MAINTENANCE.md` § "Verifying releases at build time".

### Usage from Kotlin

`Clash` does not load its native libraries automatically. The consumer chooses *when* and *from where* to load, which is the seam used for runtime-pluggable versions (download a newer `.aar`, extract its `jni/<abi>/` dir, and pass that path).

```kotlin
import io.github.oviron.libmihomo.Clash
import io.github.oviron.libmihomo.InvokeInterface
import io.github.oviron.libmihomo.TunInterface

// 1. Load once per process. Default location: the directory Android put the
//    .aar's jniLibs into when it built your APK.
Clash.load(context.applicationInfo.nativeLibraryDir)

// 2. Optional sanity check.
require(Clash.bridgeABI() == Clash.EXPECTED_BRIDGE_ABI) {
    "Loaded libclash.so reports ABI ${Clash.bridgeABI()}, expected ${Clash.EXPECTED_BRIDGE_ABI}"
}

// 3. Initialize mihomo and apply a profile in one call.
Clash.quickSetup(
    initParams = """{"homeDir": "${context.filesDir}"}""",
    setupParams = """{"profile": "$profilePath"}""",
) { result ->
    if (!result.isNullOrEmpty()) Log.e(TAG, "quickSetup failed: $result")
}

// 4. VPN-side hook: protect outbound sockets, name processes for connections.
//    Implement directly on your VpnService instance (it already has protect()).
class MyVpnService : android.net.VpnService(), TunInterface {
    override fun protect(fd: Int) { super.protect(fd) }
    override fun resolverProcess(
        protocol: Int, source: String, target: String, uid: Int,
    ): String = "" // your process resolver; "" means unknown
}

// 5. Start the TUN listener on the fd from VpnService.Builder.establish().
Clash.startTUN(
    fd = tunFd,
    cb = myVpnService,            // TunInterface
    device = "my-vpn-app",        // shown in mihomo logs / metrics
    stack = "system",             // "system" | "gvisor" | "mixed"
    address = "172.19.0.1/30",    // CIDR list, comma-separated
    dns = "1.1.1.1,1.0.0.1",      // hijacked at :53 inside the TUN
)

// 6. Push events from mihomo (connection snapshots, log lines):
Clash.setEventListener { event -> /* JSON message; see action.go for shape */ }

// 7. Stop and clean up before the VpnService dies:
Clash.stopTun()
```

Lambda overloads exist for `invokeAction`, `quickSetup`, `setEventListener`, and `startTUN`; pass `InvokeInterface`/`TunInterface` directly if you need named classes.

### Threading

- Every `Clash.*` call returns immediately; the underlying Go work runs on goroutines.
- Callbacks (`TunInterface.protect`, `TunInterface.resolverProcess`, `InvokeInterface.onResult`) are invoked on Go worker threads attached to the JVM by the library. Treat them as background; do not block them with synchronous I/O or long computations.
- `Clash.load` is the only call that must run before any other facade method. `Clash.assertReady()` is invoked by every facade method and throws `IllegalStateException` if `load` was not called (or failed).

### End-to-end usage

A minimal lifecycle:

1. Build a `VpnService.Builder`, call `.establish()`, get the int fd.
2. Pass the fd to `Clash.startTUN(fd, tunInterface, device, stack, address, dns)`.
3. Register a callback via `Clash.setEventListener` to receive push events from mihomo.
4. Drive runtime changes through `Clash.invokeAction(jsonAction, callback)`; see `src/main/jni/core/action.go` for the full action vocabulary.
5. On shutdown: `Clash.stopTun()` first, then close the VpnService.

A full reference consumer is [oviron/FlClash](https://github.com/oviron/FlClash); the bridge originally lived there before this extraction.

## API reference

| Method | Effect |
|---|---|
| `load(nativeLibDir)` | Loads `libclash.so` + `libmihomo-jni.so` from the given directory. Must be called once before any other method. |
| `isLoaded(): Boolean` | True once `load` succeeded. |
| `assertReady()` | Throws `IllegalStateException` if `load` failed or was never called. |
| `bridgeABI(): Int` | API surface version. Compare against `EXPECTED_BRIDGE_ABI`. |
| `invokeAction(data, cb)` | Dispatches a JSON action document. Async result via `cb.onResult` or lambda. |
| `quickSetup(initJson, setupJson, cb)` | One-shot mihomo init + profile apply. Result string (empty on success) via `cb`. |
| `startTUN(fd, cb, device, stack, address, dns)` | Starts the TUN listener bound to `fd`. `cb` is a `TunInterface`. |
| `stopTun()` | Stops the TUN listener. Idempotent. |
| `setEventListener(cb)` | Registers a push-event sink. Pass `null` to clear. |
| `getTraffic(): String` | Current bytes/sec snapshot (JSON). |
| `getTotalTraffic(): String` | Total bytes since process start (JSON). |
| `suspended(suspended: Boolean)` | Releases idle resources (`true`) or wakes the listener (`false`). |
| `forceGC()` | Hints the Go runtime to GC. Diagnostic. |
| `updateDNS(servers)` | Updates the system DNS fallback list (comma-separated IPs). |

The exact JSON shape of `invokeAction` payloads is defined by `src/main/jni/core/action.go`; read it for the full method vocabulary (~30 methods covering proxies, connections, providers, geo-data, probes, etc.).

## Compatibility note

While the library version is below 1.0, treat the **whole library as API-unstable**: bundled mihomo version, JSON action vocabulary, and Kotlin facade may change between minor releases. **Pin an exact version in your build** (`v0.1.0`, not `v0.+`) and re-read the CHANGELOG before bumping.

Semver guarantees begin at v1.0. The `bridgeABI()` integer is bumped on each breaking change at the native boundary so a consumer can refuse to start when the stub and `.so` disagree.

## License

This repository is licensed under **GPL-3.0**; see `LICENSE`. Upstream mihomo is GPL-3.0, statically linked into `libclash.so`. Per the FSF FAQ ("JNI counts as combined work"), downstream apps that ship `libclash.so` are also obligated under GPL-3.0.

If you are looking to drop a DPI-bypass proxy into a proprietary app without GPL obligations, see the sibling project [oviron/libbyedpi-android](https://github.com/oviron/libbyedpi-android); that one wraps byedpi (MIT) instead and ships under MIT.

## Building from source

```sh
git clone https://github.com/oviron/libmihomo-android.git
cd libmihomo-android

# Set NDK path (NDK 28+):
export ANDROID_NDK=$HOME/Library/Android/sdk/ndk/28.0.13004108

# Build libclash.so for all three ABIs:
./build-native.sh

# Package the .aar (this also builds libmihomo-jni.so via CMake):
./gradlew :assembleRelease
# Output: build/outputs/aar/libmihomo-android-release.aar
```

See `MAINTENANCE.md` for the full release procedure, GPG key rotation, and upstream-bump workflow.

## Reporting issues

GitHub Issues for bugs and feature requests. For security disclosures see `SECURITY.md`.
