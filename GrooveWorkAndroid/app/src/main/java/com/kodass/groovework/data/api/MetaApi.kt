package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.AppBuildDto
import com.kodass.groovework.data.dto.ChangelogDto
import retrofit2.http.GET

interface MetaApi {
    @GET("api/changelog")
    suspend fun changelog(): ChangelogDto

    // Номер сборки APK, выложенного на сервере (статика nginx /apps/).
    @GET("apps/version.json")
    suspend fun appBuild(): AppBuildDto
}
