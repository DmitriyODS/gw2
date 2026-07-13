package com.kodass.groovework;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.webkit.CookieManager;
import android.widget.Toast;

import androidx.core.app.RemoteInput;

import org.json.JSONObject;

import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;

// Ответ на сообщение прямо из шторки уведомления. Токенов у нативного слоя нет
// (живут в WebView), поэтому авторизуемся тем же путём, что и веб: берём
// HttpOnly refresh-cookie из CookieManager WebView, меняем её на access-токен
// через /api/auth/refresh и шлём сообщение с Bearer-заголовком.
public class MessageReplyReceiver extends BroadcastReceiver {

    @Override
    public void onReceive(Context context, Intent intent) {
        Bundle results = RemoteInput.getResultsFromIntent(intent);
        String convId = intent.getStringExtra(MessageNotifications.EXTRA_CONV_ID);
        String senderName = intent.getStringExtra(MessageNotifications.EXTRA_SENDER);
        if (results == null || convId == null) return;
        CharSequence cs = results.getCharSequence(MessageNotifications.KEY_REPLY);
        final String text = cs == null ? "" : cs.toString().trim();
        if (text.isEmpty()) return;

        final Context app = context.getApplicationContext();
        final PendingResult pending = goAsync(); // держит ресивер живым на время сети
        new Thread(() -> {
            boolean ok = false;
            try {
                ok = send(convId, text);
            } catch (Exception ignored) {
            }
            if (ok) {
                // Показываем отправленный ответ в самой переписке.
                MessageNotifications.appendSelfReply(app, convId, senderName, text);
            } else {
                MessageNotifications.repost(app, convId, senderName); // гасим спиннер
                new Handler(Looper.getMainLooper()).post(() ->
                    Toast.makeText(app, "Не удалось отправить ответ", Toast.LENGTH_SHORT).show());
            }
            pending.finish();
        }).start();
    }

    private boolean send(String convId, String text) throws Exception {
        String access = fetchAccessToken();
        if (access == null) return false;
        URL url = new URL(MainActivity.APP_URL + "/api/messenger/conversations/" + convId + "/messages");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        try {
            conn.setRequestMethod("POST");
            conn.setConnectTimeout(10_000);
            conn.setReadTimeout(10_000);
            conn.setRequestProperty("Content-Type", "application/json");
            conn.setRequestProperty("Authorization", "Bearer " + access);
            conn.setDoOutput(true);
            String body = new JSONObject().put("text", text).toString();
            conn.getOutputStream().write(body.getBytes(StandardCharsets.UTF_8));
            int code = conn.getResponseCode();
            return code >= 200 && code < 300;
        } finally {
            conn.disconnect();
        }
    }

    // Меняет refresh-cookie WebView на свежий access-токен (тело ответа
    // /api/auth/refresh несёт access_token). null — нет сессии/сеть недоступна.
    private String fetchAccessToken() throws Exception {
        String cookies = CookieManager.getInstance().getCookie(MainActivity.APP_URL);
        if (cookies == null || !cookies.contains("refresh_token")) return null;
        URL url = new URL(MainActivity.APP_URL + "/api/auth/refresh");
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        try {
            conn.setRequestMethod("POST");
            conn.setConnectTimeout(10_000);
            conn.setReadTimeout(10_000);
            conn.setRequestProperty("Content-Type", "application/json");
            conn.setRequestProperty("Cookie", cookies);
            conn.setDoOutput(true);
            conn.getOutputStream().write("{}".getBytes(StandardCharsets.UTF_8));
            int code = conn.getResponseCode();
            if (code < 200 || code >= 300) return null;
            String resp = readAll(conn.getInputStream());
            String token = new JSONObject(resp).optString("access_token", "");
            return token.isEmpty() ? null : token;
        } finally {
            conn.disconnect();
        }
    }

    private static String readAll(InputStream in) throws Exception {
        java.io.ByteArrayOutputStream out = new java.io.ByteArrayOutputStream();
        byte[] buf = new byte[4096];
        int n;
        while ((n = in.read(buf)) != -1) out.write(buf, 0, n);
        return out.toString("UTF-8");
    }
}
