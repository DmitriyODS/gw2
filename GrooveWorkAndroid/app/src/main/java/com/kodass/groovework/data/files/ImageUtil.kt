package com.kodass.groovework.data.files

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Matrix
import android.media.ExifInterface
import android.net.Uri
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
