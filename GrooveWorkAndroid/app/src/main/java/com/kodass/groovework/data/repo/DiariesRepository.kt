package com.kodass.groovework.data.repo

import com.kodass.groovework.data.api.DiariesApi
import com.kodass.groovework.data.dto.DiaryDoneRequest
import com.kodass.groovework.data.dto.DiaryDto
import com.kodass.groovework.data.dto.DiaryEntryDto
import com.kodass.groovework.data.dto.DiaryEntryRequest
import com.kodass.groovework.data.dto.DiaryNameRequest
import com.kodass.groovework.data.network.apiCall
import kotlinx.serialization.json.Json

class DiariesRepository(
    private val api: DiariesApi,
    private val json: Json,
) {
    suspend fun diaries(tab: String): List<DiaryDto> = apiCall(json) { api.list(tab) }.diaries

    suspend fun diary(id: Long): DiaryDto = apiCall(json) { api.get(id) }

    suspend fun create(name: String): DiaryDto = apiCall(json) { api.create(DiaryNameRequest(name)) }

    suspend fun rename(id: Long, name: String): DiaryDto =
        apiCall(json) { api.rename(id, DiaryNameRequest(name)) }

    suspend fun deleteDiary(id: Long) = apiCall(json) { api.delete(id) }

    suspend fun entries(
        diaryId: Long,
        archived: Boolean,
        from: String?,
        to: String?,
        search: String?,
    ): List<DiaryEntryDto> = apiCall(json) {
        api.records(
            id = diaryId,
            archived = if (archived) 1 else 0,
            from = from,
            to = to,
            search = search?.takeIf { it.isNotBlank() },
        )
    }.items

    suspend fun entry(diaryId: Long, entryId: Long): DiaryEntryDto =
        apiCall(json) { api.record(diaryId, entryId) }

    suspend fun createEntry(diaryId: Long, body: DiaryEntryRequest): DiaryEntryDto =
        apiCall(json) { api.createRecord(diaryId, body) }

    suspend fun updateEntry(diaryId: Long, entryId: Long, body: DiaryEntryRequest): DiaryEntryDto =
        apiCall(json) { api.updateRecord(diaryId, entryId, body) }

    suspend fun setDone(diaryId: Long, entryId: Long, done: Boolean): DiaryEntryDto =
        apiCall(json) { api.setDone(diaryId, entryId, DiaryDoneRequest(done)) }

    suspend fun deleteEntry(diaryId: Long, entryId: Long) =
        apiCall(json) { api.deleteRecord(diaryId, entryId) }
}
