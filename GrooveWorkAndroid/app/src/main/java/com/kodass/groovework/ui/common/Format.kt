package com.kodass.groovework.ui.common

import java.time.LocalDate
import java.time.OffsetDateTime
import java.time.ZoneId
import java.time.ZonedDateTime
import java.time.format.DateTimeFormatter
import java.util.Locale

private val RU = Locale.forLanguageTag("ru")

private val timeFmt = DateTimeFormatter.ofPattern("HH:mm")
private val dayMonthFmt = DateTimeFormatter.ofPattern("d MMM", RU)
private val shortDateFmt = DateTimeFormatter.ofPattern("dd.MM.yy")
private val fullDateFmt = DateTimeFormatter.ofPattern("d MMMM yyyy", RU)
private val dateTimeFmt = DateTimeFormatter.ofPattern("d MMM, HH:mm", RU)

fun parseIso(value: String?): ZonedDateTime? = value?.let {
    runCatching { OffsetDateTime.parse(it).atZoneSameInstant(ZoneId.systemDefault()) }.getOrNull()
}

// Время для списка чатов: сегодня — HH:mm, в этом году — «5 июн», иначе дата.
fun formatChatStamp(value: String?): String {
    val dt = parseIso(value) ?: return ""
    val today = LocalDate.now()
    return when {
        dt.toLocalDate() == today -> dt.format(timeFmt)
        dt.year == today.year -> dt.format(dayMonthFmt)
        else -> dt.format(shortDateFmt)
    }
}

fun formatTime(value: String?): String = parseIso(value)?.format(timeFmt) ?: ""

fun formatDate(value: String?): String = parseIso(value)?.format(fullDateFmt) ?: ""

fun formatDateTime(value: String?): String = parseIso(value)?.format(dateTimeFmt) ?: ""

// Заголовок дня в переписке: «Сегодня», «Вчера» или дата.
fun formatDayHeader(date: LocalDate): String {
    val today = LocalDate.now()
    return when (date) {
        today -> "Сегодня"
        today.minusDays(1) -> "Вчера"
        else -> date.format(fullDateFmt)
    }
}

fun formatFileSize(bytes: Long): String = when {
    bytes >= 1 shl 20 -> "%.1f МБ".format(bytes / 1048576.0)
    bytes >= 1 shl 10 -> "%.0f КБ".format(bytes / 1024.0)
    else -> "$bytes Б"
}

fun formatLastSeen(value: String?): String {
    val dt = parseIso(value) ?: return "был(а) давно"
    val today = LocalDate.now()
    return when {
        dt.toLocalDate() == today -> "был(а) в ${dt.format(timeFmt)}"
        dt.toLocalDate() == today.minusDays(1) -> "был(а) вчера в ${dt.format(timeFmt)}"
        else -> "был(а) ${dt.format(dayMonthFmt)}"
    }
}
