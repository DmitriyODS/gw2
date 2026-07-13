package com.kodass.groovework;

import android.app.DownloadManager;
import android.app.KeyguardManager;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.content.Intent;
import android.os.Build;
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

import java.util.ArrayList;

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

        // Полноэкранное уведомление входящего звонка запускает нас поверх
        // локскрина — показываем активность сразу, до подъёма веб-слоя.
        applyCallLaunch(getIntent());

        // Приложение открыто через системное «Поделиться» (ACTION_SEND/SEND_MULTIPLE).
        handleShareIntent(getIntent());
    }

    // Уже запущенное приложение получило новый intent («Поделиться» или звонок).
    @Override
    public void onNewIntent(Intent intent) {
        super.onNewIntent(intent);
        setIntent(intent);
        applyCallLaunch(intent);
        handleShareIntent(intent);
    }

    // Запуск полноэкранным уведомлением звонка (extra gw_call): показать
    // активность поверх заблокированного экрана и снять keyguard.
    private void applyCallLaunch(Intent intent) {
        if (intent == null || !intent.getBooleanExtra("gw_call", false)) return;
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O_MR1) {
            setShowWhenLocked(true);
            setTurnScreenOn(true);
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            KeyguardManager km = (KeyguardManager) getSystemService(KEYGUARD_SERVICE);
            if (km != null) {
                try { km.requestDismissKeyguard(this, null); } catch (Exception ignored) {}
            }
        }
    }

    // Приложение открыто через системное «Поделиться»: текст и/или файлы
    // (в т.ч. несколько) складываем в буфер плагина NativeShell. Веб-слой
    // заберёт их getSharedPayload(), когда SPA и сессия готовы — так холодный
    // старт ничего не теряет. Здесь лишь будим фронт событием.
    private void handleShareIntent(Intent intent) {
        if (intent == null) return;
        String action = intent.getAction();
        boolean single = Intent.ACTION_SEND.equals(action);
        boolean multiple = Intent.ACTION_SEND_MULTIPLE.equals(action);
        if (!single && !multiple) return;

        String text = null;
        ArrayList<Uri> uris = new ArrayList<>();
        if (single) {
            Uri stream = intent.getParcelableExtra(Intent.EXTRA_STREAM);
            if (stream != null) {
                uris.add(stream);
            } else {
                // Текстовый шаринг (без файла): EXTRA_TEXT (+ опц. SUBJECT).
                String t = intent.getStringExtra(Intent.EXTRA_TEXT);
                String subject = intent.getStringExtra(Intent.EXTRA_SUBJECT);
                if (TextUtils.isEmpty(t)) t = subject;
                else if (!TextUtils.isEmpty(subject)) t = subject + "\n" + t;
                if (!TextUtils.isEmpty(t)) text = t;
            }
        } else {
            ArrayList<Uri> list = intent.getParcelableArrayListExtra(Intent.EXTRA_STREAM);
            if (list != null) uris.addAll(list);
        }
        if (text == null && uris.isEmpty()) return;

        synchronized (NativeShellPlugin.class) {
            NativeShellPlugin.pendingShareText = text;
            NativeShellPlugin.pendingShareUris.clear();
            NativeShellPlugin.pendingShareUris.addAll(uris);
        }
        notifyShareAvailable(6);
    }

    // Будим веб-слой (тёплый старт). На холодном фронт сам дёрнет
    // getSharedPayload() при загрузке — данные ждут в буфере плагина.
    private void notifyShareAvailable(final int retriesLeft) {
        updateHandler.postDelayed(() -> {
            WebView web = bridge != null ? bridge.getWebView() : null;
            if (web == null) {
                if (retriesLeft > 0) notifyShareAvailable(retriesLeft - 1);
                return;
            }
            web.evaluateJavascript(
                "window.__gwShareAvailable=true;"
                + "window.dispatchEvent(new CustomEvent('gw:share-available'));", null);
        }, 700);
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
