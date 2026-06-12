package com.kodass.groovework.ui.calls

import android.Manifest
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.CallEnd
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.calls.CallPhase
import com.kodass.groovework.ui.common.UserAvatar

@Composable
fun IncomingCallScreen(container: AppContainer) {
    val manager = container.callManager
    val phase by manager.phase.collectAsStateWithLifecycle()
    val call = (phase as? CallPhase.Incoming)?.call ?: return
    val video = call.media == "video"
    val initiator = call.participants.firstOrNull { it.userId == call.initiatorId }

    val permissions = if (video) {
        arrayOf(Manifest.permission.RECORD_AUDIO, Manifest.permission.CAMERA)
    } else {
        arrayOf(Manifest.permission.RECORD_AUDIO)
    }
    val acceptLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { result ->
        if (result[Manifest.permission.RECORD_AUDIO] == true) {
            manager.cameraEnabled.value = video && result[Manifest.permission.CAMERA] == true
            manager.accept()
        }
    }

    // «Ответить» из шторки — сразу запрашиваем разрешения и принимаем.
    val autoAccept by manager.autoAcceptRequested.collectAsStateWithLifecycle()
    LaunchedEffect(autoAccept) {
        if (autoAccept) {
            manager.autoAcceptRequested.value = false
            acceptLauncher.launch(permissions)
        }
    }

    Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.surfaceContainerLowest) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            modifier = Modifier.fillMaxSize().padding(24.dp),
        ) {
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center,
                modifier = Modifier.weight(1f),
            ) {
                UserAvatar(
                    userId = initiator?.userId,
                    name = call.initiatorFio,
                    avatarPath = initiator?.avatarPath,
                    size = 120.dp,
                )
                Text(
                    text = call.initiatorFio ?: "Входящий звонок",
                    style = MaterialTheme.typography.headlineSmall,
                    modifier = Modifier.padding(top = 20.dp),
                )
                Text(
                    text = if (video) "Входящий видеозвонок" else "Входящий звонок",
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 6.dp),
                )
            }
            Row(
                horizontalArrangement = Arrangement.SpaceEvenly,
                modifier = Modifier
                    .fillMaxWidth()
                    .navigationBarsPadding()
                    .padding(bottom = 32.dp),
            ) {
                FloatingActionButton(
                    onClick = { manager.decline() },
                    containerColor = MaterialTheme.colorScheme.error,
                    contentColor = MaterialTheme.colorScheme.onError,
                    modifier = Modifier.size(72.dp),
                ) {
                    Icon(Icons.Filled.CallEnd, contentDescription = "Отклонить", modifier = Modifier.size(32.dp))
                }
                FloatingActionButton(
                    onClick = { acceptLauncher.launch(permissions) },
                    containerColor = MaterialTheme.colorScheme.primary,
                    contentColor = MaterialTheme.colorScheme.onPrimary,
                    modifier = Modifier.size(72.dp),
                ) {
                    Icon(Icons.Filled.Call, contentDescription = "Принять", modifier = Modifier.size(32.dp))
                }
            }
        }
    }
}
