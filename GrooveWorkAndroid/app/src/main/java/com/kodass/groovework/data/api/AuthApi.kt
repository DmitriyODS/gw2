package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.ChangeDefaultRequest
import com.kodass.groovework.data.dto.LoginRequest
import com.kodass.groovework.data.dto.SessionResponse
import com.kodass.groovework.data.dto.UserDto
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.Header
import retrofit2.http.POST
import retrofit2.http.Query

interface AuthApi {
    // Response<…> — чтобы прочитать Set-Cookie с refresh-токеном.
    @POST("api/auth/login")
    suspend fun login(@Body body: LoginRequest): Response<SessionResponse>

    @POST("api/auth/refresh")
    suspend fun refresh(@Header("Cookie") cookie: String): Response<SessionResponse>

    @POST("api/auth/change-default")
    suspend fun changeDefault(@Body body: ChangeDefaultRequest): Response<SessionResponse>

    @POST("api/auth/logout")
    suspend fun logout(): Response<Unit>

    @GET("api/users/me")
    suspend fun me(): UserDto

    @GET("api/users/directory")
    suspend fun directory(
        @Query("q") query: String? = null,
        @Query("exclude_self") excludeSelf: String? = null,
    ): List<UserDto>
}
