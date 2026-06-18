package com.kodass.groovework.data.files

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Matrix
import android.media.ExifInterface
import android.net.Uri
import android.provider.OpenableColumns
import android.webkit.MimeTypeMap
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.ByteArrayOutputStream

// Подготовка картинки под аватар: даунскейл до maxDim и сжатие в JPEG, пока не
// уложимся в maxBytes (лимит сервера — 2 МБ).
suspend fun compressForAvatar(
    context: Context,
    uri: Uri,
    maxDim: Int = 1024,
    maxBytes: Int = 2 * 1024 * 1024,
): ByteArray? = withContext(Dispatchers.IO) {
    val resolver = context.contentResolver
    val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
    resolver.openInputStream(uri)?.use { BitmapFactory.decodeStream(it, null, bounds) }
    if (bounds.outWidth <= 0 || bounds.outHeight <= 0) return@withContext null

    var sample = 1
    while (bounds.outWidth / sample > maxDim || bounds.outHeight / sample > maxDim) sample *= 2
    val opts = BitmapFactory.Options().apply { inSampleSize = sample }
    val bitmap = resolver.openInputStream(uri)?.use { BitmapFactory.decodeStream(it, null, opts) }
        ?: return@withContext null

    var quality = 90
    var bytes: ByteArray
    while (true) {
        val out = ByteArrayOutputStream()
        bitmap.compress(Bitmap.CompressFormat.JPEG, quality, out)
        bytes = out.toByteArray()
        if (bytes.size <= maxBytes || quality <= 40) break
        quality -= 10
    }
    bitmap.recycle()
    bytes
}

// Загрузка картинки с даунскейлом до maxDim и учётом EXIF-ориентации — для
// конструктора аватара (зум/обрезка).
suspend fun loadDownscaledBitmap(
    context: Context,
    uri: Uri,
    maxDim: Int = 1600,
): Bitmap? = withContext(Dispatchers.IO) {
    val resolver = context.contentResolver
    val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
    resolver.openInputStream(uri)?.use { BitmapFactory.decodeStream(it, null, bounds) }
    if (bounds.outWidth <= 0 || bounds.outHeight <= 0) return@withContext null

    var sample = 1
    while (bounds.outWidth / sample > maxDim || bounds.outHeight / sample > maxDim) sample *= 2
    val opts = BitmapFactory.Options().apply { inSampleSize = sample }
    val bitmap = resolver.openInputStream(uri)?.use { BitmapFactory.decodeStream(it, null, opts) }
        ?: return@withContext null

    val orientation = runCatching {
        resolver.openInputStream(uri)?.use { ExifInterface(it).getAttributeInt(
            ExifInterface.TAG_ORIENTATION, ExifInterface.ORIENTATION_NORMAL,
        ) }
    }.getOrNull() ?: ExifInterface.ORIENTATION_NORMAL
    applyExifRotation(bitmap, orientation)
}

private fun applyExifRotation(bitmap: Bitmap, orientation: Int): Bitmap {
    val matrix = Matrix()
    when (orientation) {
        ExifInterface.ORIENTATION_ROTATE_90 -> matrix.postRotate(90f)
        ExifInterface.ORIENTATION_ROTATE_180 -> matrix.postRotate(180f)
        ExifInterface.ORIENTATION_ROTATE_270 -> matrix.postRotate(270f)
        else -> return bitmap
    }
    val rotated = Bitmap.createBitmap(bitmap, 0, 0, bitmap.width, bitmap.height, matrix, true)
    if (rotated != bitmap) bitmap.recycle()
    return rotated
}

