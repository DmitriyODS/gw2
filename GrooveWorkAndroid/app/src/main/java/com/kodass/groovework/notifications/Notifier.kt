package com.kodass.groovework.notifications

import android.Manifest
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.app.Person
import androidx.core.content.ContextCompat
import com.kodass.groovework.MainActivity
import com.kodass.groovework.R
import com.kodass.groovework.data.dto.CallDto
import com.kodass.groovework.service.CallActionReceiver

// Каналы и построение всех уведомлений приложения.
class Notifier(private val context: Context) {
    companion object {
        const val CHANNEL_MESSAGES = "messages"
        const val CHANNEL_TASKS = "tasks"
        const val CHANNEL_CALLS = "calls"
        const val CHANNEL_SERVICE = "service"

        const val NOTIF_ID_ONLINE = 1
        const val NOTIF_ID_CALL = 2
        // Сообщения: id = 1000 + conversationId, задачи: 5000 + taskId.
        private const val MESSAGE_BASE = 1000
        private const val TASK_BASE = 5000
    }

    private val manager = NotificationManagerCompat.from(context)

    fun createChannels() {
        val nm = context.getSystemService(NotificationManager::class.java)
        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_MESSAGES, "Сообщения", NotificationManager.IMPORTANCE_HIGH).apply {
                description = "Новые сообщения в чатах"
            }
        )
        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_TASKS, "Задачи", NotificationManager.IMPORTANCE_DEFAULT).apply {
                description = "Новые задачи и комментарии"
            }
        )
        nm.createNotificationChannel(
            // Без звука канала: рингтон играет CallManager, пока идёт ринг-фаза.
            NotificationChannel(CHANNEL_CALLS, "Звонки", NotificationManager.IMPORTANCE_HIGH).apply {
                description = "Входящие и активные звонки"
                setSound(null, null)
                enableVibration(true)
            }
        )
        nm.createNotificationChannel(
            NotificationChannel(CHANNEL_SERVICE, "Подключение", NotificationManager.IMPORTANCE_MIN).apply {
                description = "Поддерживает соединение для мгновенных уведомлений"
                setShowBadge(false)
            }
        )
    }

    fun canPost(): Boolean =
        ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) ==
            PackageManager.PERMISSION_GRANTED

    private fun routeIntent(route: String, extra: Pair<String, Long>? = null): PendingIntent {
        val intent = Intent(context, MainActivity::class.java).apply {
            putExtra("route", route)
            extra?.let { putExtra(it.first, it.second) }
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP
        }
        return PendingIntent.getActivity(
            context,
            route.hashCode(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
    }

    fun showMessage(conversationId: Long, senderName: String, text: String) {
        if (!canPost()) return
        val person = Person.Builder().setName(senderName).build()
        val style = NotificationCompat.MessagingStyle(Person.Builder().setName("Вы").build())
            .addMessage(text, System.currentTimeMillis(), person)
        val notification = NotificationCompat.Builder(context, CHANNEL_MESSAGES)
            .setSmallIcon(R.drawable.ic_launcher_monochrome)
            .setStyle(style)
            .setContentTitle(senderName)
            .setContentText(text)
            .setCategory(NotificationCompat.CATEGORY_MESSAGE)
            .setAutoCancel(true)
            .setContentIntent(routeIntent("chat/$conversationId"))
            .build()
        manager.notify(MESSAGE_BASE + (conversationId % 100_000).toInt(), notification)
    }

    fun cancelMessage(conversationId: Long) {
        manager.cancel(MESSAGE_BASE + (conversationId % 100_000).toInt())
    }

    fun showTask(taskId: Long, title: String, text: String) {
        if (!canPost()) return
        val notification = NotificationCompat.Builder(context, CHANNEL_TASKS)
            .setSmallIcon(R.drawable.ic_launcher_monochrome)
            .setContentTitle(title)
            .setContentText(text)
            .setStyle(NotificationCompat.BigTextStyle().bigText(text))
            .setAutoCancel(true)
            .setContentIntent(routeIntent("task/$taskId"))
            .build()
        manager.notify(TASK_BASE + (taskId % 100_000).toInt(), notification)
    }

    fun showMissedCall(fio: String) {
        if (!canPost()) return
        val notification = NotificationCompat.Builder(context, CHANNEL_CALLS)
            .setSmallIcon(R.drawable.ic_launcher_monochrome)
            .setContentTitle("Пропущенный звонок")
            .setContentText(fio)
            .setCategory(NotificationCompat.CATEGORY_MISSED_CALL)
            .setAutoCancel(true)
            .setContentIntent(routeIntent("chats"))
            .build()
        manager.notify((System.currentTimeMillis() % 100_000).toInt() + 10_000, notification)
    }

    // Входящий звонок: CallStyle + full-screen intent — постится без FGS, это разрешено.
    fun showIncomingCall(call: CallDto) {
        if (!canPost()) return
        val fio = call.initiatorFio ?: "Входящий звонок"
        val person = Person.Builder().setName(fio).setImportant(true).build()
        val declineIntent = PendingIntent.getBroadcast(
            context,
            1,
            Intent(context, CallActionReceiver::class.java).setAction(CallActionReceiver.ACTION_DECLINE),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val answerIntent = PendingIntent.getActivity(
            context,
            2,
            Intent(context, MainActivity::class.java).apply {
                putExtra("answer_call_id", call.id)
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP
            },
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val fullScreenIntent = PendingIntent.getActivity(
            context,
            3,
            Intent(context, MainActivity::class.java).apply {
                putExtra("open_call", true)
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP
            },
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = NotificationCompat.Builder(context, CHANNEL_CALLS)
            .setSmallIcon(R.drawable.ic_launcher_monochrome)
            .setStyle(NotificationCompat.CallStyle.forIncomingCall(person, declineIntent, answerIntent))
            .setContentText(if (call.media == "video") "Входящий видеозвонок" else "Входящий звонок")
            .setCategory(NotificationCompat.CATEGORY_CALL)
            .setOngoing(true)
            .setFullScreenIntent(fullScreenIntent, true)
            .build()
        manager.notify(NOTIF_ID_CALL, notification)
    }

    // «Виджет звонка» в шторке, пока приложение свёрнуто.
    fun buildOngoingCallNotification(peerName: String, video: Boolean): Notification {
        val person = Person.Builder().setName(peerName).setImportant(true).build()
        val hangupIntent = PendingIntent.getBroadcast(
            context,
            4,
            Intent(context, CallActionReceiver::class.java).setAction(CallActionReceiver.ACTION_HANGUP),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        return NotificationCompat.Builder(context, CHANNEL_CALLS)
            .setSmallIcon(R.drawable.ic_launcher_monochrome)
            .setStyle(NotificationCompat.CallStyle.forOngoingCall(person, hangupIntent))
            .setContentText(if (video) "Видеозвонок" else "Звонок")
            .setCategory(NotificationCompat.CATEGORY_CALL)
            .setOngoing(true)
            .setUsesChronometer(true)
            .setWhen(System.currentTimeMillis())
            .setContentIntent(
                PendingIntent.getActivity(
                    context,
                    5,
                    Intent(context, MainActivity::class.java).apply {
                        putExtra("open_call", true)
                        flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_SINGLE_TOP
                    },
                    PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
                )
            )
            .build()
    }

    fun cancelCall() {
        manager.cancel(NOTIF_ID_CALL)
    }

    // Тихое уведомление фонового подключения (GatewayOnlineService).
    fun buildOnlineNotification(): Notification =
        NotificationCompat.Builder(context, CHANNEL_SERVICE)
            .setSmallIcon(R.drawable.ic_launcher_monochrome)
            .setContentTitle("Groove Work на связи")
            .setContentText("Получаем сообщения и звонки")
            .setOngoing(true)
            .setPriority(NotificationCompat.PRIORITY_MIN)
            .setContentIntent(routeIntent("chats"))
            .build()
}
