package com.kodass.groovework.calls

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import com.kodass.groovework.GrooveApp

// Кнопки уведомления звонка: отклонить входящий / завершить активный.
class CallActionReceiver : BroadcastReceiver() {
    companion object {
        const val ACTION_DECLINE = "com.kodass.groovework.CALL_DECLINE"
        const val ACTION_HANGUP = "com.kodass.groovework.CALL_HANGUP"
    }

    override fun onReceive(context: Context, intent: Intent) {
        val controller = (context.applicationContext as GrooveApp).container.callController
        when (intent.action) {
            ACTION_DECLINE -> controller.declineFromNotification(intent.getLongExtra("call_id", 0))
            ACTION_HANGUP -> controller.hangup()
        }
    }
}
