package com.kodass.groovework.service

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import com.kodass.groovework.GrooveApp
import kotlinx.coroutines.launch

// Кнопка «Завершить» в уведомлении текущего юнита.
class UnitActionReceiver : BroadcastReceiver() {
    companion object {
        const val ACTION_STOP = "com.kodass.groovework.UNIT_STOP"
    }

    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != ACTION_STOP) return
        val id = intent.getLongExtra("unit_id", 0)
        if (id == 0L) return
        val container = (context.applicationContext as GrooveApp).container
        // goAsync держит процесс receiver'а живым, пока идёт сетевой запрос —
        // иначе при медленном интернете процесс убивается и юнит не завершается.
        val pending = goAsync()
        container.appScope.launch {
            try {
                container.unitManager.stopUnitSuspend(id)
            } finally {
                pending.finish()
            }
        }
    }
}
