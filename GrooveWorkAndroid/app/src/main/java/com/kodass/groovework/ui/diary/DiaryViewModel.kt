package com.kodass.groovework.ui.diary

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.dto.DiaryDto
import com.kodass.groovework.data.dto.DiaryEntryDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.repo.DiariesRepository
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.longField
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.temporal.TemporalAdjusters

enum class DiaryViewMode { MONTH, WEEK, DAY }
enum class DiarySubtab { ACTIVE, ARCHIVE }

// Второй уровень раздела «Ежедневник» — выбранный ежедневник: активные записи
// по дню/неделе/месяцу (по умолчанию неделя) и архив выполненных.
class DiaryViewModel(
    private val diaryId: Long,
    private val repo: DiariesRepository,
    gateway: GatewayClient,
) : ViewModel() {

    var diary by mutableStateOf<DiaryDto?>(null)
        private set
    var loadingDiary by mutableStateOf(true)
        private set
    var diaryError by mutableStateOf<String?>(null)
        private set
    var diaryGone by mutableStateOf(false)
        private set

    val readonly: Boolean get() = diary?.shared == true

    var subtab by mutableStateOf(DiarySubtab.ACTIVE)
        private set
    var entries by mutableStateOf<List<DiaryEntryDto>>(emptyList())
        private set
    var archive by mutableStateOf<List<DiaryEntryDto>>(emptyList())
        private set
    var loadingEntries by mutableStateOf(false)
        private set

    var view by mutableStateOf(DiaryViewMode.WEEK)
        private set
    var cursor by mutableStateOf(LocalDate.now())
        private set
    var search by mutableStateOf("")
        private set
    var message by mutableStateOf<String?>(null)

    private var searchJob: Job? = null

    init {
        loadDiary()
        loadEntries()
        viewModelScope.launch {
            gateway.events.collect { event ->
                when (event.event) {
                    "diary:updated" -> if (event.data.longField("id") == diaryId) loadDiary()
                    "diary:deleted", "diary:unshared" -> if (event.data.longField("id") == diaryId) diaryGone = true
                    "diary_entry:created", "diary_entry:updated",
                    "diary_entry:deleted", "diary_entry:bulk-deleted" ->
                        if (event.data.longField("diary_id") == diaryId) loadEntries(silent = true)
                }
            }
        }
    }

    fun rangeStart(): LocalDate = when (view) {
        DiaryViewMode.DAY -> cursor
        DiaryViewMode.WEEK -> cursor.with(TemporalAdjusters.previousOrSame(DayOfWeek.MONDAY))
        DiaryViewMode.MONTH ->
            cursor.with(TemporalAdjusters.firstDayOfMonth())
                .with(TemporalAdjusters.previousOrSame(DayOfWeek.MONDAY))
    }

    fun rangeDays(): Int = when (view) {
        DiaryViewMode.DAY -> 1
        DiaryViewMode.WEEK -> 7
        DiaryViewMode.MONTH -> 42
    }

    fun gridDays(): List<LocalDate> {
        val start = rangeStart()
        return (0 until rangeDays()).map { start.plusDays(it.toLong()) }
    }

    fun entriesFor(date: LocalDate): List<DiaryEntryDto> =
        entries.filter { entryLocalDate(it) == date }

    fun loadDiary() {
        viewModelScope.launch {
            loadingDiary = true
            diaryError = null
            try {
                diary = repo.diary(diaryId)
            } catch (e: ApiException) {
                if (diary == null) diaryError = e.message
            } finally {
                loadingDiary = false
            }
        }
    }

    private fun loadEntries(silent: Boolean = false) {
        viewModelScope.launch {
            if (!silent) loadingEntries = true
            try {
                if (subtab == DiarySubtab.ARCHIVE) {
                    archive = repo.entries(diaryId, archived = true, from = null, to = null, search = search)
                } else {
                    val start = rangeStart()
                    val from = start.toString()
                    val to = start.plusDays(rangeDays().toLong()).toString()
                    entries = repo.entries(diaryId, archived = false, from = from, to = to, search = search)
                }
            } catch (_: ApiException) {
            } finally {
                loadingEntries = false
            }
        }
    }

    fun refresh() = loadEntries(silent = true)

    fun selectSubtab(t: DiarySubtab) {
        if (subtab == t) return
        subtab = t
        loadEntries()
    }

    fun selectView(v: DiaryViewMode) {
        if (view == v) return
        view = v
        loadEntries()
    }

    fun step(dir: Int) {
        cursor = when (view) {
            DiaryViewMode.DAY -> cursor.plusDays(dir.toLong())
            DiaryViewMode.WEEK -> cursor.plusWeeks(dir.toLong())
            DiaryViewMode.MONTH -> cursor.plusMonths(dir.toLong())
        }
        loadEntries()
    }

    fun goToday() {
        cursor = LocalDate.now()
        loadEntries()
    }

    fun openDay(date: LocalDate) {
        cursor = date
        if (view != DiaryViewMode.DAY) view = DiaryViewMode.DAY
        loadEntries()
    }

    fun updateSearch(value: String) {
        if (value == search) return
        search = value
        searchJob?.cancel()
        searchJob = viewModelScope.launch {
            delay(300)
            loadEntries()
        }
    }

    fun toggleDone(entry: DiaryEntryDto, done: Boolean) {
        viewModelScope.launch {
            try {
                repo.setDone(diaryId, entry.id, done)
                loadEntries(silent = true)
                message = if (done) "Перенесено в архив" else "Возвращено в активные"
            } catch (e: ApiException) {
                message = e.message ?: "Не удалось изменить статус"
            }
        }
    }

    // Дата по умолчанию для новой записи на выбранном дне (epoch millis, локальная полночь).
    fun defaultDateMillis(date: LocalDate): Long =
        date.atStartOfDay(java.time.ZoneId.systemDefault()).toInstant().toEpochMilli()

    fun consumeMessage() { message = null }
}
