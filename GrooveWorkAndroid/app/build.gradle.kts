import java.io.FileInputStream
import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.google.services)
}

// Номер сборки в формате ГГММДДН — единый источник правды apps/version.json в
// корне репозитория (тот же файл сервер отдаёт для проверки обновлений).
// Подставляем как versionCode: приложение узнаёт свою сборку через
// packageInfo.longVersionCode, а монотонный рост по дате удовлетворяет
// требованию Android к возрастающему versionCode при обновлении.
// Читаем через providers.fileContents (а не File.readText): при включённом
// configuration-cache только Provider API регистрирует файл как вход кэша —
// иначе изменение version.json не инвалидирует кэш и в APK уходит старый код.
val appBuildNumber: Int = run {
    val versionFile = rootProject.layout.projectDirectory.file("../apps/version.json")
    val raw = providers.fileContents(versionFile).asText.orNull
    raw?.let { Regex("\"current_build\"\\s*:\\s*(\\d+)").find(it)?.groupValues?.get(1)?.toIntOrNull() } ?: 1
}

// Подпись релиза: секреты лежат в keystore.properties (в .gitignore, шаблон —
// keystore.properties.example). Без файла release собирается без подписи, а
// debug-сборка ключа не требует. ВАЖНО: ключ один на все релизы — Android
// обновит установленное приложение только APK с той же подписью.
val keystorePropsFile = rootProject.file("keystore.properties")
val keystoreProps = Properties().apply {
    if (keystorePropsFile.exists()) FileInputStream(keystorePropsFile).use { load(it) }
}

android {
    namespace = "com.kodass.groovework"
    compileSdk {
        version = release(37)
    }

    defaultConfig {
        applicationId = "com.kodass.groovework"
        minSdk = 34
        targetSdk = 36
        versionCode = appBuildNumber
        versionName = "1.0"

        buildConfigField("String", "DEFAULT_SERVER_URL", "\"https://gw.kodass.ru\"")

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    signingConfigs {
        if (keystorePropsFile.exists()) {
            create("release") {
                storeFile = rootProject.file(keystoreProps.getProperty("storeFile"))
                storePassword = keystoreProps.getProperty("storePassword")
                keyAlias = keystoreProps.getProperty("keyAlias")
                keyPassword = keystoreProps.getProperty("keyPassword")
            }
        }
    }

    buildTypes {
        release {
            optimization {
                enable = false
            }
            if (keystorePropsFile.exists()) {
                signingConfig = signingConfigs.getByName("release")
            }
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_11
        targetCompatibility = JavaVersion.VERSION_11
    }
    buildFeatures {
        compose = true
        buildConfig = true
    }
}

dependencies {
    implementation(platform(libs.androidx.compose.bom))
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.compose.material3)
    implementation(libs.androidx.compose.material.icons.extended)
    implementation(libs.androidx.compose.ui)
    implementation(libs.androidx.compose.ui.graphics)
    implementation(libs.androidx.compose.ui.tooling.preview)
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.androidx.lifecycle.runtime.compose)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    implementation(libs.androidx.navigation.compose)
    implementation(libs.androidx.datastore.preferences)
    implementation(libs.retrofit)
    implementation(libs.retrofit.kotlinx.serialization)
    implementation(libs.okhttp)
    implementation(libs.kotlinx.serialization.json)
    implementation(libs.coil.compose)
    implementation(libs.coil.network.okhttp)
    implementation(libs.livekit.android)
    // Только VideoTrackView; аудио-визуализатор не нужен, а его транзитивный
    // noise-2.0.0 тащит libnoise.so без 16 KB-выравнивания (требование Play
    // для Android 15+).
    implementation(libs.livekit.android.compose.components) {
        exclude(group = "com.github.paramsen", module = "noise")
    }
    implementation(platform(libs.firebase.bom))
    implementation(libs.firebase.messaging)
    testImplementation(libs.junit)
    androidTestImplementation(platform(libs.androidx.compose.bom))
    androidTestImplementation(libs.androidx.compose.ui.test.junit4)
    androidTestImplementation(libs.androidx.espresso.core)
    androidTestImplementation(libs.androidx.junit)
    debugImplementation(libs.androidx.compose.ui.test.manifest)
    debugImplementation(libs.androidx.compose.ui.tooling)
}
