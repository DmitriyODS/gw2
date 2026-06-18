package com.kodass.groovework.ui.chats

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.api.AuthApi
import com.kodass.groovework.data.dto.UserDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.data.repo.MessengerRepository
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json

class ChatsViewModel(
    private val repo: MessengerRepository,
    private val authApi: AuthApi,
    private val json: Json,
) : ViewModel() {
    var loading by mutableStateOf(repo.conversations.value.isEmpty())
        private set
    var refreshing by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    // Новый чат: справочник пользователей компании.
    var directory by mutableStateOf<List<UserDto>>(emptyList())
        private set
    var directoryLoading by mutableStateOf(false)
        private set
    var directoryQuery by mutableStateOf("")
        private set
    var openingChat by mutableStateOf(false)
        private set
    var actionError by mutableStateOf<String?>(null)

    // Поиск по уже загруженному списку чатов (по имени собеседника и последнему
    // сообщению) — фильтрация локальная, без запроса к серверу.
    var chatQuery by mutableStateOf("")
        private set

    private var searchJob: Job? = null

    fun updateChatQuery(query: String) {
        chatQuery = query
    }

    // Фоновое обновление списка чатов и presence (без спиннера) — вызывается
    // периодически, пока пользователь в разделе, и при входе/смене компании.
    fun backgroundRefresh() {
        viewModelScope.launch {
            runCatching { repo.refreshConversations() }
            runCatching { repo.refreshPresence() }
            loading = false
            if (repo.conversations.value.isNotEmpty()) error = null
        }
    }

    fun refresh() {
        viewModelScope.launch {
            try {
                repo.refreshConversations()
                error = null
            } catch (e: ApiException) {
                if (repo.conversations.value.isEmpty()) error = e.message
            } finally {
                loading = false
            }
            runCatching { repo.refreshPresence() }
        }
    }

    fun pullRefresh() {
        if (refreshing) return
        viewModelScope.launch {
            refreshing = true
            try {
                repo.refreshConversations()
                error = null
            } catch (e: ApiException) {
                if (repo.conversations.value.isEmpty()) error = e.message
            } finally {
                refreshing = false
            }
            runCatching { repo.refreshPresence() }
        }
    }

    fun updateDirectoryQuery(query: String) {
        directoryQuery = query
        searchJob?.cancel()
        searchJob = viewModelScope.launch {
            delay(300)
            loadDirectory()
        }
    }

    fun loadDirectory() {
        viewModelScope.launch {
            directoryLoading = true
            try {
                directory = apiCall(json) {
                    authApi.directory(query = directoryQuery.takeIf { it.isNotBlank() }, excludeSelf = "1", all = "1")
                }
            } catch (_: Exception) {
                directory = emptyList()
            } finally {
                directoryLoading = false
            }
        }
    }

    fun togglePin(conversationId: Long) {
        viewModelScope.launch {
            try {
                repo.toggleConversationPin(conversationId)
            } catch (e: ApiException) {
                actionError = e.message
            }
        }
    }

    fun deleteConversation(conversationId: Long, scope: String) {
        viewModelScope.launch {
            try {
                repo.deleteConversation(conversationId, scope)
            } catch (e: ApiException) {
                actionError = e.message
            }
        }
    }

    fun openChatWith(userId: Long, onOpened: (Long) -> Unit) {
        if (openingChat) return
        viewModelScope.launch {
            openingChat = true
            try {
                val conversation = repo.openConversation(userId)
                runCatching { repo.refreshConversations() }
                onOpened(conversation.id)
            } catch (_: Exception) {
            } finally {
                openingChat = false
            }
        }
    }
}
