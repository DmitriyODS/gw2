package com.kodass.groovework;

import android.app.Activity;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.net.Uri;

import androidx.appcompat.app.AlertDialog;

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

// Обновление самой обёртки — аналог checkShellUpdate десктоп-клиента: UI
// приезжает с сервера при каждом деплое, здесь следим только за версией
// оболочки (apps/mobile/version.json против собственного versionCode,
// формат ГГММДДН). Новее → предлагаем скачать APK браузером; ставится
// поверх (подпись и applicationId неизменны).
final class UpdateChecker {

    private static final long CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000L;
    private static final Pattern BUILD_RE = Pattern.compile("\"current_build\"\\s*:\\s*(\\d+)");

    // Сборка, которую уже предлагали в этом процессе — не спамим диалогом.
    private static long offeredBuild = 0;

    private UpdateChecker() {}

    static void maybeCheck(Activity activity) {
        SharedPreferences prefs = activity.getSharedPreferences("app_update", Context.MODE_PRIVATE);
        long now = System.currentTimeMillis();
        if (now - prefs.getLong("last_check", 0) < CHECK_INTERVAL_MS) return;

        new Thread(() -> {
            long serverBuild = fetchServerBuild();
            if (serverBuild <= 0) return; // сети нет — проверим в следующий раз
            prefs.edit().putLong("last_check", System.currentTimeMillis()).apply();

            long ownBuild = ownBuild(activity);
            if (serverBuild <= ownBuild || offeredBuild == serverBuild) return;
            offeredBuild = serverBuild;

            activity.runOnUiThread(() -> {
                if (activity.isFinishing() || activity.isDestroyed()) return;
                new AlertDialog.Builder(activity)
                    .setTitle("Обновление приложения")
                    .setMessage("Доступна новая версия Groove Work. Скачать? Она установится поверх текущей.")
                    .setPositiveButton("Скачать", (d, w) -> activity.startActivity(
                        new Intent(Intent.ACTION_VIEW, Uri.parse(MainActivity.APP_URL + "/apps/mobile/groovework.apk"))))
                    .setNegativeButton("Позже", null)
                    .show();
            });
        }).start();
    }

    private static long ownBuild(Context context) {
        try {
            return context.getPackageManager()
                .getPackageInfo(context.getPackageName(), 0)
                .getLongVersionCode();
        } catch (Exception e) {
            return Long.MAX_VALUE; // не смогли узнать себя — не предлагаем ничего
        }
    }

    private static long fetchServerBuild() {
        HttpURLConnection conn = null;
        try {
            conn = (HttpURLConnection) new URL(MainActivity.APP_URL + "/apps/mobile/version.json").openConnection();
            conn.setConnectTimeout(10_000);
            conn.setReadTimeout(10_000);
            try (BufferedReader r = new BufferedReader(new InputStreamReader(conn.getInputStream()))) {
                StringBuilder sb = new StringBuilder();
                String line;
                while ((line = r.readLine()) != null) sb.append(line);
                Matcher m = BUILD_RE.matcher(sb);
                return m.find() ? Long.parseLong(m.group(1)) : 0;
            }
        } catch (Exception e) {
            return 0;
        } finally {
            if (conn != null) conn.disconnect();
        }
    }
}
