package com.kodass.groovework.ui.common

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil3.compose.AsyncImage

// Аватар пользователя: /uploads/<path>, иначе identicon; без userId — монограмма.
@Composable
fun UserAvatar(
    userId: Long?,
    name: String?,
    avatarPath: String?,
    size: Dp = 44.dp,
    modifier: Modifier = Modifier,
) {
    val serverUrl = LocalServerUrl.current
    if (userId != null && serverUrl.isNotEmpty()) {
        AsyncImage(
            model = avatarUrl(serverUrl, userId, avatarPath),
            contentDescription = name,
            contentScale = ContentScale.Crop,
            modifier = modifier
                .size(size)
                .clip(CircleShape)
                .background(MaterialTheme.colorScheme.surfaceContainerHighest),
        )
    } else {
        Box(
            contentAlignment = Alignment.Center,
            modifier = modifier
                .size(size)
                .clip(CircleShape)
                .background(MaterialTheme.colorScheme.primaryContainer),
        ) {
            Text(
                text = monogram(name),
                color = MaterialTheme.colorScheme.onPrimaryContainer,
                fontSize = (size.value * 0.38f).sp,
            )
        }
    }
}

private fun monogram(name: String?): String {
    val parts = name.orEmpty().trim().split(Regex("\\s+")).filter { it.isNotEmpty() }
    return when {
        parts.size >= 2 -> "${parts[0].first()}${parts[1].first()}".uppercase()
        parts.size == 1 -> parts[0].take(2).uppercase()
        else -> "•"
    }
}
