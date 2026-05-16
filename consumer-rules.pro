# Native JNI bridge — these classes/methods are looked up at runtime by
# libmihomo-jni.so via FindClass / GetMethodID. R8 in the consumer APK must
# not strip them or :remote process SIGABRTs at JNI_OnLoad. Verified by
# scripts/validate-jni-keep.sh on each release.

-keep class io.github.oviron.libmihomo.Clash { *; }
-keep class io.github.oviron.libmihomo.Clash$Companion { *; }
-keep interface io.github.oviron.libmihomo.TunInterface { *; }
-keep interface io.github.oviron.libmihomo.InvokeInterface { *; }

# Defensive: any native method declaration in this package, in case future
# bumps add classes the explicit rules above forget.
-keepclasseswithmembernames class io.github.oviron.libmihomo.** {
    native <methods>;
}
