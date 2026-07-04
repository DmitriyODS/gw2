package com.kodass.groovework.data.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// Раздел «Ежедневник» (diarysvc): личные заметки-задачи по дням. В отличие от
// календаря — фиксированный набор полей карточки и скоуп по владельцу.

@Serializable
data class DiaryDto(
    val id: Long,
    @SerialName("owner_id") val ownerId: Long = 0,
    val name: String = "",
    val position: Int = 0,
    // Для вкладки «Поделились» (чужой ежедневник, read-only).
    val shared: Boolean = false,
    @SerialName("owner_name") val ownerName: String? = null,
    @SerialName("owner_avatar") val ownerAvatar: String? = null,
    // Для чужого ежедневника: можно ли адресату отмечать записи выполненными
    // (у своих всегда true).
    @SerialName("can_check") val canCheck: Boolean = true,
    // Прогресс: количество активных и выполненных записей.
    @SerialName("active_count") val activeCount: Int = 0,
    @SerialName("done_count") val doneCount: Int = 0,
)

@Serializable
data class DiariesDto(
    val diaries: List<DiaryDto> = emptyList(),
)

@Serializable
data class DiaryEntryDto(
    val id: Long,
    @SerialName("diary_id") val diaryId: Long = 0,
    // День записи (YYYY-MM-DD). Время начала/конца — минуты от полуночи (null — без времени).
    @SerialName("entry_date") val entryDate: String = "",
    @SerialName("start_min") val startMin: Int? = null,
    @SerialName("end_min") val endMin: Int? = null,
    val title: String = "",
    val description: String = "",
    val done: Boolean = false,
    @SerialName("linked_task_id") val linkedTaskId: Long? = null,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class DiaryEntryListDto(
    val items: List<DiaryEntryDto> = emptyList(),
)

@Serializable
data class DiaryNameRequest(
    val name: String,
)

@Serializable
data class DiaryEntryRequest(
    @SerialName("entry_date") val entryDate: String,
    @SerialName("start_min") val startMin: Int? = null,
    @SerialName("end_min") val endMin: Int? = null,
    val title: String,
    val description: String = "",
)

@Serializable
data class DiaryDoneRequest(
    val done: Boolean,
)

@Serializable
data class DiaryLinkRequest(
    @SerialName("task_id") val taskId: Long?,
)

// Перенос записи на другой день и/или в другой СВОЙ ежедневник (пустые поля не меняются).
@Serializable
data class DiaryMoveRequest(
    @SerialName("diary_id") val diaryId: Long? = null,
    @SerialName("entry_date") val entryDate: String? = null,
)
