package com.kodass.groovework;

import android.app.Activity;
import android.content.Context;
import android.content.SharedPreferences;
import android.widget.Toast;

import androidx.appcompat.app.AlertDialog;

// Фоновая автопроверка обновлений обёртки — аналог checkShellUpdate
// десктоп-клиента: раз в 6 часов сверяет собственный versionCode с
// apps/mobile/version.json и предлагает поставить новую сборку (скачивание и
// установка — нативно, общий AppUpdater). Принудительная проверка без
// троттла — кнопкой в «О приложении» (NativeShellPlugin).
final class UpdateChecker {

    private static final long CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000L;

    // Сборка, которую уже предлагали в этом процессе — не спамим диалогом.
    private static long offeredBuild = 0;

    private UpdateChecker() {}

    static void maybeCheck(Activity activity) {
        SharedPreferences prefs = activity.getSharedPreferences("app_update", Context.MODE_PRIVATE);
        long now = System.currentTimeMillis();
        if (now - prefs.getLong("last_check", 0) < CHECK_INTERVAL_MS) return;

        new Thread(() -> {
            long serverBuild = AppUpdater.fetchServerBuild();
            if (serverBuild <= 0) return; // сети нет — проверим в следующий раз
            prefs.edit().putLong("last_check", System.currentTimeMillis()).apply();

            long ownBuild = AppUpdater.ownBuild(activity);
            if (serverBuild <= ownBuild || offeredBuild == serverBuild) return;
            offeredBuild = serverBuild;

            activity.runOnUiThread(() -> {
                if (activity.isFinishing() || activity.isDestroyed()) return;
                new AlertDialog.Builder(activity)
                    .setTitle("Обновление приложения")
                    .setMessage("Доступна новая версия Groove Work. Установить? Она встанет поверх текущей.")
                    .setPositiveButton("Установить", (d, w) -> downloadAndInstall(activity))
                    .setNegativeButton("Позже", null)
                    .show();
            });
        }).start();
    }

    private static void downloadAndInstall(Activity activity) {
        Toast.makeText(activity, "Скачиваем обновление…", Toast.LENGTH_SHORT).show();
        new Thread(() -> {
            try {
                AppUpdater.downloadAndInstall(activity, null);
            } catch (Exception e) {
                activity.runOnUiThread(() ->
                    Toast.makeText(activity, e.getMessage(), Toast.LENGTH_LONG).show());
            }
        }).start();
    }
}
