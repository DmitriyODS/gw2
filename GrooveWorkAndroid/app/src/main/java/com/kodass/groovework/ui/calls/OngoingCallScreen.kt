package com.kodass.groovework.ui.calls

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.VolumeOff
import androidx.compose.material.icons.automirrored.filled.VolumeUp
import androidx.compose.material.icons.filled.CallEnd
import androidx.compose.material.icons.filled.Cameraswitch
import androidx.compose.material.icons.filled.KeyboardArrowDown
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.MicOff
import androidx.compose.material.icons.filled.Videocam
import androidx.compose.material.icons.filled.VideocamOff
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
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.kodass.groovework.AppContainer
import com.kodass.groovework.calls.CallConnection
import com.kodass.groovework.calls.CallState
import com.kodass.groovework.ui.common.UserAvatar
import kotlinx.coroutines.delay

// Экран исходящего/активного звонка (поверх локскрина). Фазы:
//  Звонок… (Dialing/Ringing-Outgoing) → Соединение… (Connecting) → таймер (Active).
@Composable
fun OngoingCallScreen(container: AppContainer) {
    val controller = container.callController
    val ui by controller.ui.collectAsStateWithLifecycle()
    val room by controller.room.collectAsStateWithLifecycle()
    val state = ui.state
    val media = ui.media
    val call = state.call
    if (state is CallState.Idle) return
    val party = controller.peer

    // Таймер длительности: бежит только в Active (activeSinceMs выставлен на первом
    // аудио собеседника, не раньше — иначе секунды шли бы в тишине «Соединения»).
    val activeSince = (state as? CallState.Active)?.activeSinceMs
    var duration by remember { mutableLongStateOf(0L) }
    LaunchedEffect(activeSince) {
        while (true) {
            duration = activeSince?.let { (System.currentTimeMillis() - it) / 1000 } ?: 0L
            delay(1000)
        }
    }

    val statusText = when {
        state is CallState.Dialing || state is CallState.Ringing -> "Звонок…"
        state is CallState.Connecting -> "Соединение…"
        media.paused -> "На паузе"
        media.connection == CallConnection.Reconnecting -> "Соединение…"
        state is CallState.Active -> formatDuration(duration)
        else -> ""
    }

    // Видео: основное на весь экран, второе — в углу (PiP); тап меняет местами.
    var primaryIsRemote by remember { mutableStateOf(true) }
    val remoteTrack = media.remoteVideo
    val localVideo = if (media.cameraEnabled) media.localVideo else null
    val hasBoth = remoteTrack != null && localVideo != null
    val primaryRemote = primaryIsRemote || localVideo == null
    val primaryTrack = if (primaryRemote) remoteTrack else localVideo
    val pipTrack = if (hasBoth) (if (primaryRemote) localVideo else remoteTrack) else null

    Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.surfaceContainerLowest) {
        Box(modifier = Modifier.fillMaxSize()) {
            if (primaryTrack != null && room != null) {
                CallVideo(room = room!!, track = primaryTrack, mirror = !primaryRemote, modifier = Modifier.fillMaxSize())
            } else {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.Center,
                    modifier = Modifier.fillMaxSize(),
                ) {
                    UserAvatar(
                        userId = party?.userId,
                        name = party?.fio ?: call?.initiatorFio,
                        avatarPath = party?.avatarPath,
                        size = 110.dp,
                    )
                    Text(
                        text = party?.fio ?: call?.initiatorFio ?: "Звонок",
                        style = MaterialTheme.typography.headlineSmall,
                        modifier = Modifier.padding(top = 16.dp),
                    )
                    Text(
                        text = statusText,
                        style = MaterialTheme.typography.bodyLarge,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 6.dp),
                    )
                }
            }

            // Шапка: свернуть + имя/статус поверх видео.
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.statusBarsPadding().padding(8.dp),
            ) {
                IconButton(onClick = { controller.callUiVisible.value = false }) {
                    Icon(
                        Icons.Filled.KeyboardArrowDown,
                        contentDescription = "Свернуть",
                        tint = MaterialTheme.colorScheme.onSurface,
                    )
                }
                if (primaryTrack != null) {
                    Column(modifier = Modifier.padding(start = 4.dp)) {
                        Text(text = party?.fio ?: "", style = MaterialTheme.typography.titleMedium)
                        Text(
                            text = statusText,
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            }

            if (pipTrack != null && room != null) {
                Box(
                    modifier = Modifier
                        .align(Alignment.TopEnd)
                        .statusBarsPadding()
                        .padding(12.dp)
                        .width(110.dp)
                        .height(150.dp)
                        .clip(RoundedCornerShape(12.dp))
                        .background(MaterialTheme.colorScheme.surfaceContainerHigh)
                        .clickable { primaryIsRemote = !primaryIsRemote },
                ) {
                    CallVideo(room = room!!, track = pipTrack, mirror = primaryRemote, modifier = Modifier.fillMaxSize())
                }
            }

            // Панель управления.
            Surface(
                color = MaterialTheme.colorScheme.surfaceContainer.copy(alpha = 0.92f),
                shape = RoundedCornerShape(28.dp),
                modifier = Modifier.align(Alignment.BottomCenter).navigationBarsPadding().padding(bottom = 24.dp),
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 10.dp),
                ) {
                    ControlButton(
                        icon = if (media.micEnabled) Icons.Filled.Mic else Icons.Filled.MicOff,
                        active = media.micEnabled,
                        contentDescription = "Микрофон",
                        onClick = { controller.toggleMic() },
                    )
                    ControlButton(
                        icon = if (media.cameraEnabled) Icons.Filled.Videocam else Icons.Filled.VideocamOff,
                        active = media.cameraEnabled,
                        contentDescription = "Камера",
                        onClick = { controller.toggleCamera() },
                    )
                    if (media.cameraEnabled) {
                        ControlButton(
                            icon = Icons.Filled.Cameraswitch,
                            active = true,
                            contentDescription = "Сменить камеру",
                            onClick = { controller.flipCamera() },
                        )
                    }
                    ControlButton(
                        icon = if (media.speakerOn) Icons.AutoMirrored.Filled.VolumeUp else Icons.AutoMirrored.Filled.VolumeOff,
                        active = media.speakerOn,
                        contentDescription = "Динамик",
                        onClick = { controller.setSpeaker(!media.speakerOn) },
                    )
                    FloatingActionButton(
                        onClick = { controller.hangup() },
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
    icon: ImageVector,
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
