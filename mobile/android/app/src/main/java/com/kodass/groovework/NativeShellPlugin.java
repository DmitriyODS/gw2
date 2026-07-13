package com.kodass.groovework;

import android.content.Context;
import android.content.Intent;
import android.database.Cursor;
import android.graphics.Color;
import android.media.AudioDeviceCallback;
import android.media.AudioDeviceInfo;
import android.media.AudioManager;
import android.net.Uri;
import android.os.Build;
import android.os.PowerManager;
import android.provider.OpenableColumns;
import android.provider.Settings;
import android.util.Base64;
import android.view.Window;

import androidx.core.view.WindowCompat;
import androidx.core.view.WindowInsetsControllerCompat;

import com.getcapacitor.JSArray;
import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.util.ArrayList;

// Мост обёртки для веб-слоя (фронт зовёт через window.Capacitor.Plugins
// .NativeShell, см. front/src/utils/nativeApp.js): принудительная проверка и
// установка обновлений APK по кнопке в «О приложении» (без 6-часового троттла
// автопроверки) и окраска системных панелей под текущую тему приложения.
@CapacitorPlugin(name = "NativeShell")
public class NativeShellPlugin extends Plugin {

    private AudioManager am() {
        return (AudioManager) getContext().getSystemService(Context.AUDIO_SERVICE);
    }

    // Смена набора аудио-устройств (подключили/убрали гарнитуру, BT) — будим
    // веб-слой, чтобы он перечитал доступные маршруты и показал выбор.
    @Override
    public void load() {
        try {
            am().registerAudioDeviceCallback(new AudioDeviceCallback() {
                @Override public void onAudioDevicesAdded(AudioDeviceInfo[] a) {
                    notifyListeners("audioDevicesChanged", new JSObject());
                }
                @Override public void onAudioDevicesRemoved(AudioDeviceInfo[] r) {
                    notifyListeners("audioDevicesChanged", new JSObject());
                }
            }, null);
        } catch (Exception ignored) {}
    }

