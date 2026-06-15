package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.YougileConnectRequest
import com.kodass.groovework.data.dto.YougileLoginRequest
import com.kodass.groovework.data.dto.YougileNamedDto
import com.kodass.groovework.data.dto.YougileRefDto
import com.kodass.groovework.data.dto.YougileRotateRequest
import com.kodass.groovework.data.dto.YougileSettingsDto
import com.kodass.groovework.data.dto.YougileStatusDto
import kotlinx.serialization.json.JsonObject
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Query

// Интеграция YouGile (tasksvc). Личный коннект + админ-визард компании.
interface YougileApi {
    @GET("api/yougile/status")
    suspend fun status(): YougileStatusDto

    @POST("api/yougile/account")
    suspend fun connect(@Body body: YougileConnectRequest)

    @DELETE("api/yougile/account")
    suspend fun disconnect()

    @POST("api/yougile/account/rotate")
    suspend fun rotate(@Body body: YougileRotateRequest)

    @POST("api/yougile/companies/lookup")
    suspend fun lookupCompanies(@Body body: YougileLoginRequest): List<YougileRefDto>

    @GET("api/yougile/projects")
    suspend fun projects(): List<YougileNamedDto>

    @GET("api/yougile/boards")
    suspend fun boards(@Query("projectId") projectId: String): List<YougileNamedDto>

    @GET("api/yougile/columns")
    suspend fun columns(@Query("boardId") boardId: String): List<YougileNamedDto>

    @GET("api/yougile/company-settings")
    suspend fun companySettings(): YougileSettingsDto

    // Тело — JsonObject с явными null для очистки (kotlinx explicitNulls=false
    // иначе вырезал бы null, а бэк различает «ключ передан» / «не передан»).
    @PUT("api/yougile/company-settings")
    suspend fun updateCompanySettings(@Body body: JsonObject): YougileSettingsDto

    @POST("api/yougile/reset")
    suspend fun reset(): YougileSettingsDto
}
