package com.kodass.groovework;

import android.app.Notification;
import android.app.PendingIntent;
import android.app.Service;
import android.content.Intent;
import android.content.pm.ServiceInfo;
import android.os.Build;
import android.os.IBinder;
import android.os.PowerManager;

import androidx.annotation.Nullable;
import androidx.core.app.NotificationCompat;

// Foreground-сервис на время активного звонка: держит процесс живым и CPU
// разбуженным (partial wake lock), чтобы WebRTC-соединение в WebView не
// умирало при блокировке экрана и в Doze. Управляется из NativeShell
// (startCallService/stopCallService — веб-слой зовёт по фазам звонка).
public class CallForegroundService extends Service {

    static final int NOTIF_ID = 44001;
    // Видеозвонок: с Android 11 камера в фоне работает только у сервиса с
    // типом camera — без него картинка замирала при блокировке экрана.
    static final String EXTRA_CAMERA = "camera";
    private PowerManager.WakeLock wakeLock;

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        Notification n = buildNotification();

        // Android 14+ (особенно Samsung One UI на Android 16) жёстко ограничивает
        // microphone-FGS: старт из фона или до реального захвата микрофона кидает
        // SecurityException/ForegroundServiceStartNotAllowedException. НЕ роняем
        // приложение: пробуем с типом, при отказе — без типа, иначе тихо
        // останавливаемся (звонок продолжится, просто без гарантии жизни при
        // заблокированном экране). stopSelf до 5-сек таймаута снимает требование
        // системы вызвать startForeground — иначе был бы отдельный краш.
        // Повторный старт (включили/выключили камеру) обновляет тип сервиса.
        // Отказ с типом camera (нет разрешения на камеру) — остаёмся с микрофоном.
        boolean camera = intent != null && intent.getBooleanExtra(EXTRA_CAMERA, false);
        boolean started = false;
        if (camera && Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            started = tryStart(n, ServiceInfo.FOREGROUND_SERVICE_TYPE_MICROPHONE
                | ServiceInfo.FOREGROUND_SERVICE_TYPE_CAMERA);
        }
        if (!started && Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            started = tryStart(n, ServiceInfo.FOREGROUND_SERVICE_TYPE_MICROPHONE);
        }
        if (!started) started = tryStart(n, 0);
        if (!started) {
            stopSelf();
            return START_NOT_STICKY;
        }

        try {
            if (wakeLock == null) {
                PowerManager pm = (PowerManager) getSystemService(POWER_SERVICE);
                wakeLock = pm.newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "gw:call");
                wakeLock.setReferenceCounted(false);
            }
            if (!wakeLock.isHeld()) wakeLock.acquire(2 * 60 * 60 * 1000L); // предохранитель 2ч
        } catch (Throwable ignored) {}
        return START_NOT_STICKY;
    }

    private boolean tryStart(Notification n, int type) {
        try {
            if (type != 0) startForeground(NOTIF_ID, n, type);
            else startForeground(NOTIF_ID, n);
            return true;
        } catch (Throwable t) {
            return false;
        }
    }

    private Notification buildNotification() {
        Intent open = new Intent(this, MainActivity.class)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_SINGLE_TOP);
        PendingIntent tap = PendingIntent.getActivity(
            this, 0, open, PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
        return new NotificationCompat.Builder(this, "calls_incoming")
            .setSmallIcon(R.drawable.ic_launcher_foreground)
            .setContentTitle("Идёт звонок")
            .setContentText("Groove Work")
            .setOngoing(true)
            .setCategory(NotificationCompat.CATEGORY_CALL)
            .setContentIntent(tap)
            .build();
    }

    @Override
    public void onDestroy() {
        if (wakeLock != null && wakeLock.isHeld()) wakeLock.release();
        super.onDestroy();
    }

    @Nullable
    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }
}
