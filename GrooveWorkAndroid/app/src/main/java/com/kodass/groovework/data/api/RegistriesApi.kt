package com.kodass.groovework.data.api

import com.kodass.groovework.data.dto.BulkDeleteRequest
import com.kodass.groovework.data.dto.RecordDataRequest
import com.kodass.groovework.data.dto.RecordListDto
import com.kodass.groovework.data.dto.RegistriesDto
import com.kodass.groovework.data.dto.RegistryDto
import com.kodass.groovework.data.dto.RegistryRecordDto
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

// Пользовательская часть реестров (registrysvc): просмотр структуры и работа с
// записями. Управление полями (PUT .../fields) — админская часть, в мобилку не
// выносим.
interface RegistriesApi {
    @GET("api/registries")
    suspend fun list(): RegistriesDto

    @GET("api/registries/{id}")
    suspend fun get(@Path("id") id: Long): RegistryDto

    @GET("api/registries/{id}/records")
    suspend fun records(
        @Path("id") id: Long,
        @Query("search") search: String? = null,
        @Query("sort") sort: String = "",
        @Query("order") order: String = "desc",
        @Query("page") page: Int = 1,
        @Query("per_page") perPage: Int = 30,
    ): RecordListDto

    @GET("api/registries/{id}/records/{rid}")
    suspend fun record(@Path("id") id: Long, @Path("rid") rid: Long): RegistryRecordDto

    @POST("api/registries/{id}/records")
    suspend fun createRecord(@Path("id") id: Long, @Body body: RecordDataRequest): RegistryRecordDto

    @PATCH("api/registries/{id}/records/{rid}")
    suspend fun updateRecord(
        @Path("id") id: Long,
        @Path("rid") rid: Long,
        @Body body: RecordDataRequest,
    ): RegistryRecordDto

    @DELETE("api/registries/{id}/records/{rid}")
    suspend fun deleteRecord(@Path("id") id: Long, @Path("rid") rid: Long)

    @POST("api/registries/{id}/records/bulk-delete")
    suspend fun bulkDelete(@Path("id") id: Long, @Body body: BulkDeleteRequest)

    @Multipart
    @POST("api/registries/uploads")
    suspend fun upload(@Part file: MultipartBody.Part): UploadedFileDto
}
