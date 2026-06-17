package com.kodass.groovework.ui.appearance

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import com.kodass.groovework.ui.theme.fromHex
import com.kodass.groovework.ui.theme.toHex

// Палитра-конструктор: выбор цвета по модели HSV (квадрат насыщенность/яркость +
// ползунок оттенка). Аналог нативного color-input на вебе. Возвращает #RRGGBB.
@Composable
fun ColorPickerDialog(
    title: String,
    initial: String,
    onDismiss: () -> Unit,
    onConfirm: (String) -> Unit,
) {
    val hsv = remember(initial) {
        val c = Color.fromHex(initial)
        val argb = android.graphics.Color.rgb(
            (c.red * 255).toInt(), (c.green * 255).toInt(), (c.blue * 255).toInt(),
        )
        FloatArray(3).also { android.graphics.Color.colorToHSV(argb, it) }
    }
    var hue by remember { mutableStateOf(hsv[0]) }
    var sat by remember { mutableStateOf(hsv[1]) }
    var value by remember { mutableStateOf(hsv[2]) }

    val current = Color(android.graphics.Color.HSVToColor(floatArrayOf(hue, sat, value)))

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(14.dp)) {
                SatValueBox(
                    hue = hue, sat = sat, value = value,
                    onChange = { s, v -> sat = s; value = v },
                )
                HueSlider(hue = hue, onChange = { hue = it })
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Surface(color = current, shape = CircleShape, modifier = Modifier.size(36.dp)) {}
                    Text(
                        text = current.toHex(),
                        style = MaterialTheme.typography.bodyLarge,
                        modifier = Modifier.padding(start = 12.dp),
                    )
                }
            }
        },
        confirmButton = { TextButton(onClick = { onConfirm(current.toHex()) }) { Text("Готово") } },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Отмена") } },
    )
}

@Composable
private fun SatValueBox(hue: Float, sat: Float, value: Float, onChange: (Float, Float) -> Unit) {
    var box by remember { mutableStateOf(IntSize.Zero) }
    val pureHue = Color(android.graphics.Color.HSVToColor(floatArrayOf(hue, 1f, 1f)))
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(170.dp)
            .clip(RoundedCornerShape(12.dp))
            .pointerInput(Unit) {
                awaitPointerEventScope {
                    while (true) {
                        val event = awaitPointerEvent()
                        if (box.width == 0 || box.height == 0) continue
                        val pos = event.changes.first().position
                        val s = (pos.x / box.width).coerceIn(0f, 1f)
                        val v = (1f - pos.y / box.height).coerceIn(0f, 1f)
                        onChange(s, v)
                    }
                }
            },
    ) {
        Canvas(modifier = Modifier.fillMaxWidth().height(170.dp)) {
            box = IntSize(size.width.toInt(), size.height.toInt())
            drawRect(Brush.horizontalGradient(listOf(Color.White, pureHue)))
            drawRect(Brush.verticalGradient(listOf(Color.Transparent, Color.Black)))
            val cx = sat * size.width
            val cy = (1f - value) * size.height
            drawCircle(Color.White, radius = 11f, center = Offset(cx, cy), style = androidx.compose.ui.graphics.drawscope.Stroke(width = 4f))
        }
    }
}

@Composable
private fun HueSlider(hue: Float, onChange: (Float) -> Unit) {
    var w by remember { mutableStateOf(0) }
    val hueColors = remember {
        (0..360 step 30).map { Color(android.graphics.Color.HSVToColor(floatArrayOf(it.toFloat(), 1f, 1f))) }
    }
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(28.dp)
            .clip(RoundedCornerShape(14.dp))
            .pointerInput(Unit) {
                awaitPointerEventScope {
                    while (true) {
                        val event = awaitPointerEvent()
                        if (w == 0) continue
                        val x = event.changes.first().position.x.coerceIn(0f, w.toFloat())
                        onChange((x / w) * 360f)
                    }
                }
            },
    ) {
        Canvas(modifier = Modifier.fillMaxWidth().height(28.dp)) {
            w = size.width.toInt()
            drawRect(Brush.horizontalGradient(hueColors))
            val cx = (hue / 360f) * size.width
            drawCircle(Color.White, radius = 12f, center = Offset(cx, size.height / 2f), style = androidx.compose.ui.graphics.drawscope.Stroke(width = 4f))
        }
    }
}
