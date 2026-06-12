package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.AttachmentDto
import com.kodass.groovework.data.dto.ConversationItemDto
import com.kodass.groovework.data.dto.ForwardRequest
import com.kodass.groovework.data.dto.MessageDto
import com.kodass.groovework.data.dto.OpenConversationRequest
import com.kodass.groovework.data.dto.OpenedConversationDto
import com.kodass.groovework.data.dto.PresenceDto
import com.kodass.groovework.data.dto.SendMessageRequest
import com.kodass.groovework.data.dto.UnreadDto
import okhttp3.MultipartBody
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.Multipart
import retrofit2.http.POST
import retrofit2.http.Part
import retrofit2.http.Path
import retrofit2.http.Query

interface MessengerApi {
    @GET("api/messenger/conversations")
    suspend fun conversations(): List<ConversationItemDto>

    @POST("api/messenger/conversations")
    suspend fun openConversation(@Body body: OpenConversationRequest): OpenedConversationDto

    @GET("api/messenger/conversations/{id}/messages")
    suspend fun messages(
        @Path("id") conversationId: Long,
        @Query("before_id") beforeId: Long? = null,
        @Query("after_id") afterId: Long? = null,
        @Query("limit") limit: Int = 50,
    ): List<MessageDto>

    @POST("api/messenger/conversations/{id}/messages")
    suspend fun send(@Path("id") conversationId: Long, @Body body: SendMessageRequest): MessageDto

    @POST("api/messenger/conversations/{id}/read")
    suspend fun markRead(@Path("id") conversationId: Long)

    @GET("api/messenger/unread")
    suspend fun unread(): UnreadDto

    @GET("api/messenger/presence")
    suspend fun presence(): PresenceDto

    @Multipart
    @POST("api/messenger/uploads")
    suspend fun upload(@Part file: MultipartBody.Part): AttachmentDto

    @DELETE("api/messenger/messages/{id}")
    suspend fun deleteMessage(@Path("id") messageId: Long, @Query("scope") scope: String = "me")

    @POST("api/messenger/forward")
    suspend fun forward(@Body body: ForwardRequest)
}
