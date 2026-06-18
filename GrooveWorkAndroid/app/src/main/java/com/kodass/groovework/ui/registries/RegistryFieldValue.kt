package com.kodass.groovework.ui.registries

import android.content.Intent
import android.net.Uri
import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.gestures.rememberTransformableState
import androidx.compose.foundation.gestures.transformable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckBox
import androidx.compose.material.icons.filled.CheckBoxOutlineBlank
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Download
import androidx.compose.material.icons.filled.InsertDriveFile
import androidx.compose.material.icons.filled.Link
import androidx.compose.material.icons.filled.OpenInNew
import androidx.compose.material3.AssistChip
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.RegistryFieldDto
import com.kodass.groovework.data.dto.RegistryFieldType
import com.kodass.groovework.data.dto.UploadedFileDto
import com.kodass.groovework.ui.common.LocalServerUrl
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement

private fun uploadUrl(serverUrl: String, path: String): String =
    serverUrl.trimEnd('/') + "/uploads/" + path.trimStart('/')

@Composable
fun RegistryFieldValue(
    container: AppContainer,
    field: RegistryFieldDto,
    value: JsonElement?,
) {
    val serverUrl = LocalServerUrl.current
    when (field.type) {
        RegistryFieldType.IMAGE -> {
            val file = value.asUploadedFile()
            if (file == null) EmptyValue() else ImageValue(serverUrl, file)
        }
        RegistryFieldType.FILE -> {
            val file = value.asUploadedFile()
            if (file == null) EmptyValue() else FileValue(container, serverUrl, file)
        }
        RegistryFieldType.CHECKBOX -> {
            val on = value.asBool()
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                Icon(
                    if (on) Icons.Filled.CheckBox else Icons.Filled.CheckBoxOutlineBlank,
                    contentDescription = null,
                    tint = if (on) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.size(20.dp),
                )
                Text(if (on) "Да" else "Нет", style = MaterialTheme.typography.bodyLarge)
            }
        }
        RegistryFieldType.SELECT -> {
            val chips = value.asSelectValues()
            if (chips.isEmpty()) EmptyValue()
            else FlowRow(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                chips.forEach { AssistChip(onClick = {}, label = { Text(it) }) }
            }
        }
        RegistryFieldType.LINK -> {
            val url = value.asStringOrNull()
            if (url.isNullOrBlank()) EmptyValue() else LinkValue(url)
        }
        RegistryFieldType.DATETIME -> {
            val text = formatDateTime(value.asStringOrNull(), field.config)
            if (text.isBlank()) EmptyValue() else Text(text, style = MaterialTheme.typography.bodyLarge)
        }
        else -> {
            val text = value.asStringOrNull()
            if (text.isNullOrBlank()) EmptyValue() else Text(text, style = MaterialTheme.typography.bodyLarge)
        }
    }
}

@Composable
private fun EmptyValue() {
    Text("—", style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
}

@Composable
private fun ImageValue(serverUrl: String, file: UploadedFileDto) {
    var viewer by remember { mutableStateOf(false) }
    val url = uploadUrl(serverUrl, file.path)
    AsyncImage(
        model = url,
        contentDescription = file.name,
        contentScale = ContentScale.Crop,
        modifier = Modifier
            .heightIn(max = 200.dp)
            .widthIn(max = 260.dp)
            .clip(RoundedCornerShape(12.dp))
            .background(MaterialTheme.colorScheme.surfaceContainerHighest)
            .clickable { viewer = true },
    )
    if (viewer) RegistryImageViewer(url = url, caption = file.name) { viewer = false }
}

@Composable
private fun FileValue(container: AppContainer, serverUrl: String, file: UploadedFileDto) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var busy by remember { mutableStateOf(false) }
    val url = uploadUrl(serverUrl, file.path)

    Surface(
        color = MaterialTheme.colorScheme.surfaceContainerHigh,
        shape = RoundedCornerShape(12.dp),
        onClick = {
            if (busy) return@Surface
            busy = true
            scope.launch {
                try {
                    container.downloader.download(url, file.name.ifBlank { "file" }, file.mime, toImages = false) {}
                    Toast.makeText(context, "Сохранено в «Загрузки»", Toast.LENGTH_SHORT).show()
                } catch (_: Exception) {
                    Toast.makeText(context, "Не удалось скачать", Toast.LENGTH_SHORT).show()
                } finally {
                    busy = false
                }
            }
        },
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 10.dp),
        ) {
            Icon(Icons.Filled.InsertDriveFile, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
            Text(
                file.name.ifBlank { "Файл" },
                style = MaterialTheme.typography.bodyLarge,
                modifier = Modifier.weight(1f, fill = false),
            )
            if (busy) {
                CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                Icon(Icons.Filled.Download, contentDescription = "Скачать", tint = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

@Composable
private fun LinkValue(url: String) {
    val context = LocalContext.current
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(6.dp)) {
        Icon(
            Icons.Filled.Link,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.primary,
            modifier = Modifier.size(20.dp),
        )
        Text(
            url,
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.primary,
            textDecoration = TextDecoration.Underline,
            modifier = Modifier
                .weight(1f, fill = false)
                .clickable {
                    runCatching {
                        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                    }
                },
        )
        IconButton(onClick = {
            runCatching { context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url))) }
        }) {
            Icon(Icons.Filled.OpenInNew, contentDescription = "Открыть", modifier = Modifier.size(20.dp))
        }
    }
}

// Полноэкранный просмотр картинки реестра (зум/панорама/крестик).
@Composable
private fun RegistryImageViewer(url: String, caption: String, onDismiss: () -> Unit) {
    var scale by remember { mutableFloatStateOf(1f) }
    var offset by remember { mutableStateOf(Offset.Zero) }
    val transformState = rememberTransformableState { _, zoomChange, panChange, _ ->
        scale = (scale * zoomChange).coerceIn(1f, 5f)
        offset = if (scale > 1f) offset + panChange else Offset.Zero
    }
    androidx.compose.ui.window.Dialog(
        onDismissRequest = onDismiss,
        properties = androidx.compose.ui.window.DialogProperties(usePlatformDefaultWidth = false),
    ) {
        Box(modifier = Modifier.fillMaxSize().background(Color.Black)) {
            AsyncImage(
                model = url,
                contentDescription = caption,
                contentScale = ContentScale.Fit,
                modifier = Modifier
                    .fillMaxSize()
                    .graphicsLayer {
                        scaleX = scale; scaleY = scale
                        translationX = offset.x; translationY = offset.y
                    }
                    .transformable(transformState)
                    .pointerInput(Unit) {
                        detectTapGestures(onDoubleTap = {
                            if (scale > 1f) { scale = 1f; offset = Offset.Zero } else scale = 2.5f
                        })
                    },
            )
            Row(modifier = Modifier.fillMaxWidth().statusBarsPadding().padding(4.dp)) {
                IconButton(onClick = onDismiss) {
                    Icon(Icons.Filled.Close, contentDescription = "Закрыть", tint = Color.White)
                }
                Spacer(modifier = Modifier.weight(1f))
            }
        }
    }
}
