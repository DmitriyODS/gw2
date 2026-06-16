package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.AddMemberRequest
import com.kodass.groovework.data.dto.CompanyCreateRequest
import com.kodass.groovework.data.dto.CompanyDto
import com.kodass.groovework.data.dto.CompanyListDto
import com.kodass.groovework.data.dto.CreateCompanyUserRequest
import com.kodass.groovework.data.dto.CreateInviteRequest
import com.kodass.groovework.data.dto.GrooveSettingsDto
import com.kodass.groovework.data.dto.InviteCodeDto
import com.kodass.groovework.data.dto.InvitePreviewDto
import com.kodass.groovework.data.dto.OkMessageDto
import com.kodass.groovework.data.dto.RoleIdRequest
import com.kodass.groovework.data.dto.RoleRef
import com.kodass.groovework.data.dto.SessionResponse
import com.kodass.groovework.data.dto.UpdateCompanyUserRequest
import com.kodass.groovework.data.dto.UserDto
import com.kodass.groovework.data.dto.WeekendSettingsDto
import kotlinx.serialization.json.JsonObject
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import retrofit2.http.Query

// Раздел «Компании» (authsvc). Создатель компании управляет участниками/сотрудниками,
// любой администратор компании — настройками; super-админ может всё.
interface CompaniesApi {
    // ── Компании ──────────────────────────────────────────────────────────────
    @GET("api/companies/mine")
    suspend fun mine(): CompanyListDto

    @GET("api/companies/{id}")
    suspend fun company(@Path("id") companyId: Long): CompanyDto

    @POST("api/companies")
    suspend fun create(@Body body: CompanyCreateRequest): CompanyDto

    // PATCH фич/настроек: JsonObject с явными ключами (бэк различает «передан ключ»).
    @PATCH("api/companies/{id}")
    suspend fun update(@Path("id") companyId: Long, @Body body: JsonObject): CompanyDto

    @DELETE("api/companies/{id}")
    suspend fun delete(@Path("id") companyId: Long): OkMessageDto

    @GET("api/roles")
    suspend fun roles(): List<RoleRef>

    // ── Участники ───────────────────────────────────────────────────────────────
    @GET("api/companies/{id}/members")
    suspend fun members(@Path("id") companyId: Long): List<UserDto>

    @GET("api/companies/{id}/members/candidates")
    suspend fun candidates(@Path("id") companyId: Long, @Query("q") query: String): List<UserDto>

    @POST("api/companies/{id}/members")
    suspend fun addMember(@Path("id") companyId: Long, @Body body: AddMemberRequest): OkMessageDto

    @PATCH("api/companies/{id}/members/{userId}")
    suspend fun setMemberRole(
        @Path("id") companyId: Long,
        @Path("userId") userId: Long,
        @Body body: RoleIdRequest,
    ): OkMessageDto

    @DELETE("api/companies/{id}/members/{userId}")
    suspend fun removeMember(@Path("id") companyId: Long, @Path("userId") userId: Long): OkMessageDto

    // ── Сотрудники компании (создатель) ──────────────────────────────────────────
    @POST("api/companies/{id}/users")
    suspend fun createCompanyUser(
        @Path("id") companyId: Long,
        @Body body: CreateCompanyUserRequest,
    ): UserDto

    @PATCH("api/companies/{id}/users/{userId}")
    suspend fun updateCompanyMember(
        @Path("id") companyId: Long,
        @Path("userId") userId: Long,
        @Body body: UpdateCompanyUserRequest,
    ): UserDto

    @POST("api/companies/{id}/users/{userId}/reset-password")
    suspend fun resetCompanyMemberPassword(
        @Path("id") companyId: Long,
        @Path("userId") userId: Long,
    ): OkMessageDto

    // ── Email-приглашения ────────────────────────────────────────────────────────
    @POST("api/companies/{id}/invites")
    suspend fun createInvite(@Path("id") companyId: Long, @Body body: CreateInviteRequest): OkMessageDto

    @GET("api/companies/invites/{token}")
    suspend fun invitePreview(@Path("token") token: String): InvitePreviewDto

    // accept/join возвращают сессию + refresh-cookie → Response для чтения Set-Cookie.
    @POST("api/companies/invites/{token}/accept")
    suspend fun acceptInvite(@Path("token") token: String): Response<SessionResponse>

    @POST("api/companies/join/{code}")
    suspend fun joinByCode(@Path("code") code: String): Response<SessionResponse>

    // ── Настройки компании (per-company id) ───────────────────────────────────────
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
