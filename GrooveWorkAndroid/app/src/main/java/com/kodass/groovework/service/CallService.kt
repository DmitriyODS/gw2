package com.kodass.groovework.service

import android.app.Service
import android.content.Intent
import android.content.pm.PackageManager
import android.content.pm.ServiceInfo
import android.os.IBinder
import androidx.core.content.ContextCompat
import com.kodass.groovework.GrooveApp
import com.kodass.groovework.data.calls.CallPhase
import com.kodass.groovework.notifications.Notifier
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

// Foreground-сервис активного звонка: микрофон/камера живут при свёрнутом
// приложении, в шторке — CallStyle-уведомление с кнопкой завершения.
class CallService : Service() {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private var watchJob: Job? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val container = (application as GrooveApp).container
        val manager = container.callManager
        val call = manager.currentCall
        if (call == null) {
            stopSelf()
            return START_NOT_STICKY
        }
        val peerName = manager.peer?.fio ?: call.initiatorFio ?: "Звонок"
        val video = call.media == "video"

        var types = ServiceInfo.FOREGROUND_SERVICE_TYPE_PHONE_CALL
        if (granted(android.Manifest.permission.RECORD_AUDIO)) {
            types = types or ServiceInfo.FOREGROUND_SERVICE_TYPE_MICROPHONE
        }
        if (video && granted(android.Manifest.permission.CAMERA)) {
            types = types or ServiceInfo.FOREGROUND_SERVICE_TYPE_CAMERA
        }
        startForeground(
            Notifier.NOTIF_ID_CALL,
            container.notifier.buildOngoingCallNotification(peerName, video),
            types,
        )

        watchJob?.cancel()
        watchJob = scope.launch {
            manager.phase.collect { phase ->
                if (phase is CallPhase.Idle) {
                    stopForeground(STOP_FOREGROUND_REMOVE)
                    stopSelf()
                }
            }
        }
        return START_NOT_STICKY
    }

    private fun granted(permission: String): Boolean =
        ContextCompat.checkSelfPermission(this, permission) == PackageManager.PERMISSION_GRANTED

    override fun onDestroy() {
        watchJob?.cancel()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null
}
