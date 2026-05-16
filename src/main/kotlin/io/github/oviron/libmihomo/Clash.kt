package io.github.oviron.libmihomo

import java.io.File

/** Native bridge to embedded mihomo; see README for lifecycle + threading. */
object Clash {
    const val EXPECTED_BRIDGE_ABI: Int = 1

    @Volatile
    private var initFailure: Throwable? = null

    @Volatile
    private var loaded: Boolean = false

    @Synchronized
    fun load(nativeLibDir: String) {
        if (loaded) return
        initFailure = try {
            System.load(resolveLib(nativeLibDir, "libclash.so"))
            System.load(resolveLib(nativeLibDir, "libmihomo-jni.so"))
            val abi = nativeBridgeABI()
            if (abi != EXPECTED_BRIDGE_ABI) {
                throw IllegalStateException(
                    "libclash bridge ABI mismatch: facade expects " +
                            "$EXPECTED_BRIDGE_ABI, .so reports $abi"
                )
            }
            loaded = true
            null
        } catch (t: Throwable) {
            t
        }
    }

    fun isLoaded(): Boolean = loaded

    fun assertReady() {
        if (loaded) return
        val err = initFailure
        throw IllegalStateException(
            "Clash not loaded: call Clash.load(nativeLibDir) first" +
                    (err?.let { ": ${it.message}" } ?: ""),
            err,
        )
    }

    private fun resolveLib(dir: String, name: String): String {
        val f = File(dir, name)
        if (!f.exists()) throw IllegalStateException("missing ${f.absolutePath}")
        return f.absolutePath
    }

    fun bridgeABI(): Int {
        assertReady()
        return nativeBridgeABI()
    }

    fun invokeAction(data: String, cb: InvokeInterface) {
        assertReady()
        nativeInvokeAction(data, cb)
    }

    fun invokeAction(data: String, onResult: (String?) -> Unit) =
        invokeAction(data, lambdaInvoke(onResult))

    fun quickSetup(initParams: String, setupParams: String, cb: InvokeInterface) {
        assertReady()
        nativeQuickSetup(initParams, setupParams, cb)
    }

    fun quickSetup(
        initParams: String,
        setupParams: String,
        onResult: (String?) -> Unit,
    ) = quickSetup(initParams, setupParams, lambdaInvoke(onResult))

    fun startTUN(
        fd: Int,
        cb: TunInterface,
        device: String,
        stack: String,
        address: String,
        dns: String,
    ) {
        assertReady()
        nativeStartTUN(fd, cb, device, stack, address, dns)
    }

    fun startTUN(
        fd: Int,
        protect: (Int) -> Unit,
        resolverProcess: (protocol: Int, source: String, target: String, uid: Int) -> String,
        device: String,
        stack: String,
        address: String,
        dns: String,
    ) = startTUN(fd, lambdaTun(protect, resolverProcess), device, stack, address, dns)

    fun stopTun() {
        assertReady()
        nativeStopTun()
    }

    fun setEventListener(cb: InvokeInterface?) {
        assertReady()
        nativeSetEventListener(cb)
    }

    fun setEventListener(onResult: ((String?) -> Unit)?) =
        setEventListener(onResult?.let { lambdaInvoke(it) })

    fun getTraffic(): String {
        assertReady()
        return nativeGetTraffic()
    }

    fun getTotalTraffic(): String {
        assertReady()
        return nativeGetTotalTraffic()
    }

    fun suspended(suspended: Boolean) {
        assertReady()
        nativeSuspended(suspended)
    }

    fun forceGC() {
        assertReady()
        nativeForceGC()
    }

    fun updateDNS(servers: String) {
        assertReady()
        nativeUpdateDNS(servers)
    }

    private fun lambdaInvoke(onResult: (String?) -> Unit) =
        object : InvokeInterface {
            override fun onResult(result: String?) = onResult.invoke(result)
        }

    private fun lambdaTun(
        protect: (Int) -> Unit,
        resolverProcess: (Int, String, String, Int) -> String,
    ) = object : TunInterface {
        override fun protect(fd: Int) = protect.invoke(fd)
        override fun resolverProcess(
            protocol: Int, source: String, target: String, uid: Int,
        ): String = resolverProcess.invoke(protocol, source, target, uid)
    }

    @JvmStatic
    private external fun nativeBridgeABI(): Int

    @JvmStatic
    private external fun nativeInvokeAction(data: String, cb: InvokeInterface)

    @JvmStatic
    private external fun nativeQuickSetup(
        initParams: String,
        setupParams: String,
        cb: InvokeInterface,
    )

    @JvmStatic
    private external fun nativeStartTUN(
        fd: Int,
        cb: TunInterface,
        device: String,
        stack: String,
        address: String,
        dns: String,
    )

    @JvmStatic
    private external fun nativeStopTun()

    @JvmStatic
    private external fun nativeSetEventListener(cb: InvokeInterface?)

    @JvmStatic
    private external fun nativeGetTraffic(): String

    @JvmStatic
    private external fun nativeGetTotalTraffic(): String

    @JvmStatic
    private external fun nativeSuspended(suspended: Boolean)

    @JvmStatic
    private external fun nativeForceGC()

    @JvmStatic
    private external fun nativeUpdateDNS(servers: String)
}
