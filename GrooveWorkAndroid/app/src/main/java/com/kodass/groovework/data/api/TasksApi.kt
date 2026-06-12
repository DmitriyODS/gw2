package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.CommentDto
import com.kodass.groovework.data.dto.CommentRequest
import com.kodass.groovework.data.dto.CommentsDto
import com.kodass.groovework.data.dto.CreateTaskRequest
import com.kodass.groovework.data.dto.DeptRef
import com.kodass.groovework.data.dto.FavoriteDto
import com.kodass.groovework.data.dto.PagedTasksDto
import com.kodass.groovework.data.dto.SetColorRequest
import com.kodass.groovework.data.dto.SetResponsibleRequest
import com.kodass.groovework.data.dto.SetStageRequest
import com.kodass.groovework.data.dto.StageDto
import com.kodass.groovework.data.dto.TaskDto
import com.kodass.groovework.data.dto.UpdateTaskRequest
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import retrofit2.http.Query

interface TasksApi {
    @GET("api/tasks")
    suspend fun tasks(
        @Query("tab") tab: String = "active",
        @Query("search") search: String? = null,
        @Query("sort") sort: String = "last_activity",
        @Query("page") page: Int = 1,
        @Query("per_page") perPage: Int = 30,
    ): PagedTasksDto

    @GET("api/tasks/{id}")
    suspend fun task(@Path("id") id: Long): TaskDto

    @POST("api/tasks")
    suspend fun create(@Body body: CreateTaskRequest): TaskDto

    @PATCH("api/tasks/{id}")
    suspend fun update(@Path("id") id: Long, @Body body: UpdateTaskRequest): TaskDto

    @POST("api/tasks/{id}/favorite")
    suspend fun toggleFavorite(@Path("id") id: Long): FavoriteDto

    @POST("api/tasks/{id}/archive")
    suspend fun archive(@Path("id") id: Long): TaskDto

    @POST("api/tasks/{id}/restore")
    suspend fun restore(@Path("id") id: Long): TaskDto

    @PUT("api/tasks/{id}/color")
    suspend fun setColor(@Path("id") id: Long, @Body body: SetColorRequest)

    @PATCH("api/tasks/{id}/responsible")
    suspend fun setResponsible(@Path("id") id: Long, @Body body: SetResponsibleRequest): TaskDto

    @PATCH("api/tasks/{id}/stage")
    suspend fun setStage(@Path("id") id: Long, @Body body: SetStageRequest): TaskDto

    @GET("api/tasks/{id}/comments")
    suspend fun comments(@Path("id") taskId: Long): CommentsDto

    @POST("api/tasks/{id}/comments")
    suspend fun addComment(@Path("id") taskId: Long, @Body body: CommentRequest): CommentDto

    @PATCH("api/tasks/{id}/comments/{commentId}")
    suspend fun updateComment(
        @Path("id") taskId: Long,
        @Path("commentId") commentId: Long,
        @Body body: CommentRequest,
    ): CommentDto

    @DELETE("api/tasks/{id}/comments/{commentId}")
    suspend fun deleteComment(@Path("id") taskId: Long, @Path("commentId") commentId: Long)

    @GET("api/stages")
    suspend fun stages(): List<StageDto>

    @GET("api/departments")
    suspend fun departments(): List<DeptRef>
}
