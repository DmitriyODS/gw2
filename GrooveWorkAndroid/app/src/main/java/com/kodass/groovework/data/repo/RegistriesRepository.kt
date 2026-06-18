package com.kodass.groovework.data.repo

import com.kodass.groovework.data.api.RegistriesApi
import com.kodass.groovework.data.dto.BulkDeleteRequest
import com.kodass.groovework.data.dto.RecordDataRequest
import com.kodass.groovework.data.dto.RecordListDto
import com.kodass.groovework.data.dto.RegistryDto
import com.kodass.groovework.data.dto.RegistryRecordDto
import com.kodass.groovework.data.dto.UploadedFileDto
import com.kodass.groovework.data.network.apiCall
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.MultipartBody
import okhttp3.RequestBody.Companion.toRequestBody

class RegistriesRepository(
    private val api: RegistriesApi,
    private val json: Json,
) {
    suspend fun registries(): List<RegistryDto> = apiCall(json) { api.list() }.registries

    suspend fun registry(id: Long): RegistryDto = apiCall(json) { api.get(id) }

    suspend fun records(
        registryId: Long,
        search: String?,
        sort: String,
        order: String,
        page: Int,
        perPage: Int = PER_PAGE,
    ): RecordListDto = apiCall(json) {
        api.records(
            id = registryId,
            search = search?.takeIf { it.isNotBlank() },
            sort = sort,
            order = order,
            page = page,
            perPage = perPage,
        )
    }

    suspend fun record(registryId: Long, recordId: Long): RegistryRecordDto =
        apiCall(json) { api.record(registryId, recordId) }

    suspend fun createRecord(registryId: Long, data: JsonObject): RegistryRecordDto =
        apiCall(json) { api.createRecord(registryId, RecordDataRequest(data)) }

    suspend fun updateRecord(registryId: Long, recordId: Long, data: JsonObject): RegistryRecordDto =
        apiCall(json) { api.updateRecord(registryId, recordId, RecordDataRequest(data)) }

    suspend fun deleteRecord(registryId: Long, recordId: Long) =
        apiCall(json) { api.deleteRecord(registryId, recordId) }

    suspend fun bulkDelete(registryId: Long, ids: List<Long>) =
        apiCall(json) { api.bulkDelete(registryId, BulkDeleteRequest(ids)) }

    // Загрузка файла/картинки для поля image/file. Возвращает метаданные, которые
    // кладутся как значение поля в data записи.
    suspend fun upload(fileName: String, mimeType: String, bytes: ByteArray): UploadedFileDto {
        val body = bytes.toRequestBody(mimeType.toMediaTypeOrNull())
        val part = MultipartBody.Part.createFormData("file", fileName, body)
        return apiCall(json) { api.upload(part) }
    }

    companion object {
        const val PER_PAGE = 30
    }
}
