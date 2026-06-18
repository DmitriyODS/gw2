package com.kodass.groovework.ui.registries

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.AttachFile
import androidx.compose.material.icons.outlined.CheckBox
import androidx.compose.material.icons.outlined.Checklist
import androidx.compose.material.icons.outlined.Event
import androidx.compose.material.icons.outlined.Image
import androidx.compose.material.icons.outlined.Link
import androidx.compose.material.icons.outlined.Notes
import androidx.compose.material.icons.outlined.Tag
import androidx.compose.ui.graphics.vector.ImageVector
import com.kodass.groovework.data.dto.RegistryFieldConfig
import com.kodass.groovework.data.dto.RegistryFieldDto
import com.kodass.groovework.data.dto.RegistryFieldType
import com.kodass.groovework.data.dto.UploadedFileDto
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import java.time.Instant
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter

// Утилиты типов полей реестра — зеркало front/src/utils/registryFields.js.

fun fieldIcon(type: String): ImageVector = when (type) {
    RegistryFieldType.NUMBER -> Icons.Outlined.Tag
    RegistryFieldType.SELECT -> Icons.Outlined.Checklist
    RegistryFieldType.CHECKBOX -> Icons.Outlined.CheckBox
    RegistryFieldType.DATETIME -> Icons.Outlined.Event
    RegistryFieldType.LINK -> Icons.Outlined.Link
    RegistryFieldType.IMAGE -> Icons.Outlined.Image
    RegistryFieldType.FILE -> Icons.Outlined.AttachFile
    else -> Icons.Outlined.Notes
}

fun isSortable(type: String): Boolean =
    type in setOf(
        RegistryFieldType.TEXT,
        RegistryFieldType.NUMBER,
        RegistryFieldType.DATETIME,
        RegistryFieldType.LINK,
    )

// ── Извлечение значений из JsonElement ──

// contentOrNull уже возвращает null для JsonNull.
fun JsonElement?.asStringOrNull(): String? = (this as? JsonPrimitive)?.contentOrNull

fun JsonElement?.asBool(): Boolean = (this as? JsonPrimitive)?.booleanOrNull ?: false

fun JsonElement?.asSelectValues(): List<String> = when (this) {
    is JsonArray -> mapNotNull { (it as? JsonPrimitive)?.contentOrNull }
    is JsonPrimitive -> contentOrNull?.let { listOf(it) } ?: emptyList()
    else -> emptyList()
}

fun JsonElement?.asUploadedFile(): UploadedFileDto? {
    val obj = this as? JsonObject ?: return null
    val path = (obj["path"] as? JsonPrimitive)?.contentOrNull ?: return null
    if (path.isBlank()) return null
    return UploadedFileDto(
        path = path,
        name = (obj["name"] as? JsonPrimitive)?.contentOrNull ?: "",
        mime = (obj["mime"] as? JsonPrimitive)?.contentOrNull ?: "",
        size = (obj["size"] as? JsonPrimitive)?.contentOrNull?.toLongOrNull() ?: 0,
    )
}

// Компактное текстовое представление значения (карточка/таблица).
fun textValue(field: RegistryFieldDto, value: JsonElement?): String {
    if (value == null || value is JsonNull) return ""
    return when (field.type) {
        RegistryFieldType.CHECKBOX -> if (value.asBool()) "Да" else "Нет"
        RegistryFieldType.SELECT -> value.asSelectValues().joinToString(", ")
        RegistryFieldType.DATETIME -> formatDateTime(value.asStringOrNull(), field.config)
        RegistryFieldType.IMAGE -> value.asUploadedFile()?.name?.ifBlank { "Картинка" } ?: "Картинка"
        RegistryFieldType.FILE -> value.asUploadedFile()?.name?.ifBlank { "Файл" } ?: "Файл"
        else -> (value as? JsonPrimitive)?.contentOrNull ?: ""
    }
}

// Форматирование ISO-даты по включённым частям конфига (год / день-месяц / время).
fun formatDateTime(iso: String?, config: RegistryFieldConfig): String {
    if (iso.isNullOrBlank()) return ""
    val dt = parseInstant(iso)?.atZone(ZoneId.systemDefault()) ?: return iso
    val parts = mutableListOf<String>()
    when {
        config.monthDay && config.year -> parts.add(dt.format(DATE_FULL))
        config.monthDay -> parts.add(dt.format(DATE_DAY_MONTH))
        config.year -> parts.add(dt.year.toString())
    }
    if (config.time) parts.add(dt.format(TIME_HM))
    return parts.joinToString(" ")
}

private fun parseInstant(iso: String): Instant? = runCatching {
    Instant.parse(iso)
}.recoverCatching {
    OffsetDateTime.parse(iso).toInstant()
}.getOrNull()

private val DATE_FULL = DateTimeFormatter.ofPattern("dd.MM.yyyy")
private val DATE_DAY_MONTH = DateTimeFormatter.ofPattern("dd.MM")
private val TIME_HM = DateTimeFormatter.ofPattern("HH:mm")
