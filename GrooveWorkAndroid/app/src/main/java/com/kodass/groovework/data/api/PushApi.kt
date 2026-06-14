package com.kodass.groovework.data.api

import kotlinx.serialization.Serializable
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.POST

@Serializable
data class DeviceTokenRequest(
    val token: String,
    val platform: String = "android",
)

interface PushApi {
    @POST("api/push/register")
    suspend fun register(@Body body: DeviceTokenRequest): Response<Unit>

    @POST("api/push/unregister")
    suspend fun unregister(@Body body: DeviceTokenRequest): Response<Unit>
}
