# JNI bridge — keep the Kotlin facade so R8 in the consumer app does not
# strip the `external fun` declarations that bind to libclash.so.
-keep class io.github.oviron.libmihomo.Clash { *; }
-keep class io.github.oviron.libmihomo.Clash$Companion { *; }
