package com.kodass.groovework.ui.calls

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CallEnd
import androidx.compose.material.icons.filled.Cameraswitch
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.MicOff
import androidx.compose.material.icons.filled.Videocam
import androidx.compose.material.icons.filled.VideocamOff
import androidx.compose.material.icons.automirrored.filled.VolumeUp
import androidx.compose.material.icons.automirrored.filled.VolumeOff
import androidx.compose.material3.FilledIconButton
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.IconButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.calls.CallPhase
import com.kodass.groovework.ui.common.UserAvatar
import kotlinx.coroutines.delay

@Composable
fun CallScreen(container: AppContainer) {
    val manager = container.callManager
    val phase by manager.phase.collectAsStateWithLifecycle()
    val call = when (val p = phase) {
        is CallPhase.Outgoing -> p.call
        is CallPhase.Active -> p.call
        else -> return
    }
    val isOutgoing = phase is CallPhase.Outgoing
    val peer = manager.peer
    val micEnabled by manager.micEnabled.collectAsStateWithLifecycle()
    val cameraEnabled by manager.cameraEnabled.collectAsStateWithLifecycle()
    val speakerOn by manager.speakerOn.collectAsStateWithLifecycle()
    val localTrack by manager.localVideoTrack.collectAsStateWithLifecycle()
    val remoteTracks by manager.remoteVideoTracks.collectAsStateWithLifecycle()
    val activeSince by manager.activeSince.collectAsStateWithLifecycle()
    val room = manager.roomOrNull

    // Таймер длительности.
    var duration by remember { mutableLongStateOf(0L) }
    LaunchedEffect(activeSince) {
        while (true) {
            duration = activeSince?.let { (System.currentTimeMillis() - it) / 1000 } ?: 0L
            delay(1000)
        }
    }

    Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.surfaceContainerLowest) {
        Box(modifier = Modifier.fillMaxSize()) {
            // Видео собеседника на весь экран, иначе — аватар и имя.
            val remoteTrack = remoteTracks.values.firstOrNull()
            if (remoteTrack != null && room != null) {
                VideoTrackView(room = room, track = remoteTrack, modifier = Modifier.fillMaxSize())
            } else {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.Center,
                    modifier = Modifier.fillMaxSize(),
                ) {
                    UserAvatar(
                        userId = peer?.userId,
                        name = peer?.fio,
                        avatarPath = peer?.avatarPath,
                        size = 110.dp,
                    )
                    Text(
                        text = peer?.fio ?: "Звонок",
                        style = MaterialTheme.typography.headlineSmall,
                        modifier = Modifier.padding(top = 16.dp),
                    )
                    Text(
                        text = if (isOutgoing) "Звоним…" else formatDuration(duration),
                        style = MaterialTheme.typography.bodyLarge,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 6.dp),
                    )
                }
            }

            // Шапка: свернуть + имя/таймер поверх видео.
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .statusBarsPadding()
                    .padding(8.dp),
            ) {
                IconButton(onClick = { manager.callUiVisible.value = false }) {
                    Icon(
                        Icons.Filled.KeyboardArrowDown,
                        contentDescription = "Свернуть",
                        tint = MaterialTheme.colorScheme.onSurface,
                    )
                }
                if (remoteTrack != null) {
                    Column(modifier = Modifier.padding(start = 4.dp)) {
                        Text(text = peer?.fio ?: "", style = MaterialTheme.typography.titleMedium)
                        Text(
                            text = if (isOutgoing) "Звоним…" else formatDuration(duration),
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            }

            // Локальное превью.
            if (cameraEnabled && localTrack != null && room != null) {
                Box(
                    modifier = Modifier
                        .align(Alignment.TopEnd)
                        .statusBarsPadding()
                        .padding(12.dp)
                        .width(110.dp)
                        .height(150.dp)
                        .clip(RoundedCornerShape(12.dp))
                        .background(MaterialTheme.colorScheme.surfaceContainerHigh),
                ) {
                    VideoTrackView(room = room, track = localTrack, modifier = Modifier.fillMaxSize())
                }
            }

            // Панель управления.
            Surface(
                color = MaterialTheme.colorScheme.surfaceContainer.copy(alpha = 0.92f),
                shape = RoundedCornerShape(28.dp),
                modifier = Modifier
                    .align(Alignment.BottomCenter)
                    .navigationBarsPadding()
                    .padding(bottom = 24.dp),
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 10.dp),
                ) {
                    ControlButton(
                        icon = if (micEnabled) Icons.Filled.Mic else Icons.Filled.MicOff,
                        active = micEnabled,
                        contentDescription = "Микрофон",
                        onClick = { manager.toggleMic() },
                    )
                    ControlButton(
                        icon = if (cameraEnabled) Icons.Filled.Videocam else Icons.Filled.VideocamOff,
                        active = cameraEnabled,
                        contentDescription = "Камера",
                        onClick = { manager.toggleCamera() },
                    )
                    if (cameraEnabled) {
                        ControlButton(
                            icon = Icons.Filled.Cameraswitch,
                            active = true,
                            contentDescription = "Сменить камеру",
                            onClick = { manager.flipCamera() },
                        )
                    }
                    ControlButton(
                        icon = if (speakerOn) Icons.AutoMirrored.Filled.VolumeUp else Icons.AutoMirrored.Filled.VolumeOff,
                        active = speakerOn,
                        contentDescription = "Динамик",
                        onClick = { manager.setSpeaker(!speakerOn) },
                    )
                    FloatingActionButton(
                        onClick = { manager.hangup() },
                        containerColor = MaterialTheme.colorScheme.error,
                        contentColor = MaterialTheme.colorScheme.onError,
                        modifier = Modifier.size(56.dp),
                    ) {
                        Icon(Icons.Filled.CallEnd, contentDescription = "Завершить")
                    }
                }
            }
        }
    }
}

@Composable
private fun ControlButton(
    icon: androidx.compose.ui.graphics.vector.ImageVector,
    active: Boolean,
    contentDescription: String,
    onClick: () -> Unit,
) {
    FilledIconButton(
        onClick = onClick,
        colors = IconButtonDefaults.filledIconButtonColors(
            containerColor = if (active) MaterialTheme.colorScheme.secondaryContainer
            else MaterialTheme.colorScheme.surfaceContainerHighest,
            contentColor = if (active) MaterialTheme.colorScheme.onSecondaryContainer
            else MaterialTheme.colorScheme.onSurfaceVariant,
        ),
        modifier = Modifier.size(48.dp),
    ) {
        Icon(icon, contentDescription = contentDescription)
    }
}

private fun formatDuration(seconds: Long): String {
    val m = seconds / 60
    val s = seconds % 60
    return "%d:%02d".format(m, s)
}