// Вырезает квадрат из исходной картинки (координаты в пикселях исходника),
// масштабирует до outSize и сжимает в JPEG под лимит размера.
suspend fun cropSquareToJpeg(
    source: Bitmap,
    srcLeft: Int,
    srcTop: Int,
    srcSize: Int,
    outSize: Int = 600,
    maxBytes: Int = 2 * 1024 * 1024,
): ByteArray = withContext(Dispatchers.IO) {
    val left = srcLeft.coerceIn(0, (source.width - 1).coerceAtLeast(0))
    val top = srcTop.coerceIn(0, (source.height - 1).coerceAtLeast(0))
    val size = srcSize.coerceAtMost(minOf(source.width - left, source.height - top)).coerceAtLeast(1)
    val cropped = Bitmap.createBitmap(source, left, top, size, size)
    val scaled = if (size != outSize) {
        Bitmap.createScaledBitmap(cropped, outSize, outSize, true).also {
            if (it != cropped) cropped.recycle()
        }
    } else {
        cropped
    }
    var quality = 92
    var bytes: ByteArray
    while (true) {
        val out = ByteArrayOutputStream()
        scaled.compress(Bitmap.CompressFormat.JPEG, quality, out)
        bytes = out.toByteArray()
        if (bytes.size <= maxBytes || quality <= 40) break
        quality -= 10
    }
    scaled.recycle()
    bytes
}

// Вырезает произвольный прямоугольник (координаты в пикселях исходника),
// масштабирует длинную сторону до maxDim и сжимает в JPEG под лимит размера.
// Для свободной обрезки картинок реестра (поле image не обязано быть квадратным).
suspend fun cropRectToJpeg(
    source: Bitmap,
    srcLeft: Int,
    srcTop: Int,
    srcWidth: Int,
    srcHeight: Int,
    maxDim: Int = 1600,
    maxBytes: Int = 4 * 1024 * 1024,
): ByteArray = withContext(Dispatchers.IO) {
    val left = srcLeft.coerceIn(0, (source.width - 1).coerceAtLeast(0))
    val top = srcTop.coerceIn(0, (source.height - 1).coerceAtLeast(0))
    val width = srcWidth.coerceIn(1, source.width - left)
    val height = srcHeight.coerceIn(1, source.height - top)
    val cropped = Bitmap.createBitmap(source, left, top, width, height)

    val longest = maxOf(width, height)
    val scaled = if (longest > maxDim) {
        val factor = maxDim.toFloat() / longest
        Bitmap.createScaledBitmap(
            cropped,
            (width * factor).toInt().coerceAtLeast(1),
            (height * factor).toInt().coerceAtLeast(1),
            true,
        ).also { if (it != cropped) cropped.recycle() }
    } else {
        cropped
    }

    var quality = 92
    var bytes: ByteArray
    while (true) {
        val out = ByteArrayOutputStream()
        scaled.compress(Bitmap.CompressFormat.JPEG, quality, out)
        bytes = out.toByteArray()
        if (bytes.size <= maxBytes || quality <= 40) break
        quality -= 10
    }
    scaled.recycle()
    bytes
}

// Содержимое выбранного файла произвольного типа (для поля «файл»): сырые байты
// + отображаемое имя + MIME. null — если поток недоступен.
data class PickedFile(val bytes: ByteArray, val name: String, val mime: String)

suspend fun readPickedFile(
    context: Context,
    uri: Uri,
): PickedFile? = withContext(Dispatchers.IO) {
    val resolver = context.contentResolver
    val bytes = resolver.openInputStream(uri)?.use { it.readBytes() } ?: return@withContext null
    val name = queryDisplayName(context, uri) ?: "file"
    val mime = resolver.getType(uri)
        ?: MimeTypeMap.getSingleton().getMimeTypeFromExtension(name.substringAfterLast('.', ""))
        ?: "application/octet-stream"
    PickedFile(bytes, name, mime)
}

private fun queryDisplayName(context: Context, uri: Uri): String? {
    return runCatching {
        context.contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)
            ?.use { cursor ->
                if (cursor.moveToFirst()) {
                    val idx = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
                    if (idx >= 0) cursor.getString(idx) else null
                } else null
            }
    }.getOrNull() ?: uri.lastPathSegment?.substringAfterLast('/')
}
