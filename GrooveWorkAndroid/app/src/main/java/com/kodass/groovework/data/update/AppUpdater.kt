package com.kodass.groovework.data.update

import android.app.Application
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.provider.Settings
import androidx.core.content.FileProvider
import com.kodass.groovework.data.api.MetaApi
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.data.network.normalizeServerUrl
import com.kodass.groovework.data.session.SessionManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.Request
import java.io.File

// Состояние проверки/скачивания обновления приложения. Живёт в AppContainer
// (один на процесс), чтобы фаза ReadyToInstall пережила уход с экрана «О
// приложении» — кнопка продолжает предлагать установку.
sealed interface UpdateState {
    data object Idle : UpdateState
    data object Checking : UpdateState
    data object UpToDate : UpdateState
    data class Available(val build: Long) : UpdateState
    data class Downloading(val progress: Float) : UpdateState
    // APK скачан и ждёт запуска установки (или пользователь отклонил системный
    // диалог установки — кнопка остаётся «Установить обновление»).
    data class ReadyToInstall(val build: Long) : UpdateState
    data class Failed(val message: String) : UpdateState
}

// Обновление приложения «по воздуху»: сравнивает свою сборку (versionCode) с
// номером сборки APK на сервере (apps/version.json), качает apps/groovework.apk
// во внутренний каталог и запускает системную установку через FileProvider.
class AppUpdater(
    private val app: Application,
    private val metaApi: MetaApi,
    private val downloadHttp: OkHttpClient,
    private val sessionManager: SessionManager,
    private val json: Json,
    private val scope: CoroutineScope,
) {
    private val _state = MutableStateFlow<UpdateState>(UpdateState.Idle)
    val state: StateFlow<UpdateState> = _state

    private val prefs = app.getSharedPreferences("app_update", Context.MODE_PRIVATE)

    val currentBuild: Long = runCatching {
        app.packageManager.getPackageInfo(app.packageName, 0).longVersionCode
    }.getOrDefault(0L)

    private val apkFile: File
        get() = File(app.getExternalFilesDir(null), "groovework-update.apk")

    init {
        // Мусор после успешного обновления: скачанный APK со сборкой не новее
        // установленной (или нечитаемый) больше не нужен.
        scope.launch(Dispatchers.IO) {
            val file = apkFile
            if (file.exists()) {
                val info = app.packageManager.getPackageArchiveInfo(file.path, 0)
                if (info == null || info.longVersionCode <= currentBuild) file.delete()
            }
        }
    }

    // Автопроверка при старте (после появления сессии): не чаще раза в 6 часов,
    // ошибки — молча (ручная проверка на экране «О приложении» покажет их сама).
    fun autoCheck() {
        // UpToDate не блокирует будущие проверки: приложение живёт в памяти
        // днями, а релиз новой сборки может выйти позже старта.
        when (_state.value) {
            is UpdateState.Idle, is UpdateState.UpToDate -> Unit
            else -> return
        }
        val now = System.currentTimeMillis()
        if (now - prefs.getLong(KEY_LAST_CHECK, 0L) < AUTO_CHECK_INTERVAL_MS) return
        scope.launch {
            try {
                val serverBuild = apiCall(json) { metaApi.appBuild() }.currentBuild
                prefs.edit().putLong(KEY_LAST_CHECK, System.currentTimeMillis()).apply()
                if (serverBuild > currentBuild) _state.value = UpdateState.Available(serverBuild)
            } catch (_: Exception) {
            }
        }
    }

    fun check() {
        when (_state.value) {
            is UpdateState.Checking, is UpdateState.Downloading -> return
            else -> Unit
        }
        scope.launch {
            _state.value = UpdateState.Checking
            try {
                val serverBuild = apiCall(json) { metaApi.appBuild() }.currentBuild
                _state.value = if (serverBuild > currentBuild) {
                    UpdateState.Available(serverBuild)
                } else {
                    UpdateState.UpToDate
                }
            } catch (_: Exception) {
                _state.value = UpdateState.Failed("Не удалось проверить обновления")
            }
        }
    }

    fun download() {
        val available = _state.value as? UpdateState.Available ?: return
        scope.launch {
            _state.value = UpdateState.Downloading(0f)
            try {
                downloadApk { progress -> _state.value = UpdateState.Downloading(progress) }
                // Проверяем скачанное перед установкой: это наш пакет и сборка не
                // ниже заявленной — иначе файл битый либо сервер ещё раздаёт старый
                // APK (version.json при деплое обновляется раньше самого файла).
                if (!verifyApk(available.build)) {
                    apkFile.delete()
                    _state.value = UpdateState.Failed("Файл обновления повреждён — попробуйте ещё раз")
                    return@launch
                }
                _state.value = UpdateState.ReadyToInstall(available.build)
                launchInstall()
            } catch (_: Exception) {
                _state.value = UpdateState.Failed("Не удалось скачать обновление")
            }
        }
    }

    // Скачивание из обязательного диалога обновления: сборку помнит вызывающий,
    // поэтому стартует и из Available, и повторно после Failed.
    fun downloadBuild(build: Long) {
        when (_state.value) {
            is UpdateState.Checking, is UpdateState.Downloading -> return
            is UpdateState.ReadyToInstall -> {
                launchInstall()
                return
            }
            else -> Unit
        }
        _state.value = UpdateState.Available(build)
        download()
    }

    // Повторный запуск установки скачанного APK (после отклонённого диалога).
    fun install() {
        if (_state.value !is UpdateState.ReadyToInstall) return
        launchInstall()
    }

    private fun launchInstall() {
        val file = apkFile
        if (!file.exists()) {
            _state.value = UpdateState.Idle
            return
        }
        // С Android 8+ установка из стороннего источника требует явного
        // разрешения пользователя — отправляем в системные настройки, состояние
        // остаётся ReadyToInstall, чтобы повторить установку после возврата.
        if (!app.packageManager.canRequestPackageInstalls()) {
            val intent = Intent(
                Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,
                Uri.parse("package:${app.packageName}"),
            ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            runCatching { app.startActivity(intent) }
            return
        }
        val uri = FileProvider.getUriForFile(app, "${app.packageName}.fileprovider", file)
        val intent = Intent(Intent.ACTION_VIEW)
            .setDataAndType(uri, "application/vnd.android.package-archive")
            .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION or Intent.FLAG_ACTIVITY_NEW_TASK)
        runCatching { app.startActivity(intent) }
    }

    private suspend fun verifyApk(build: Long): Boolean = withContext(Dispatchers.IO) {
        val info = app.packageManager.getPackageArchiveInfo(apkFile.path, 0)
        info != null && info.packageName == app.packageName && info.longVersionCode >= build
    }

    private suspend fun downloadApk(onProgress: (Float) -> Unit) = withContext(Dispatchers.IO) {
        val base = normalizeServerUrl(sessionManager.serverUrl.value)
        val response = downloadHttp.newCall(Request.Builder().url("$base/apps/mobile/groovework.apk").build()).execute()
        response.use {
            if (!response.isSuccessful) error("HTTP ${response.code}")
            val body = response.body ?: error("Пустой ответ сервера")
            val total = body.contentLength()
            apkFile.outputStream().use { out ->
                body.byteStream().use { input ->
                    val buffer = ByteArray(64 * 1024)
                    var readTotal = 0L
                    onProgress(if (total > 0) 0f else -1f)
                    while (true) {
                        val read = input.read(buffer)
                        if (read < 0) break
                        out.write(buffer, 0, read)
                        readTotal += read
                        onProgress(if (total > 0) (readTotal.toFloat() / total).coerceIn(0f, 1f) else -1f)
                    }
                    out.flush()
                }
            }
        }
    }

    private companion object {
        const val KEY_LAST_CHECK = "last_check"
        const val AUTO_CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000L
    }
}
