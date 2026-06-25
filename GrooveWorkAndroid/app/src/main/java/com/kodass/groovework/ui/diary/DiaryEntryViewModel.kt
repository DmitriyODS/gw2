package com.kodass.groovework.ui.diary

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.dto.DiaryDto
import com.kodass.groovework.data.dto.DiaryEntryDto
import com.kodass.groovework.data.dto.DiaryEntryRequest
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.repo.DiariesRepository
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId

class DiaryEntryViewModel(
    private val diaryId: Long,
    private val entryId: Long,
    private val defaultDateMillis: Long,
    private val repo: DiariesRepository,
) : ViewModel() {

    val isNew: Boolean = entryId == 0L

    var diary by mutableStateOf<DiaryDto?>(null)
        private set
    private var record: DiaryEntryDto? = null

    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    val readonly: Boolean get() = diary?.shared == true

    var editing by mutableStateOf(isNew)
        private set
    var saving by mutableStateOf(false)
        private set
    var message by mutableStateOf<String?>(null)

    var title by mutableStateOf("")
    var description by mutableStateOf("")
    var date by mutableStateOf<LocalDate>(LocalDate.now())
        private set
    var startMin by mutableStateOf<Int?>(null)
        private set
    var endMin by mutableStateOf<Int?>(null)
        private set
    var done by mutableStateOf(false)
        private set

    init {
        load()
    }

    fun load() {
        viewModelScope.launch {
            loading = true
            error = null
            try {
                diary = repo.diary(diaryId)
                if (!isNew) {
                    val rec = repo.entry(diaryId, entryId)
                    record = rec
                    fill(rec)
                } else {
                    date = if (defaultDateMillis > 0) {
                        Instant.ofEpochMilli(defaultDateMillis).atZone(ZoneId.systemDefault()).toLocalDate()
                    } else LocalDate.now()
                }
            } catch (e: ApiException) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    private fun fill(rec: DiaryEntryDto) {
        title = rec.title
        description = rec.description
        date = runCatching { LocalDate.parse(rec.entryDate) }.getOrDefault(LocalDate.now())
        startMin = rec.startMin
        endMin = rec.endMin
        done = rec.done
    }

    fun updateDate(value: LocalDate) { date = value }
    fun setStart(min: Int?) { startMin = min }
    fun setEnd(min: Int?) { endMin = min }

    fun startEdit() { editing = true }
    fun cancelEdit() {
        record?.let { fill(it) }
        editing = false
    }

    fun save(onSuccess: () -> Unit) {
        if (title.isBlank()) { message = "Укажите название записи"; return }
        viewModelScope.launch {
            saving = true
            try {
                val body = DiaryEntryRequest(
                    entryDate = date.toString(),
                    startMin = startMin,
                    endMin = endMin,
                    title = title.trim(),
                    description = description.trim(),
                )
                if (isNew) {
                    repo.createEntry(diaryId, body)
                    message = "Запись добавлена"
                } else {
                    val updated = repo.updateEntry(diaryId, entryId, body)
                    record = updated
                    fill(updated)
                    editing = false
                    message = "Запись сохранена"
                }
                onSuccess()
            } catch (e: ApiException) {
                message = e.message ?: "Не удалось сохранить запись"
            } finally {
                saving = false
            }
        }
    }

    fun toggleDone(onSuccess: () -> Unit) {
        if (isNew) return
        viewModelScope.launch {
            try {
                repo.setDone(diaryId, entryId, !done)
                done = !done
                message = if (done) "Перенесено в архив" else "Возвращено в активные"
                onSuccess()
            } catch (e: ApiException) {
                message = e.message ?: "Не удалось изменить статус"
            }
        }
    }

    fun delete(onSuccess: () -> Unit) {
        if (isNew) return
        viewModelScope.launch {
            try {
                repo.deleteEntry(diaryId, entryId)
                message = "Запись удалена"
                onSuccess()
            } catch (e: ApiException) {
                message = e.message ?: "Не удалось удалить запись"
            }
        }
    }

    fun consumeMessage() { message = null }
}
