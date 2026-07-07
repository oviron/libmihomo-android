/* Vendored cgo header for libclash.so produced by build-native.sh.
 *
 * Hand-merged from the per-ABI cgo outputs so a single header serves all
 * supported architectures (arm64-v8a, armeabi-v7a, x86_64). The only
 * cgo-emitted per-arch differences are the size of GoInt/GoUint and a
 * pointer-size sanity assertion, both gated by __LP64__ below.
 *
 * Keep this file in sync with src/main/jni/core/lib.go `//export` declarations.
 * Bump LIBCLASH_EXPECTED_BRIDGE_ABI when adding, removing, or changing a signature.
 */

#ifndef VENDORED_LIBCLASH_H
#define VENDORED_LIBCLASH_H

#include <stddef.h>

#ifndef GO_CGO_EXPORT_PROLOGUE_H
#define GO_CGO_EXPORT_PROLOGUE_H

#ifndef GO_CGO_GOSTRING_TYPEDEF
typedef struct { const char *p; ptrdiff_t n; } _GoString_;
extern size_t _GoStringLen(_GoString_ s);
extern const char *_GoStringPtr(_GoString_ s);
#endif

#endif

#include <stdlib.h>

#ifndef GO_CGO_PROLOGUE_H
#define GO_CGO_PROLOGUE_H

typedef signed char GoInt8;
typedef unsigned char GoUint8;
typedef short GoInt16;
typedef unsigned short GoUint16;
typedef int GoInt32;
typedef unsigned int GoUint32;
typedef long long GoInt64;
typedef unsigned long long GoUint64;
#if defined(__LP64__) || defined(_WIN64) || defined(__x86_64__) || defined(__aarch64__)
typedef GoInt64 GoInt;
typedef GoUint64 GoUint;
#else
typedef GoInt32 GoInt;
typedef GoUint32 GoUint;
#endif
typedef size_t GoUintptr;
typedef float GoFloat32;
typedef double GoFloat64;
#ifdef _MSC_VER
#if !defined(__cplusplus) || _MSVC_LANG <= 201402L
#include <complex.h>
typedef _Fcomplex GoComplex64;
typedef _Dcomplex GoComplex128;
#else
#include <complex>
typedef std::complex<float> GoComplex64;
typedef std::complex<double> GoComplex128;
#endif
#else
typedef float _Complex GoComplex64;
typedef double _Complex GoComplex128;
#endif

typedef char _check_for_pointer_matching_GoInt[sizeof(void*) == sizeof(GoInt) ? 1 : -1];

#ifndef GO_CGO_GOSTRING_TYPEDEF
typedef _GoString_ GoString;
#endif
typedef void *GoMap;
typedef void *GoChan;
typedef struct { void *t; void *v; } GoInterface;
typedef struct { void *data; GoInt len; GoInt cap; } GoSlice;

#endif

#ifdef __cplusplus
extern "C" {
#endif

extern void invokeAction(void* callback, char* paramsChar);
extern void quickSetup(void* callback, char* initParamsChar, char* setupParamsChar);
extern void startTUN(void* callback, int fd, char* deviceChar, char* stackChar, char* addressChar, char* dnsChar, int mtu);
extern void stopTun(void);
extern void setEventListener(void* listener);
extern char* getTraffic(void);
extern char* getTotalTraffic(void);
extern void suspend(GoUint8 suspended);
extern void forceGC(void);
extern void updateDns(char* s);
extern int  bridgeABI(void);

#ifdef __cplusplus
}
#endif

#define LIBCLASH_EXPECTED_BRIDGE_ABI 2

_Static_assert(LIBCLASH_EXPECTED_BRIDGE_ABI == 2,
               "Update native-lib.cpp + Kotlin EXPECTED_BRIDGE_ABI together when bumping.");

#endif /* VENDORED_LIBCLASH_H */
