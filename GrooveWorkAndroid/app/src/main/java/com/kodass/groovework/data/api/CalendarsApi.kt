package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.BulkDeleteRequest
import com.kodass.groovework.data.dto.CalendarDto
import com.kodass.groovework.data.dto.CalendarEntryDto
import com.kodass.groovework.data.dto.CalendarsDto
import com.kodass.groovework.data.dto.EntryDataRequest
import com.kodass.groovework.data.dto.EntryListDto
import com.kodass.groovework.data.dto.UploadedFileDto
import okhttp3.MultipartBody
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.Multipart
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.Part
import retrofit2.http.Path
import retrofit2.http.Query

// Пользовательская часть календарей (calendarsvc): просмотр структуры и работа с
// записями. Управление полями (PUT .../fields) — админская часть, в мобилку не
// выносим.
interface CalendarsApi {
    @GET("api/calendars")
    suspend fun list(): CalendarsDto

    @GET("api/calendars/{id}")
    suspend fun get(@Path("id") id: Long): CalendarDto

    // Записи за диапазон дат (ISO-8601) для просмотра по дню/неделе/месяцу.
    @GET("api/calendars/{id}/records")
    suspend fun records(
        @Path("id") id: Long,
        @Query("from") from: String? = null,
        @Query("to") to: String? = null,
        @Query("search") search: String? = null,
    ): EntryListDto

    @GET("api/calendars/{id}/records/{rid}")
    suspend fun record(@Path("id") id: Long, @Path("rid") rid: Long): CalendarEntryDto

    @POST("api/calendars/{id}/records")
    suspend fun createRecord(@Path("id") id: Long, @Body body: EntryDataRequest): CalendarEntryDto

    @PATCH("api/calendars/{id}/records/{rid}")
    suspend fun updateRecord(
        @Path("id") id: Long,
        @Path("rid") rid: Long,
        @Body body: EntryDataRequest,
    ): CalendarEntryDto

    @DELETE("api/calendars/{id}/records/{rid}")
    suspend fun deleteRecord(@Path("id") id: Long, @Path("rid") rid: Long)

    @POST("api/calendars/{id}/records/bulk-delete")
    suspend fun bulkDelete(@Path("id") id: Long, @Body body: BulkDeleteRequest)

    @Multipart
    @POST("api/calendars/uploads")
    suspend fun upload(@Part file: MultipartBody.Part): UploadedFileDto
}
