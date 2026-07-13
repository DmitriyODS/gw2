package com.kodass.groovework;

import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Intent;
import android.media.RingtoneManager;

import androidx.core.app.NotificationCompat;

import com.google.firebase.messaging.RemoteMessage;

import java.util.Map;

// FCM-сервис приложения (заменяет сервис Capacitor-плагина в манифесте,
// наследуясь от него — регистрация токенов и JS-события работают как есть).
//
// Звонки pushsvc шлёт data-only + high priority (см. buildMessage в
// back-go/push/internal/fcm): onMessageReceived вызывается даже при убитом
// приложении, и уведомление входящего звонка строим сами — иначе звонок при
// свёрнутом приложении беззвучен. Сообщения/задачи/кудосы/портал приходят с
// notification-payload — их показывает системный трей независимо от состояния
// приложения (надёжно на всех вендорах), здесь не трогаем.
public class PushMessagingService extends com.capacitorjs.plugins.pushnotifications.MessagingService {

    private static final int CALL_NOTIFICATION_ID = 43001;
    // Ринг-фаза звонка — 45 секунд; после уведомление уже неактуально.
    private static final long CALL_TIMEOUT_MS = 60_000;

    @Override
    public void onMessageReceived(RemoteMessage message) {
        super.onMessageReceived(message);

        Map<String, String> data = message.getData();
        if (!"call".equals(data.get("type"))) return;

        Intent open = new Intent(this, MainActivity.class)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_SINGLE_TOP);
        PendingIntent tap = PendingIntent.getActivity(
            this, 0, open, PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);

        String title = data.containsKey("title") ? data.get("title") : "Входящий звонок";
        String body = data.containsKey("body") ? data.get("body") : "";

        NotificationCompat.Builder builder = new NotificationCompat.Builder(this, "calls_incoming")
            .setSmallIcon(R.drawable.ic_launcher_foreground)
            .setContentTitle(title)
            .setContentText(body)
            .setCategory(NotificationCompat.CATEGORY_CALL)
            .setPriority(NotificationCompat.PRIORITY_MAX)
            .setSound(RingtoneManager.getDefaultUri(RingtoneManager.TYPE_RINGTONE))
            .setAutoCancel(true)
            .setTimeoutAfter(CALL_TIMEOUT_MS)
            .setContentIntent(tap);

        NotificationManager nm = getSystemService(NotificationManager.class);
        if (nm != null && nm.areNotificationsEnabled()) {
            nm.notify(CALL_NOTIFICATION_ID, builder.build());
        }
    }
}
