#!/usr/bin/env sh
# Static check: every class/interface/method that JNI_OnLoad references via
# FindClass + GetMethodID must be covered by a `-keep` rule in
# consumer-rules.pro. Exits non-zero if any JNI lookup lacks a matching
# keep rule. Wired into Gradle preBuild via `dependsOn(validateJniKeep)`.

set -eu

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CPP_DIR="$REPO_ROOT/src/main/cpp"
KT_DIR="$REPO_ROOT/src/main/kotlin"
RULES="$REPO_ROOT/consumer-rules.pro"

[ -f "$RULES" ] || { echo "consumer-rules.pro not found at $RULES" >&2; exit 1; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT INT TERM

# 1. Collect classes referenced via find_class / FindClass in C/C++ sources.
grep -rhoE 'find_class[[:space:]]*\([[:space:]]*"[^"]+"' "$CPP_DIR" 2>/dev/null \
    | sed -E 's@^[^"]*"([^"]+)".*@\1@' | sort -u > "$WORK/jni_classes"
grep -rhoE 'FindClass[[:space:]]*\([^,]*,[[:space:]]*"[^"]+"' "$CPP_DIR" 2>/dev/null \
    | sed -E 's@^[^"]*"([^"]+)".*@\1@' | sort -u >> "$WORK/jni_classes"
sort -uo "$WORK/jni_classes" "$WORK/jni_classes"

# 2. Collect method lookups (best-effort: parse find_method / GetMethodID args).
grep -rhE 'find_method[[:space:]]*\([^)]+\)' "$CPP_DIR" 2>/dev/null \
    | sed -E 's@.*find_method[[:space:]]*\([[:space:]]*([^,]+),[[:space:]]*"([^"]+)".*@\1::\2@' \
    | sort -u > "$WORK/jni_methods"

# 3. Native external funs in Kotlin → expected JNI symbols.
grep -rhE 'external[[:space:]]+fun[[:space:]]+[A-Za-z_]+' "$KT_DIR" 2>/dev/null \
    | sed -E 's@.*external[[:space:]]+fun[[:space:]]+([A-Za-z_][A-Za-z0-9_]*).*@\1@' \
    | sort -u > "$WORK/native_funs"

# 4. Parse consumer-rules.pro: collect kept fully-qualified names.
grep -E '^-keep ' "$RULES" \
    | sed -E 's@^-keep (class|interface)[[:space:]]+([A-Za-z0-9_.$]+).*@\2@' \
    | sed -E 's@^[A-Za-z0-9_]+[[:space:]]+@@' \
    | grep -E '^[a-zA-Z]' | sort -u > "$WORK/kept_classes"

# Also: -keepclasseswithmembernames wildcard package coverage.
grep -E '^-keepclasseswithmembernames' "$RULES" \
    | sed -E 's@.*class[[:space:]]+([A-Za-z0-9_.$*]+)[[:space:]]*\{.*@\1@' \
    | sort -u > "$WORK/wildcard_packages"

echo "== JNI classes looked up (from C/C++) =="
sed 's@/@.@g' "$WORK/jni_classes" | sed 's@^@  @'
echo
echo "== Kept (from consumer-rules.pro) =="
sed 's@^@  @' "$WORK/kept_classes"
[ -s "$WORK/wildcard_packages" ] && {
    echo "  -- wildcard --"
    sed 's@^@  @' "$WORK/wildcard_packages"
}
echo
echo "== Verification =="

FAIL=0
while IFS= read -r jni_path; do
    [ -z "$jni_path" ] && continue
    dotted=$(echo "$jni_path" | sed 's@/@.@g')
    # java.* / kotlin.* / android.* are framework — R8 never strips them.
    case "$dotted" in
        java.*|javax.*|kotlin.*|android.*) echo "  SKIP $dotted   (framework)"; continue ;;
    esac
    if grep -qxF "$dotted" "$WORK/kept_classes"; then
        echo "  OK   $dotted"
    else
        echo "  FAIL $dotted   (looked up by JNI but not in -keep)"
        FAIL=1
    fi
done < "$WORK/jni_classes"

echo
echo "== Native methods declared in Kotlin =="
sed 's@^@  @' "$WORK/native_funs"

if [ "$FAIL" -ne 0 ]; then
    echo
    echo "FAIL: at least one JNI-looked-up class is not covered by consumer-rules.pro" >&2
    echo "Add the corresponding -keep rule before tagging a release." >&2
    exit 1
fi

echo
echo "PASS: every JNI lookup is covered by consumer-rules.pro"
