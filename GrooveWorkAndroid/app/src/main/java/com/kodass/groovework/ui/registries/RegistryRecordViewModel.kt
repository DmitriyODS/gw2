package com.kodass.groovework.ui.registries

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.dto.RegistryDto
import com.kodass.groovework.data.dto.RegistryRecordDto
import com.kodass.groovework.data.dto.UploadedFileDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.repo.RegistriesRepository
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement

class RegistryRecordViewModel(
    private val registryId: Long,
    private val recordId: Long,
    private val repo: RegistriesRepository,
    private val json: Json,
) : ViewModel() {

    val isNew: Boolean = recordId == 0L

    var registry by mutableStateOf<RegistryDto?>(null)
        private set
    private var record: RegistryRecordDto? = null

    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    var editing by mutableStateOf(isNew)
        private set
    var saving by mutableStateOf(false)
        private set
    var message by mutableStateOf<String?>(null)

    // Значения полей по строковому ключу поля; отсутствие ключа = пустое значение.
    private val form = mutableStateMapOf<String, JsonElement>()

    // Поля, для которых сейчас идёт загрузка файла/картинки.
    val uploading = mutableStateMapOf<String, Boolean>()

    init {
        load()
    }

    fun load() {
        viewModelScope.launch {
            loading = true
            error = null
            try {
                registry = repo.registry(registryId)
                if (!isNew) {
                    val rec = repo.record(registryId, recordId)
                    record = rec
                    fillForm(rec)
                }
            } catch (e: ApiException) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    private fun fillForm(rec: RegistryRecordDto) {
        form.clear()
        rec.data.forEach { (k, v) -> form[k] = v }
    }

    fun value(key: String): JsonElement? = form[key]

    fun setValue(key: String, value: JsonElement?) {
        if (value == null) form.remove(key) else form[key] = value
    }

    fun startEdit() {
        editing = true
    }

    fun cancelEdit() {
        record?.let { fillForm(it) }
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
        viewModelScope.launch {
            saving = true
            try {
                val data = JsonObject(form.toMap())
                if (isNew) {
                    repo.createRecord(registryId, data)
                    message = "Запись добавлена"
                } else {
                    val updated = repo.updateRecord(registryId, recordId, data)
                    record = updated
                    fillForm(updated)
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
