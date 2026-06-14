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

// Foreground-сервис звонка. Два режима:
//  INCOMING — поднимается из пуша (тип phoneCall, mic/cam из фона нельзя):
//    держит процесс живым во время звона и постит full-screen-уведомление.
//  ONGOING  — после ответа/при исходящем (тип phoneCall|microphone|camera,
//    стартует с переднего плана): микрофон/камера живут при свёрнутом
//    приложении, в шторке — тихое CallStyle-уведомление.
class CallService : Service() {
    companion object {
        const val EXTRA_MODE = "mode"
        const val MODE_INCOMING = "incoming"
        const val MODE_ONGOING = "ongoing"
    }

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
        val mode = intent?.getStringExtra(EXTRA_MODE) ?: MODE_ONGOING

        if (mode == MODE_INCOMING) {
            // Из фона разрешён только тип phoneCall — микрофон/камеру поднимем
            // после ответа, уже с переднего плана.
            startForeground(
                Notifier.NOTIF_ID_INCOMING,
                container.notifier.buildIncomingCallNotification(call),
                ServiceInfo.FOREGROUND_SERVICE_TYPE_PHONE_CALL,
            )
        } else {
            container.notifier.cancelIncoming()
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
                Notifier.NOTIF_ID_ONGOING,
                container.notifier.buildOngoingCallNotification(peerName, video),
                types,
            )
        }

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
