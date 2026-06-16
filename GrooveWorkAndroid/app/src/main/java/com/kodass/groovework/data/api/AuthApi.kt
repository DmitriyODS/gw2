package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.ChangeDefaultRequest
import com.kodass.groovework.data.dto.ForgotPasswordRequest
import com.kodass.groovework.data.dto.LoginRequest
import com.kodass.groovework.data.dto.RegisterRequest
import com.kodass.groovework.data.dto.RegisterResultDto
import com.kodass.groovework.data.dto.ResetPasswordRequest
import com.kodass.groovework.data.dto.ResetPasswordResultDto
import com.kodass.groovework.data.dto.SelectCompanyRequest
import com.kodass.groovework.data.dto.SessionResponse
import com.kodass.groovework.data.dto.StatusDto
import com.kodass.groovework.data.dto.SuggestLoginDto
import com.kodass.groovework.data.dto.SwitchCompanyRequest
import com.kodass.groovework.data.dto.UpdateMeRequest
import com.kodass.groovework.data.dto.UserDto
import com.kodass.groovework.data.dto.VerifyEmailRequest
import okhttp3.MultipartBody
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.Header
import retrofit2.http.Multipart
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.Part
import retrofit2.http.Path
import retrofit2.http.Query

interface AuthApi {
    // Response<…> — чтобы прочитать Set-Cookie с refresh-токеном.
    @POST("api/auth/login")
    suspend fun login(@Body body: LoginRequest): Response<SessionResponse>

    // Регистрация: сессию НЕ выдаёт (201 {status, email}); дальше — verify-email.
    @POST("api/auth/register")
    suspend fun register(@Body body: RegisterRequest): RegisterResultDto

    // Live-подсказка свободного логина по ФИО (публичный).
    @GET("api/auth/suggest-login")
    suspend fun suggestLogin(@Query("fio") fio: String): SuggestLoginDto

    // Подтверждение email (token из ссылки ИЛИ email+code) → сессия + refresh-cookie.
    @POST("api/auth/verify-email")
    suspend fun verifyEmail(@Body body: VerifyEmailRequest): Response<SessionResponse>

    @POST("api/auth/resend-verification")
    suspend fun resendVerification(@Body body: ForgotPasswordRequest): StatusDto

    @POST("api/auth/forgot-password")
    suspend fun forgotPassword(@Body body: ForgotPasswordRequest): StatusDto

    @POST("api/auth/reset-password")
    suspend fun resetPassword(@Body body: ResetPasswordRequest): ResetPasswordResultDto

    // Завершение многокомпанийного login-gate: select_token из login + выбранная компания.
    @POST("api/auth/select-company")
    suspend fun selectCompany(@Body body: SelectCompanyRequest): Response<SessionResponse>

    // Смена активной компании в существующей сессии (перевыпуск токенов).
    @POST("api/auth/switch-company")
    suspend fun switchCompany(@Body body: SwitchCompanyRequest): Response<SessionResponse>

    @POST("api/auth/refresh")
    suspend fun refresh(@Header("Cookie") cookie: String): Response<SessionResponse>

    @POST("api/auth/change-default")
    suspend fun changeDefault(@Body body: ChangeDefaultRequest): Response<SessionResponse>

    @POST("api/auth/logout")
    suspend fun logout(): Response<Unit>

    @GET("api/users/me")
    suspend fun me(): UserDto

    @PATCH("api/users/me")
    suspend fun updateMe(@Body body: UpdateMeRequest): UserDto

    @Multipart
    @POST("api/users/me/avatar")
    suspend fun uploadAvatar(@Part file: MultipartBody.Part): UserDto

    @DELETE("api/users/me/avatar")
    suspend fun deleteAvatar(): UserDto

    // all="1" — глобальный справочник (видимые сотрудники всех компаний),
    // нужен для старта чата/звонка с сотрудником другой компании.
    @GET("api/users/directory")
    suspend fun directory(
        @Query("q") query: String? = null,
        @Query("exclude_self") excludeSelf: String? = null,
        @Query("all") all: String? = null,
    ): List<UserDto>

    @GET("api/users/directory/{id}")
    suspend fun directoryUser(@Path("id") id: Long): UserDto
}
