package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.ActiveCallDto
import com.kodass.groovework.data.dto.CallTokenDto
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path

interface CallsApi {
    // Токен входа/возврата в живой звонок — устойчивый путь ответа на входящий,
    // не зависящий от WS-обмена call:accept/call:accepted (работает и из убитого
    // приложения, поднятого пушем).
    @POST("api/calls/{id}/token")
    suspend fun token(@Path("id") callId: Long): CallTokenDto

    @GET("api/calls/active")
    suspend fun active(): ActiveCallDto
}
