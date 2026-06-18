package com.kodass.groovework.ui.registries

import android.content.Context
import android.graphics.Bitmap
import android.net.Uri
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.gestures.detectDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Rect
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.PathFillType
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.core.content.FileProvider
import com.kodass.groovework.data.files.cropRectToJpeg
import com.kodass.groovework.data.files.loadDownscaledBitmap
import kotlinx.coroutines.launch
import java.io.File
import kotlin.math.abs
import kotlin.math.roundToInt

// Временный URI для снимка камеры (cacheDir/camera, отдаётся через FileProvider).
fun createCameraImageUri(context: Context): Uri {
    val dir = File(context.cacheDir, "camera").apply { mkdirs() }
    // Имя стабильно в рамках одной сессии съёмки — без Date/random (не критично:
    // файл перезаписывается следующим снимком, потребляется сразу после возврата).
    val file = File.createTempFile("shot_", ".jpg", dir)
    return FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
}

private enum class Handle { TL, TR, BL, BR, MOVE, NONE }

// Свободная обрезка прямоугольником: рамка с 4 угловыми маркерами, можно тянуть
// углы и двигать рамку целиком. Результат — JPEG выбранной области.
@Composable
fun ImageCropDialog(
    uri: Uri,
    onCancel: () -> Unit,
    onCropped: (ByteArray) -> Unit,
) {
    val context = LocalContext.current
    val scope = androidx.compose.runtime.rememberCoroutineScope()
    var bitmap by remember { mutableStateOf<Bitmap?>(null) }
    var processing by remember { mutableStateOf(false) }

    LaunchedEffect(uri) { bitmap = loadDownscaledBitmap(context, uri) }
    val imageBitmap = remember(bitmap) { bitmap?.asImageBitmap() }

    var boxSize by remember { mutableStateOf(Size.Zero) }
    // Прямоугольник изображения внутри Canvas (contain-fit) и текущая рамка обрезки.
    var imgRect by remember { mutableStateOf(Rect.Zero) }
    var cropL by remember { mutableFloatStateOf(0f) }
    var cropT by remember { mutableFloatStateOf(0f) }
    var cropR by remember { mutableFloatStateOf(0f) }
    var cropB by remember { mutableFloatStateOf(0f) }

    // Пересчёт contain-fit и стартовой рамки при готовности картинки/размера.
    LaunchedEffect(bitmap, boxSize) {
        val bmp = bitmap ?: return@LaunchedEffect
        if (boxSize.width <= 0f || boxSize.height <= 0f) return@LaunchedEffect
        val scale = minOf(boxSize.width / bmp.width, boxSize.height / bmp.height)
        val w = bmp.width * scale
        val h = bmp.height * scale
        val left = (boxSize.width - w) / 2f
        val top = (boxSize.height - h) / 2f
        imgRect = Rect(left, top, left + w, top + h)
        // Стартовая рамка — 90% картинки.
        val inset = minOf(w, h) * 0.05f
        cropL = left + inset
        cropT = top + inset
        cropR = left + w - inset
        cropB = top + h - inset
    }

    Dialog(
        onDismissRequest = { if (!processing) onCancel() },
        properties = DialogProperties(usePlatformDefaultWidth = false),
    ) {
        Surface(modifier = Modifier.fillMaxSize(), color = Color(0xFF101014)) {
            Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
                Text(
                    text = "Обрезка фото",
                    style = MaterialTheme.typography.titleLarge,
                    color = Color.White,
                    modifier = Modifier.padding(top = 24.dp, bottom = 4.dp),
                )
                Text(
                    text = "Тяните за углы рамки или двигайте её целиком.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = Color.White.copy(alpha = 0.7f),
                )

                Box(
                    modifier = Modifier.fillMaxWidth().weight(1f).padding(top = 12.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    val img = imageBitmap
                    if (img == null) {
                        CircularProgressIndicator(color = Color.White)
                    } else {
                        val minSize = 64f
                        val touch = 56f
                        var active by remember { mutableStateOf(Handle.NONE) }
                        Canvas(
                            modifier = Modifier
                                .fillMaxSize()
                                .onSizeChanged { boxSize = Size(it.width.toFloat(), it.height.toFloat()) }
                                .pointerInput(imgRect) {
                                    detectDragGestures(
                                        onDragStart = { pos ->
                                            active = pickHandle(pos, cropL, cropT, cropR, cropB, touch)
                                        },
                                        onDragEnd = { active = Handle.NONE },
                                        onDragCancel = { active = Handle.NONE },
                                    ) { change, drag ->
                                        change.consume()
                                        val r = imgRect
                                        when (active) {
                                            Handle.TL -> {
                                                cropL = (cropL + drag.x).coerceIn(r.left, cropR - minSize)
                                                cropT = (cropT + drag.y).coerceIn(r.top, cropB - minSize)
                                            }
                                            Handle.TR -> {
                                                cropR = (cropR + drag.x).coerceIn(cropL + minSize, r.right)
                                                cropT = (cropT + drag.y).coerceIn(r.top, cropB - minSize)
                                            }
                                            Handle.BL -> {
                                                cropL = (cropL + drag.x).coerceIn(r.left, cropR - minSize)
                                                cropB = (cropB + drag.y).coerceIn(cropT + minSize, r.bottom)
                                            }
                                            Handle.BR -> {
                                                cropR = (cropR + drag.x).coerceIn(cropL + minSize, r.right)
                                                cropB = (cropB + drag.y).coerceIn(cropT + minSize, r.bottom)
                                            }
                                            Handle.MOVE -> {
                                                val w = cropR - cropL
                                                val h = cropB - cropT
                                                val nl = (cropL + drag.x).coerceIn(r.left, r.right - w)
                                                val nt = (cropT + drag.y).coerceIn(r.top, r.bottom - h)
                                                cropL = nl; cropT = nt; cropR = nl + w; cropB = nt + h
                                            }
                                            Handle.NONE -> {}
                                        }
                                    }
                                },
                        ) {
                            if (boxSize != size) boxSize = size
                            // Картинка целиком (contain).
                            drawImage(
                                image = img,
                                dstOffset = IntOffset(imgRect.left.roundToInt(), imgRect.top.roundToInt()),
                                dstSize = IntSize(imgRect.width.roundToInt(), imgRect.height.roundToInt()),
                            )
                            // Затемнение вне рамки.
                            val mask = Path().apply {
                                addRect(Rect(0f, 0f, size.width, size.height))
                                addRect(Rect(cropL, cropT, cropR, cropB))
                                fillType = PathFillType.EvenOdd
                            }
                            drawPath(mask, color = Color.Black.copy(alpha = 0.55f))
                            // Рамка + угловые маркеры.
                            drawRect(
                                color = Color.White.copy(alpha = 0.95f),
                                topLeft = Offset(cropL, cropT),
                                size = Size(cropR - cropL, cropB - cropT),
                                style = Stroke(width = 2.dp.toPx()),
                            )
                            val hs = 10.dp.toPx()
                            listOf(
                                Offset(cropL, cropT), Offset(cropR, cropT),
                                Offset(cropL, cropB), Offset(cropR, cropB),
                            ).forEach { c ->
                                drawRect(
                                    color = Color.White,
                                    topLeft = Offset(c.x - hs / 2, c.y - hs / 2),
                                    size = Size(hs, hs),
                                )
                            }
                        }
                    }
                }

                Row(
                    modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    TextButton(
                        onClick = { if (!processing) onCancel() },
                        modifier = Modifier.weight(1f),
                    ) {
                        Text("Отмена", color = Color.White)
                    }
                    Button(
                        onClick = {
                            val bmp = bitmap ?: return@Button
                            val r = imgRect
                            if (r.width <= 0f) return@Button
                            processing = true
                            // view-координаты рамки → пиксели исходника.
                            val scale = bmp.width / r.width
                            val srcLeft = ((cropL - r.left) * scale).roundToInt()
                            val srcTop = ((cropT - r.top) * scale).roundToInt()
                            val srcW = ((cropR - cropL) * scale).roundToInt()
                            val srcH = ((cropB - cropT) * scale).roundToInt()
                            scope.launch {
                                val bytes = cropRectToJpeg(bmp, srcLeft, srcTop, srcW, srcH)
                                onCropped(bytes)
                            }
                        },
                        enabled = !processing && imageBitmap != null,
                        modifier = Modifier.weight(1f),
                    ) {
                        Text(if (processing) "Сохраняю…" else "Готово")
                    }
                }
            }
        }
    }
}

private fun pickHandle(
    pos: Offset,
    l: Float, t: Float, r: Float, b: Float,
    radius: Float,
): Handle {
    fun near(x: Float, y: Float) = abs(pos.x - x) <= radius && abs(pos.y - y) <= radius
    return when {
        near(l, t) -> Handle.TL
        near(r, t) -> Handle.TR
        near(l, b) -> Handle.BL
        near(r, b) -> Handle.BR
        pos.x in l..r && pos.y in t..b -> Handle.MOVE
        else -> Handle.NONE
    }
}
