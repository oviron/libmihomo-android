package io.github.oviron.libmihomo

interface TunInterface {
    fun protect(fd: Int)

    fun resolverProcess(
        protocol: Int,
        source: String,
        target: String,
        uid: Int,
    ): String
}
