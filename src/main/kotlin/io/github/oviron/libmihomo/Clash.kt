package io.github.oviron.libmihomo

/**
 * Native bridge to the embedded mihomo core.
 *
 * The shared object `libclash.so` is statically linked against the upstream
 * [metacubex/mihomo](https://github.com/MetaCubeX/mihomo) (GPL-3.0). All
 * exports below are `//export`-marked Go functions compiled with
 * `-buildmode=c-shared`.
 *
 * Callbacks crossing the JNI boundary are passed as `Long` opaque handles
 * (jobject GlobalRef in the consumer's wrapper). The consumer is responsible
 * for the per-callback lifetime — see `release_object` in the C bridge.
 *
 * Threading: all `external fun` here are non-blocking dispatchers — actual
 * work runs on Go goroutines.
 */
object Clash {
    init {
        System.loadLibrary("clash")
    }

    /**
     * Integer identifying the JNI/bridge API surface of the bundled `libclash.so`.
     * Bumped on breaking changes (signature, semantics, exports added/removed).
     * Consumers can compare against [EXPECTED_BRIDGE_ABI] at startup to detect
     * mismatched library + stub pairings.
     */
    @JvmStatic
    external fun bridgeABI(): Int

    /**
     * The bridge ABI this stub was generated against. Cross-check at startup:
     * if `bridgeABI() != EXPECTED_BRIDGE_ABI` the host APK was built against a
     * different libmihomo-android release than the .so it loads.
     */
    const val EXPECTED_BRIDGE_ABI: Int = 1

    /**
     * Dispatches a JSON action to the mihomo dispatcher. The `data` payload is
     * an `Action` document (`{"id", "method", "data"}`). The result is
     * forwarded asynchronously through `callback` — see the C `result` glue.
     */
    @JvmStatic
    external fun invokeAction(callback: Long, data: String)

    /**
     * Initializes mihomo and applies the first profile in one round-trip.
     * `initParams` and `setupParams` are JSON documents matching mihomo's
     * standard init / setup contract.
     */
    @JvmStatic
    external fun quickSetup(callback: Long, initParams: String, setupParams: String)

    /**
     * Starts the TUN listener bound to `fd` (an Android VpnService.Builder fd).
     * `device` is the interface label surfaced in mihomo logs. `stack` is one
     * of "system" | "gvisor" | "mixed". `address` is a comma-separated CIDR
     * list (IPv4/IPv6). `dns` is a comma-separated host list hijacked at :53.
     */
    @JvmStatic
    external fun startTUN(
        callback: Long,
        fd: Int,
        device: String,
        stack: String,
        address: String,
        dns: String,
    )

    /** Stops the TUN listener and releases the kernel fd. Idempotent. */
    @JvmStatic
    external fun stopTun()

    /**
     * Installs a callback that receives push events from mihomo (connections
     * snapshots, log lines, etc.). Pass 0 to clear. Only one listener per
     * process is supported — the latest call wins.
     */
    @JvmStatic
    external fun setEventListener(callback: Long)

    /** Bytes/sec totals at the moment of the call. Snapshot, not a stream. */
    @JvmStatic
    external fun getTraffic(): String

    /** Total bytes since the process started. */
    @JvmStatic
    external fun getTotalTraffic(): String

    /** Releases idle resources without stopping the TUN. Safe to call often. */
    @JvmStatic
    external fun suspend()

    /** Hints the Go runtime to GC right now. Diagnostic. */
    @JvmStatic
    external fun forceGC()

    /** Updates the system DNS list mihomo falls back on for unresolved domains. */
    @JvmStatic
    external fun updateDns(servers: String)
}
