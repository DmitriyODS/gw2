package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.AppBuildDto
import com.kodass.groovework.data.dto.ChangelogDto
import retrofit2.http.GET

interface MetaApi {
    @GET("api/changelog")
    suspend fun changelog(): ChangelogDto

    // Номер сборки НОВОГО мобильного приложения (Capacitor-обёртка, канал
    // /apps/mobile/ — статика nginx). Это нативное приложение заморожено:
    // обновление ведёт на новый канал; старый /apps/version.json больше не
    // опрашиваем. Пока /apps/mobile/ не выложен, запрос падает 404 — проверка
    // молчит и всплывашка обновления не показывается.
    @GET("apps/mobile/version.json")
    suspend fun appBuild(): AppBuildDto
}
