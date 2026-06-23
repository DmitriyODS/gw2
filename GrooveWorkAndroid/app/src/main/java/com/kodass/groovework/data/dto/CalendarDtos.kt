package com.kodass.groovework.data.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

// Типы полей карточки записи календаря совпадают с реестрами — переиспользуем
// RegistryFieldType / RegistryFieldConfig (см. RegistryDtos.kt). Встроенное
// обязательное поле «Дата и время» (event_at) живёт отдельной колонкой и в этот
// набор не входит.

@Serializable
data class CalendarFieldDto(
    val id: Long,
    @SerialName("calendar_id") val calendarId: Long = 0,
    val label: String = "",
    val type: String = RegistryFieldType.TEXT,
    val config: RegistryFieldConfig = RegistryFieldConfig(),
    val position: Int = 0,
    @SerialName("col_span") val colSpan: Int = 1,
    @SerialName("row_span") val rowSpan: Int = 1,
    @SerialName("show_in_table") val showInTable: Boolean = true,
    // Условная видимость: поле показывать, только когда значение поля
    // visibleFieldId равно visibleValue (для checkbox — "true"). null — всегда.
    @SerialName("visible_field_id") val visibleFieldId: Long? = null,
    @SerialName("visible_value") val visibleValue: String? = null,
) {
    // Строковый ключ поля в Entry.data.
    val key: String get() = id.toString()

    // Адаптер для переиспования компонентов реестров (RegistryFieldInput /
    // RegistryFieldValue / textValue), работающих с RegistryFieldDto.
    fun asRegistryField(): RegistryFieldDto = RegistryFieldDto(
        id = id,
        label = label,
        type = type,
        config = config,
        position = position,
        colSpan = colSpan,
        rowSpan = rowSpan,
        showInTable = showInTable,
    )
}

@Serializable
data class CalendarDto(
    val id: Long,
    @SerialName("company_id") val companyId: Long? = null,
    val name: String = "",
    val position: Int = 0,
    val fields: List<CalendarFieldDto> = emptyList(),
)

@Serializable
data class CalendarsDto(
    val calendars: List<CalendarDto> = emptyList(),
)

@Serializable
data class CalendarEntryDto(
    val id: Long,
    @SerialName("calendar_id") val calendarId: Long = 0,
    // Дата/время записи (ISO-8601, без секунд). По ней запись попадает в день.
    @SerialName("event_at") val eventAt: String? = null,
    val data: JsonObject = JsonObject(emptyMap()),
    @SerialName("created_by") val createdBy: Long? = null,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class EntryListDto(
    val items: List<CalendarEntryDto> = emptyList(),
)

@Serializable
data class EntryDataRequest(
    @SerialName("event_at") val eventAt: String,
    val data: JsonObject,
)
