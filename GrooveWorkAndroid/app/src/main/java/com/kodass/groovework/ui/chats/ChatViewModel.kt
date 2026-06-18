package com.kodass.groovework.ui.chats

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.dto.AttachmentDto
import com.kodass.groovework.data.dto.MessageDto
import com.kodass.groovework.data.dto.TaskDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.repo.MessengerRepository
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.ws.GatewayEvent
import com.kodass.groovework.data.ws.longField
import com.kodass.groovework.data.ws.objField
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromJsonElement

private const val PAGE_SIZE = 50
private const val MAX_UPLOAD_BYTES = 25L * 1024 * 1024

class ChatViewModel(
    private val repo: MessengerRepository,
    session: SessionManager,
    private val json: Json,
    val conversationId: Long,
) : ViewModel() {
    val myUserId: Long? = (session.authState.value as? AuthState.LoggedIn)?.claims?.userId

    // Сообщения от новых к старым — под reverseLayout LazyColumn.
    var messages by mutableStateOf<List<MessageDto>>(emptyList())
        private set
    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var loadingMore by mutableStateOf(false)
        private set
    var hasMore by mutableStateOf(true)
        private set

    var input by mutableStateOf("")
    var replyTo by mutableStateOf<MessageDto?>(null)
    var editingMessage by mutableStateOf<MessageDto?>(null)
        private set
    var attachedTask by mutableStateOf<TaskDto?>(null)
    var pendingAttachment by mutableStateOf<AttachmentDto?>(null)
        private set
    var uploading by mutableStateOf(false)
        private set
    var sending by mutableStateOf(false)
        private set
    var actionError by mutableStateOf<String?>(null)

    // Закреплённые сообщения диалога (баннер сверху).
    var pinnedMessages by mutableStateOf<List<MessageDto>>(emptyList())
        private set

    init {
        loadInitial()
        loadPinned()
        viewModelScope.launch {
            repo.events.collect { handleEvent(it) }
        }
    }

    fun loadInitial() {
        viewModelScope.launch {
            loading = true
            error = null
            try {
                val batch = repo.messages(conversationId, limit = PAGE_SIZE)
                messages = batch.sortedByDescending { it.id }
                hasMore = batch.size >= PAGE_SIZE
                runCatching { repo.markRead(conversationId) }
            } catch (e: ApiException) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun loadMore() {
        viewModelScope.launch { fetchOlder() }
    }

    private suspend fun fetchOlder(): Boolean {
        if (loadingMore || !hasMore || messages.isEmpty()) return false
        loadingMore = true
        return try {
            val beforeId = messages.last().id
            val batch = repo.messages(conversationId, beforeId = beforeId, limit = PAGE_SIZE)
            val known = messages.map { it.id }.toHashSet()
            messages = messages + batch.sortedByDescending { it.id }.filter { it.id !in known }
            hasMore = batch.size >= PAGE_SIZE
            true
        } catch (_: Exception) {
            false
        } finally {
            loadingMore = false
        }
    }

    // Индекс сообщения в списке, при необходимости догружая историю (для перехода
    // к отвеченному сообщению). null — если не найдено даже после полной загрузки.
    suspend fun ensureLoaded(messageId: Long): Int? {
        var index = messages.indexOfFirst { it.id == messageId }
        var guard = 0
        while (index < 0 && hasMore && guard < 20) {
            if (!fetchOlder()) break
            index = messages.indexOfFirst { it.id == messageId }
            guard++
        }
        return index.takeIf { it >= 0 }
    }

    private fun loadPinned() {
        viewModelScope.launch {
            runCatching { pinnedMessages = repo.pinnedMessages(conversationId) }
        }
    }

    fun togglePin(message: MessageDto) {
        viewModelScope.launch {
            try {
                repo.toggleMessagePin(message.id)
                loadPinned()
            } catch (e: ApiException) {
                actionError = e.message
            }
        }
    }

    val canSend: Boolean
        get() = !sending && !uploading &&
            (input.isNotBlank() || pendingAttachment != null || attachedTask != null)

    // Начать редактирование своего текстового сообщения — подставляем текст в поле.
    fun startEdit(message: MessageDto) {
        editingMessage = message
        replyTo = null
        input = message.text ?: ""
    }

    fun cancelEdit() {
        editingMessage = null
        input = ""
    }

    fun send() {
        val editing = editingMessage
        if (editing != null) {
            saveEdit(editing)
            return
        }
        if (!canSend) return
        stopTyping()
        val text = input.trim().takeIf { it.isNotEmpty() }
        val attachment = pendingAttachment
        val attachmentIds = attachment?.let { listOf(it.id) }
        val reply = replyTo
        val task = attachedTask
        // Очищаем поле сразу, не дожидаясь ответа сети: при медленном соединении
        // запрос уже ушёл, и пользователь не должен видеть «не очистившийся» текст
        // или отправлять повторно.
        input = ""
        replyTo = null
        pendingAttachment = null
        attachedTask = null
        viewModelScope.launch {
            sending = true
            actionError = null
            try {
                val message = repo.send(conversationId, text, attachmentIds, reply?.id, task?.id)
                prepend(message)
            } catch (e: ApiException) {
                // Не удалось отправить — возвращаем введённое для повтора, если поле
                // ещё пустое (пользователь не начал печатать новое).
                if (input.isBlank()) input = text ?: ""
                replyTo = reply
                pendingAttachment = attachment
                attachedTask = task
                actionError = e.message
            } finally {
                sending = false
            }
        }
    }

    private fun saveEdit(message: MessageDto) {
        val text = input.trim()
        if (text.isEmpty() || sending) return
        if (text == message.text) { cancelEdit(); return }
        viewModelScope.launch {
            sending = true
            actionError = null
            try {
                val updated = repo.updateMessage(message.id, text)
                messages = messages.map { if (it.id == updated.id) updated else it }
                editingMessage = null
                input = ""
            } catch (e: ApiException) {
                actionError = e.message
            } finally {
                sending = false
            }
        }
    }

    fun attachFile(fileName: String, mimeType: String, bytes: ByteArray) {
        if (bytes.size > MAX_UPLOAD_BYTES) {
            actionError = "Файл больше 25 МБ"
            return
        }
        viewModelScope.launch {
            uploading = true
            actionError = null
            try {
                pendingAttachment = repo.upload(fileName, mimeType, bytes)
            } catch (e: ApiException) {
                actionError = e.message
            } finally {
                uploading = false
            }
        }
    }

    fun clearAttachment() {
        pendingAttachment = null
    }

    // «Печатает…»: шлём собеседнику не чаще раза в 3с, авто-«перестал» через 4с
    // тишины (и сразу при отправке/выходе).
    private var typingActive = false
    private var lastTypingEmit = 0L
    private var typingStopJob: Job? = null

    fun onInputChange(value: String) {
        input = value
        if (value.isNotBlank()) notifyTyping()
    }

    private fun notifyTyping() {
        val now = System.currentTimeMillis()
        if (!typingActive || now - lastTypingEmit > 3000) {
            typingActive = true
            lastTypingEmit = now
            repo.sendTyping(conversationId, true)
        }
        typingStopJob?.cancel()
        typingStopJob = viewModelScope.launch {
            delay(4000)
            stopTyping()
        }
    }

    private fun stopTyping() {
        typingStopJob?.cancel()
        if (typingActive) {
            typingActive = false
            repo.sendTyping(conversationId, false)
        }
    }

    override fun onCleared() {
        stopTyping()
        super.onCleared()
    }

    // Сообщение, выбранное свайпом для пересылки (открывает шит выбора чата).
    var forwardTarget by mutableStateOf<MessageDto?>(null)
    var forwarding by mutableStateOf(false)
        private set

    fun forward(message: MessageDto, targetConversationId: Long, onDone: () -> Unit) {
        if (forwarding) return
        viewModelScope.launch {
            forwarding = true
            try {
                repo.forward(message.id, targetConversationId)
                forwardTarget = null
                onDone()
            } catch (e: ApiException) {
                actionError = e.message
            } finally {
                forwarding = false
            }
        }
    }

    fun deleteMessage(message: MessageDto, forAll: Boolean) {
        viewModelScope.launch {
            try {
                repo.deleteMessage(message.id, if (forAll) "all" else "me")
                messages = messages.filter { it.id != message.id }
            } catch (e: ApiException) {
                actionError = e.message
            }
        }
    }

    private fun prepend(message: MessageDto) {
        if (messages.any { it.id == message.id }) return
        messages = (listOf(message) + messages).sortedByDescending { it.id }
    }

    private fun handleEvent(event: GatewayEvent) {
        when (event.event) {
            "message:new" -> {
                if (event.data.longField("conversation_id") != conversationId) return
                val message = decodeMessage(event.data.objField("message")) ?: return
                prepend(message)
                if (message.senderId != myUserId) {
                    viewModelScope.launch { runCatching { repo.markRead(conversationId) } }
                }
            }
            "message:updated", "message:pin" -> {
                if (event.data.longField("conversation_id") != conversationId) return
                val message = decodeMessage(event.data.objField("message")) ?: return
                messages = messages.map { if (it.id == message.id) message else it }
                if (event.event == "message:pin") loadPinned()
            }
            "message:deleted" -> {
                if (event.data.longField("conversation_id") != conversationId) return
                val messageId = event.data.longField("message_id") ?: return
                messages = messages.filter { it.id != messageId }
            }
            "message:read" -> {
                if (event.data.longField("conversation_id") != conversationId) return
                val readerId = event.data.longField("reader_id")
                if (readerId != null && readerId != myUserId) {
                    // Собеседник прочитал: помечаем свои непрочитанные.
                    messages = messages.map {
                        if (it.senderId == myUserId && it.readAt == null) it.copy(readAt = "read") else it
                    }
                }
            }
        }
    }

    private fun decodeMessage(element: kotlinx.serialization.json.JsonElement?): MessageDto? =
        element?.let { runCatching { json.decodeFromJsonElement<MessageDto>(it) }.getOrNull() }
}
