package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.DiariesDto
import com.kodass.groovework.data.dto.DiaryDoneRequest
import com.kodass.groovework.data.dto.DiaryDto
import com.kodass.groovework.data.dto.DiaryEntryDto
import com.kodass.groovework.data.dto.DiaryEntryListDto
import com.kodass.groovework.data.dto.DiaryEntryRequest
import com.kodass.groovework.data.dto.DiaryLinkRequest
import com.kodass.groovework.data.dto.DiaryMoveRequest
import com.kodass.groovework.data.dto.DiaryNameRequest
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Query

// Личные ежедневники (diarysvc). Шаринг (ссылки/адресный доступ) и связь
// записи с задачей — пока веб-функции; в мобилку выносим просмотр/ведение
// своих ежедневников и read-only вкладку «Поделились».
interface DiariesApi {
    // tab: "mine" | "shared".
    @GET("api/diaries")
    suspend fun list(@Query("tab") tab: String): DiariesDto

    @GET("api/diaries/{id}")
    suspend fun get(@Path("id") id: Long): DiaryDto

    @POST("api/diaries")
    suspend fun create(@Body body: DiaryNameRequest): DiaryDto

    @PATCH("api/diaries/{id}")
    suspend fun rename(@Path("id") id: Long, @Body body: DiaryNameRequest): DiaryDto

    @DELETE("api/diaries/{id}")
    suspend fun delete(@Path("id") id: Long)

    // Записи: активные (archived=0) за диапазон дат либо весь архив (archived=1).
    @GET("api/diaries/{id}/records")
    suspend fun records(
        @Path("id") id: Long,
        @Query("archived") archived: Int? = null,
        @Query("from") from: String? = null,
        @Query("to") to: String? = null,
        @Query("search") search: String? = null,
    ): DiaryEntryListDto

    @GET("api/diaries/{id}/records/{rid}")
    suspend fun record(@Path("id") id: Long, @Path("rid") rid: Long): DiaryEntryDto

    @POST("api/diaries/{id}/records")
    suspend fun createRecord(@Path("id") id: Long, @Body body: DiaryEntryRequest): DiaryEntryDto

    @PATCH("api/diaries/{id}/records/{rid}")
    suspend fun updateRecord(
        @Path("id") id: Long,
        @Path("rid") rid: Long,
        @Body body: DiaryEntryRequest,
    ): DiaryEntryDto

    @PATCH("api/diaries/{id}/records/{rid}/done")
    suspend fun setDone(
        @Path("id") id: Long,
        @Path("rid") rid: Long,
        @Body body: DiaryDoneRequest,
    ): DiaryEntryDto

    // Перенос записи (владелец обоих ежедневников).
    @PATCH("api/diaries/{id}/records/{rid}/move")
    suspend fun moveRecord(
        @Path("id") id: Long,
        @Path("rid") rid: Long,
        @Body body: DiaryMoveRequest,
    ): DiaryEntryDto

    @PATCH("api/diaries/{id}/records/{rid}/link")
    suspend fun setLink(
        @Path("id") id: Long,
        @Path("rid") rid: Long,
        @Body body: DiaryLinkRequest,
    ): DiaryEntryDto

    @DELETE("api/diaries/{id}/records/{rid}")
    suspend fun deleteRecord(@Path("id") id: Long, @Path("rid") rid: Long)
}
