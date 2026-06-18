package com.kodass.groovework.ui.registries

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.dto.RegistryDto
import com.kodass.groovework.data.dto.RegistryRecordDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.repo.RegistriesRepository
import com.kodass.groovework.data.repo.RegistriesRepository.Companion.PER_PAGE
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.longField
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromJsonElement

// Второй уровень раздела «Реестры» — структура и записи одного реестра.
class RegistryRecordsViewModel(
    private val registryId: Long,
    private val repo: RegistriesRepository,
    private val session: SessionManager,
    gateway: GatewayClient,
    private val json: Json,
) : ViewModel() {

    var registry by mutableStateOf<RegistryDto?>(null)
        private set
    var loadingRegistry by mutableStateOf(true)
        private set
    var registryError by mutableStateOf<String?>(null)
        private set
    // Реестр удалён (сокет-событие) — экран сам уходит назад.
    var registryGone by mutableStateOf(false)
        private set

    var records by mutableStateOf<List<RegistryRecordDto>>(emptyList())
        private set
    var total by mutableStateOf(0)
        private set
    var loadingRecords by mutableStateOf(false)
        private set
    var refreshing by mutableStateOf(false)
        private set
    var loadingMore by mutableStateOf(false)
        private set
    var recordsError by mutableStateOf<String?>(null)
        private set

    var search by mutableStateOf("")
        private set
    var sort by mutableStateOf("") // "" → по дате создания
        private set
    var order by mutableStateOf("desc")
        private set
    private var page = 1
    val hasMore: Boolean get() = records.size < total

    var selectedIds by mutableStateOf<Set<Long>>(emptySet())
        private set
    var message by mutableStateOf<String?>(null)

    private var searchJob: Job? = null

    init {
        loadRegistry()
        loadRecords(reset = true)
        viewModelScope.launch {
            gateway.events.collect { event ->
                when (event.event) {
                    "registry:updated" ->
                        if (forThisRegistry(event.data.longField("id"), event.data.longField("company_id"))) loadRegistry()
                    "registry:deleted" ->
                        if (forThisRegistry(event.data.longField("id"), event.data.longField("company_id"))) registryGone = true
                    "record:created" -> onRecordEvent(event.data) { refresh() }
                    "record:updated" -> onRecordEvent(event.data) {
                        val updated = runCatching { json.decodeFromJsonElement<RegistryRecordDto>(event.data!!) }.getOrNull()
                        if (updated != null) records = records.map { if (it.id == updated.id) updated else it }
                    }
                    "record:deleted" -> onRecordEvent(event.data) {
                        val id = event.data.longField("id") ?: return@onRecordEvent
                        if (records.any { it.id == id }) {
                            records = records.filter { it.id != id }
                            total = (total - 1).coerceAtLeast(0)
                        }
                    }
                    "record:bulk-deleted" -> onRecordEvent(event.data) { refresh() }
                }
            }
        }
    }

    private fun forThisRegistry(id: Long?, companyId: Long?): Boolean =
        id == registryId && companyMatches(session, companyId)

    private inline fun onRecordEvent(data: kotlinx.serialization.json.JsonElement?, block: () -> Unit) {
        if (data.longField("registry_id") == registryId && companyMatches(session, data.longField("company_id"))) block()
    }

    fun loadRegistry() {
        viewModelScope.launch {
            loadingRegistry = true
            registryError = null
            try {
                registry = repo.registry(registryId)
            } catch (e: ApiException) {
                if (registry == null) registryError = e.message
            } finally {
                loadingRegistry = false
            }
        }
    }

    private fun loadRecords(reset: Boolean) {
        viewModelScope.launch {
            if (reset && records.isEmpty()) loadingRecords = true
            recordsError = null
            try {
                val r = repo.records(registryId, search, sort, order, 1, PER_PAGE)
                page = 1
                records = r.items
                total = r.total
            } catch (e: ApiException) {
                if (records.isEmpty()) recordsError = e.message
            } finally {
                loadingRecords = false
            }
        }
    }

    private fun resetAndFetch() {
        records = emptyList()
        total = 0
        page = 1
        loadRecords(reset = true)
    }

    fun refresh() {
        viewModelScope.launch {
            refreshing = true
            try {
                val r = repo.records(registryId, search, sort, order, 1, PER_PAGE)
                page = 1
                records = r.items
                total = r.total
                pruneSelection()
            } catch (_: ApiException) {
            } finally {
                refreshing = false
            }
        }
    }

    fun loadMore() {
        if (loadingMore || !hasMore) return
        viewModelScope.launch {
            loadingMore = true
            try {
                val next = page + 1
                val r = repo.records(registryId, search, sort, order, next, PER_PAGE)
                page = next
                val known = records.map { it.id }.toSet()
                records = records + r.items.filter { it.id !in known }
                total = r.total
            } catch (_: ApiException) {
            } finally {
                loadingMore = false
            }
        }
    }

    fun updateSearch(value: String) {
        if (value == search) return
        search = value
        searchJob?.cancel()
        searchJob = viewModelScope.launch {
            delay(300)
            resetAndFetch()
        }
    }

    fun setSortField(value: String) {
        if (value == sort) return
        sort = value
        resetAndFetch()
    }

    fun toggleOrder() {
        order = if (order == "asc") "desc" else "asc"
        resetAndFetch()
    }

    fun toggleRow(id: Long) {
        selectedIds = selectedIds.toMutableSet().apply { if (!add(id)) remove(id) }
    }

    fun toggleAll() {
        val all = records.map { it.id }.toSet()
        selectedIds = if (selectedIds.containsAll(all) && all.isNotEmpty()) emptySet() else all
    }

    fun clearSelection() {
        selectedIds = emptySet()
    }

    private fun pruneSelection() {
        val ids = records.map { it.id }.toSet()
        selectedIds = selectedIds.filterTo(mutableSetOf()) { it in ids }
    }

    fun bulkDelete() {
        val ids = selectedIds.toList()
        if (ids.isEmpty()) return
        viewModelScope.launch {
            try {
                repo.bulkDelete(registryId, ids)
                records = records.filter { it.id !in ids }
                total = (total - ids.size).coerceAtLeast(0)
                clearSelection()
                message = "Удалено записей: ${ids.size}"
            } catch (e: ApiException) {
                message = e.message
            }
        }
    }

    fun consumeMessage() {
        message = null
    }
}
