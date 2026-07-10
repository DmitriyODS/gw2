package com.kodass.groovework;

import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.net.http.SslError;
import android.os.Bundle;
import android.webkit.SslErrorHandler;
import android.webkit.WebView;

import com.getcapacitor.BridgeActivity;
import com.getcapacitor.BridgeWebViewClient;

public class MainActivity extends BridgeActivity {

    // Прод-адрес платформы. UI приезжает с сервера (server.url в
    // capacitor.config.json), здесь адрес нужен каналу обновлений обёртки.
    static final String APP_URL = "https://gw.kodass.ru";

    @Override
    public void onCreate(Bundle savedInstanceState) {
        registerPlugin(NativeShellPlugin.class);
        super.onCreate(savedInstanceState);
        createNotificationChannels();

        // Capacitor ведёт onReceivedError/onReceivedHttpError на server.errorPath,
        // но НЕ обрабатывает onReceivedSslError: TLS-сбой на холодном старте
        // (несинхронизированные часы, captive portal) оставлял голый белый/чёрный
        // WebView без фолбэка. Заворачиваем на ту же страницу ошибки — она сама
        // повторяет подключение с бэкоффом.
        this.bridge.setWebViewClient(new BridgeWebViewClient(this.bridge) {
            @Override
            public void onReceivedSslError(WebView view, SslErrorHandler handler, SslError error) {
                handler.cancel();
                String errorUrl = bridge.getErrorUrl();
                if (errorUrl != null) {
                    view.loadUrl(errorUrl);
                }
            }
        });
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
