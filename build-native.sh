#!/usr/bin/env bash
#
# Builds libclash.so for all three supported ABIs and drops them under
# src/main/jniLibs/<abi>/. AGP `:assembleRelease` then bundles them into the
# .aar without itself invoking Go.
#
# Required environment:
#   ANDROID_NDK — path to NDK 28+ (e.g. $ANDROID_HOME/ndk/28.0.13004108)
#
# Optional:
#   GO_TAGS — default "with_gvisor,cmfa"
#   GO_LDFLAGS — default "-w -s"
#   ABIS — space-separated list, default "arm64-v8a armeabi-v7a x86_64"

set -euo pipefail

cd "$(dirname "$0")"

: "${ANDROID_NDK:?ANDROID_NDK must point at an NDK 28+ install}"
GO_TAGS="${GO_TAGS:-with_gvisor,cmfa}"
# Reproducible-build flags: -buildid= zeros Go's build ID; -trimpath strips
# absolute source paths from the binary so the same source tree on different
# checkout directories produces byte-identical output. Together with a fixed
# SOURCE_DATE_EPOCH (used by clang strip via Android NDK) this yields a
# deterministic .so — same source SHA → same artifact SHA → no surprise
# divergence between a contributor's local rebuild and the GitHub Release.
GO_LDFLAGS="${GO_LDFLAGS:--w -s -buildid=}"
ABIS="${ABIS:-arm64-v8a armeabi-v7a x86_64}"
# Fixed epoch for clang/objcopy timestamps. Same value across all rebuilds.
export SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-1700000000}"

case "$(uname -s)" in
    Darwin) NDK_HOST=darwin-x86_64 ;;
    Linux)  NDK_HOST=linux-x86_64 ;;
    *) echo "Unsupported host OS: $(uname -s)" >&2; exit 1 ;;
esac

TOOLCHAIN_BIN="$ANDROID_NDK/toolchains/llvm/prebuilt/$NDK_HOST/bin"
if [ ! -d "$TOOLCHAIN_BIN" ]; then
    echo "NDK toolchain not found: $TOOLCHAIN_BIN" >&2
    exit 1
fi

JNI_ROOT="src/main/jniLibs"
CORE_DIR="src/main/jni/core"

abi_to_cc() {
    case "$1" in
        arm64-v8a)    echo "aarch64-linux-android21-clang" ;;
        armeabi-v7a)  echo "armv7a-linux-androideabi21-clang" ;;
        x86_64)       echo "x86_64-linux-android21-clang" ;;
        *) echo "Unknown ABI: $1" >&2; exit 1 ;;
    esac
}

abi_to_goarch() {
    case "$1" in
        arm64-v8a)   echo "arm64" ;;
        armeabi-v7a) echo "arm" ;;
        x86_64)      echo "amd64" ;;
        *) echo "Unknown ABI: $1" >&2; exit 1 ;;
    esac
}

for abi in $ABIS; do
    cc="$TOOLCHAIN_BIN/$(abi_to_cc "$abi")"
    goarch="$(abi_to_goarch "$abi")"
    out_dir="$JNI_ROOT/$abi"
    mkdir -p "$out_dir"

    echo ">> Building libclash.so for $abi (GOARCH=$goarch, CC=$(basename "$cc"))"
    repo_root="$PWD"
    (
        cd "$CORE_DIR"
        GOOS=android GOARCH="$goarch" CGO_ENABLED=1 \
            CC="$cc" CFLAGS="-O3 -Werror" \
            go build \
                -trimpath \
                -ldflags="$GO_LDFLAGS" \
                -tags="$GO_TAGS" \
                -buildmode=c-shared \
                -o "$repo_root/$out_dir/libclash.so" \
                .
    )
    # Strip the generated header — Android AGP only needs the .so itself.
    rm -f "$out_dir/libclash.h"
done

echo ">> Done. Artifacts:"
find "$JNI_ROOT" -name 'libclash.so' -exec ls -la {} \;
