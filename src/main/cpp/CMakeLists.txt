cmake_minimum_required(VERSION 3.22.1)

project("mihomo-jni")

if (NOT "${CMAKE_BUILD_TYPE}" STREQUAL "Debug")
    set(CMAKE_BUILD_TYPE Release CACHE STRING "Build type" FORCE)
    add_compile_options(-O3 -flto -g0 -fno-exceptions -fno-rtti)
    add_link_options(-flto -Wl,--gc-sections,--strip-all)
endif ()

set(LIB_CLASH_PATH "${CMAKE_SOURCE_DIR}/../jniLibs/${ANDROID_ABI}/libclash.so")

if (NOT EXISTS ${LIB_CLASH_PATH})
    message(FATAL_ERROR
        "libclash.so missing for ABI ${ANDROID_ABI}.\n"
        "Run `bash build-native.sh` to compile libclash.so via cgo before the AAR build.")
endif ()

include_directories(${CMAKE_SOURCE_DIR})
link_directories(${CMAKE_SOURCE_DIR}/../jniLibs/${ANDROID_ABI})

add_library(${CMAKE_PROJECT_NAME} SHARED
        jni_helper.cpp
        native-lib.cpp)

target_link_libraries(${CMAKE_PROJECT_NAME} clash)
