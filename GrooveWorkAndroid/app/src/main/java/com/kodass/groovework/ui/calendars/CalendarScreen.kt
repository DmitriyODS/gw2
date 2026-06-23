package com.kodass.groovework.ui.calendars

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.ChevronLeft
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.CalendarEntryDto
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.RefreshOnResume
import com.kodass.groovework.ui.common.SearchField
import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.util.Locale

private val RU = Locale("ru")
private val WEEKDAYS = listOf("Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CalendarScreen(
    container: AppContainer,
    calendarId: Long,
    onBack: () -> Unit,
    onOpenEntry: (calendarId: Long, entryId: Long, dateMillis: Long) -> Unit,
) {
    val viewModel: CalendarViewModel = viewModel {
        CalendarViewModel(calendarId, container.calendarsRepo, container.sessionManager, container.gateway)
    }
    RefreshOnResume { viewModel.refresh() }

    LaunchedEffect(viewModel.calendarGone) {
        if (viewModel.calendarGone) onBack()
    }

    val calendar = viewModel.calendar

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(calendar?.name ?: "Календарь", maxLines = 1, overflow = TextOverflow.Ellipsis) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Назад")
                    }
                },
            )
        },
        floatingActionButton = {
            if (calendar != null && viewModel.view == CalendarView.DAY) {
                FloatingActionButton(onClick = {
                    onOpenEntry(calendarId, 0L, viewModel.defaultDateMillis(viewModel.cursor))
                }) {
                    Icon(Icons.Filled.Add, contentDescription = "Добавить запись")
                }
            }
        },
    ) { padding ->
        Box(modifier = Modifier.fillMaxSize().padding(padding)) {
            when {
                viewModel.loadingCalendar && calendar == null -> CenteredLoading()
                viewModel.calendarError != null && calendar == null ->
                    ErrorState(viewModel.calendarError!!, onRetry = { viewModel.loadCalendar() })
                calendar != null -> Column(modifier = Modifier.fillMaxSize()) {
                    ViewSelector(viewModel.view, viewModel::selectView)
                    PeriodBar(
                        label = periodLabel(viewModel),
                        onPrev = { viewModel.step(-1) },
                        onNext = { viewModel.step(1) },
                        onToday = { viewModel.goToday() },
                    )
                    SearchField(
                        value = viewModel.search,
                        onValueChange = viewModel::updateSearch,
                        placeholder = "Поиск по записям…",
                    )
                    Box(modifier = Modifier.weight(1f)) {
                        when (viewModel.view) {
                            CalendarView.MONTH -> MonthGrid(viewModel, calendarId, onOpenEntry)
                            CalendarView.WEEK -> WeekList(viewModel, calendarId, onOpenEntry)
                            CalendarView.DAY -> DayList(viewModel, calendarId, onOpenEntry)
                        }
                        if (viewModel.loadingEntries && viewModel.entries.isEmpty()) {
                            CircularProgressIndicator(
                                modifier = Modifier.align(Alignment.TopCenter).padding(top = 12.dp).size(28.dp),
                                strokeWidth = 2.dp,
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ViewSelector(view: CalendarView, onSelect: (CalendarView) -> Unit) {
    Row(
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        FilterChip(selected = view == CalendarView.MONTH, onClick = { onSelect(CalendarView.MONTH) }, label = { Text("Месяц") })
        FilterChip(selected = view == CalendarView.WEEK, onClick = { onSelect(CalendarView.WEEK) }, label = { Text("Неделя") })
        FilterChip(selected = view == CalendarView.DAY, onClick = { onSelect(CalendarView.DAY) }, label = { Text("День") })
    }
}

@Composable
private fun PeriodBar(label: String, onPrev: () -> Unit, onNext: () -> Unit, onToday: () -> Unit) {
    Row(
        modifier = Modifier.fillMaxWidth().padding(horizontal = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        IconButton(onClick = onPrev) { Icon(Icons.Filled.ChevronLeft, contentDescription = "Назад") }
        Text(
            label,
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.SemiBold,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
            textAlign = TextAlign.Center,
        )
        IconButton(onClick = onNext) { Icon(Icons.Filled.ChevronRight, contentDescription = "Вперёд") }
        TextButton(onClick = onToday) { Text("Сегодня") }
    }
}

// ── Месяц: сетка 6×7 ──
@Composable
private fun MonthGrid(
    viewModel: CalendarViewModel,
    calendarId: Long,
    onOpenEntry: (Long, Long, Long) -> Unit,
) {
    val days = viewModel.gridDays()
    val weeks = days.chunked(7)
    val currentMonth = viewModel.cursor.monthValue
    val today = LocalDate.now()

    Column(modifier = Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(horizontal = 8.dp)) {
        Row(modifier = Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
            WEEKDAYS.forEach { wd ->
                Text(
                    wd,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.weight(1f),
                )
            }
        }
        weeks.forEach { week ->
            Row(modifier = Modifier.fillMaxWidth()) {
                week.forEach { day ->
                    MonthCell(
                        day = day,
                        inMonth = day.monthValue == currentMonth,
                        isToday = day == today,
                        entries = viewModel.entriesFor(day),
                        onClick = { viewModel.openDay(day) },
                        onEntryClick = { e -> onOpenEntry(calendarId, e.id, 0L) },
                        modifier = Modifier.weight(1f),
                    )
                }
            }
        }
    }
}

@Composable
private fun MonthCell(
    day: LocalDate,
    inMonth: Boolean,
    isToday: Boolean,
    entries: List<CalendarEntryDto>,
    onClick: () -> Unit,
    onEntryClick: (CalendarEntryDto) -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .height(84.dp)
            .padding(2.dp)
            .clickable { onClick() }
            .background(
                if (inMonth) MaterialTheme.colorScheme.surfaceContainerLow else MaterialTheme.colorScheme.surface,
                RoundedCornerShape(8.dp),
            )
            .padding(3.dp),
    ) {
        Box(
            modifier = Modifier
                .size(20.dp)
                .then(
                    if (isToday) Modifier.background(MaterialTheme.colorScheme.primary, RoundedCornerShape(50)) else Modifier,
                ),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                day.dayOfMonth.toString(),
                style = MaterialTheme.typography.labelSmall,
                color = when {
                    isToday -> MaterialTheme.colorScheme.onPrimary
                    inMonth -> MaterialTheme.colorScheme.onSurface
                    else -> MaterialTheme.colorScheme.onSurfaceVariant
                },
            )
        }
        val shown = entries.take(2)
        shown.forEach { e ->
            Text(
                hhmm(e.eventAt),
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onPrimaryContainer,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                fontSize = 9.sp,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 2.dp)
                    .background(MaterialTheme.colorScheme.primaryContainer, RoundedCornerShape(4.dp))
                    .clickable { onEntryClick(e) }
                    .padding(horizontal = 4.dp, vertical = 1.dp),
            )
        }
        if (entries.size > shown.size) {
            Text(
                "+${entries.size - shown.size}",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                fontSize = 9.sp,
                modifier = Modifier.padding(top = 1.dp, start = 4.dp),
            )
        }
    }
}

// ── Неделя: вертикальный список 7 дней ──
@Composable
private fun WeekList(
    viewModel: CalendarViewModel,
    calendarId: Long,
    onOpenEntry: (Long, Long, Long) -> Unit,
) {
    val days = viewModel.gridDays()
    val today = LocalDate.now()
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        items(days, key = { it.toString() }) { day ->
            DayCard(
                day = day,
                isToday = day == today,
                entries = viewModel.entriesFor(day),
                onAdd = { onOpenEntry(calendarId, 0L, viewModel.defaultDateMillis(day)) },
                onEntryClick = { e -> onOpenEntry(calendarId, e.id, 0L) },
                calendarName = viewModel.calendar,
            )
        }
    }
}

@Composable
private fun DayCard(
    day: LocalDate,
    isToday: Boolean,
    entries: List<CalendarEntryDto>,
    onAdd: () -> Unit,
    onEntryClick: (CalendarEntryDto) -> Unit,
    calendarName: com.kodass.groovework.data.dto.CalendarDto?,
) {
    Surface(
        shape = MaterialTheme.shapes.large,
        color = MaterialTheme.colorScheme.surfaceContainerLow,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(start = 14.dp, top = 10.dp, end = 6.dp, bottom = 10.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    day.format(DAY_HEADER).replaceFirstChar { it.titlecase(RU) },
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = if (isToday) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,
                    modifier = Modifier.weight(1f),
                )
                IconButton(onClick = onAdd) { Icon(Icons.Filled.Add, contentDescription = "Добавить") }
            }
            if (entries.isEmpty()) {
                Text(
                    "Нет записей",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 2.dp),
                )
            } else {
                entries.forEach { e ->
                    EntryRow(e, calendarName, onClick = { onEntryClick(e) })
                }
            }
        }
    }
}

// ── День: хронологический список ──
@Composable
private fun DayList(
    viewModel: CalendarViewModel,
    calendarId: Long,
    onOpenEntry: (Long, Long, Long) -> Unit,
) {
    val entries = viewModel.entriesFor(viewModel.cursor)
    if (entries.isEmpty()) {
        Box(modifier = Modifier.fillMaxSize().padding(32.dp), contentAlignment = Alignment.Center) {
            Text(
                "На этот день записей нет",
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        return
    }
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        items(entries, key = { it.id }) { e ->
            Surface(
                onClick = { onOpenEntry(calendarId, e.id, 0L) },
                shape = MaterialTheme.shapes.large,
                color = MaterialTheme.colorScheme.surfaceContainerLow,
                modifier = Modifier.fillMaxWidth(),
            ) {
                EntryRow(e, viewModel.calendar, onClick = { onOpenEntry(calendarId, e.id, 0L) }, padded = true)
            }
        }
    }
}

@Composable
private fun EntryRow(
    entry: CalendarEntryDto,
    calendar: com.kodass.groovework.data.dto.CalendarDto?,
    onClick: () -> Unit,
    padded: Boolean = false,
) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onClick() }
            .padding(
                horizontal = if (padded) 14.dp else 0.dp,
                vertical = if (padded) 12.dp else 6.dp,
            ),
    ) {
        Text(
            hhmm(entry.eventAt),
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.primary,
            modifier = Modifier.padding(end = 12.dp),
        )
        Column(modifier = Modifier.weight(1f)) {
            Text(
                entryTitle(calendar, entry),
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            val sub = entrySubtitle(calendar, entry)
            if (sub.isNotBlank()) {
                Text(
                    sub,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

// ── Подпись периода ──
private fun periodLabel(viewModel: CalendarViewModel): String = when (viewModel.view) {
    CalendarView.DAY -> viewModel.cursor.format(DAY_FULL).replaceFirstChar { it.titlecase(RU) }
    CalendarView.WEEK -> {
        val start = viewModel.rangeStart()
        val end = start.plusDays(6)
        "${start.format(DM)} – ${end.format(DM)} ${end.year}"
    }
    CalendarView.MONTH -> viewModel.cursor.format(MONTH_YEAR).replaceFirstChar { it.titlecase(RU) }
}

private val DAY_FULL = DateTimeFormatter.ofPattern("EEEE, d MMMM yyyy", RU)
private val DAY_HEADER = DateTimeFormatter.ofPattern("EEEE, d MMMM", RU)
private val MONTH_YEAR = DateTimeFormatter.ofPattern("LLLL yyyy", RU)
private val DM = DateTimeFormatter.ofPattern("d MMM", RU)
