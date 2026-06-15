package com.kodass.groovework.calls

import android.Manifest
import android.app.Notification
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.app.Person
import androidx.core.content.ContextCompat
import com.kodass.groovework.CallActivity
import com.kodass.groovework.R
import com.kodass.groovework.data.dto.CallDto
import com.kodass.groovework.notifications.Notifier

// Строит все уведомления звонка (каналы регистрирует Notifier.createChannels):
//  • входящий — CallStyle.forIncomingCall + full-screen intent (поверх локскрина);
//  • активный — CallStyle.forOngoingCall + хронометр → системный чип-таймер в
//    статус-баре на Android 12+ (FGS типа phoneCall обязателен — см. CallForegroundService).
class CallNotifications(private val context: Context) {
    private val manager = NotificationManagerCompat.from(context)

    fun canPost(): Boolean =
        ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) ==
            PackageManager.PERMISSION_GRANTED

    // ── Входящий ──────────────────────────────────────────────────────────────

    fun buildIncomingCallNotification(call: CallDto): Notification {
        val fio = call.initiatorFio ?: "Входящий звонок"
        val video = call.media == "video"
        val person = Person.Builder().setName(fio).setImportant(true).build()
        val declineIntent = PendingIntent.getBroadcast(
            context, 1,
            Intent(context, CallActionReceiver::class.java)
                .setAction(CallActionReceiver.ACTION_DECLINE)
                .putExtra("call_id", call.id),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val answerIntent = PendingIntent.getActivity(
            context, 2,
            callActivityIntent {
                putExtra("answer_call_id", call.id)
                putExtra("answer_call_video", video)
                putExtra("answer_call_fio", fio)
            },
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val fullScreenIntent = PendingIntent.getActivity(
            context, 3, callActivityIntent { putExtra("open_call", true) },
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        return NotificationCompat.Builder(context, Notifier.CHANNEL_CALLS_INCOMING)
            .setSmallIcon(R.drawable.ic_launcher_monochrome)
            .setStyle(NotificationCompat.CallStyle.forIncomingCall(person, declineIntent, answerIntent))
            .setContentText(if (video) "Входящий видеозвонок" else "Входящий звонок")
            .setCategory(NotificationCompat.CATEGORY_CALL)
            .setOngoing(true)
            .setFullScreenIntent(fullScreenIntent, true)
            .build()
    }

    // Показать входящий напрямую (без FGS) — запасной путь, когда старт
    // foreground-сервиса из фона запрещён (Doze/понижен приоритет пуша).
    fun showIncomingCallStandalone(call: CallDto) {
        if (!canPost()) return
        manager.notify(Notifier.NOTIF_ID_INCOMING, buildIncomingCallNotification(call))
    }

    // ── Активный / исходящий ───────────────────────────────────────────────────

    // peerName — подпись; video — тип; activeSinceMs != null → хронометр (чип
    // в статус-баре); иначе показываем статус «Звоним…»/«Соединение…».
    fun buildOngoingCallNotification(
        peerName: String,
        video: Boolean,
        activeSinceMs: Long?,
        status: String?,
    ): Notification {
        val person = Person.Builder().setName(peerName).setImportant(true).build()
        val hangupIntent = PendingIntent.getBroadcast(
            context, 4,
            Intent(context, CallActionReceiver::class.java).setAction(CallActionReceiver.ACTION_HANGUP),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val openIntent = PendingIntent.getActivity(
            context, 5, callActivityIntent { putExtra("open_call", true) },
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val builder = NotificationCompat.Builder(context, Notifier.CHANNEL_CALLS_ONGOING)
            .setSmallIcon(R.drawable.ic_launcher_monochrome)
            .setStyle(NotificationCompat.CallStyle.forOngoingCall(person, hangupIntent))
            .setContentText(status ?: if (video) "Видеозвонок" else "Звонок")
            .setCategory(NotificationCompat.CATEGORY_CALL)
            .setOngoing(true)
            .setOnlyAlertOnce(true)
            .setContentIntent(openIntent)
        if (activeSinceMs != null) {
            builder.setUsesChronometer(true).setWhen(activeSinceMs)
        } else {
            builder.setShowWhen(false)
        }
        return builder.build()
    }

    // Заглушка: удовлетворяет контракт FGS (startForeground обязателен после
    // startForegroundService), когда звонок завершился до старта сервиса.
    fun buildPlaceholderCallNotification(): Notification =
        NotificationCompat.Builder(context, Notifier.CHANNEL_CALLS_ONGOING)
            .setSmallIcon(R.drawable.ic_launcher_monochrome)
            .setContentTitle("Звонок")
            .setOngoing(false)
            .build()

    fun showMissedCall(fio: String) {
        if (!canPost()) return
        val notification = NotificationCompat.Builder(context, Notifier.CHANNEL_CALLS_INCOMING)
            .setSmallIcon(R.drawable.ic_launcher_monochrome)
            .setContentTitle("Пропущенный звонок")
            .setContentText(fio)
            .setCategory(NotificationCompat.CATEGORY_MISSED_CALL)
            .setAutoCancel(true)
            .build()
        manager.notify((System.currentTimeMillis() % 100_000).toInt() + 10_000, notification)
    }

    fun cancelIncoming() = manager.cancel(Notifier.NOTIF_ID_INCOMING)

    fun cancelCall() {
        manager.cancel(Notifier.NOTIF_ID_INCOMING)
        manager.cancel(Notifier.NOTIF_ID_ONGOING)
    }

    private inline fun callActivityIntent(extras: Intent.() -> Unit): Intent =
        Intent(context, CallActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP
            extras()
        }
}
