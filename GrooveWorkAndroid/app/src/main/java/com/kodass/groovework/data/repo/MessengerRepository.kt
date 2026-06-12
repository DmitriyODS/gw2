package com.kodass.groovework.data.repo

import com.kodass.groovework.data.api.MessengerApi
import com.kodass.groovework.data.dto.AttachmentDto
import com.kodass.groovework.data.dto.ConversationItemDto
import com.kodass.groovework.data.dto.ForwardRequest
import com.kodass.groovework.data.dto.MessageDto
import com.kodass.groovework.data.dto.OpenConversationRequest
import com.kodass.groovework.data.dto.OpenedConversationDto
import com.kodass.groovework.data.dto.SendMessageRequest
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.GatewayEvent
import com.kodass.groovework.data.ws.boolField
import com.kodass.groovework.data.ws.longField
import com.kodass.groovework.data.ws.objField
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromJsonElement
import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.MultipartBody
import okhttp3.RequestBody.Companion.toRequestBody

// Кэш диалогов + presence; патчится сокет-событиями gatewaysvc, как stores/messenger.js на вебе.
class MessengerRepository(
    private val api: MessengerApi,
    gateway: GatewayClient,
    private val session: SessionManager,
    private val json: Json,
    scope: CoroutineScope,
) {
    private val _conversations = MutableStateFlow<List<ConversationItemDto>>(emptyList())
    val conversations: StateFlow<List<ConversationItemDto>> = _conversations

    private val _onlineUsers = MutableStateFlow<Set<Long>>(emptySet())
    val onlineUsers: StateFlow<Set<Long>> = _onlineUsers

    // Сквозной поток событий мессенджера для открытого чата.
    private val _events = MutableSharedFlow<GatewayEvent>(extraBufferCapacity = 128)
    val events: SharedFlow<GatewayEvent> = _events

    init {
        scope.launch {
            gateway.events.collect { handleEvent(it) }
        }
    }

    suspend fun refreshConversations() {
        _conversations.value = apiCall(json) { api.conversations() }
    }

    suspend fun refreshPresence() {
        _onlineUsers.value = apiCall(json) { api.presence() }.online.toSet()
    }

    suspend fun messages(conversationId: Long, beforeId: Long? = null, limit: Int = 50): List<MessageDto> =
        apiCall(json) { api.messages(conversationId, beforeId = beforeId, limit = limit) }

    suspend fun send(conversationId: Long, text: String?, attachmentIds: List<Long>?, replyToId: Long?): MessageDto {
        val message = apiCall(json) {
            api.send(conversationId, SendMessageRequest(text, attachmentIds, replyToId))
        }
        patchLastMessage(message, incrementUnread = false)
        return message
    }

    suspend fun markRead(conversationId: Long) {
        apiCall(json) { api.markRead(conversationId) }
        _conversations.value = _conversations.value.map {
            if (it.id == conversationId) it.copy(unreadCount = 0) else it
        }
    }

    suspend fun openConversation(userId: Long): OpenedConversationDto =
        apiCall(json) { api.openConversation(OpenConversationRequest(userId)) }

    suspend fun upload(fileName: String, mimeType: String, bytes: ByteArray): AttachmentDto {
        val body = bytes.toRequestBody(mimeType.toMediaTypeOrNull())
        val part = MultipartBody.Part.createFormData("file", fileName, body)
        return apiCall(json) { api.upload(part) }
    }

    suspend fun deleteMessage(messageId: Long, scope: String) {
        apiCall(json) { api.deleteMessage(messageId, scope) }
    }

    suspend fun forward(messageId: Long, conversationId: Long) {
        apiCall(json) { api.forward(ForwardRequest(messageId, listOf(conversationId))) }
    }

    val totalUnread: Int
        get() = _conversations.value.sumOf { it.unreadCount }

    private suspend fun handleEvent(event: GatewayEvent) {
        when (event.event) {
            "message:new" -> {
                val message = event.data.objField("message")?.let {
                    runCatching { json.decodeFromJsonElement<MessageDto>(it) }.getOrNull()
                }
                if (message != null) patchLastMessage(message, incrementUnread = true)
            }
            "presence:update" -> {
                val userId = event.data.longField("user_id")
                val online = event.data.boolField("online")
                if (userId != null && online != null) {
                    _onlineUsers.value =
                        if (online) _onlineUsers.value + userId else _onlineUsers.value - userId
                }
            }
            "message:read", "message:updated", "message:deleted",
            "conversation:deleted", "conversation:pin", "message:pin" -> {
                runCatching { refreshConversations() }
            }
        }
        _events.emit(event)
    }

    private fun patchLastMessage(message: MessageDto, incrementUnread: Boolean) {
        val myId = (session.authState.value as? AuthState.LoggedIn)?.claims?.userId
        val fromOther = message.senderId == null || message.senderId != myId
        val current = _conversations.value
        val exists = current.any { it.id == message.conversationId }
        if (!exists) return
        _conversations.value = current.map { conv ->
            if (conv.id != message.conversationId) conv
            else conv.copy(
                lastMessage = message,
                lastMessageAt = message.createdAt,
                unreadCount = if (incrementUnread && fromOther) conv.unreadCount + 1 else conv.unreadCount,
            )
        }.sortedWith(
            compareByDescending<ConversationItemDto> { it.isPinned }
                .thenByDescending { it.lastMessageAt ?: "" }
        )
    }
}
