package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.GrooveSettingsDto
import com.kodass.groovework.data.dto.InviteCodeDto
import com.kodass.groovework.data.dto.WeekendSettingsDto
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path

// Настройки компании (authsvc). DIRECTOR+ — своей компании.
interface CompaniesApi {
    @GET("api/companies/{id}/weekend-settings")
    suspend fun weekendSettings(@Path("id") companyId: Long): WeekendSettingsDto

    @PUT("api/companies/{id}/weekend-settings")
    suspend fun updateWeekendSettings(
        @Path("id") companyId: Long,
        @Body body: WeekendSettingsDto,
    ): WeekendSettingsDto

    @GET("api/companies/{id}/groove-settings")
    suspend fun grooveSettings(@Path("id") companyId: Long): GrooveSettingsDto

    @PUT("api/companies/{id}/groove-settings")
    suspend fun updateGrooveSettings(
        @Path("id") companyId: Long,
        @Body body: GrooveSettingsDto,
    ): GrooveSettingsDto

    @GET("api/companies/{id}/invite")
    suspend fun invite(@Path("id") companyId: Long): InviteCodeDto

    @POST("api/companies/{id}/invite")
    suspend fun regenerateInvite(@Path("id") companyId: Long): InviteCodeDto
}
