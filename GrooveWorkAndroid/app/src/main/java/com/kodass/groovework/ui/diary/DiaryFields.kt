package com.kodass.groovework.ui.diary

import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.pointer.pointerInput
import com.kodass.groovework.data.dto.DiaryEntryDto
import java.time.LocalDate

// Горизонтальный свайп для переключения вкладок: влево — onNext, вправо — onPrev.
fun Modifier.swipeTabs(onPrev: () -> Unit, onNext: () -> Unit): Modifier = this.pointerInput(Unit) {
    var total = 0f
    val threshold = 90f
    detectHorizontalDragGestures(
        onDragStart = { total = 0f },
        onDragEnd = {
            if (total <= -threshold) onNext()
            else if (total >= threshold) onPrev()
        },
        onHorizontalDrag = { _, dragAmount -> total += dragAmount },
    )
}

// Группировка записей по дню (для архива по дням). Порядок групп сохраняет
// порядок прихода (бэкенд отдаёт архив по дате убыв.).
fun groupByDay(entries: List<DiaryEntryDto>): List<Pair<LocalDate, List<DiaryEntryDto>>> {
    val map = LinkedHashMap<LocalDate, MutableList<DiaryEntryDto>>()
    for (e in entries) {
        val d = entryLocalDate(e) ?: continue
        map.getOrPut(d) { mutableListOf() }.add(e)
    }
    return map.entries.map { it.key to it.value }
}

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
