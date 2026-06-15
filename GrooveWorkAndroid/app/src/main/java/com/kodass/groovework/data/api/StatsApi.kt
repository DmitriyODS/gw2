package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.ProfileStatsDto
import com.kodass.groovework.data.dto.StatsCommonDto
import com.kodass.groovework.data.dto.StatsExtendedDto
import retrofit2.http.GET
import retrofit2.http.Query

interface StatsApi {
    // Личная статистика за период (YYYY-MM-DD).
    @GET("api/stats/profile")
    suspend fun profile(
        @Query("from") from: String,
        @Query("to") to: String,
    ): ProfileStatsDto

    @GET("api/stats/common")
    suspend fun common(
        @Query("from") from: String,
        @Query("to") to: String,
        @Query("company_id") companyId: Long? = null,
    ): StatsCommonDto

    @GET("api/stats/extended")
    suspend fun extended(
        @Query("from") from: String,
        @Query("to") to: String,
        @Query("company_id") companyId: Long? = null,
    ): StatsExtendedDto
}
