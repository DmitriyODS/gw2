package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.AiIndexingDto
import com.kodass.groovework.data.dto.AiReindexDto
import com.kodass.groovework.data.dto.AiSettingsDto
import com.kodass.groovework.data.dto.AiSettingsUpdate
import com.kodass.groovework.data.dto.AiTestDto
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path

// Нейро-функции компании (aisvc; nginx-regex /api/companies/<id>/ai-settings).
interface AiApi {
    @GET("api/companies/{id}/ai-settings")
    suspend fun settings(@Path("id") companyId: Long): AiSettingsDto

    @PUT("api/companies/{id}/ai-settings")
    suspend fun updateSettings(
        @Path("id") companyId: Long,
        @Body body: AiSettingsUpdate,
    ): AiSettingsDto

    @POST("api/companies/{id}/ai-settings/test")
    suspend fun test(@Path("id") companyId: Long): AiTestDto

    @GET("api/companies/{id}/ai-settings/indexing")
    suspend fun indexing(@Path("id") companyId: Long): AiIndexingDto

    @POST("api/companies/{id}/ai-settings/reindex-tasks")
    suspend fun reindex(@Path("id") companyId: Long): AiReindexDto
}
