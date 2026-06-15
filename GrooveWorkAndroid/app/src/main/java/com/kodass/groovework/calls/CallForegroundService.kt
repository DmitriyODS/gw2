package com.kodass.groovework.calls

import android.app.Notification
import android.app.Service
import android.content.Intent
import android.content.pm.PackageManager
import android.content.pm.ServiceInfo
import android.os.IBinder
import androidx.core.content.ContextCompat
import com.kodass.groovework.GrooveApp
import com.kodass.groovework.notifications.Notifier
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

// Foreground-сервис звонка. Три режима:
//  INCOMING       — из пуша (mic/cam из фона нельзя): тип phoneCall, держит
//                   процесс живым во время звона и постит full-screen-уведомление.
//  ONGOING        — после ответа / при исходящем: тип phoneCall (микрофон во время
//                   foreground-разговора работает и так).
//  ONGOING_MEDIA  — разговор активен (мы на переднем плане): повышаем тип до
//                   phoneCall|microphone|camera, чтобы медиа жили при свёрнутом
//                   приложении (промоушн строго из foreground — иначе SecurityException).
class CallForegroundService : Service() {
    companion object {
        const val EXTRA_MODE = "mode"
        const val MODE_INCOMING = "incoming"
        const val MODE_ONGOING = "ongoing"
        const val MODE_ONGOING_MEDIA = "ongoing_media"
    }

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private var watchJob: Job? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val controller = (application as GrooveApp).container.callController
        if (controller.currentCall == null) {
            // Контракт FGS: startForeground обязателен после startForegroundService
            // (даже если тут же останавливаемся) — иначе краш.
            runCatching {
                startForeground(
                    Notifier.NOTIF_ID_ONGOING,
                    controller.notifications.buildPlaceholderCallNotification(),
                    ServiceInfo.FOREGROUND_SERVICE_TYPE_PHONE_CALL,
                )
            }
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
            return START_NOT_STICKY
        }

        when (intent?.getStringExtra(EXTRA_MODE) ?: MODE_ONGOING) {
            MODE_INCOMING -> {
                val call = controller.currentCall
                if (call != null) {
                    startForeground(
                        Notifier.NOTIF_ID_INCOMING,
                        controller.notifications.buildIncomingCallNotification(call),
                        ServiceInfo.FOREGROUND_SERVICE_TYPE_PHONE_CALL,
                    )
                }
            }
            MODE_ONGOING_MEDIA -> {
                controller.notifications.cancelIncoming()
                var types = ServiceInfo.FOREGROUND_SERVICE_TYPE_PHONE_CALL
                if (granted(android.Manifest.permission.RECORD_AUDIO)) {
                    types = types or ServiceInfo.FOREGROUND_SERVICE_TYPE_MICROPHONE
                }
                if (controller.ui.value.state.video && granted(android.Manifest.permission.CAMERA)) {
                    types = types or ServiceInfo.FOREGROUND_SERVICE_TYPE_CAMERA
                }
                startForeground(Notifier.NOTIF_ID_ONGOING, ongoingNotification(), types)
            }
            else -> { // MODE_ONGOING
                controller.notifications.cancelIncoming()
                startForeground(
                    Notifier.NOTIF_ID_ONGOING,
                    ongoingNotification(),
                    ServiceInfo.FOREGROUND_SERVICE_TYPE_PHONE_CALL,
                )
            }
        }

        // Один наблюдатель на инстанс: обновляет активное уведомление (статус →
        // хронометр) и останавливает сервис, когда звонок завершён.
        if (watchJob?.isActive != true) {
            watchJob = scope.launch {
                controller.ui.collect { ui ->
                    when (val st = ui.state) {
                        is CallState.Idle -> {
                            stopForeground(STOP_FOREGROUND_REMOVE)
                            stopSelf()
                        }
                        is CallState.Ringing -> if (st.direction != CallDirection.Incoming) refreshOngoing(controller)
                        else -> refreshOngoing(controller)
                    }
                }
            }
        }
        return START_NOT_STICKY
    }

    private fun refreshOngoing(controller: com.kodass.groovework.calls.CallController) {
        if (controller.notifications.canPost()) {
            ContextCompat.getSystemService(this, android.app.NotificationManager::class.java)
                ?.notify(Notifier.NOTIF_ID_ONGOING, ongoingNotification())
        }
    }

    private fun ongoingNotification(): Notification {
        val controller = (application as GrooveApp).container.callController
        val st = controller.ui.value.state
        val peerName = controller.peer?.fio ?: st.call?.initiatorFio ?: "Звонок"
        val status = when (st) {
            is CallState.Active -> null
            is CallState.Connecting -> "Соединение…"
            else -> "Звоним…"
        }
        val activeSince = (st as? CallState.Active)?.activeSinceMs
        return controller.notifications.buildOngoingCallNotification(peerName, st.video, activeSince, status)
    }

    private fun granted(permission: String): Boolean =
        ContextCompat.checkSelfPermission(this, permission) == PackageManager.PERMISSION_GRANTED

    override fun onDestroy() {
        watchJob?.cancel()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null
}
