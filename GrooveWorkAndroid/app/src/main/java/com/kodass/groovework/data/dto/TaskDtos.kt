package com.kodass.groovework.data.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class DeptRef(
    val id: Long,
    val name: String = "",
)

@Serializable
data class StageDto(
    val id: Long,
    val name: String = "",
    val color: String? = null,
    val order: Int = 0,
    @SerialName("company_id") val companyId: Long? = null,
)

@Serializable
data class TaskDto(
    val id: Long,
    val name: String = "",
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("received_at") val receivedAt: String? = null,
    @SerialName("author_id") val authorId: Long? = null,
    val author: UserRef? = null,
    @SerialName("responsible_user_id") val responsibleUserId: Long? = null,
    val responsible: UserRef? = null,
    @SerialName("department_id") val departmentId: Long? = null,
    val department: DeptRef? = null,
    @SerialName("stage_id") val stageId: Long? = null,
    val stage: StageDto? = null,
    val deadline: String? = null,
    @SerialName("is_archived") val isArchived: Boolean = false,
    @SerialName("archived_at") val archivedAt: String? = null,
    @SerialName("is_favorite") val isFavorite: Boolean = false,
    val color: String? = null,
    @SerialName("link_yougile") val linkYougile: String? = null,
    @SerialName("has_units") val hasUnits: Boolean = false,
    @SerialName("active_users") val activeUsers: List<UserRef> = emptyList(),
)

@Serializable
data class PagedTasksDto(
    val items: List<TaskDto> = emptyList(),
    val page: Int = 1,
    @SerialName("per_page") val perPage: Int = 30,
    val total: Int = 0,
)

@Serializable
data class CreateTaskRequest(
    val name: String,
    @SerialName("department_id") val departmentId: Long,
    @SerialName("responsible_user_id") val responsibleUserId: Long? = null,
    @SerialName("stage_id") val stageId: Long? = null,
    val deadline: String? = null,
)

@Serializable
data class UpdateTaskRequest(
    val name: String? = null,
    @SerialName("department_id") val departmentId: Long? = null,
    val deadline: String? = null,
)

@Serializable
data class SetResponsibleRequest(
    @SerialName("responsible_user_id") val responsibleUserId: Long?,
)

@Serializable
data class SetStageRequest(
    @SerialName("stage_id") val stageId: Long?,
)

@Serializable
data class SetColorRequest(val color: String?)

@Serializable
data class FavoriteDto(
    @SerialName("is_favorite") val isFavorite: Boolean = false,
)

@Serializable
data class CommentDto(
    val id: Long,
    @SerialName("task_id") val taskId: Long,
    @SerialName("author_id") val authorId: Long? = null,
    val author: UserRef? = null,
    val text: String = "",
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class CommentsDto(val items: List<CommentDto> = emptyList())

@Serializable
data class CommentRequest(val text: String)
