package com.kodass.groovework;

import android.Manifest;
import android.content.ContentValues;
import android.content.Context;
import android.content.Intent;
import android.content.pm.ShortcutInfo;
import android.content.pm.ShortcutManager;
import android.database.Cursor;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.graphics.Color;
import android.graphics.drawable.Icon;
import android.media.AudioDeviceCallback;
import android.media.AudioDeviceInfo;
import android.media.AudioManager;
import android.media.MediaScannerConnection;
import android.net.Uri;
import android.os.Build;
import android.os.Environment;
import android.os.PowerManager;
import android.provider.MediaStore;
import android.provider.OpenableColumns;
import android.provider.Settings;
import android.util.Base64;
import android.view.Window;
import android.widget.Toast;

import androidx.core.view.WindowCompat;
import androidx.core.view.WindowInsetsControllerCompat;

import com.getcapacitor.JSArray;
import com.getcapacitor.JSObject;
import com.getcapacitor.PermissionState;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;
import com.getcapacitor.annotation.Permission;
import com.getcapacitor.annotation.PermissionCallback;

import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.ArrayList;

// Мост обёртки для веб-слоя (фронт зовёт через window.Capacitor.Plugins
// .NativeShell, см. front/src/utils/nativeApp.js): принудительная проверка и
// установка обновлений APK по кнопке в «О приложении» (без 6-часового троттла
// автопроверки), сохранение созданных в браузере файлов и окраска системных
// панелей под текущую тему приложения.
@CapacitorPlugin(
    name = "NativeShell",
    permissions = {
        // Только Android 9 и ниже: там запись в общие «Загрузки» идёт по
        // файловому пути и требует разрешения. С Android 10 пишем через
        // MediaStore — разрешение не нужно и не запрашивается.
        @Permission(alias = "storage", strings = { Manifest.permission.WRITE_EXTERNAL_STORAGE })
    }
)
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

    // ── Сохранение файла, собранного в вебе (выгрузки, экспорты) ───────────
    // WebView скачивает только http(s)-ссылки (их ведёт DownloadListener в
    // MainActivity): blob:/data: он игнорирует МОЛЧА, поэтому любой файл,
    // который фронт строит сам (xlsx, docx, png доски, архив бэкапа), в обёртке
    // просто не сохранялся. Здесь принимаем его содержимое base64 и кладём в
    // общие «Загрузки» — как это делает системный загрузчик.
    @PluginMethod
    public void saveFile(PluginCall call) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.Q
            && getPermissionState("storage") != PermissionState.GRANTED) {
            requestPermissionForAlias("storage", call, "storagePermissionCallback");
            return;
        }
        writeToDownloads(call);
    }

    @PermissionCallback
    private void storagePermissionCallback(PluginCall call) {
        if (getPermissionState("storage") != PermissionState.GRANTED) {
            call.reject("Нет доступа к памяти устройства");
            return;
        }
        writeToDownloads(call);
    }

    private void writeToDownloads(PluginCall call) {
        final String name = safeName(call.getString("name", "file"));
        final String mime = call.getString("mimeType", "application/octet-stream");
        final String data = call.getString("data", "");
        new Thread(() -> {
            try {
                byte[] bytes = Base64.decode(data, Base64.DEFAULT);
                String saved = Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q
                    ? saveViaMediaStore(name, mime, bytes)
                    : saveViaFile(name, bytes);
                toast("Сохранено в «Загрузки»: " + saved);
                JSObject ret = new JSObject();
                ret.put("name", saved);
                call.resolve(ret);
            } catch (Throwable e) {
                call.reject("Не удалось сохранить файл");
            }
        }).start();
    }

    // Android 10+: файл создаёт MediaStore — он же разводит одинаковые имена
    // («Отчёт(1).xlsx»). IS_PENDING держит запись скрытой, пока идёт запись:
    // иначе менеджер файлов покажет недокачанный файл.
    private String saveViaMediaStore(String name, String mime, byte[] bytes) throws Exception {
        ContentValues values = new ContentValues();
        values.put(MediaStore.Downloads.DISPLAY_NAME, name);
        values.put(MediaStore.Downloads.MIME_TYPE, mime);
        values.put(MediaStore.Downloads.IS_PENDING, 1);
        Uri uri = getContext().getContentResolver()
            .insert(MediaStore.Downloads.EXTERNAL_CONTENT_URI, values);
        if (uri == null) throw new Exception("no uri");
        try (OutputStream out = getContext().getContentResolver().openOutputStream(uri)) {
            if (out == null) throw new Exception("no stream");
            out.write(bytes);
        }
        values.clear();
        values.put(MediaStore.Downloads.IS_PENDING, 0);
        getContext().getContentResolver().update(uri, values, null, null);
        return name;
    }

    // Android 9 и ниже: обычный файл в общей папке «Загрузки» + сканирование,
    // иначе он не появится в менеджере файлов до перезагрузки.
    private String saveViaFile(String name, byte[] bytes) throws Exception {
        File dir = Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_DOWNLOADS);
        if (!dir.exists() && !dir.mkdirs()) throw new Exception("no dir");
        File file = uniqueFile(dir, name);
        try (FileOutputStream out = new FileOutputStream(file)) {
            out.write(bytes);
        }
        MediaScannerConnection.scanFile(getContext(), new String[] { file.getAbsolutePath() }, null, null);
        return file.getName();
    }

    private static File uniqueFile(File dir, String name) {
        File file = new File(dir, name);
        if (!file.exists()) return file;
        int dot = name.lastIndexOf('.');
        String base = dot > 0 ? name.substring(0, dot) : name;
        String ext = dot > 0 ? name.substring(dot) : "";
        for (int i = 1; i < 1000; i++) {
            file = new File(dir, base + "(" + i + ")" + ext);
            if (!file.exists()) return file;
        }
        return file;
    }

    // Имя приходит из пользовательского текста (название доски, реестра): в нём
    // не должно быть разделителей пути.
    private static String safeName(String name) {
        String clean = name == null ? "" : name.replaceAll("[\\\\/:*?\"<>|\\x00-\\x1f]", " ").trim();
        return clean.isEmpty() ? "file" : clean;
    }

    private void toast(String text) {
        if (getActivity() == null) return;
        getActivity().runOnUiThread(() ->
            Toast.makeText(getContext(), text, Toast.LENGTH_SHORT).show());
    }

    @PluginMethod
    public void getInfo(PluginCall call) {
        JSObject ret = new JSObject();
        ret.put("build", AppUpdater.ownBuild(getContext()));
        call.resolve(ret);
    }

    /* Ярлык раздела на домашнем экране телефона: система показывает свой
       диалог подтверждения, поэтому «успех» здесь означает лишь то, что
       предложение отправлено лаунчеру. Значок рисует веб-слой (глиф раздела в
       цветах текущей темы) и передаёт сюда base64-PNG — держать копии иконок
       двадцати разделов в ресурсах ради этого незачем.
       supported=false — лаунчер закреплять ярлыки не умеет (бывает на старых
       и нестандартных оболочках), и веб-слой объясняет это человеку. */
    @PluginMethod
    public void pinShortcut(PluginCall call) {
        String path = call.getString("path", "");
        String label = call.getString("label", "");
        if (path == null || path.isEmpty() || label == null || label.isEmpty()) {
            call.reject("Нужны путь раздела и название");
            return;
        }

        ShortcutManager sm = getContext().getSystemService(ShortcutManager.class);
        JSObject ret = new JSObject();
        if (sm == null || !sm.isRequestPinShortcutSupported()) {
            ret.put("supported", false);
            call.resolve(ret);
            return;
        }

        Intent intent = new Intent(getContext(), MainActivity.class)
            .setAction(Intent.ACTION_VIEW)
            .putExtra(MainActivity.EXTRA_PATH, path)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TOP);

        ShortcutInfo.Builder shortcut = new ShortcutInfo.Builder(getContext(), "gw:" + path)
            .setShortLabel(label)
            .setLongLabel(label)
            .setIntent(intent)
            .setIcon(shortcutIcon(call.getString("icon")));

        sm.requestPinShortcut(shortcut.build(), null);
        ret.put("supported", true);
        call.resolve(ret);
    }

    // Значок ярлыка: рисунок веб-слоя, а если он не доехал — иконка приложения.
    private Icon shortcutIcon(String base64Png) {
        if (base64Png != null && !base64Png.isEmpty()) {
            try {
                byte[] png = Base64.decode(base64Png, Base64.DEFAULT);
                Bitmap bmp = BitmapFactory.decodeByteArray(png, 0, png.length);
                // Адаптивный значок: края съедает маска лаунчера, поэтому
                // веб-слой рисует содержимое с запасом по краям.
                if (bmp != null) return Icon.createWithAdaptiveBitmap(bmp);
            } catch (Exception ignored) {}
        }
        return Icon.createWithResource(getContext(), R.mipmap.ic_launcher);
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
