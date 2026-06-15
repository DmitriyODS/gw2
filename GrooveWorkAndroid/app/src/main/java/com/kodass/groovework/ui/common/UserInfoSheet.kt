package com.kodass.groovework.ui.common

import android.content.Intent
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Chat
import androidx.compose.material.icons.filled.AlternateEmail
import androidx.compose.material.icons.filled.Badge
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.Email
import androidx.compose.material.icons.filled.Phone
import androidx.compose.material.icons.filled.Videocam
import androidx.compose.material3.AssistChip
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.core.net.toUri
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.UserDto

// Карточка пользователя «снизу» (тап по шапке чата): данные, статус онлайн и
// действия. Полный профиль догружаем из directory, пока показываем то, что есть.
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun UserInfoSheet(
    container: AppContainer,
    userId: Long,
    fallback: UserDto?,
    online: Boolean,
    canCall: Boolean,
    onAudioCall: () -> Unit,
    onVideoCall: () -> Unit,
    onDismiss: () -> Unit,
    onWrite: (() -> Unit)? = null,
) {
    val context = LocalContext.current
    var user by remember(userId) { mutableStateOf(fallback) }
    LaunchedEffect(userId) {
        runCatching { container.authApi.directoryUser(userId) }.getOrNull()?.let { user = it }
    }
    val u = user

    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 20.dp)
                .navigationBarsPadding(),
        ) {
            UserAvatar(
                userId = userId,
                name = u?.fio,
                avatarPath = u?.avatarPath,
                size = 96.dp,
            )
            Text(
                text = u?.fio ?: "Пользователь",
                style = MaterialTheme.typography.headlineSmall,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.padding(top = 12.dp),
            )
            u?.role?.name?.let { roleName ->
                AssistChip(
                    onClick = {},
                    enabled = false,
                    label = { Text(roleName) },
                    modifier = Modifier.padding(top = 6.dp),
                )
            }
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.padding(top = 4.dp),
            ) {
                Box(
                    modifier = Modifier
                        .size(8.dp)
                        .clip(CircleShape)
                        .then(
                            Modifier.background(
                                if (online) MaterialTheme.colorScheme.primary
                                else MaterialTheme.colorScheme.outline
                            )
                        ),
                )
                Text(
                    text = if (online) "в сети" else formatLastSeen(u?.lastSeenAt),
                    style = MaterialTheme.typography.bodyMedium,
                    color = if (online) MaterialTheme.colorScheme.primary
                    else MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(start = 6.dp),
                )
            }

            Column(modifier = Modifier.fillMaxWidth().padding(top = 16.dp)) {
                u?.post?.takeIf { it.isNotBlank() }?.let {
                    InfoRow(Icons.Filled.Badge, "Должность", it)
                }
                u?.login?.takeIf { it.isNotBlank() }?.let {
                    InfoRow(Icons.Filled.AlternateEmail, "Логин", "@$it")
                }
                u?.phone?.takeIf { it.isNotBlank() }?.let { phone ->
                    InfoRow(Icons.Filled.Phone, "Телефон", phone) {
                        runCatching { context.startActivity(Intent(Intent.ACTION_DIAL, "tel:$phone".toUri())) }
                    }
                }
                u?.email?.takeIf { it.isNotBlank() }?.let { email ->
                    InfoRow(Icons.Filled.Email, "Email", email) {
                        runCatching { context.startActivity(Intent(Intent.ACTION_SENDTO, "mailto:$email".toUri())) }
                    }
                }
            }

            if (onWrite != null) {
                FilledTonalButton(
                    onClick = onWrite,
                    modifier = Modifier.fillMaxWidth().padding(top = 20.dp),
                ) {
                    Icon(Icons.AutoMirrored.Filled.Chat, contentDescription = null, modifier = Modifier.size(18.dp))
                    Text("Написать", modifier = Modifier.padding(start = 8.dp))
                }
            }
            if (canCall) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.fillMaxWidth().padding(top = 12.dp, bottom = 12.dp),
                ) {
                    FilledTonalButton(onClick = onAudioCall, modifier = Modifier.weight(1f)) {
                        Icon(Icons.Filled.Call, contentDescription = null, modifier = Modifier.size(18.dp))
                        Text("Аудио", modifier = Modifier.padding(start = 8.dp))
                    }
                    FilledTonalButton(onClick = onVideoCall, modifier = Modifier.weight(1f)) {
                        Icon(Icons.Filled.Videocam, contentDescription = null, modifier = Modifier.size(18.dp))
                        Text("Видео", modifier = Modifier.padding(start = 8.dp))
                    }
                }
            } else {
                Box(modifier = Modifier.padding(bottom = 12.dp))
            }
        }
    }
}

@Composable
private fun InfoRow(icon: ImageVector, label: String, value: String, onClick: (() -> Unit)? = null) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceContainerHigh,
        shape = RoundedCornerShape(12.dp),
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp)
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(horizontal = 14.dp, vertical = 12.dp),
        ) {
            Icon(icon, contentDescription = null, tint = MaterialTheme.colorScheme.primary, modifier = Modifier.size(20.dp))
            Column(modifier = Modifier.padding(start = 12.dp)) {
                Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                Text(value, style = MaterialTheme.typography.bodyMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
            }
        }
    }
}
