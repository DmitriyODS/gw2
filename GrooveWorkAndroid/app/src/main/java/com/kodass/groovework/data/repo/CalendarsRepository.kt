package com.kodass.groovework.data.repo

import com.kodass.groovework.data.api.CalendarsApi
import com.kodass.groovework.data.dto.BulkDeleteRequest
import com.kodass.groovework.data.dto.CalendarDto
import com.kodass.groovework.data.dto.CalendarEntryDto
import com.kodass.groovework.data.dto.EntryDataRequest
import com.kodass.groovework.data.dto.UploadedFileDto
import com.kodass.groovework.data.network.apiCall
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.MultipartBody
import okhttp3.RequestBody.Companion.toRequestBody

class CalendarsRepository(
    private val api: CalendarsApi,
    private val json: Json,
) {
    suspend fun calendars(): List<CalendarDto> = apiCall(json) { api.list() }.calendars

    suspend fun calendar(id: Long): CalendarDto = apiCall(json) { api.get(id) }

    suspend fun entries(
        calendarId: Long,
        from: String?,
        to: String?,
        search: String?,
    ): List<CalendarEntryDto> = apiCall(json) {
        api.records(
            id = calendarId,
            from = from,
            to = to,
            search = search?.takeIf { it.isNotBlank() },
        )
    }.items

    suspend fun entry(calendarId: Long, entryId: Long): CalendarEntryDto =
        apiCall(json) { api.record(calendarId, entryId) }

    suspend fun createEntry(calendarId: Long, eventAt: String, data: JsonObject): CalendarEntryDto =
        apiCall(json) { api.createRecord(calendarId, EntryDataRequest(eventAt, data)) }

    suspend fun updateEntry(calendarId: Long, entryId: Long, eventAt: String, data: JsonObject): CalendarEntryDto =
        apiCall(json) { api.updateRecord(calendarId, entryId, EntryDataRequest(eventAt, data)) }

    suspend fun deleteEntry(calendarId: Long, entryId: Long) =
        apiCall(json) { api.deleteRecord(calendarId, entryId) }

    suspend fun bulkDelete(calendarId: Long, ids: List<Long>) =
        apiCall(json) { api.bulkDelete(calendarId, BulkDeleteRequest(ids)) }

    // Загрузка файла/картинки для поля image/file. Возвращает метаданные, которые
    // кладутся как значение поля в data записи.
    suspend fun upload(fileName: String, mimeType: String, bytes: ByteArray): UploadedFileDto {
        val body = bytes.toRequestBody(mimeType.toMediaTypeOrNull())
        val part = MultipartBody.Part.createFormData("file", fileName, body)
        return apiCall(json) { api.upload(part) }
    }
}
