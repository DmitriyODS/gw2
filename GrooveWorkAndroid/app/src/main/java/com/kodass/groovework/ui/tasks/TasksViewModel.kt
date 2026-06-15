package com.kodass.groovework.ui.tasks

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
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

// Вкладки задач — общий список для экрана (свайп-пейджер) и VM.
val taskTabs = listOf("active" to "Активные", "favorites" to "Избранные", "archive" to "Архив")

// Состояние на каждую вкладку (свайп между Активные/Избранные/Архив — у каждой
// свой список, пагинация и индикаторы, чтобы вкладки не «прыгали» при свайпе).
class TasksViewModel(
    private val repo: TasksRepository,
    gateway: GatewayClient,
    private val json: Json,
) : ViewModel() {
    var tab by mutableStateOf("active")
        private set
    var search by mutableStateOf("")
        private set

    private val itemsState = mutableStateMapOf<String, List<TaskDto>>()
    private val totalState = mutableStateMapOf<String, Int>()
    private val loadingState = mutableStateMapOf<String, Boolean>()
    private val refreshingState = mutableStateMapOf<String, Boolean>()
    private val loadingMoreState = mutableStateMapOf<String, Boolean>()
    private val errorState = mutableStateMapOf<String, String?>()
    private val pageState = mutableMapOf<String, Int>()
    private val loaded = mutableSetOf<String>()

    fun items(tabKey: String): List<TaskDto> = itemsState[tabKey] ?: emptyList()
    fun isLoading(tabKey: String): Boolean = loadingState[tabKey] ?: true
    fun isRefreshing(tabKey: String): Boolean = refreshingState[tabKey] ?: false
    fun isLoadingMore(tabKey: String): Boolean = loadingMoreState[tabKey] ?: false
    fun errorOf(tabKey: String): String? = errorState[tabKey]
    fun hasMore(tabKey: String): Boolean = items(tabKey).size < (totalState[tabKey] ?: 0)

    var departments by mutableStateOf<List<DeptRef>>(emptyList())
        private set
    var creating by mutableStateOf(false)
        private set
    var createError by mutableStateOf<String?>(null)

    private var searchJob: Job? = null

    init {
        ensureLoaded("active")
        // Локальные мутации из карточки задачи — мгновенно в загруженные вкладки;
        // членство (избранное/архив) могло измениться → тихо перезагружаем.
        viewModelScope.launch {
            repo.taskChanges.collect { updated ->
                loaded.forEach { key ->
                    if (items(key).any { it.id == updated.id }) {
                        itemsState[key] = items(key).map { if (it.id == updated.id) updated else it }
                    }
                }
                refreshLoaded()
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
                        loaded.forEach { key ->
                            itemsState[key] = items(key).map {
                                if (it.id == updated.id) updated.copy(color = it.color) else it
                            }
                        }
                    }
                    "task:deleted" -> {
                        val id = event.data.longField("task_id") ?: return@collect
                        loaded.forEach { key -> itemsState[key] = items(key).filter { it.id != id } }
                    }
                    "task:created", "task:archived", "task:restored" -> refreshLoaded()
                }
            }
        }
    }

    fun selectTab(key: String) {
        if (tab != key) tab = key
        ensureLoaded(key)
    }

    fun ensureLoaded(key: String) {
        if (key in loaded) return
        loadFirst(key)
    }

    fun reload(key: String) = loadFirst(key)

    private fun loadFirst(key: String) {
        viewModelScope.launch {
            if (items(key).isEmpty()) loadingState[key] = true
            errorState[key] = null
            try {
                pageState[key] = 1
                val r = repo.tasks(key, search, 1, PER_PAGE)
                itemsState[key] = r.items
                totalState[key] = r.total
                loaded.add(key)
            } catch (e: ApiException) {
                if (items(key).isEmpty()) errorState[key] = e.message
            } finally {
                loadingState[key] = false
            }
        }
    }

    fun pullRefresh(key: String) {
        if (isRefreshing(key)) return
        viewModelScope.launch {
            refreshingState[key] = true
            try {
                pageState[key] = 1
                val r = repo.tasks(key, search, 1, PER_PAGE)
                itemsState[key] = r.items
                totalState[key] = r.total
                loaded.add(key)
                errorState[key] = null
            } catch (e: ApiException) {
                if (items(key).isEmpty()) errorState[key] = e.message
            } finally {
                refreshingState[key] = false
            }
        }
    }

    // Тихое фоновое обновление всех загруженных вкладок (без спиннеров) — вызывается
    // периодически и при входе/смене компании, чтобы данные были актуальны.
    fun backgroundRefresh() = refreshLoaded()

    private fun refreshLoaded() = loaded.toList().forEach { silentRefresh(it) }

    private fun silentRefresh(key: String) {
        viewModelScope.launch {
            try {
                val size = items(key).size.coerceAtLeast(PER_PAGE)
                val r = repo.tasks(key, search, 1, size)
                itemsState[key] = r.items
                totalState[key] = r.total
                loaded.add(key)
            } catch (_: Exception) {
            }
        }
    }

    fun setSearchValue(value: String) {
        search = value
        loaded.clear() // под новый запрос перезагрузятся все вкладки (текущая — сразу)
        searchJob?.cancel()
        searchJob = viewModelScope.launch {
            delay(350)
            loadFirst(tab)
        }
    }

    fun loadMore(key: String) {
        if (isLoadingMore(key) || isLoading(key) || !hasMore(key)) return
        viewModelScope.launch {
            loadingMoreState[key] = true
            try {
                val next = (pageState[key] ?: 1) + 1
                val r = repo.tasks(key, search, next, PER_PAGE)
                pageState[key] = next
                val known = items(key).map { it.id }.toHashSet()
                itemsState[key] = items(key) + r.items.filter { it.id !in known }
                totalState[key] = r.total
            } catch (_: Exception) {
            } finally {
                loadingMoreState[key] = false
            }
        }
    }

    fun toggleFavorite(task: TaskDto) {
        val newFav = !task.isFavorite
        loaded.forEach { key ->
            itemsState[key] = items(key).map { if (it.id == task.id) it.copy(isFavorite = newFav) else it }
        }
        viewModelScope.launch {
            try {
                val isFavorite = repo.toggleFavorite(task.id)
                if (!isFavorite) {
                    itemsState["favorites"] = items("favorites").filter { it.id != task.id }
                } else if ("favorites" in loaded) {
                    silentRefresh("favorites")
                }
            } catch (_: Exception) {
                loaded.forEach { key ->
                    itemsState[key] = items(key).map { if (it.id == task.id) it.copy(isFavorite = task.isFavorite) else it }
                }
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
                reload("active")
                onDone(task)
            } catch (e: ApiException) {
                createError = e.message
            } finally {
                creating = false
            }
        }
    }
}
