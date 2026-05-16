plugins {
    id("com.android.library") version "8.12.2"
    id("org.jetbrains.kotlin.android") version "2.2.10"
}

android {
    namespace = "io.github.oviron.libmihomo"
    compileSdk = 36
    ndkVersion = "28.0.13004108"

    defaultConfig {
        minSdk = 21
        ndk {
            abiFilters += listOf("arm64-v8a", "armeabi-v7a", "x86_64")
        }
        consumerProguardFiles("consumer-rules.pro")
        externalNativeBuild {
            cmake {
                arguments += listOf("-DANDROID_STL=c++_static")
                cppFlags += "-std=c++17"
            }
        }
    }

    externalNativeBuild {
        cmake {
            path = file("src/main/cpp/CMakeLists.txt")
            version = "3.22.1"
        }
    }

    sourceSets {
        getByName("main") {
            jniLibs.srcDirs("src/main/jniLibs")
            kotlin.srcDirs("src/main/kotlin")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    packaging {
        jniLibs.useLegacyPackaging = false
    }

    buildTypes {
        release {
            isMinifyEnabled = false
        }
    }
}

val validateJniKeep by tasks.registering(Exec::class) {
    workingDir = projectDir
    commandLine("sh", "scripts/validate-jni-keep.sh")
}

afterEvaluate {
    tasks.named("preBuild") { dependsOn(validateJniKeep) }
}
