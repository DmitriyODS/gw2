package com.kodass.groovework;

import android.app.DownloadManager;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.content.Intent;
import android.net.Uri;
import android.net.http.SslError;
import android.os.Bundle;
import android.os.Environment;
import android.os.Handler;
import android.os.Looper;
import android.text.TextUtils;
import android.webkit.CookieManager;
import android.webkit.SslErrorHandler;
import android.webkit.URLUtil;
import android.webkit.WebView;
import android.widget.Toast;

import com.getcapacitor.BridgeActivity;
import com.getcapacitor.BridgeWebViewClient;

import org.json.JSONObject;

public class MainActivity extends BridgeActivity {

    // Прод-адрес платформы. UI приезжает с сервера (server.url в
    // capacitor.config.json), здесь адрес нужен каналу обновлений обёртки.
    static final String APP_URL = "https://gw.kodass.ru";

    // Почасовая проверка обновлений обёртки: onResume не срабатывает, пока
    // приложение постоянно открыто, — тикаем таймером (троттл в UpdateChecker).
    private final Handler updateHandler = new Handler(Looper.getMainLooper());
    private final Runnable updateTick = new Runnable() {
        @Override
        public void run() {
            UpdateChecker.maybeCheck(MainActivity.this);
            updateHandler.postDelayed(this, 60 * 60 * 1000L);
        }
    };

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

        // WebView сам файлы не скачивает — без DownloadListener клики по
        // ссылкам с download (кнопка «Скачать» в просмотрщике картинок,
        // вложения, экспорты) молча игнорируются. Передаём системному
        // DownloadManager: файл уходит в «Загрузки» с уведомлением о ходе.
        this.bridge.getWebView().setDownloadListener((url, userAgent, contentDisposition, mimeType, contentLength) -> {
            if (!url.startsWith("http")) return; // blob:/data: DownloadManager не умеет
            try {
                DownloadManager.Request req = new DownloadManager.Request(Uri.parse(url));
                req.setMimeType(mimeType);
                req.addRequestHeader("User-Agent", userAgent);
                String cookies = CookieManager.getInstance().getCookie(url);
                if (cookies != null) {
                    req.addRequestHeader("Cookie", cookies);
                }
                String fileName = URLUtil.guessFileName(url, contentDisposition, mimeType);
                req.setDestinationInExternalPublicDir(Environment.DIRECTORY_DOWNLOADS, fileName);
                req.setNotificationVisibility(DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED);
                ((DownloadManager) getSystemService(DOWNLOAD_SERVICE)).enqueue(req);
                Toast.makeText(this, "Скачивание: " + fileName, Toast.LENGTH_SHORT).show();
            } catch (Exception e) {
                Toast.makeText(this, "Не удалось скачать файл", Toast.LENGTH_SHORT).show();
            }
        });

        // Первая проверка обновления — сразу при запуске, дальше — раз в час.
        updateHandler.post(updateTick);

        // Приложение открыто через системное «Поделиться» (ACTION_SEND) —
        // пробрасываем текст в веб-слой, где пользователь выберет получателя.
        handleShareIntent(getIntent());
    }

    // Уже запущенное приложение получило новое «Поделиться».
    @Override
    public void onNewIntent(Intent intent) {
        super.onNewIntent(intent);
        setIntent(intent);
        handleShareIntent(intent);
    }

    // Извлекаем расшаренный текст (Intent.EXTRA_TEXT + опц. EXTRA_SUBJECT) и
    // доставляем в JS. WebView на холодном старте ещё грузит SPA — доставляем с
    // задержкой и повтором; фронт также читает window.__gwSharedText при mount.
    private void handleShareIntent(Intent intent) {
        if (intent == null || !Intent.ACTION_SEND.equals(intent.getAction())) return;
        String type = intent.getType();
        if (type == null || !type.startsWith("text/")) return; // пока только текст
        String text = intent.getStringExtra(Intent.EXTRA_TEXT);
        String subject = intent.getStringExtra(Intent.EXTRA_SUBJECT);
        if (TextUtils.isEmpty(text)) text = subject;
        else if (!TextUtils.isEmpty(subject)) text = subject + "\n" + text;
        if (TextUtils.isEmpty(text)) return;
        deliverShareToWeb(text, 8);
    }

    private void deliverShareToWeb(final String text, final int retriesLeft) {
        updateHandler.postDelayed(() -> {
            WebView web = bridge != null ? bridge.getWebView() : null;
            if (web == null) {
                if (retriesLeft > 0) deliverShareToWeb(text, retriesLeft - 1);
                return;
            }
            String payload;
            try {
                payload = new JSONObject().put("text", text).toString();
            } catch (Exception e) {
                return;
            }
            // Ставим глобал (фронт читает его при mount, если слушателя ещё нет)
            // и шлём событие тем, кто уже слушает.
            String js = "window.__gwSharedText=" + payload + ";"
                + "window.dispatchEvent(new CustomEvent('gw:shared-text',{detail:" + payload + "}));";
            web.evaluateJavascript(js, null);
        }, 1200);
    }

    @Override
    public void onDestroy() {
        updateHandler.removeCallbacks(updateTick);
        super.onDestroy();
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

        NotificationChannel kudos = new NotificationChannel(
            "kudos", "Кудосы", NotificationManager.IMPORTANCE_DEFAULT);
        kudos.setDescription("Входящие переводы кудосов от коллег");
        nm.createNotificationChannel(kudos);

        NotificationChannel portal = new NotificationChannel(
            "portal", "Портал", NotificationManager.IMPORTANCE_DEFAULT);
        portal.setDescription("Новые посты корпоративного портала");
        nm.createNotificationChannel(portal);
    }
}
