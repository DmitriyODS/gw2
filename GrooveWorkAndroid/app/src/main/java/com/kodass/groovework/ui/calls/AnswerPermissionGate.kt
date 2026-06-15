package com.kodass.groovework.ui.calls

import android.Manifest
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.kodass.groovework.AppContainer
import com.kodass.groovework.calls.PendingAnswer
import com.kodass.groovework.ui.common.UserAvatar

// Шлюз ответа на звонок: спрашивает доступ к микрофону/камере (+ Bluetooth для
// маршрутизации на гарнитуру) и вызывает answer(). Работает в любой фазе — в том
// числе когда звонок поднят пушем из убитого приложения и экрана входящего нет.
@Composable
fun AnswerPermissionGate(container: AppContainer, pending: PendingAnswer) {
    val controller = container.callController
    val permissions = buildList {
        add(Manifest.permission.RECORD_AUDIO)
        if (pending.video) add(Manifest.permission.CAMERA)
        add(Manifest.permission.BLUETOOTH_CONNECT) // маршрут на BT-гарнитуру (не обязателен для ответа)
    }.toTypedArray()

    val launcher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { result ->
        if (result[Manifest.permission.RECORD_AUDIO] == true) {
            controller.answer(
                callId = pending.callId,
                video = pending.video && result[Manifest.permission.CAMERA] == true,
                fio = pending.fio,
            )
        } else {
            // Без микрофона отвечать нельзя — отклоняем и гасим уведомление.
            controller.declineFromNotification(pending.callId)
        }
        controller.pendingAnswer.value = null
    }
    LaunchedEffect(pending.callId) { launcher.launch(permissions) }

    Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.surfaceContainerLowest) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = androidx.compose.foundation.layout.Arrangement.Center,
            modifier = Modifier.fillMaxSize().padding(24.dp),
        ) {
            UserAvatar(userId = null, name = pending.fio, avatarPath = null, size = 120.dp)
            Text(
                text = pending.fio ?: "Звонок",
                style = MaterialTheme.typography.headlineSmall,
                modifier = Modifier.padding(top = 20.dp),
            )
            CircularProgressIndicator(modifier = Modifier.padding(top = 24.dp))
            Text(
                text = "Соединение…",
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 16.dp),
            )
        }
    }
}
