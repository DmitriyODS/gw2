package com.kodass.groovework.ui.common

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.repeatOnLifecycle

// Однократное обновление при каждом входе в раздел и возврате приложения на
// передний план (раздел перерисовывается при переключении вкладок нижней панели).
// Живые изменения внутри раздела доставляет WebSocket — поллинга нет.
@Composable
fun RefreshOnResume(key: Any? = Unit, block: suspend () -> Unit) {
    val owner = LocalLifecycleOwner.current
    LaunchedEffect(owner, key) {
        owner.lifecycle.repeatOnLifecycle(Lifecycle.State.RESUMED) {
            block()
        }
    }
}
