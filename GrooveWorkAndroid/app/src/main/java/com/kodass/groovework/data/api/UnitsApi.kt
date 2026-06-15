package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.CreateUnitRequest
import com.kodass.groovework.data.dto.UnitDto
import com.kodass.groovework.data.dto.UnitTypeDto
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path

interface UnitsApi {
    @GET("api/tasks/{id}/units")
    suspend fun taskUnits(@Path("id") taskId: Long): List<UnitDto>

    @POST("api/tasks/{id}/units")
    suspend fun createUnit(@Path("id") taskId: Long, @Body body: CreateUnitRequest): UnitDto

    // null → нет активного юнита (бэкенд отдаёт JSON null со статусом 200).
    @GET("api/units/active")
    suspend fun activeUnit(): UnitDto?

    @POST("api/units/{id}/stop")
    suspend fun stopUnit(@Path("id") unitId: Long): UnitDto

    @DELETE("api/units/{id}")
    suspend fun deleteUnit(@Path("id") unitId: Long)

    @GET("api/unit-types")
    suspend fun unitTypes(): List<UnitTypeDto>
}
