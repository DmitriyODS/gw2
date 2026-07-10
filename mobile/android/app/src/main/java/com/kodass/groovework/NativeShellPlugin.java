package com.kodass.groovework;

import android.content.Context;
import android.content.Intent;
import android.graphics.Color;
import android.net.Uri;
import android.provider.Settings;
import android.view.Window;

import androidx.core.view.WindowCompat;
import androidx.core.view.WindowInsetsControllerCompat;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

// Мост обёртки для веб-слоя (фронт зовёт через window.Capacitor.Plugins
// .NativeShell, см. front/src/utils/nativeApp.js): принудительная проверка и
// установка обновлений APK по кнопке в «О приложении» (без 6-часового троттла
// автопроверки) и окраска системных панелей под текущую тему приложения.
@CapacitorPlugin(name = "NativeShell")
public class NativeShellPlugin extends Plugin {

    @PluginMethod
    public void getInfo(PluginCall call) {
        JSObject ret = new JSObject();
        ret.put("build", AppUpdater.ownBuild(getContext()));
        call.resolve(ret);
    }

    @PluginMethod
    public void checkUpdate(PluginCall call) {
        new Thread(() -> {
            long server = AppUpdater.fetchServerBuild();
            if (server <= 0) {
                call.reject("Не удалось проверить обновления — проверьте интернет");
                return;
            }
            long own = AppUpdater.ownBuild(getContext());
            JSObject ret = new JSObject();
            ret.put("current", own);
            ret.put("latest", server);
            ret.put("updateAvailable", server > own);
            call.resolve(ret);
        }).start();
    }

    @PluginMethod
    public void installUpdate(PluginCall call) {
        Context ctx = getContext();
        // Установка из стороннего источника требует явного разрешения — ведём
        // в системные настройки; фронт попросит повторить после возврата.
        if (!ctx.getPackageManager().canRequestPackageInstalls()) {
            Intent intent = new Intent(
                Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,
                Uri.parse("package:" + ctx.getPackageName())
            ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            ctx.startActivity(intent);
            JSObject ret = new JSObject();
            ret.put("status", "needs_permission");
            call.resolve(ret);
            return;
        }
        new Thread(() -> {
            try {
                AppUpdater.downloadAndInstall(ctx, value -> {
                    JSObject ev = new JSObject();
                    ev.put("progress", value);
                    notifyListeners("updateProgress", ev);
                });
                JSObject ret = new JSObject();
                ret.put("status", "installing");
                call.resolve(ret);
            } catch (Exception e) {
                call.reject(e.getMessage());
            }
        }).start();
    }

    // Красит статус-бар и системную навигацию под тему веб-приложения:
    // color — hex фона, dark — тёмная ли тема (true → светлые иконки).
    // Работает благодаря opt-out от edge-to-edge (values-v35/styles.xml).
    @PluginMethod
    public void setSystemBars(PluginCall call) {
        String color = call.getString("color", "#1A1C1E");
        boolean dark = Boolean.TRUE.equals(call.getBoolean("dark", true));
        getActivity().runOnUiThread(() -> {
            try {
                Window w = getActivity().getWindow();
                int parsed = Color.parseColor(color);
                w.setStatusBarColor(parsed);
                w.setNavigationBarColor(parsed);
                WindowInsetsControllerCompat ic = WindowCompat.getInsetsController(w, w.getDecorView());
                ic.setAppearanceLightStatusBars(!dark);
                ic.setAppearanceLightNavigationBars(!dark);
                call.resolve();
            } catch (Exception e) {
                call.reject("bad color");
            }
        });
    }
}
