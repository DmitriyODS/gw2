package com.kodass.groovework.ui.tasks

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.api.AuthApi
import com.kodass.groovework.data.dto.CommentDto
import com.kodass.groovework.data.dto.StageDto
import com.kodass.groovework.data.dto.TaskDto
import com.kodass.groovework.data.dto.UnitDto
import com.kodass.groovework.data.dto.UpdateTaskRequest
import com.kodass.groovework.data.dto.UpdateUnitRequest
import com.kodass.groovework.data.dto.UserDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.data.repo.TasksRepository
import com.kodass.groovework.data.repo.UnitsRepository
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.longField
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromJsonElement

class TaskDetailViewModel(
    private val repo: TasksRepository,
    private val unitsRepo: UnitsRepository,
    private val authApi: AuthApi,
    session: SessionManager,
    gateway: GatewayClient,
    private val json: Json,
    val taskId: Long,
) : ViewModel() {
    val myUserId: Long? = (session.authState.value as? AuthState.LoggedIn)?.claims?.userId
    private val myRoleLevel: Int = (session.authState.value as? AuthState.LoggedIn)?.claims?.roleLevel ?: 1

    var task by mutableStateOf<TaskDto?>(null)
        private set
    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var actionError by mutableStateOf<String?>(null)

    var comments by mutableStateOf<List<CommentDto>>(emptyList())
        private set
    var stages by mutableStateOf<List<StageDto>>(emptyList())
        private set
    var directory by mutableStateOf<List<UserDto>>(emptyList())
        private set

    var units by mutableStateOf<List<UnitDto>>(emptyList())
        private set
    var unitsLoading by mutableStateOf(false)
        private set

    fun canManageUnit(unit: UnitDto): Boolean = unit.userId == myUserId || myRoleLevel >= 2

    var commentInput by mutableStateOf("")
    var sendingComment by mutableStateOf(false)
        private set

    init {
        load()
        viewModelScope.launch {
            gateway.events.collect { event ->
                when (event.event) {
                    "task:updated" -> {
                        val updated = runCatching {
                            json.decodeFromJsonElement<TaskDto>(event.data ?: return@collect)
                        }.getOrNull() ?: return@collect
                        if (updated.id == taskId) {
                            task = updated.copy(color = task?.color)
                        }
                    }
                    "comment:new", "comment:updated" -> {
                        val comment = runCatching {
                            json.decodeFromJsonElement<CommentDto>(event.data ?: return@collect)
                        }.getOrNull() ?: return@collect
                        if (comment.taskId != taskId) return@collect
                        comments = if (comments.any { it.id == comment.id }) {
                            comments.map { if (it.id == comment.id) comment else it }
                        } else {
                            comments + comment
                        }
                    }
                    "comment:deleted" -> {
                        if (event.data.longField("task_id") != taskId) return@collect
                        val commentId = event.data.longField("comment_id") ?: return@collect
                        comments = comments.filter { it.id != commentId }
                    }
                    "task:archived", "task:restored" -> {
                        if (event.data.longField("task_id") == taskId) refreshTask()
                    }
                    "unit:started" -> {
                        val unit = runCatching {
                            json.decodeFromJsonElement<UnitDto>(event.data ?: return@collect)
                        }.getOrNull() ?: return@collect
                        if (unit.taskId == taskId && units.none { it.id == unit.id }) {
                            units = listOf(unit) + units
                        }
                    }
                    "unit:stopped" -> {
                        if (event.data.longField("task_id") != taskId) return@collect
                        // datetime_end приходит в событии — перезагружаем список,
                        // чтобы строка перестала тикать и показала длительность.
                        loadUnits()
                    }
                    "unit:updated" -> {
                        val unit = runCatching {
                            json.decodeFromJsonElement<UnitDto>(event.data ?: return@collect)
                        }.getOrNull() ?: return@collect
                        if (unit.taskId == taskId) {
                            units = units.map { if (it.id == unit.id) unit else it }
                        }
                    }
                    "unit:deleted" -> {
                        val unitId = event.data.longField("unit_id") ?: return@collect
                        units = units.filter { it.id != unitId }
                    }
                }
            }
        }
    }

    fun loadUnits() {
        viewModelScope.launch {
            unitsLoading = true
            try {
                units = unitsRepo.taskUnits(taskId)
            } catch (_: Exception) {
            } finally {
                unitsLoading = false
            }
        }
    }

    fun deleteUnit(unit: UnitDto) {
        viewModelScope.launch {
            try {
                unitsRepo.deleteUnit(unit.id)
                units = units.filter { it.id != unit.id }
            } catch (e: ApiException) {
                actionError = e.message
            }
        }
    }

    fun updateUnit(unitId: Long, body: UpdateUnitRequest, onResult: (Result<Unit>) -> Unit) {
        viewModelScope.launch {
            try {
                val updated = unitsRepo.updateUnit(unitId, body)
                units = units.map { if (it.id == updated.id) updated else it }
                onResult(Result.success(Unit))
            } catch (e: ApiException) {
                onResult(Result.failure(e))
            }
        }
    }

    fun load() {
        viewModelScope.launch {
            loading = true
            error = null
            try {
                task = repo.task(taskId)
                comments = repo.comments(taskId)
            } catch (e: ApiException) {
                error = e.message
            } finally {
                loading = false
            }
            runCatching { stages = repo.stages() }
        }
    }

    private fun refreshTask() {
        viewModelScope.launch { runCatching { task = repo.task(taskId) } }
    }

    fun loadDirectory() {
        if (directory.isNotEmpty()) return
        viewModelScope.launch {
            runCatching { directory = apiCall(json) { authApi.directory() } }
        }
    }

    private fun mutate(block: suspend () -> TaskDto) {
        viewModelScope.launch {
            actionError = null
            try {
                task = block()
                task?.let { repo.notifyTaskChanged(it) }
            } catch (e: ApiException) {
                actionError = e.message
            }
        }
    }

    fun rename(name: String) = mutate { repo.update(taskId, UpdateTaskRequest(name = name)) }

    fun setDeadline(date: String?) = mutate { repo.update(taskId, UpdateTaskRequest(deadline = date)) }

    fun setStage(stageId: Long?) = mutate { repo.setStage(taskId, stageId) }

    fun setResponsible(userId: Long?) = mutate { repo.setResponsible(taskId, userId) }

    // Завершение задачи: отправляем в архив и закрываем экран. onDone зовём
    // только при успехе — при отказе (например, активный юнит) остаёмся на
    // экране и показываем actionError.
    fun complete(onDone: () -> Unit) {
        viewModelScope.launch {
            actionError = null
            try {
                task = repo.archive(taskId)
                task?.let { repo.notifyTaskChanged(it) }
                onDone()
            } catch (e: ApiException) {
                actionError = e.message
            }
        }
    }

    fun restore() = mutate { repo.restore(taskId) }

    fun toggleFavorite() {
        val current = task ?: return
        task = current.copy(isFavorite = !current.isFavorite)
        viewModelScope.launch {
            try {
                val isFavorite = repo.toggleFavorite(taskId)
                task = task?.copy(isFavorite = isFavorite)
                task?.let { repo.notifyTaskChanged(it) }
            } catch (_: Exception) {
                task = current
            }
        }
    }

    fun setColor(color: String?) {
        val current = task ?: return
        task = current.copy(color = color)
        viewModelScope.launch {
            try {
                repo.setColor(taskId, color)
                task?.let { repo.notifyTaskChanged(it) }
            } catch (_: Exception) {
                task = current
            }
        }
    }

    fun addComment() {
        val text = commentInput.trim()
        if (text.isEmpty() || sendingComment) return
        viewModelScope.launch {
            sendingComment = true
            try {
                val comment = repo.addComment(taskId, text)
                if (comments.none { it.id == comment.id }) comments = comments + comment
                commentInput = ""
            } catch (e: ApiException) {
                actionError = e.message
            } finally {
                sendingComment = false
            }
        }
    }

    fun deleteComment(comment: CommentDto) {
        viewModelScope.launch {
            try {
                repo.deleteComment(taskId, comment.id)
                comments = comments.filter { it.id != comment.id }
            } catch (e: ApiException) {
                actionError = e.message
            }
        }
    }
}
