package com.kodass.groovework.ui.common

import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.gestures.rememberTransformableState
import androidx.compose.foundation.gestures.transformable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Download
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import coil3.compose.AsyncImage
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.AttachmentDto
import com.kodass.groovework.data.files.DownloadState
import kotlinx.coroutines.launch

// Полноэкранный просмотр картинки из чата: щипок/двойной тап — зум и панорама,
// крестик — закрыть, иконка скачивания — сохранить в галерею с прогрессом.
@Composable
fun ImageViewer(
    container: AppContainer,
    attachment: AttachmentDto,
    onDismiss: () -> Unit,
) {
    val serverUrl = LocalServerUrl.current
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val fullUrl = serverUrl.trimEnd('/') + "/" + attachment.url.trimStart('/')

    var dl by remember { mutableStateOf<DownloadState>(DownloadState.Idle) }
    var scale by remember { mutableFloatStateOf(1f) }
    var offset by remember { mutableStateOf(Offset.Zero) }
    val transformState = rememberTransformableState { _, zoomChange, panChange, _ ->
        scale = (scale * zoomChange).coerceIn(1f, 5f)
        offset = if (scale > 1f) offset + panChange else Offset.Zero
    }

    fun startDownload() {
        if (dl is DownloadState.Running) return
        dl = DownloadState.Running(-1f)
        scope.launch {
            try {
                val uri = container.downloader.download(
                    url = fullUrl,
                    fileName = attachment.fileName,
                    mime = attachment.mimeType,
                    toImages = true,
                ) { p -> dl = DownloadState.Running(p) }
                dl = DownloadState.Done(uri, attachment.mimeType)
                Toast.makeText(context, "Сохранено в галерею", Toast.LENGTH_SHORT).show()
            } catch (_: Exception) {
                dl = DownloadState.Idle
                Toast.makeText(context, "Не удалось скачать", Toast.LENGTH_SHORT).show()
            }
        }
    }

    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false),
    ) {
        Box(modifier = Modifier.fillMaxSize().background(Color.Black)) {
            AsyncImage(
                model = fullUrl,
                contentDescription = attachment.fileName,
                contentScale = ContentScale.Fit,
                modifier = Modifier
                    .fillMaxSize()
                    .graphicsLayer {
                        scaleX = scale
                        scaleY = scale
                        translationX = offset.x
                        translationY = offset.y
                    }
                    .transformable(transformState)
                    .pointerInput(Unit) {
                        detectTapGestures(
                            onDoubleTap = {
                                if (scale > 1f) {
                                    scale = 1f
                                    offset = Offset.Zero
                                } else {
                                    scale = 2.5f
                                }
                            },
                        )
                    },
            )

            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .fillMaxWidth()
                    .statusBarsPadding()
                    .padding(horizontal = 4.dp, vertical = 4.dp),
            ) {
                IconButton(onClick = onDismiss) {
                    Icon(Icons.Filled.Close, contentDescription = "Закрыть", tint = Color.White)
                }
                Spacer(modifier = Modifier.weight(1f))
                when (val s = dl) {
                    is DownloadState.Running -> Box(
                        modifier = Modifier.size(48.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        if (s.progress >= 0f) {
                            CircularProgressIndicator(
                                progress = { s.progress },
                                color = Color.White,
                                modifier = Modifier.size(24.dp),
                            )
                        } else {
                            CircularProgressIndicator(color = Color.White, modifier = Modifier.size(24.dp))
                        }
                    }
                    else -> IconButton(onClick = { startDownload() }) {
                        Icon(Icons.Filled.Download, contentDescription = "Скачать", tint = Color.White)
                    }
                }
            }
        }
    }
}
