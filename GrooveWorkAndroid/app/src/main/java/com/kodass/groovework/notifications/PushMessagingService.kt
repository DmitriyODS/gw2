package com.kodass.groovework.notifications

import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.kodass.groovework.GrooveApp
import com.kodass.groovework.data.dto.CallDto

// Приём пуш-уведомлений FCM. Сервер (pushsvc) шлёт их только офлайн-получателям
// (на переднем плане событие приходит по WebSocket). data-поля строит pushsvc:
// type=message|task|call, channel, title, body + специфичные id.
class PushMessagingService : FirebaseMessagingService() {

    override fun onNewToken(token: String) {
        (application as GrooveApp).container.pushTokens.register(token)
    }

    override fun onMessageReceived(message: RemoteMessage) {
        val data = message.data
        val notifier = (application as GrooveApp).container.notifier
        val title = data["title"] ?: message.notification?.title.orEmpty()
        val body = data["body"] ?: message.notification?.body.orEmpty()

        when (data["type"]) {
            "call" -> {
                val callId = data["call_id"]?.toLongOrNull() ?: return
                notifier.showIncomingCall(
                    CallDto(
                        id = callId,
                        media = data["media"] ?: "audio",
                        initiatorId = data["caller_id"]?.toLongOrNull() ?: 0,
                        initiatorFio = data["caller"] ?: body.ifBlank { null },
                    )
                )
            }
            "message" -> {
                val conversationId = data["conversation_id"]?.toLongOrNull() ?: return
                notifier.showMessage(conversationId, title.ifBlank { "Новое сообщение" }, body)
            }
            "task" -> {
                val taskId = data["task_id"]?.toLongOrNull() ?: return
                notifier.showTask(taskId, title.ifBlank { "Новая задача" }, body)
            }
        }
    }
}
