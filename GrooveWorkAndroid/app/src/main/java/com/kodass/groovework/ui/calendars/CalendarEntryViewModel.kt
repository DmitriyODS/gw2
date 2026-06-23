package com.kodass.groovework.ui.calendars

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.dto.CalendarDto
import com.kodass.groovework.data.dto.CalendarEntryDto
import com.kodass.groovework.data.dto.CalendarFieldDto
import com.kodass.groovework.data.dto.UploadedFileDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.repo.CalendarsRepository
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import java.time.Instant
import java.time.temporal.ChronoUnit

class CalendarEntryViewModel(
    private val calendarId: Long,
    private val entryId: Long,
    private val defaultDateMillis: Long,
    private val repo: CalendarsRepository,
    private val json: Json,
) : ViewModel() {

    val isNew: Boolean = entryId == 0L

    var calendar by mutableStateOf<CalendarDto?>(null)
        private set
    private var record: CalendarEntryDto? = null

    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    var editing by mutableStateOf(isNew)
        private set
    var saving by mutableStateOf(false)
        private set
    var message by mutableStateOf<String?>(null)

    // Дата/время записи (epoch millis). Обязательное встроенное поле.
    var eventAtMillis by mutableStateOf<Long?>(null)
        private set

    private val form = mutableStateMapOf<String, JsonElement>()
    val uploading = mutableStateMapOf<String, Boolean>()

    init {
        load()
    }

    fun load() {
        viewModelScope.launch {
            loading = true
            error = null
            try {
                calendar = repo.calendar(calendarId)
                if (!isNew) {
                    val rec = repo.entry(calendarId, entryId)
                    record = rec
                    fill(rec)
                } else {
                    eventAtMillis = defaultDateMillis.takeIf { it > 0 } ?: System.currentTimeMillis()
                }
            } catch (e: ApiException) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    private fun fill(rec: CalendarEntryDto) {
        form.clear()
        rec.data.forEach { (k, v) -> form[k] = v }
        eventAtMillis = parseEventInstant(rec.eventAt)?.toEpochMilli() ?: System.currentTimeMillis()
    }

    // Видимые поля при текущих значениях (условная видимость).
    fun visibleFields(): List<CalendarFieldDto> =
        (calendar?.fields ?: emptyList()).filter { isFieldVisible(it, form) }

    fun value(key: String): JsonElement? = form[key]

    fun setValue(key: String, value: JsonElement?) {
        if (value == null) form.remove(key) else form[key] = value
    }

    fun setEventAt(millis: Long) {
        eventAtMillis = millis
    }

    fun startEdit() {
        editing = true
    }

    fun cancelEdit() {
        record?.let { fill(it) }
        editing = false
    }

    fun uploadFile(key: String, fileName: String, mime: String, bytes: ByteArray) {
        viewModelScope.launch {
            uploading[key] = true
            try {
                val meta: UploadedFileDto = repo.upload(fileName, mime, bytes)
                form[key] = json.encodeToJsonElement(meta)
            } catch (e: ApiException) {
                message = e.message ?: "Не удалось загрузить файл"
            } finally {
                uploading[key] = false
            }
        }
    }

    fun save(onSuccess: () -> Unit) {
        val millis = eventAtMillis
        if (millis == null) {
            message = "Укажите дату и время записи"
            return
        }
        viewModelScope.launch {
            saving = true
            try {
                // Только значения видимых полей — скрытые условием очищаются.
                val visibleKeys = visibleFields().map { it.key }.toSet()
                val data = JsonObject(form.filterKeys { it in visibleKeys })
                val iso = Instant.ofEpochMilli(millis).truncatedTo(ChronoUnit.MINUTES).toString()
                if (isNew) {
                    repo.createEntry(calendarId, iso, data)
                    message = "Запись добавлена"
                } else {
                    val updated = repo.updateEntry(calendarId, entryId, iso, data)
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

    fun consumeMessage() {
        message = null
    }
}
