package com.kodass.groovework.notifications

import android.Manifest
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.app.Person
import androidx.core.content.ContextCompat
import com.kodass.groovework.MainActivity
import com.kodass.groovework.R

// Каналы и построение всех уведомлений приложения.
class Notifier(private val context: Context) {
    companion object {
        const val CHANNEL_MESSAGES = "messages"
        const val CHANNEL_TASKS = "tasks"
        // Раздельные каналы звонка: входящий — HIGH + full-screen; активный —
        // DEFAULT и тихий (только в шторке, без heads-up).
        const val CHANNEL_CALLS_INCOMING = "calls_incoming"
        const val CHANNEL_CALLS_ONGOING = "calls_ongoing"
        const val CHANNEL_SERVICE = "service"
        const val CHANNEL_UNIT = "unit"

        const val NOTIF_ID_ONLINE = 1
        // Разные id для входящего и активного — иначе активное «наследует»
        // heads-up входящего и зависает поверх.
        const val NOTIF_ID_INCOMING = 2
        const val NOTIF_ID_ONGOING = 3
        const val NOTIF_ID_UNIT = 4
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
            // Без звука/вибрации канала: рингтон и вибрацию ведёт IncomingRinger,
            // пока идёт ринг-фаза. Канал HIGH — для full-screen intent.
            NotificationChannel(CHANNEL_CALLS_INCOMING, "Входящие звонки", NotificationManager.IMPORTANCE_HIGH).apply {
                description = "Входящие звонки"
                setSound(null, null)
                enableVibration(false)
            }
        )
        nm.createNotificationChannel(
            // Активный звонок — тихо в шторке (без heads-up).
            NotificationChannel(CHANNEL_CALLS_ONGOING, "Активный звонок", NotificationManager.IMPORTANCE_DEFAULT).apply {
                description = "Текущий звонок"
                setSound(null, null)
                enableVibration(false)
            }
        )
        nm.createNotificationChannel(
            // Текущий юнит — тихо в шторке (без heads-up), с отсчётом времени.
            NotificationChannel(CHANNEL_UNIT, "Текущий юнит", NotificationManager.IMPORTANCE_LOW).apply {
                description = "Отсчёт времени активного юнита"
                setSound(null, null)
                enableVibration(false)
                setShowBadge(false)
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

    // Android 14+: full-screen intent звонка может быть не разрешён. Без него
    // «звонилка» всё равно звучит и вибрирует (IncomingRinger), но экран звонка
    // поверх локскрина не развернётся автоматически. Используется MainScreen для
    // разового онбординга разрешения.
    fun canUseFullScreenIntent(): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.UPSIDE_DOWN_CAKE) return true
        return context.getSystemService(NotificationManager::class.java).canUseFullScreenIntent()
    }

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

    // Уведомления звонка (входящий/активный/пропущенный) строит CallNotifications
    // в пакете calls; здесь — только каналы (createChannels) и проверка FSI.

    // Ongoing-уведомление текущего юнита: отсчёт времени (хронометр от старта) +
    // кнопка «Завершить» и переход в модалку юнита по тапу.
    fun showUnit(unit: com.kodass.groovework.data.dto.UnitDto) {
        if (!canPost()) return
        val startMillis = com.kodass.groovework.ui.common.parseIso(unit.datetimeStart)
            ?.toInstant()?.toEpochMilli() ?: System.currentTimeMillis()
        val stopIntent = PendingIntent.getBroadcast(
            context,
            6,
            Intent(context, com.kodass.groovework.service.UnitActionReceiver::class.java)
                .setAction(com.kodass.groovework.service.UnitActionReceiver.ACTION_STOP)
                .putExtra("unit_id", unit.id),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = NotificationCompat.Builder(context, CHANNEL_UNIT)
            .setSmallIcon(R.drawable.ic_launcher_monochrome)
            .setContentTitle("Текущий юнит")
            .setContentText(unit.name)
            .setOngoing(true)
            .setOnlyAlertOnce(true)
            .setUsesChronometer(true)
            .setWhen(startMillis)
            .setShowWhen(true)
            .setCategory(NotificationCompat.CATEGORY_STOPWATCH)
            .setContentIntent(routeIntent("unit"))
            .addAction(0, "Завершить", stopIntent)
            .build()
        manager.notify(NOTIF_ID_UNIT, notification)
    }

    fun cancelUnit() {
        manager.cancel(NOTIF_ID_UNIT)
    }

    // Тихое уведомление фонового подключения (если когда-то понадобится FGS связи).
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
