package com.kodass.groovework.ui.diary

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.dto.DiaryDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.repo.DiariesRepository
import com.kodass.groovework.data.ws.GatewayClient
import kotlinx.coroutines.launch

enum class DiaryTab(val key: String) { MINE("mine"), SHARED("shared") }

// Первый уровень раздела «Ежедневник» — список ежедневников: вкладки «Мои» и
// «Поделились» (чужие, открытые мне read-only).
class DiariesListViewModel(
    private val repo: DiariesRepository,
    gateway: GatewayClient,
) : ViewModel() {

    var tab by mutableStateOf(DiaryTab.MINE)
        private set
    var diaries by mutableStateOf<List<DiaryDto>>(emptyList())
        private set
    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var creating by mutableStateOf(false)
        private set
    var message by mutableStateOf<String?>(null)

    init {
        load(initial = true)
        viewModelScope.launch {
            gateway.events.collect { event ->
                when (event.event) {
                    "diary:created", "diary:updated", "diary:deleted",
                    "diary:shared", "diary:unshared" -> load(initial = false)
                }
            }
        }
    }

    fun selectTab(t: DiaryTab) {
        if (tab == t) return
        tab = t
        diaries = emptyList()
        load(initial = true)
    }

    fun load(initial: Boolean) {
        viewModelScope.launch {
            if (initial) loading = true
            error = null
            try {
                diaries = repo.diaries(tab.key)
            } catch (e: ApiException) {
                if (diaries.isEmpty()) error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun createDiary(name: String, onCreated: (Long) -> Unit) {
        val trimmed = name.trim()
        if (trimmed.isEmpty()) return
        viewModelScope.launch {
            creating = true
            try {
                val d = repo.create(trimmed)
                if (tab == DiaryTab.MINE) diaries = diaries + d
                onCreated(d.id)
            } catch (e: ApiException) {
                message = e.message ?: "Не удалось создать ежедневник"
            } finally {
                creating = false
            }
        }
    }

    fun consumeMessage() { message = null }
}
