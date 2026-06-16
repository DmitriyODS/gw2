package com.kodass.groovework.ui.common

import android.content.ClipData
import androidx.compose.runtime.Composable
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.platform.ClipEntry
import androidx.compose.ui.platform.LocalClipboard
import kotlinx.coroutines.launch

// Копирование текста в буфер обмена через новый suspend-API Clipboard
// (LocalClipboardManager.setText устарел). Возвращает синхронный колбэк для onClick.
@Composable
fun rememberClipboardCopy(): (String) -> Unit {
    val clipboard = LocalClipboard.current
    val scope = rememberCoroutineScope()
    return { text ->
        scope.launch {
            clipboard.setClipEntry(ClipEntry(ClipData.newPlainText("text", text)))
        }
    }
}
