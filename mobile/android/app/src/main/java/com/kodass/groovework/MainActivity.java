package com.kodass.groovework;

import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.os.Bundle;

import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {

    // Прод-адрес платформы. UI приезжает с сервера (server.url в
    // capacitor.config.json), здесь адрес нужен каналу обновлений обёртки.
    static final String APP_URL = "https://gw.kodass.ru";

    @Override
    public void onCreate(Bundle savedInstanceState) {
        registerPlugin(NativeShellPlugin.class);
        super.onCreate(savedInstanceState);
        createNotificationChannels();
    }

    @Override
    public void onResume() {
        super.onResume();
        // Обновление самой обёртки (UI обновляется деплоем сервера сам):
        // apps/mobile/version.json против собственного versionCode.
        UpdateChecker.maybeCheck(this);
    }

    // Id каналов совпадают и с channel_id, который pushsvc кладёт в
    // FCM-сообщения (messages/tasks), и с каналами прежнего нативного
    // приложения — при установке поверх пользовательские настройки
    // звука/вибрации сохраняются.
    private void createNotificationChannels() {
        NotificationManager nm = getSystemService(NotificationManager.class);

        NotificationChannel messages = new NotificationChannel(
            "messages", "Сообщения", NotificationManager.IMPORTANCE_HIGH);
        messages.setDescription("Новые сообщения в чатах");
        nm.createNotificationChannel(messages);

        NotificationChannel tasks = new NotificationChannel(
            "tasks", "Задачи", NotificationManager.IMPORTANCE_DEFAULT);
        tasks.setDescription("Назначенные задачи");
        nm.createNotificationChannel(tasks);

        NotificationChannel calls = new NotificationChannel(
            "calls_incoming", "Входящие звонки", NotificationManager.IMPORTANCE_HIGH);
        calls.setDescription("Входящие звонки и видеозвонки");
        nm.createNotificationChannel(calls);
    }
}
