package com.kodass.groovework.service

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import com.kodass.groovework.GrooveApp

// Кнопка «Завершить» в уведомлении текущего юнита.
class UnitActionReceiver : BroadcastReceiver() {
    companion object {
        const val ACTION_STOP = "com.kodass.groovework.UNIT_STOP"
    }

    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != ACTION_STOP) return
        val id = intent.getLongExtra("unit_id", 0)
        if (id != 0L) {
            (context.applicationContext as GrooveApp).container.unitManager.stopUnit(id)
        }
    }
}
