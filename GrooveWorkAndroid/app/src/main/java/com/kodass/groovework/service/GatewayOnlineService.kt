package com.kodass.groovework.service

import android.app.Service
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.IBinder
import com.kodass.groovework.GrooveApp
import com.kodass.groovework.notifications.Notifier

// Держит WS-подключение к gatewaysvc, пока пользователь залогинен:
// сообщения, задачи и звонки доходят даже при свёрнутом приложении.
class GatewayOnlineService : Service() {
    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val container = (application as GrooveApp).container
        startForeground(
            Notifier.NOTIF_ID_ONLINE,
            container.notifier.buildOnlineNotification(),
            ServiceInfo.FOREGROUND_SERVICE_TYPE_REMOTE_MESSAGING,
        )
        container.gateway.start()
        return START_STICKY
    }

    override fun onDestroy() {
        (application as GrooveApp).container.gateway.stop()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null
}
