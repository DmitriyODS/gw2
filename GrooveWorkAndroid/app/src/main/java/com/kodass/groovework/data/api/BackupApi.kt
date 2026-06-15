package com.kodass.groovework.data.api

import okhttp3.MultipartBody
import retrofit2.http.Multipart
import retrofit2.http.POST
import retrofit2.http.Part

// Резервная копия (authsvc). Экспорт ZIP качаем через Downloader (нужен прогресс
// и сохранение в файл); здесь — только ДЕСТРУКТИВНЫЙ импорт.
interface BackupApi {
    @Multipart
    @POST("api/backup/import")
    suspend fun import(@Part file: MultipartBody.Part)
}