    // ── Звонок: foreground-сервис (жизнь при блокировке) ───────────────────
    @PluginMethod
    public void startCallService(PluginCall call) {
        // Старт FGS может быть запрещён (фон/прошивка One UI) — не роняем звонок.
        try {
            Context ctx = getContext();
            Intent i = new Intent(ctx, CallForegroundService.class);
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) ctx.startForegroundService(i);
            else ctx.startService(i);
        } catch (Throwable ignored) {}
        call.resolve();
    }

    @PluginMethod
    public void stopCallService(PluginCall call) {
        try {
            Context ctx = getContext();
            ctx.stopService(new Intent(ctx, CallForegroundService.class));
        } catch (Throwable ignored) {}
        call.resolve();
    }

    // Экран гаснет у уха во время АУДИО-звонка (датчик приближения). Держим
    // proximity wake lock — он сам гасит/зажигает экран по сенсору; звонок при
    // этом продолжается (процесс жив за счёт CallForegroundService).
    private PowerManager.WakeLock proximityLock;

    @PluginMethod
    public void setProximityLock(PluginCall call) {
        boolean on = Boolean.TRUE.equals(call.getBoolean("on", false));
        if (getActivity() == null) { call.resolve(); return; }
        getActivity().runOnUiThread(() -> {
            try {
                PowerManager pm = (PowerManager) getContext().getSystemService(Context.POWER_SERVICE);
                if (proximityLock == null) {
                    proximityLock = pm.newWakeLock(
                        PowerManager.PROXIMITY_SCREEN_OFF_WAKE_LOCK, "gw:proximity");
                    proximityLock.setReferenceCounted(false);
                }
                if (on && !proximityLock.isHeld()) proximityLock.acquire(2 * 60 * 60 * 1000L);
                else if (!on && proximityLock.isHeld()) proximityLock.release();
            } catch (Exception ignored) {}
            call.resolve();
        });
    }

    // Показ активности поверх локскрина: включаем ТОЛЬКО на входящий звонок; на
    // активном/idle сбрасываем — тогда блокировка во время разговора ведёт себя
    // штатно (экран гаснет, звонок живёт за счёт foreground-сервиса, приложение
    // не закрывается).
    @PluginMethod
    public void setShowOverLock(PluginCall call) {
        boolean on = Boolean.TRUE.equals(call.getBoolean("on", false));
        if (getActivity() == null) { call.resolve(); return; }
        getActivity().runOnUiThread(() -> {
            try {
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O_MR1) {
                    getActivity().setShowWhenLocked(on);
                    getActivity().setTurnScreenOn(on);
                }
            } catch (Exception ignored) {}
            call.resolve();
        });
    }

    // ── Аудио-маршрутизация звонка ─────────────────────────────────────────
    @PluginMethod
    public void audioStart(PluginCall call) {
        try { am().setMode(AudioManager.MODE_IN_COMMUNICATION); } catch (Exception ignored) {}
        call.resolve();
    }

    @PluginMethod
    public void audioStop(PluginCall call) {
        AudioManager m = am();
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                m.clearCommunicationDevice();
            } else {
                m.setSpeakerphoneOn(false);
                if (m.isBluetoothScoOn()) { m.stopBluetoothSco(); m.setBluetoothScoOn(false); }
            }
            m.setMode(AudioManager.MODE_NORMAL);
        } catch (Exception ignored) {}
        call.resolve();
    }

    @PluginMethod
    public void audioListDevices(PluginCall call) {
        AudioManager m = am();
        JSArray arr = new JSArray();
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                AudioDeviceInfo cur = m.getCommunicationDevice();
                java.util.LinkedHashSet<String> seen = new java.util.LinkedHashSet<>();
                for (AudioDeviceInfo d : m.getAvailableCommunicationDevices()) {
                    String route = routeOf(d.getType());
                    if (route == null || !seen.add(route)) continue;
                    JSObject o = new JSObject();
                    o.put("route", route);
                    o.put("current", cur != null && cur.getId() == d.getId());
                    arr.put(o);
                }
            } else {
                addRoute(arr, "earpiece");
                addRoute(arr, "speaker");
                if (m.isWiredHeadsetOn()) addRoute(arr, "wired");
                if (m.isBluetoothScoAvailableOffCall()) addRoute(arr, "bluetooth");
            }
        } catch (Exception ignored) {}
        JSObject ret = new JSObject();
        ret.put("devices", arr);
        call.resolve(ret);
    }

    @PluginMethod
    public void audioSetRoute(PluginCall call) {
        String route = call.getString("route", "");
        AudioManager m = am();
        boolean ok = false;
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                for (AudioDeviceInfo d : m.getAvailableCommunicationDevices()) {
                    if (route.equals(routeOf(d.getType()))) { ok = m.setCommunicationDevice(d); break; }
                }
            } else {
                switch (route) {
                    case "speaker":
                        if (m.isBluetoothScoOn()) { m.stopBluetoothSco(); m.setBluetoothScoOn(false); }
                        m.setSpeakerphoneOn(true); ok = true; break;
                    case "bluetooth":
                        m.setSpeakerphoneOn(false); m.startBluetoothSco(); m.setBluetoothScoOn(true); ok = true; break;
                    default: // earpiece / wired
                        if (m.isBluetoothScoOn()) { m.stopBluetoothSco(); m.setBluetoothScoOn(false); }
                        m.setSpeakerphoneOn(false); ok = true;
                }
            }
        } catch (Exception ignored) {}
        JSObject ret = new JSObject();
        ret.put("ok", ok);
        call.resolve(ret);
    }

    @PluginMethod
    public void audioGetRoute(PluginCall call) {
        AudioManager m = am();
        String route = "earpiece";
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                AudioDeviceInfo d = m.getCommunicationDevice();
                if (d != null) { String r = routeOf(d.getType()); if (r != null) route = r; }
            } else {
                if (m.isBluetoothScoOn()) route = "bluetooth";
                else if (m.isSpeakerphoneOn()) route = "speaker";
                else if (m.isWiredHeadsetOn()) route = "wired";
            }
        } catch (Exception ignored) {}
        JSObject ret = new JSObject();
        ret.put("route", route);
        call.resolve(ret);
    }

    private static String routeOf(int type) {
        switch (type) {
            case AudioDeviceInfo.TYPE_BUILTIN_EARPIECE: return "earpiece";
            case AudioDeviceInfo.TYPE_BUILTIN_SPEAKER: return "speaker";
            case AudioDeviceInfo.TYPE_WIRED_HEADSET:
            case AudioDeviceInfo.TYPE_WIRED_HEADPHONES:
            case AudioDeviceInfo.TYPE_USB_HEADSET:
            case AudioDeviceInfo.TYPE_USB_DEVICE: return "wired";
            case AudioDeviceInfo.TYPE_BLUETOOTH_SCO:
            case AudioDeviceInfo.TYPE_BLE_HEADSET: return "bluetooth";
            default: return null;
        }
    }

    private static void addRoute(JSArray arr, String route) {
        JSObject o = new JSObject();
        o.put("route", route);
        o.put("current", false);
        arr.put(o);
    }

    // ── Входящий шаринг из системного «Поделиться» (заполняет MainActivity) ──
    // Pull-модель: полезная нагрузка живёт здесь, пока веб-слой не заберёт её
    // getSharedPayload() — так холодный старт не теряет данные (фронт дёргает
    // метод, когда SPA и сессия готовы).
    static String pendingShareText = null;
    static final ArrayList<Uri> pendingShareUris = new ArrayList<>();
    // Защита памяти/бриджа: файлы крупнее серверного лимита не тащим (25 МБ +
    // небольшой запас) — фронт покажет, что файл слишком большой.
    private static final long MAX_SHARE_FILE = 26L * 1024 * 1024;

    // Отдаёт расшаренный контент (текст + файлы base64) и очищает буфер.
    @PluginMethod
    public void getSharedPayload(PluginCall call) {
        new Thread(() -> {
            String text;
            ArrayList<Uri> uris;
            synchronized (NativeShellPlugin.class) {
                text = pendingShareText;
                uris = new ArrayList<>(pendingShareUris);
                pendingShareText = null;
                pendingShareUris.clear();
            }
            JSObject ret = new JSObject();
            if (text != null) ret.put("text", text);
            JSArray files = new JSArray();
            Context ctx = getContext();
            for (Uri uri : uris) {
                try {
                    JSObject f = readUri(ctx, uri);
                    if (f != null) files.put(f);
                } catch (Exception ignored) {}
            }
            ret.put("files", files);
            call.resolve(ret);
        }).start();
    }

    private JSObject readUri(Context ctx, Uri uri) throws Exception {
        String name = "файл";
        long size = -1;
        try (Cursor c = ctx.getContentResolver().query(uri, null, null, null, null)) {
            if (c != null && c.moveToFirst()) {
                int ni = c.getColumnIndex(OpenableColumns.DISPLAY_NAME);
                int si = c.getColumnIndex(OpenableColumns.SIZE);
                if (ni >= 0 && !c.isNull(ni)) name = c.getString(ni);
                if (si >= 0 && !c.isNull(si)) size = c.getLong(si);
            }
        } catch (Exception ignored) {}
        String mime = ctx.getContentResolver().getType(uri);
        if (mime == null) mime = "application/octet-stream";
        if (size > MAX_SHARE_FILE) return tooLarge(name, mime);

        ByteArrayOutputStream bos = new ByteArrayOutputStream();
        try (InputStream in = ctx.getContentResolver().openInputStream(uri)) {
            if (in == null) return null;
            byte[] buf = new byte[8192];
            int n; long total = 0;
            while ((n = in.read(buf)) != -1) {
                total += n;
                if (total > MAX_SHARE_FILE) return tooLarge(name, mime); // size был неизвестен
                bos.write(buf, 0, n);
            }
        }
        JSObject f = new JSObject();
        f.put("name", name);
        f.put("mimeType", mime);
        f.put("size", bos.size());
        f.put("data", Base64.encodeToString(bos.toByteArray(), Base64.NO_WRAP));
        return f;
    }

    private JSObject tooLarge(String name, String mime) {
        JSObject f = new JSObject();
        f.put("name", name);
        f.put("mimeType", mime);
        f.put("tooLarge", true);
        return f;
    }

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
