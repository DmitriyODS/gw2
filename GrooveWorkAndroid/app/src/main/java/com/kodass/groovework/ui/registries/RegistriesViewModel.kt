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
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.longField
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromJsonElement

class RegistriesViewModel(
    private val repo: RegistriesRepository,
    private val session: SessionManager,
    gateway: GatewayClient,
    private val json: Json,
) : ViewModel() {

    var registries by mutableStateOf<List<RegistryDto>>(emptyList())
        private set
    var selectedId by mutableStateOf<Long?>(null)
        private set
    var loadingRegistries by mutableStateOf(true)
        private set
    var registriesError by mutableStateOf<String?>(null)
        private set

    val selected: RegistryDto? get() = registries.firstOrNull { it.id == selectedId }

    // ── Записи выбранного реестра ──
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

    // Массовый выбор записей (для удаления).
    var selectedIds by mutableStateOf<Set<Long>>(emptySet())
        private set
    var message by mutableStateOf<String?>(null)

    private var searchJob: Job? = null

    init {
        loadRegistries(initial = true)
        viewModelScope.launch {
            gateway.events.collect { event ->
                when (event.event) {
                    "registry:created", "registry:updated", "registry:deleted" ->
                        if (companyMatches(event.data.longField("company_id"))) loadRegistries(initial = false)
                    "record:created" -> onRecordEvent(event.data.longField("registry_id"), event.data.longField("company_id")) { refresh() }
                    "record:updated" -> onRecordEvent(event.data.longField("registry_id"), event.data.longField("company_id")) {
                        val updated = runCatching { json.decodeFromJsonElement<RegistryRecordDto>(event.data!!) }.getOrNull()
                        if (updated != null) records = records.map { if (it.id == updated.id) updated else it }
                    }
                    "record:deleted" -> onRecordEvent(event.data.longField("registry_id"), event.data.longField("company_id")) {
                        val id = event.data.longField("id") ?: return@onRecordEvent
                        if (records.any { it.id == id }) {
                            records = records.filter { it.id != id }
                            total = (total - 1).coerceAtLeast(0)
                        }
                    }
                    "record:bulk-deleted" -> onRecordEvent(event.data.longField("registry_id"), event.data.longField("company_id")) { refresh() }
                }
            }
        }
    }

    private fun companyMatches(eventCompanyId: Long?): Boolean {
        val mine = (session.authState.value as? AuthState.LoggedIn)?.claims?.companyId ?: return true
        return eventCompanyId == null || eventCompanyId == mine
    }

    private inline fun onRecordEvent(registryId: Long?, companyId: Long?, block: () -> Unit) {
        if (registryId == selectedId && companyMatches(companyId)) block()
    }

    fun loadRegistries(initial: Boolean) {
        viewModelScope.launch {
            if (initial) loadingRegistries = true
            registriesError = null
            try {
                val list = repo.registries()
                registries = list
                // Сохраняем выбор, если реестр всё ещё существует; иначе — первый.
                val keep = selectedId?.takeIf { id -> list.any { it.id == id } }
                val next = keep ?: list.firstOrNull()?.id
                if (next != selectedId) {
                    selectedId = next
                    resetAndFetch()
                } else if (initial && next != null) {
                    resetAndFetch()
                }
            } catch (e: ApiException) {
                if (registries.isEmpty()) registriesError = e.message
            } finally {
                loadingRegistries = false
            }
        }
    }

    fun select(id: Long) {
        if (id == selectedId) return
        selectedId = id
        search = ""
        sort = ""
        order = "desc"
        clearSelection()
        resetAndFetch()
    }

    private fun resetAndFetch() {
        records = emptyList()
        total = 0
        page = 1
        loadRecords(reset = true)
    }

    private fun loadRecords(reset: Boolean) {
        val registryId = selectedId ?: return
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

    fun refresh() {
        val registryId = selectedId ?: return
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
        val registryId = selectedId ?: return
        if (loadingMore || !hasMore) return
        viewModelScope.launch {
            loadingMore = true
            try {
                val next = page + 1
                val r = repo.records(registryId, search, sort, order, next, PER_PAGE)
                page = next
                // Дедуп по id — сокет-события могли уже что-то вставить.
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

    // ── Выбор записей ──
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
        val registryId = selectedId ?: return
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
