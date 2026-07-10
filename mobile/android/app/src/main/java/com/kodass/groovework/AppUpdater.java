package com.kodass.groovework;

import android.content.Context;
import android.content.Intent;
import android.content.pm.PackageInfo;
import android.net.Uri;

import androidx.core.content.FileProvider;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

// Обновление самой обёртки «по воздуху» (UI приезжает с сервера при обычном
// деплое): сверка versionCode с /apps/mobile/version.json, скачивание
// /apps/mobile/groovework.apk и запуск системной установки поверх (подпись и
// applicationId неизменны). Общая логика для кнопки в «О приложении»
// (NativeShellPlugin) и фоновой автопроверки (UpdateChecker).
final class AppUpdater {

    interface Progress {
        void onProgress(float value); // 0..1, отрицательное — размер неизвестен
    }

    private static final Pattern BUILD_RE = Pattern.compile("\"current_build\"\\s*:\\s*(\\d+)");

    private AppUpdater() {}

    static long ownBuild(Context ctx) {
        try {
            return ctx.getPackageManager().getPackageInfo(ctx.getPackageName(), 0).getLongVersionCode();
        } catch (Exception e) {
            return Long.MAX_VALUE; // не смогли узнать себя — обновления не предлагаем
        }
    }

    // Номер сборки на сервере; 0 — сеть/парсинг не удались.
    static long fetchServerBuild() {
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

    // Скачивает APK, проверяет его (наш пакет, сборка новее установленной)
    // и запускает системный установщик. Бросает Exception с понятным
    // пользователю текстом.
    static void downloadAndInstall(Context ctx, Progress progress) throws Exception {
        File apk = new File(ctx.getExternalFilesDir(null), "groovework-update.apk");

        HttpURLConnection conn = null;
        try {
            conn = (HttpURLConnection) new URL(MainActivity.APP_URL + "/apps/mobile/groovework.apk").openConnection();
            conn.setConnectTimeout(15_000);
            conn.setReadTimeout(30_000);
            long total = conn.getContentLengthLong();
            try (InputStream in = conn.getInputStream(); FileOutputStream out = new FileOutputStream(apk)) {
                byte[] buf = new byte[64 * 1024];
                long readTotal = 0;
                int read;
                while ((read = in.read(buf)) >= 0) {
                    out.write(buf, 0, read);
                    readTotal += read;
                    if (progress != null) {
                        progress.onProgress(total > 0 ? Math.min(1f, (float) readTotal / total) : -1f);
                    }
                }
                out.flush();
            }
        } catch (Exception e) {
            apk.delete();
            throw new Exception("Не удалось скачать обновление — проверьте интернет");
        } finally {
            if (conn != null) conn.disconnect();
        }

        // Битый файл либо сервер ещё раздаёт старую сборку (version.json при
        // деплое мог обновиться раньше самого APK) — не предлагаем установку.
        PackageInfo info = ctx.getPackageManager().getPackageArchiveInfo(apk.getPath(), 0);
        if (info == null || !ctx.getPackageName().equals(info.packageName)
            || info.getLongVersionCode() <= ownBuild(ctx)) {
            apk.delete();
            throw new Exception("Файл обновления повреждён — попробуйте ещё раз");
        }

        Uri uri = FileProvider.getUriForFile(ctx, ctx.getPackageName() + ".fileprovider", apk);
        Intent intent = new Intent(Intent.ACTION_VIEW)
            .setDataAndType(uri, "application/vnd.android.package-archive")
            .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_ACTIVITY_NEW_TASK);
        ctx.startActivity(intent);
    }
}
