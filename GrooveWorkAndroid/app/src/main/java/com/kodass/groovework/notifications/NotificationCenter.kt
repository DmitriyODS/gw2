package com.kodass.groovework.notifications

import com.kodass.groovework.data.dto.CommentDto
import com.kodass.groovework.data.dto.MessageDto
import com.kodass.groovework.data.dto.TaskDto
import com.kodass.groovework.data.repo.MessengerRepository
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.GatewayEvent
import com.kodass.groovework.data.ws.longField
import com.kodass.groovework.data.ws.objField
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromJsonElement

// Решает, какие сокет-события превращать в уведомления в шторке.
class NotificationCenter(
    private val notifier: Notifier,
    gateway: GatewayClient,
    private val messengerRepo: MessengerRepository,
    private val session: SessionManager,
    private val json: Json,
    scope: CoroutineScope,
) {
    // Состояние UI: открыт ли app и какой чат на экране.
    val appForeground = MutableStateFlow(false)
    val activeConversationId = MutableStateFlow<Long?>(null)

    private val myUserId: Long?
        get() = (session.authState.value as? AuthState.LoggedIn)?.claims?.userId

    init {
        scope.launch {
            gateway.events.collect { handle(it) }
        }
    }

    private fun handle(event: GatewayEvent) {
        when (event.event) {
            "message:new" -> {
                val message = event.data.objField("message")?.let {
                    runCatching { json.decodeFromJsonElement<MessageDto>(it) }.getOrNull()
                } ?: return
                if (message.senderId != null && message.senderId == myUserId) return
                val chatOnScreen = appForeground.value &&
                    activeConversationId.value == message.conversationId
                if (chatOnScreen) return
                val conversation = messengerRepo.conversations.value
                    .firstOrNull { it.id == message.conversationId }
                val sender = when {
                    conversation?.isPetChat == true -> conversation.petName ?: "Питомец"
                    conversation?.isDevChat == true -> "Техподдержка"
                    else -> conversation?.otherUser?.fio ?: "Новое сообщение"
                }
                notifier.showMessage(message.conversationId, sender, messagePreview(message))
            }
            "message:read" -> {
                // Прочитали на другом устройстве — убираем уведомление диалога.
                val conversationId = event.data.longField("conversation_id") ?: return
                val readerId = event.data.longField("reader_id")
                if (readerId == myUserId) notifier.cancelMessage(conversationId)
            }
            "task:created" -> {
                val task = runCatching {
                    json.decodeFromJsonElement<TaskDto>(event.data ?: return)
                }.getOrNull() ?: return
                if (task.authorId == myUserId) return
                notifier.showTask(task.id, "Новая задача", task.name)
            }
            "comment:new" -> {
                val comment = runCatching {
                    json.decodeFromJsonElement<CommentDto>(event.data ?: return)
                }.getOrNull() ?: return
                if (comment.authorId == myUserId) return
                if (appForeground.value) return
                val author = comment.author?.fio ?: "Комментарий"
                notifier.showTask(comment.taskId, "$author — комментарий к задаче", comment.text)
            }
        }
    }

    private fun messagePreview(message: MessageDto): String {
        message.text?.takeIf { it.isNotBlank() }?.let { return it }
        return when {
            message.kind == "call" -> "Звонок"
            message.task != null -> "Задача: ${message.task.name}"
            message.attachments.isNotEmpty() -> "Вложение: ${message.attachments.first().fileName}"
            else -> "Сообщение"
        }
    }
}
