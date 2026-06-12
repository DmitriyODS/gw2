package com.kodass.groovework.ui.tasks

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.dto.CreateTaskRequest
import com.kodass.groovework.data.dto.DeptRef
import com.kodass.groovework.data.dto.TaskDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.repo.TasksRepository
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.longField
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromJsonElement

private const val PER_PAGE = 30

class TasksViewModel(
    private val repo: TasksRepository,
    gateway: GatewayClient,
    private val json: Json,
) : ViewModel() {
    var tab by mutableStateOf("active")
        private set
    var search by mutableStateOf("")
        private set
    var items by mutableStateOf<List<TaskDto>>(emptyList())
        private set
    var loading by mutableStateOf(true)
        private set
    var refreshing by mutableStateOf(false)
        private set
    var loadingMore by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var total by mutableIntStateOf(0)
        private set

    var departments by mutableStateOf<List<DeptRef>>(emptyList())
        private set
    var creating by mutableStateOf(false)
        private set
    var createError by mutableStateOf<String?>(null)

    private var page = 1
    private var searchJob: Job? = null

    val hasMore: Boolean
        get() = items.size < total

    init {
        reload()
        // Изменения из карточки задачи (цвет, избранное, архив и т.д.) — мгновенно в список.
        viewModelScope.launch {
            repo.taskChanges.collect { updated ->
                if (items.none { it.id == updated.id }) {
                    silentRefresh()
                    return@collect
                }
                items = items
                    .map { if (it.id == updated.id) updated else it }
                    .filter { task ->
                        when (tab) {
                            "favorites" -> task.isFavorite
                            "archive" -> task.isArchived
                            else -> !task.isArchived
                        }
                    }
            }
        }
        viewModelScope.launch {
            gateway.events.collect { event ->
                when (event.event) {
                    "task:updated" -> {
                        val updated = runCatching {
                            json.decodeFromJsonElement<TaskDto>(event.data ?: return@collect)
                        }.getOrNull() ?: return@collect
                        // В броадкастах вырезан личный цвет — сохраняем свой.
                        items = items.map {
                            if (it.id == updated.id) updated.copy(color = it.color) else it
                        }
                    }
                    "task:deleted" -> {
                        val id = event.data.longField("task_id") ?: return@collect
                        items = items.filter { it.id != id }
                    }
                    "task:created", "task:archived", "task:restored" -> silentRefresh()
                }
            }
        }
    }

    fun setTabValue(value: String) {
        if (tab == value) return
        tab = value
        reload()
    }

    fun setSearchValue(value: String) {
        search = value
        searchJob?.cancel()
        searchJob = viewModelScope.launch {
            delay(350)
            reload()
        }
    }

    fun reload() {
        viewModelScope.launch {
            loading = true
            error = null
            try {
                page = 1
                val result = repo.tasks(tab, search, page, PER_PAGE)
                items = result.items
                total = result.total
            } catch (e: ApiException) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    // Pull-to-refresh: список остаётся на экране, крутится только индикатор сверху.
    fun pullRefresh() {
        if (refreshing) return
        viewModelScope.launch {
            refreshing = true
            try {
                page = 1
                val result = repo.tasks(tab, search, page, PER_PAGE)
                items = result.items
                total = result.total
                error = null
            } catch (e: ApiException) {
                if (items.isEmpty()) error = e.message
            } finally {
                refreshing = false
            }
        }
    }

    private fun silentRefresh() {
        viewModelScope.launch {
            try {
                val result = repo.tasks(tab, search, 1, PER_PAGE.coerceAtLeast(items.size))
                items = result.items
                total = result.total
            } catch (_: Exception) {
            }
        }
    }

    fun loadMore() {
        if (loadingMore || loading || !hasMore) return
        viewModelScope.launch {
            loadingMore = true
            try {
                val result = repo.tasks(tab, search, page + 1, PER_PAGE)
                page += 1
                val known = items.map { it.id }.toHashSet()
                items = items + result.items.filter { it.id !in known }
                total = result.total
            } catch (_: Exception) {
            } finally {
                loadingMore = false
            }
        }
    }

    fun toggleFavorite(task: TaskDto) {
        val before = items
        items = items.map { if (it.id == task.id) it.copy(isFavorite = !task.isFavorite) else it }
        viewModelScope.launch {
            try {
                val isFavorite = repo.toggleFavorite(task.id)
                if (tab == "favorites" && !isFavorite) {
                    items = items.filter { it.id != task.id }
                }
            } catch (_: Exception) {
                items = before
            }
        }
    }

    fun loadDepartments() {
        if (departments.isNotEmpty()) return
        viewModelScope.launch {
            runCatching { departments = repo.departments() }
        }
    }

    fun create(name: String, departmentId: Long, deadline: String?, onDone: (TaskDto) -> Unit) {
        if (creating) return
        viewModelScope.launch {
            creating = true
            createError = null
            try {
                val task = repo.create(CreateTaskRequest(name = name, departmentId = departmentId, deadline = deadline))
                reload()
                onDone(task)
            } catch (e: ApiException) {
                createError = e.message
            } finally {
                creating = false
            }
        }
    }
}
