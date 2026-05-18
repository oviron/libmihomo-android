#!/usr/bin/env bash
# Verify libclash.so exports the exact 11 JNI entrypoints the Kotlin
# facade calls. Drift here = runtime UnsatisfiedLinkError on the
# consumer side, so this is a release gate, not a hint.
set -euo pipefail

EXPECTED=(
  invokeAction
  startTUN
  quickSetup
  stopTun
  setEventListener
  getTraffic
  getTotalTraffic
  suspend
  forceGC
  updateDns
  bridgeABI
)
pattern=$(printf '|%s' "${EXPECTED[@]}")
pattern="${pattern:1}"

fail=0
for abi in arm64-v8a armeabi-v7a x86_64; do
  so="src/main/jniLibs/$abi/libclash.so"
  if [ ! -f "$so" ]; then
    echo "missing: $so" >&2
    fail=1
    continue
  fi
  count=$(nm -D "$so" 2>/dev/null | grep -cE " T (${pattern})$" || true)
  if [ "$count" -ne "${#EXPECTED[@]}" ]; then
    echo "FAIL $so: expected ${#EXPECTED[@]} JNI exports, got $count" >&2
    nm -D "$so" 2>/dev/null | grep ' T ' >&2 || true
    fail=1
  else
    echo "OK   $so: all ${#EXPECTED[@]} JNI exports present"
  fi
done
exit "$fail"
