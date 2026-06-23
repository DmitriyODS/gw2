package com.kodass.groovework.ui.calendars

import com.kodass.groovework.data.dto.CalendarDto
import com.kodass.groovework.data.dto.CalendarEntryDto
import com.kodass.groovework.data.dto.CalendarFieldDto
import com.kodass.groovework.ui.registries.textValue
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull
import java.time.Instant
import java.time.LocalDate
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.format.DateTimeFormatter

// Утилиты раздела «Календари» — зеркало front/src/utils/calendarFields.js.

// isFieldVisible — показывать ли поле при текущих значениях. Правило условной
// видимости: поле-источник visibleFieldId должно иметь значение, равное
// visibleValue. Для checkbox-источника visibleValue == "true", для select —
// выбранный вариант. Без условия — поле видно всегда.
fun isFieldVisible(field: CalendarFieldDto, data: Map<String, JsonElement>): Boolean {
    val src = field.visibleFieldId ?: return true
    val target = field.visibleValue ?: ""
    return when (val v = data[src.toString()]) {
        is JsonArray -> v.any { (it as? JsonPrimitive)?.contentOrNull == target }
        is JsonPrimitive -> {
            val b = v.booleanOrNull
            if (b != null) b.toString() == target else (v.contentOrNull ?: "") == target
        }
        else -> "" == target
    }
}

// entryTitle — заголовок записи для плитки/списка: первое поле «показывать в
// таблице» (иначе первое вообще); пусто → запасной текст.
fun entryTitle(calendar: CalendarDto?, entry: CalendarEntryDto, fallback: String = "Запись"): String {
    val fields = calendar?.fields ?: emptyList()
    val pick = fields.firstOrNull { it.showInTable } ?: fields.firstOrNull()
    if (pick != null) {
        val v = textValue(pick.asRegistryField(), entry.data[pick.key])
        if (v.isNotBlank()) return v
    }
    return fallback
}

// entrySubtitle — второе «таблничное» поле (для режима «День»).
fun entrySubtitle(calendar: CalendarDto?, entry: CalendarEntryDto): String {
    val fields = (calendar?.fields ?: emptyList()).filter { it.showInTable }
    val f = fields.getOrNull(1) ?: return ""
    return textValue(f.asRegistryField(), entry.data[f.key])
}

fun parseEventInstant(iso: String?): Instant? {
    if (iso.isNullOrBlank()) return null
    return runCatching { Instant.parse(iso) }
        .recoverCatching { OffsetDateTime.parse(iso).toInstant() }
        .getOrNull()
}

// Локальная дата записи (для группировки по дням).
fun entryLocalDate(entry: CalendarEntryDto): LocalDate? =
    parseEventInstant(entry.eventAt)?.atZone(ZoneId.systemDefault())?.toLocalDate()

// Время записи без секунд.
fun hhmm(iso: String?): String {
    val inst = parseEventInstant(iso) ?: return ""
    return inst.atZone(ZoneId.systemDefault()).format(TIME_HM)
}

private val TIME_HM = DateTimeFormatter.ofPattern("HH:mm")
