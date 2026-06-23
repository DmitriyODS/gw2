package com.kodass.groovework.ui.calendars

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.dto.CalendarDto
import com.kodass.groovework.data.dto.CalendarEntryDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.repo.CalendarsRepository
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.longField
import com.kodass.groovework.ui.registries.companyMatches
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.ZoneId
import java.time.temporal.TemporalAdjusters

enum class CalendarView { MONTH, WEEK, DAY }

// Второй уровень раздела «Календари» — выбранный календарь: записи за период
// (месяц/неделя/день), привязанные к дате/времени.
class CalendarViewModel(
    private val calendarId: Long,
    private val repo: CalendarsRepository,
    private val session: SessionManager,
    gateway: GatewayClient,
) : ViewModel() {

    var calendar by mutableStateOf<CalendarDto?>(null)
        private set
    var loadingCalendar by mutableStateOf(true)
        private set
    var calendarError by mutableStateOf<String?>(null)
        private set
    var calendarGone by mutableStateOf(false)
        private set

    var entries by mutableStateOf<List<CalendarEntryDto>>(emptyList())
        private set
    var loadingEntries by mutableStateOf(false)
        private set

    var view by mutableStateOf(CalendarView.MONTH)
        private set
    var cursor by mutableStateOf(LocalDate.now())
        private set
    var search by mutableStateOf("")
        private set

    private var searchJob: Job? = null
    private val zone: ZoneId = ZoneId.systemDefault()

    init {
        loadCalendar()
        loadEntries()
        viewModelScope.launch {
            gateway.events.collect { event ->
                when (event.event) {
                    "calendar:updated" ->
                        if (forThis(event.data.longField("id"), event.data.longField("company_id"))) loadCalendar()
                    "calendar:deleted" ->
                        if (forThis(event.data.longField("id"), event.data.longField("company_id"))) calendarGone = true
                    "entry:created", "entry:updated", "entry:deleted", "entry:bulk-deleted" ->
                        if (event.data.longField("calendar_id") == calendarId &&
                            companyMatches(session, event.data.longField("company_id"))
                        ) loadEntries(silent = true)
                }
            }
        }
    }

    private fun forThis(id: Long?, companyId: Long?): Boolean =
        id == calendarId && companyMatches(session, companyId)

    // ── Диапазон видимого периода [from, to) ──
    fun rangeStart(): LocalDate = when (view) {
        CalendarView.DAY -> cursor
        CalendarView.WEEK -> cursor.with(TemporalAdjusters.previousOrSame(DayOfWeek.MONDAY))
        CalendarView.MONTH ->
            cursor.with(TemporalAdjusters.firstDayOfMonth())
                .with(TemporalAdjusters.previousOrSame(DayOfWeek.MONDAY))
    }

    fun rangeDays(): Int = when (view) {
        CalendarView.DAY -> 1
        CalendarView.WEEK -> 7
        CalendarView.MONTH -> 42
    }

    fun gridDays(): List<LocalDate> {
        val start = rangeStart()
        return (0 until rangeDays()).map { start.plusDays(it.toLong()) }
    }

    fun entriesFor(date: LocalDate): List<CalendarEntryDto> =
        entries.filter { entryLocalDate(it) == date }

    fun loadCalendar() {
        viewModelScope.launch {
            loadingCalendar = true
            calendarError = null
            try {
                calendar = repo.calendar(calendarId)
            } catch (e: ApiException) {
                if (calendar == null) calendarError = e.message
            } finally {
                loadingCalendar = false
            }
        }
    }

    private fun loadEntries(silent: Boolean = false) {
        viewModelScope.launch {
            if (!silent) loadingEntries = true
            try {
                val start = rangeStart()
                val from = start.atStartOfDay(zone).toInstant().toString()
                val to = start.plusDays(rangeDays().toLong()).atStartOfDay(zone).toInstant().toString()
                entries = repo.entries(calendarId, from, to, search)
            } catch (_: ApiException) {
            } finally {
                loadingEntries = false
            }
        }
    }

    fun refresh() = loadEntries(silent = true)

    fun selectView(v: CalendarView) {
        if (view == v) return
        view = v
        loadEntries()
    }

    fun step(dir: Int) {
        cursor = when (view) {
            CalendarView.DAY -> cursor.plusDays(dir.toLong())
            CalendarView.WEEK -> cursor.plusWeeks(dir.toLong())
            CalendarView.MONTH -> cursor.plusMonths(dir.toLong())
        }
        loadEntries()
    }

    fun goToday() {
        cursor = LocalDate.now()
        loadEntries()
    }

    fun openDay(date: LocalDate) {
        cursor = date
        if (view != CalendarView.DAY) {
            view = CalendarView.DAY
        }
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

    // Дефолтная дата/время для новой записи на выбранном дне (09:00).
    fun defaultDateMillis(date: LocalDate): Long =
        date.atTime(9, 0).atZone(zone).toInstant().toEpochMilli()
}
