package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.ChangelogDto
import retrofit2.http.GET

interface MetaApi {
    @GET("api/changelog")
    suspend fun changelog(): ChangelogDto
}
