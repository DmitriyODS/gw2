package com.kodass.groovework.data.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

// Типы полей реестра — набор синхронизирован с Go-доменом
// (back-go/registry/internal/domain/models.go) и фронтом (utils/registryFields.js).
object RegistryFieldType {
    const val TEXT = "text"
    const val NUMBER = "number"
    const val SELECT = "select"
    const val CHECKBOX = "checkbox"
    const val DATETIME = "datetime"
    const val LINK = "link"
    const val IMAGE = "image"
    const val FILE = "file"
}

// Конфиг поля: объединённый набор ключей под все типы (лишние игнорируются
// благодаря ignoreUnknownKeys; дефолты datetime — все части включены).
@Serializable
data class RegistryFieldConfig(
    val multiline: Boolean = false,
    val pattern: String? = null,
    val options: List<String> = emptyList(),
    val multiple: Boolean = false,
    val year: Boolean = true,
    @SerialName("month_day") val monthDay: Boolean = true,
    val time: Boolean = true,
)

@Serializable
data class RegistryFieldDto(
    val id: Long,
    @SerialName("registry_id") val registryId: Long = 0,
    val label: String = "",
    val type: String = RegistryFieldType.TEXT,
    val config: RegistryFieldConfig = RegistryFieldConfig(),
    val position: Int = 0,
    @SerialName("col_span") val colSpan: Int = 1,
    @SerialName("row_span") val rowSpan: Int = 1,
    @SerialName("show_in_table") val showInTable: Boolean = true,
) {
    // Строковый ключ поля в Record.data (домен хранит data по строковому id).
    val key: String get() = id.toString()
}

@Serializable
data class RegistryDto(
    val id: Long,
    @SerialName("company_id") val companyId: Long? = null,
    val name: String = "",
    val position: Int = 0,
    val fields: List<RegistryFieldDto> = emptyList(),
)

@Serializable
data class RegistriesDto(
    val registries: List<RegistryDto> = emptyList(),
)

@Serializable
data class RegistryRecordDto(
    val id: Long,
    @SerialName("registry_id") val registryId: Long = 0,
    // Значения по строковому id поля; тип значения зависит от поля (строка,
    // число, bool, массив строк, либо объект UploadedFile для image/file).
    val data: JsonObject = JsonObject(emptyMap()),
    @SerialName("created_by") val createdBy: Long? = null,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class RecordListDto(
    val items: List<RegistryRecordDto> = emptyList(),
    val total: Int = 0,
    val page: Int = 1,
    @SerialName("per_page") val perPage: Int = 30,
)

// Метаданные загруженного файла/картинки — значение поля image/file в data.
@Serializable
data class UploadedFileDto(
    val path: String = "",
    val name: String = "",
    val mime: String = "",
    val size: Long = 0,
)

@Serializable
data class RecordDataRequest(val data: JsonObject)

@Serializable
data class BulkDeleteRequest(val ids: List<Long>)
