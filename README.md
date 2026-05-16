# libmihomo-android

Standalone Android library that embeds **mihomo** ([metacubex/mihomo](https://github.com/MetaCubeX/mihomo)) — the maintained Clash-Meta successor — as `libclash.so` per ABI, plus a thin Kotlin JNI facade. Drop the `.aar` into any Android app that needs a configurable proxy core (VPN clients, network testing, traffic shaping) without forking ClashMetaForAndroid or maintaining your own cgo build pipeline.

**Status:** v0.1.0 is a low-level JNI binding. The API surface mirrors the cgo `//export` boundary of the bridge 1:1. A higher-level typed Kotlin DSL (Config builder, sealed `Outbound` / `Rule` types) is on the v0.2 roadmap once we see real adopters' needs.

**Why this repo exists.** ClashMetaForAndroid uses an in-tree cgo bridge that only its APK consumes. KaringX's `libclash-vpn-service` is closed-source ([issue #350](https://github.com/KaringX/karing/issues/350)). [AndroidLibV2rayLite](https://github.com/2dust/AndroidLibV2rayLite) is the only comparable OSS prior art and it wraps v2ray, not mihomo. So this is — to our knowledge — the first publicly maintained Android JNI library for mihomo. It started as the bridge inside [oviron/FlClash](https://github.com/oviron/FlClash) and was extracted so other clients can use the same `.aar`.

## What's in each release

Each `v*` tag attaches three files to its GitHub Release:

| File | Description |
|---|---|
| `libmihomo-android-vX.Y.Z.aar` | Android library, all three ABIs (arm64-v8a + armeabi-v7a + x86_64) plus the Kotlin facade |
| `libmihomo-android-vX.Y.Z.aar.sha256` | SHA-256 checksum |
| `libmihomo-android-vX.Y.Z.aar.asc` | GPG detached signature |

Verify before consuming:

```sh
sha256sum -c libmihomo-android-vX.Y.Z.aar.sha256

# One-time: import maintainer public key from this repo
gpg --import oviron-signing.pub.asc
gpg --verify libmihomo-android-vX.Y.Z.aar.asc libmihomo-android-vX.Y.Z.aar
# Expected: Good signature from "oviron <awdonkin@gmail.com>"
```

Public key fingerprint: `1139 C91B 6525 883E 6783 DCF0 4A94 DA48 8A4C 5033`. Cross-check against the maintainer's GitHub profile (https://github.com/oviron) or `keys.openpgp.org` before trusting it.

## Requirements

- **minSdk 21** (Android 5.0+). The `.so` files are built with `-DANDROID_PLATFORM=android-21`; older devices fail at `System.loadLibrary` time.
- **AGP 8.5.1+** in the host APK — needed so it is itself 16 KB page-aligned for Android 15+. The `.so` files in this library are built 16 KB-aligned by NDK 28.
- **ABIs**: arm64-v8a, armeabi-v7a, x86_64. No 32-bit x86 (`x86`) build is shipped — it's irrelevant for current Android devices.

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

A safer variant verifies SHA-256 before trusting the download — see `MAINTENANCE.md` § "Verifying releases at build time".

### Usage from Kotlin

```kotlin
import io.github.oviron.libmihomo.Clash

// Optional but recommended: catch mismatched .so + stub at startup.
require(Clash.bridgeABI() == Clash.EXPECTED_BRIDGE_ABI) {
    "libmihomo-android bridge ABI mismatch: stub=${Clash.EXPECTED_BRIDGE_ABI}, .so=${Clash.bridgeABI()}"
}

// Initialize mihomo and apply a profile in one call.
Clash.quickSetup(
    callback = registerGlobalCallback { result ->
        if (result.isNotEmpty()) Log.e(TAG, "quickSetup failed: $result")
    },
    initParams = """{"homeDir": "${context.filesDir}"}""",
    setupParams = """{"profile": "${profilePath}"}""",
)

// Start the TUN listener on a file descriptor obtained from VpnService.Builder.
Clash.startTUN(
    callback = registerGlobalCallback { /* push-stream of TUN events */ },
    fd = tunFd,
    device = "my-vpn-app",        // shown in mihomo logs / metrics
    stack = "system",              // "system" | "gvisor" | "mixed"
    address = "172.19.0.1/30",     // CIDR list, comma-separated
    dns = "1.1.1.1,1.0.0.1",       // hijacked at :53 inside the TUN
)

// Stop and clean up before the VpnService dies:
Clash.stopTun()
```

### End-to-end usage

A minimal lifecycle:

1. Build a `VpnService.Builder`, call `.establish()`, get the int fd.
2. Pass the fd to `Clash.startTUN(fd, device, stack, address, dns)`.
3. Register a callback via `Clash.setEventListener(handle)` to receive push events from mihomo (connection snapshots, log lines).
4. Drive runtime changes through `Clash.invokeAction(handle, jsonAction)` — see `src/main/jni/core/action.go` for the full action vocabulary.
5. On shutdown: `Clash.stopTun()` first, then close the VpnService.

A full reference consumer is [oviron/FlClash](https://github.com/oviron/FlClash) — the bridge originally lived there before this extraction.

### Callbacks (`Long` opaque handles)

Every JNI entry-point that accepts `callback: Long` expects a JNI `GlobalRef` int that points at a consumer-side object. The library passes the handle back through the C glue when results are ready (in `src/main/jni/core/bridge.{c,h}`). Lifetime is the consumer's responsibility: register the global ref before passing the handle, release it from the same C glue when the result fires.

If you do not have any custom JNI code yet, the simplest pattern is:

```kotlin
// in your existing JNI shim
extern "C" JNIEXPORT void JNICALL
Java_my_app_CallbackBridge_release(JNIEnv* env, jobject /* this */, jlong handle) {
    env->DeleteGlobalRef(reinterpret_cast<jobject>(handle));
}
```

…and call it from the C-side release glue. The FlClash reference consumer above has a working implementation you can copy.

## API reference

| Method | Effect |
|---|---|
| `bridgeABI(): Int` | API surface version. Compare against `EXPECTED_BRIDGE_ABI` at startup. |
| `invokeAction(handle, jsonAction)` | Dispatches a JSON action document (`{"id", "method", "data"}`). Async, result via callback. |
| `quickSetup(handle, initJson, setupJson)` | One-shot mihomo init + profile apply. Returns error string or empty on success. |
| `startTUN(handle, fd, device, stack, address, dns)` | Starts the TUN listener bound to `fd`. `stack` is `"system" \| "gvisor" \| "mixed"`. |
| `stopTun()` | Stops the TUN listener. Idempotent. |
| `setEventListener(handle)` | Registers a push-event sink. Pass `0` to clear. |
| `getTraffic(): String` | Current bytes/sec snapshot (JSON). |
| `getTotalTraffic(): String` | Total bytes since process start (JSON). |
| `suspend()` | Releases idle resources without stopping the TUN. Safe to call often. |
| `forceGC()` | Hints the Go runtime to GC. Diagnostic. |
| `updateDns(servers)` | Updates the system DNS fallback list. |

The exact JSON shape of `invokeAction` payloads is defined by `src/main/jni/core/action.go` — read it for the full method vocabulary (~30 methods covering proxies, connections, providers, geo-data, probes, etc.).

## Compatibility note

While the library version is below 1.0, treat the **whole library as API-unstable**: bundled mihomo version, JSON action vocabulary, and Kotlin facade may change between minor releases. The 11 `external fun` signatures are not expected to change during 0.x but no guarantee is made. **Pin an exact version in your build** (`v0.1.0`, not `v0.+`) and re-read the CHANGELOG before bumping.

Semver guarantees begin at v1.0. The `bridgeABI()` integer is bumped on each breaking change so a consumer can refuse to start when the stub and `.so` disagree.

## License

This repository is licensed under **GPL-3.0** — see `LICENSE`. Upstream mihomo is GPL-3.0, statically linked into `libclash.so`. Per the FSF FAQ ("JNI counts as combined work"), downstream apps that ship `libclash.so` are also obligated under GPL-3.0.

If you are looking to drop a DPI-bypass proxy into a proprietary app without GPL obligations, see the sibling project [oviron/libbyedpi-android](https://github.com/oviron/libbyedpi-android) — that one wraps byedpi (MIT) instead and ships under MIT.

## Building from source

```sh
git clone https://github.com/oviron/libmihomo-android.git
cd libmihomo-android

# Set NDK path (NDK 28+):
export ANDROID_NDK=$HOME/Library/Android/sdk/ndk/28.0.13004108

# Build libclash.so for all three ABIs:
./build-native.sh

# Package the .aar:
./gradlew :assembleRelease
# Output: build/outputs/aar/libmihomo-android-release.aar
```

See `MAINTENANCE.md` for the full release procedure, GPG key rotation, and upstream-bump workflow.

## Reporting issues

GitHub Issues for bugs and feature requests. For security disclosures see `SECURITY.md`.
