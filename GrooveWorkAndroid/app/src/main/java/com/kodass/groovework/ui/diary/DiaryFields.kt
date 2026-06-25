package com.kodass.groovework.ui.diary

import com.kodass.groovework.data.dto.DiaryEntryDto
import java.time.LocalDate

// Утилиты раздела «Ежедневник».

// Локальная дата записи (день, к которому привязана). entry_date — "YYYY-MM-DD".
fun entryLocalDate(entry: DiaryEntryDto): LocalDate? =
    runCatching { LocalDate.parse(entry.entryDate) }.getOrNull()

private fun hhmm(min: Int): String {
    val m = min.coerceIn(0, 23 * 60 + 59)
    return "%02d:%02d".format(m / 60, m % 60)
}

// Время записи: "" если не задано, "HH:mm" или "HH:mm–HH:mm".
fun entryTime(entry: DiaryEntryDto): String {
    val s = entry.startMin ?: return ""
    val end = entry.endMin ?: return hhmm(s)
    return "${hhmm(s)}–${hhmm(end)}"
}
