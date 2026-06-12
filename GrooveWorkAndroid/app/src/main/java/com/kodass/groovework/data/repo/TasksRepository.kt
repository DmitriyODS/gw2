package com.kodass.groovework.data.repo

import com.kodass.groovework.data.api.TasksApi
import com.kodass.groovework.data.dto.CommentDto
import com.kodass.groovework.data.dto.CommentRequest
import com.kodass.groovework.data.dto.CreateTaskRequest
import com.kodass.groovework.data.dto.DeptRef
import com.kodass.groovework.data.dto.PagedTasksDto
import com.kodass.groovework.data.dto.SetColorRequest
import com.kodass.groovework.data.dto.SetResponsibleRequest
import com.kodass.groovework.data.dto.SetStageRequest
import com.kodass.groovework.data.dto.StageDto
import com.kodass.groovework.data.dto.TaskDto
import com.kodass.groovework.data.dto.UpdateTaskRequest
import com.kodass.groovework.data.network.apiCall
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.serialization.json.Json

class TasksRepository(
    private val api: TasksApi,
    private val json: Json,
) {
    // Личные изменения (цвет, избранное) не приходят сокетом — карточка задачи
    // оповещает список напрямую, чтобы он обновлялся без перезахода.
    private val _taskChanges = MutableSharedFlow<TaskDto>(extraBufferCapacity = 32)
    val taskChanges: SharedFlow<TaskDto> = _taskChanges

    fun notifyTaskChanged(task: TaskDto) {
        _taskChanges.tryEmit(task)
    }

    suspend fun tasks(tab: String, search: String?, page: Int, perPage: Int = 30): PagedTasksDto =
        apiCall(json) { api.tasks(tab = tab, search = search?.takeIf { it.isNotBlank() }, page = page, perPage = perPage) }

    suspend fun task(id: Long): TaskDto = apiCall(json) { api.task(id) }

    suspend fun create(body: CreateTaskRequest): TaskDto = apiCall(json) { api.create(body) }

    suspend fun update(id: Long, body: UpdateTaskRequest): TaskDto = apiCall(json) { api.update(id, body) }

    suspend fun toggleFavorite(id: Long): Boolean = apiCall(json) { api.toggleFavorite(id) }.isFavorite

    suspend fun archive(id: Long): TaskDto = apiCall(json) { api.archive(id) }

    suspend fun restore(id: Long): TaskDto = apiCall(json) { api.restore(id) }

    suspend fun setColor(id: Long, color: String?) = apiCall(json) { api.setColor(id, SetColorRequest(color)) }

    suspend fun setResponsible(id: Long, userId: Long?): TaskDto =
        apiCall(json) { api.setResponsible(id, SetResponsibleRequest(userId)) }

    suspend fun setStage(id: Long, stageId: Long?): TaskDto =
        apiCall(json) { api.setStage(id, SetStageRequest(stageId)) }

    suspend fun comments(taskId: Long): List<CommentDto> = apiCall(json) { api.comments(taskId) }.items

    suspend fun addComment(taskId: Long, text: String): CommentDto =
        apiCall(json) { api.addComment(taskId, CommentRequest(text)) }

    suspend fun updateComment(taskId: Long, commentId: Long, text: String): CommentDto =
        apiCall(json) { api.updateComment(taskId, commentId, CommentRequest(text)) }

    suspend fun deleteComment(taskId: Long, commentId: Long) =
        apiCall(json) { api.deleteComment(taskId, commentId) }

    suspend fun stages(): List<StageDto> = apiCall(json) { api.stages() }

    suspend fun departments(): List<DeptRef> = apiCall(json) { api.departments() }
}
