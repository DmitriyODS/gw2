package com.kodass.groovework.ui.profile

import android.graphics.Bitmap
import android.net.Uri
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.gestures.detectTransformGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.Row
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
import com.kodass.groovework.data.files.cropSquareToJpeg
import com.kodass.groovework.data.files.loadDownscaledBitmap
import kotlinx.coroutines.launch
import kotlin.math.max
import kotlin.math.roundToInt

// Конструктор аватара: зум (щипком) и панорамирование внутри круглой области,
// затем обрезка выбранного квадрата в JPEG.
@Composable
fun AvatarCropDialog(
    uri: Uri,
    onCancel: () -> Unit,
    onCropped: (ByteArray) -> Unit,
) {
    val context = LocalContext.current
    val scope = androidx.compose.runtime.rememberCoroutineScope()
    var bitmap by remember { mutableStateOf<Bitmap?>(null) }
    var processing by remember { mutableStateOf(false) }

    var zoom by remember { mutableFloatStateOf(1f) }
    var offset by remember { mutableStateOf(Offset.Zero) }
    var viewportPx by remember { mutableFloatStateOf(0f) }

    LaunchedEffect(uri) {
        bitmap = loadDownscaledBitmap(context, uri)
    }
    val imageBitmap = remember(bitmap) { bitmap?.asImageBitmap() }

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
                    text = "Двигайте и масштабируйте, чтобы выбрать область.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = Color.White.copy(alpha = 0.7f),
                )

                Box(
                    modifier = Modifier.fillMaxWidth().weight(1f),
                    contentAlignment = Alignment.Center,
                ) {
                    val img = imageBitmap
                    if (img == null) {
                        CircularProgressIndicator(color = Color.White)
                    } else {
                        Canvas(
                            modifier = Modifier
                                .fillMaxWidth()
                                .aspectRatio(1f)
                                .onSizeChanged { viewportPx = it.width.toFloat() }
                                .pointerInput(bitmap) {
                                    detectTransformGestures { _, pan, gestureZoom, _ ->
                                        val bmp = bitmap ?: return@detectTransformGestures
                                        val v = viewportPx
                                        if (v <= 0f) return@detectTransformGestures
                                        val newZoom = (zoom * gestureZoom).coerceIn(1f, 6f)
                                        val baseScale = max(v / bmp.width, v / bmp.height)
                                        val eff = baseScale * newZoom
                                        val maxX = ((bmp.width * eff - v) / 2).coerceAtLeast(0f)
                                        val maxY = ((bmp.height * eff - v) / 2).coerceAtLeast(0f)
                                        zoom = newZoom
                                        offset = Offset(
                                            (offset.x + pan.x).coerceIn(-maxX, maxX),
                                            (offset.y + pan.y).coerceIn(-maxY, maxY),
                                        )
                                    }
                                },
                        ) {
                            val bmp = bitmap ?: return@Canvas
                            val v = size.width
                            val baseScale = max(v / bmp.width, v / bmp.height)
                            val eff = baseScale * zoom
                            val dispW = bmp.width * eff
                            val dispH = bmp.height * eff
                            val left = (v - dispW) / 2 + offset.x
                            val top = (v - dispH) / 2 + offset.y
                            drawImage(
                                image = img,
                                dstOffset = IntOffset(left.roundToInt(), top.roundToInt()),
                                dstSize = IntSize(dispW.roundToInt(), dispH.roundToInt()),
                            )
                            // Затемнение вне круга + контур круга.
                            val mask = Path().apply {
                                addRect(Rect(0f, 0f, size.width, size.height))
                                addOval(Rect(0f, 0f, size.width, size.height))
                                fillType = PathFillType.EvenOdd
                            }
                            drawPath(mask, color = Color.Black.copy(alpha = 0.55f))
                            drawCircle(
                                color = Color.White.copy(alpha = 0.9f),
                                radius = size.width / 2,
                                center = Offset(size.width / 2, size.height / 2),
                                style = Stroke(width = 2.dp.toPx()),
                            )
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
                            val v = viewportPx
                            if (v <= 0f) return@Button
                            processing = true
                            val baseScale = max(v / bmp.width, v / bmp.height)
                            val eff = baseScale * zoom
                            val dispW = bmp.width * eff
                            val dispH = bmp.height * eff
                            val left = (v - dispW) / 2 + offset.x
                            val top = (v - dispH) / 2 + offset.y
                            val srcLeft = ((0f - left) / eff).roundToInt()
                            val srcTop = ((0f - top) / eff).roundToInt()
                            val srcSize = (v / eff).roundToInt()
                            scope.launch {
                                val bytes = cropSquareToJpeg(bmp, srcLeft, srcTop, srcSize)
                                onCropped(bytes)
                            }
                        },
                        enabled = !processing && imageBitmap != null,
                        modifier = Modifier.weight(1f),
                    ) {
                        Text(if (processing) "Сохраняю…" else "Сохранить")
                    }
                }
            }
        }
    }
}
