#include <jni.h>
#include <cstring>

#include "jni_helper.h"
#include "libclash.h"
#include "bridge.h"

extern "C"
JNIEXPORT void JNICALL
Java_io_github_oviron_libmihomo_Clash_nativeStartTUN(JNIEnv *env, jclass clazz, jint fd,
                                                     jobject cb, jstring device, jstring stack,
                                                     jstring address, jstring dns, jint mtu) {
    const auto interface = new_global(cb);
    startTUN(interface, fd, get_string(device), get_string(stack), get_string(address),
             get_string(dns), mtu);
}

extern "C"
JNIEXPORT void JNICALL
Java_io_github_oviron_libmihomo_Clash_nativeStopTun(JNIEnv *env, jclass clazz) {
    stopTun();
}

extern "C"
JNIEXPORT void JNICALL
Java_io_github_oviron_libmihomo_Clash_nativeForceGC(JNIEnv *env, jclass clazz) {
    forceGC();
}

extern "C"
JNIEXPORT void JNICALL
Java_io_github_oviron_libmihomo_Clash_nativeUpdateDNS(JNIEnv *env, jclass clazz, jstring servers) {
    updateDns(get_string(servers));
}

extern "C"
JNIEXPORT void JNICALL
Java_io_github_oviron_libmihomo_Clash_nativeInvokeAction(JNIEnv *env, jclass clazz, jstring data,
                                                         jobject cb) {
    const auto interface = new_global(cb);
    invokeAction(interface, get_string(data));
}

extern "C"
JNIEXPORT void JNICALL
Java_io_github_oviron_libmihomo_Clash_nativeSetEventListener(JNIEnv *env, jclass clazz,
                                                             jobject cb) {
    if (cb != nullptr) {
        const auto interface = new_global(cb);
        setEventListener(interface);
    } else {
        setEventListener(nullptr);
    }
}

extern "C"
JNIEXPORT jstring JNICALL
Java_io_github_oviron_libmihomo_Clash_nativeGetTraffic(JNIEnv *env, jclass clazz) {
    auto traffic = getTraffic();
    const auto result = new_string(traffic);
    release_string(&traffic);
    return result;
}

extern "C"
JNIEXPORT jstring JNICALL
Java_io_github_oviron_libmihomo_Clash_nativeGetTotalTraffic(JNIEnv *env, jclass clazz) {
    auto traffic = getTotalTraffic();
    const auto result = new_string(traffic);
    release_string(&traffic);
    return result;
}

extern "C"
JNIEXPORT void JNICALL
Java_io_github_oviron_libmihomo_Clash_nativeSuspended(JNIEnv *env, jclass clazz,
                                                      jboolean suspended) {
    suspend(suspended);
}

extern "C"
JNIEXPORT void JNICALL
Java_io_github_oviron_libmihomo_Clash_nativeQuickSetup(JNIEnv *env, jclass clazz,
                                                       jstring init_params,
                                                       jstring setup_params, jobject cb) {
    const auto interface = new_global(cb);
    quickSetup(interface, get_string(init_params), get_string(setup_params));
}

extern "C"
JNIEXPORT jint JNICALL
Java_io_github_oviron_libmihomo_Clash_nativeBridgeABI(JNIEnv *env, jclass clazz) {
    return bridgeABI();
}


static jmethodID m_tun_interface_protect;
static jmethodID m_tun_interface_resolve_process;
static jmethodID m_invoke_interface_result;


static void release_jni_object_impl(void *obj) {
    ATTACH_JNI();
    del_global(static_cast<jobject>(obj));
}

static void free_string_impl(char *str) {
    free(str);
}

static void call_tun_interface_protect_impl(void *tun_interface, const int fd) {
    ATTACH_JNI();
    env->CallVoidMethod(static_cast<jobject>(tun_interface),
                        m_tun_interface_protect,
                        fd);
}

static char *
call_tun_interface_resolve_process_impl(void *tun_interface, const int protocol,
                                        const char *source,
                                        const char *target,
                                        const int uid) {
    if (tun_interface == nullptr) {
        return strdup("");
    }
    ATTACH_JNI();
    const auto packageName = reinterpret_cast<jstring>(env->CallObjectMethod(
            static_cast<jobject>(tun_interface),
            m_tun_interface_resolve_process,
            protocol,
            new_string(source),
            new_string(target),
            uid));
    return get_string(packageName);
}

static void call_invoke_interface_result_impl(void *invoke_interface, const char *data) {
    ATTACH_JNI();
    env->CallVoidMethod(static_cast<jobject>(invoke_interface),
                        m_invoke_interface_result,
                        new_string(data));
}

extern "C"
JNIEXPORT jint JNICALL
JNI_OnLoad(JavaVM *vm, void *) {
    JNIEnv *env = nullptr;
    if (vm->GetEnv(reinterpret_cast<void **>(&env), JNI_VERSION_1_6) != JNI_OK) {
        return JNI_ERR;
    }

    initialize_jni(vm, env);

    const auto c_tun_interface = find_class("io/github/oviron/libmihomo/TunInterface");
    const auto c_invoke_interface = find_class("io/github/oviron/libmihomo/InvokeInterface");

    m_tun_interface_protect = find_method(c_tun_interface, "protect", "(I)V");
    m_tun_interface_resolve_process = find_method(c_tun_interface, "resolverProcess",
                                                  "(ILjava/lang/String;Ljava/lang/String;I)Ljava/lang/String;");
    m_invoke_interface_result = find_method(c_invoke_interface, "onResult",
                                            "(Ljava/lang/String;)V");

    protect_func = &call_tun_interface_protect_impl;
    resolve_process_func = &call_tun_interface_resolve_process_impl;
    result_func = &call_invoke_interface_result_impl;
    release_object_func = &release_jni_object_impl;
    free_string_func = &free_string_impl;

    return JNI_VERSION_1_6;
}
