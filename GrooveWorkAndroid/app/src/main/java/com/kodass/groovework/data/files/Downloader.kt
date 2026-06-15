package com.kodass.groovework.data.files

import android.content.ContentValues
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Environment
import android.provider.MediaStore
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.OkHttpClient
import okhttp3.Request

// Состояние скачивания одного вложения (показываем прогресс прямо в чате/просмотрщике).
sealed interface DownloadState {
    data object Idle : DownloadState
    // progress 0..1; -1f — размер неизвестен (нет Content-Length).
    data class Running(val progress: Float) : DownloadState
    data class Done(val uri: Uri, val mime: String) : DownloadState
    data class Failed(val message: String) : DownloadState
}

// Скачивание файлов внутри приложения (без браузера) в общий MediaStore:
// картинки — в Pictures/GrooveWork (видны в галерее), остальное — в
// Downloads/GrooveWork. Прогресс отдаётся через onProgress (minSdk 34 — scoped
// storage, разрешения не нужны).
class Downloader(
    private val context: Context,
    private val http: OkHttpClient,
) {
    suspend fun download(
        url: String,
        fileName: String,
        mime: String,
        toImages: Boolean,
        onProgress: (Float) -> Unit,
    ): Uri = withContext(Dispatchers.IO) {
        val response = http.newCall(Request.Builder().url(url).build()).execute()
        response.use {
            if (!response.isSuccessful) error("HTTP ${response.code}")
            val body = response.body ?: error("Пустой ответ сервера")
            val total = body.contentLength()

            val resolver = context.contentResolver
            val collection = if (toImages) {
                MediaStore.Images.Media.getContentUri(MediaStore.VOLUME_EXTERNAL_PRIMARY)
            } else {
                MediaStore.Downloads.getContentUri(MediaStore.VOLUME_EXTERNAL_PRIMARY)
            }
            val relativeDir = (if (toImages) Environment.DIRECTORY_PICTURES else Environment.DIRECTORY_DOWNLOADS) +
                "/GrooveWork"

            val values = ContentValues().apply {
                put(MediaStore.MediaColumns.DISPLAY_NAME, fileName.ifBlank { "file" })
                if (mime.isNotBlank()) put(MediaStore.MediaColumns.MIME_TYPE, mime)
                put(MediaStore.MediaColumns.RELATIVE_PATH, relativeDir)
                put(MediaStore.MediaColumns.IS_PENDING, 1)
            }
            val uri = resolver.insert(collection, values) ?: error("Не удалось создать файл")
            try {
                resolver.openOutputStream(uri)?.use { out ->
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
                } ?: error("Не удалось записать файл")
                values.clear()
                values.put(MediaStore.MediaColumns.IS_PENDING, 0)
                resolver.update(uri, values, null, null)
                uri
            } catch (e: Exception) {
                runCatching { resolver.delete(uri, null, null) }
                throw e
            }
        }
    }
}

// Открыть скачанный файл сторонним приложением (content:// из MediaStore).
fun openDownloadedFile(context: Context, uri: Uri, mime: String) {
    val intent = Intent(Intent.ACTION_VIEW).apply {
        setDataAndType(uri, mime.ifBlank { "*/*" })
        addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION or Intent.FLAG_ACTIVITY_NEW_TASK)
    }
    runCatching { context.startActivity(intent) }
}
