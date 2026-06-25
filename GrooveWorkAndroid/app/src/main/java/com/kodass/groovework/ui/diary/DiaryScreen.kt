package com.kodass.groovework.ui.diary

import android.widget.Toast
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
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Undo
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.ChevronLeft
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material.icons.outlined.RadioButtonUnchecked
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.PrimaryTabRow
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.DiaryEntryDto
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
fun DiaryScreen(
    container: AppContainer,
    diaryId: Long,
    onBack: () -> Unit,
    onOpenEntry: (diaryId: Long, entryId: Long, dateMillis: Long) -> Unit,
) {
    val viewModel: DiaryViewModel = viewModel {
        DiaryViewModel(diaryId, container.diariesRepo, container.gateway)
    }
    RefreshOnResume { viewModel.refresh() }

    val context = LocalContext.current
    LaunchedEffect(viewModel.message) {
        viewModel.message?.let {
            Toast.makeText(context, it, Toast.LENGTH_SHORT).show()
            viewModel.consumeMessage()
        }
    }
    LaunchedEffect(viewModel.diaryGone) { if (viewModel.diaryGone) onBack() }

    val diary = viewModel.diary

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(diary?.name ?: "Ежедневник", maxLines = 1, overflow = TextOverflow.Ellipsis) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Назад")
                    }
                },
            )
        },
        floatingActionButton = {
            if (diary != null && !viewModel.readonly &&
                viewModel.subtab == DiarySubtab.ACTIVE && viewModel.view == DiaryViewMode.DAY
            ) {
                FloatingActionButton(onClick = {
                    onOpenEntry(diaryId, 0L, viewModel.defaultDateMillis(viewModel.cursor))
                }) {
                    Icon(Icons.Filled.Add, contentDescription = "Добавить запись")
                }
            }
        },
    ) { padding ->
        Box(modifier = Modifier.fillMaxSize().padding(padding)) {
            when {
                viewModel.loadingDiary && diary == null -> CenteredLoading()
                viewModel.diaryError != null && diary == null ->
                    ErrorState(viewModel.diaryError!!, onRetry = { viewModel.loadDiary() })
                diary != null -> Column(modifier = Modifier.fillMaxSize()) {
                    PrimaryTabRow(selectedTabIndex = viewModel.subtab.ordinal) {
                        Tab(
                            selected = viewModel.subtab == DiarySubtab.ACTIVE,
                            onClick = { viewModel.selectSubtab(DiarySubtab.ACTIVE) },
                            text = { Text("Активные") },
                        )
                        Tab(
                            selected = viewModel.subtab == DiarySubtab.ARCHIVE,
                            onClick = { viewModel.selectSubtab(DiarySubtab.ARCHIVE) },
                            text = { Text("Архив") },
                        )
                    }

                    if (viewModel.subtab == DiarySubtab.ACTIVE) {
                        ViewSelector(viewModel.view, viewModel::selectView)
                        PeriodBar(
                            label = periodLabel(viewModel),
                            onPrev = { viewModel.step(-1) },
                            onNext = { viewModel.step(1) },
                            onToday = { viewModel.goToday() },
                        )
                    }
                    SearchField(
                        value = viewModel.search,
                        onValueChange = viewModel::updateSearch,
                        placeholder = "Поиск по записям…",
                    )

                    Box(
                        modifier = Modifier.weight(1f).swipeTabs(
                            onPrev = { viewModel.selectSubtab(DiarySubtab.ACTIVE) },
                            onNext = { viewModel.selectSubtab(DiarySubtab.ARCHIVE) },
                        ),
                    ) {
                        if (viewModel.subtab == DiarySubtab.ARCHIVE) {
                            ArchiveList(viewModel, diaryId, onOpenEntry)
                        } else when (viewModel.view) {
                            DiaryViewMode.MONTH -> MonthGrid(viewModel, diaryId, onOpenEntry)
                            DiaryViewMode.WEEK -> WeekList(viewModel, diaryId, onOpenEntry)
                            DiaryViewMode.DAY -> DayList(viewModel, diaryId, onOpenEntry)
                        }
                        if (viewModel.loadingEntries && viewModel.entries.isEmpty() && viewModel.archive.isEmpty()) {
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

    DiaryDayDialog(viewModel, diaryId, onOpenEntry)
}

@Composable
private fun ViewSelector(view: DiaryViewMode, onSelect: (DiaryViewMode) -> Unit) {
    Row(
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        FilterChip(selected = view == DiaryViewMode.MONTH, onClick = { onSelect(DiaryViewMode.MONTH) }, label = { Text("Месяц") })
        FilterChip(selected = view == DiaryViewMode.WEEK, onClick = { onSelect(DiaryViewMode.WEEK) }, label = { Text("Неделя") })
        FilterChip(selected = view == DiaryViewMode.DAY, onClick = { onSelect(DiaryViewMode.DAY) }, label = { Text("День") })
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

// ── Месяц ──
@Composable
private fun MonthGrid(viewModel: DiaryViewModel, diaryId: Long, onOpenEntry: (Long, Long, Long) -> Unit) {
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
                        count = viewModel.entriesFor(day).size,
                        onClick = { viewModel.openDayDialog(day) },
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
    count: Int,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .height(72.dp)
            .padding(2.dp)
            .clickable { onClick() }
            .background(
                if (inMonth) MaterialTheme.colorScheme.surfaceContainerLow else MaterialTheme.colorScheme.surface,
                RoundedCornerShape(8.dp),
            )
            .padding(4.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            modifier = Modifier
                .size(22.dp)
                .then(if (isToday) Modifier.background(MaterialTheme.colorScheme.primary, RoundedCornerShape(50)) else Modifier),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                day.dayOfMonth.toString(),
                style = MaterialTheme.typography.labelMedium,
                color = when {
                    isToday -> MaterialTheme.colorScheme.onPrimary
                    inMonth -> MaterialTheme.colorScheme.onSurface
                    else -> MaterialTheme.colorScheme.onSurfaceVariant
                },
            )
        }
        if (count > 0) {
            Box(
                modifier = Modifier
                    .padding(top = 4.dp)
                    .background(MaterialTheme.colorScheme.primaryContainer, RoundedCornerShape(50))
                    .padding(horizontal = 7.dp, vertical = 1.dp),
            ) {
                Text(
                    count.toString(),
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onPrimaryContainer,
                    fontWeight = FontWeight.Bold,
                )
            }
        }
    }
}

// ── Неделя ──
@Composable
private fun WeekList(viewModel: DiaryViewModel, diaryId: Long, onOpenEntry: (Long, Long, Long) -> Unit) {
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
                readonly = viewModel.readonly,
                onAdd = { onOpenEntry(diaryId, 0L, viewModel.defaultDateMillis(day)) },
                onEntryClick = { e -> onOpenEntry(diaryId, e.id, 0L) },
                onDone = { e -> viewModel.toggleDone(e, true) },
            )
        }
    }
}

@Composable
private fun DayCard(
    day: LocalDate,
    isToday: Boolean,
    entries: List<DiaryEntryDto>,
    readonly: Boolean,
    onAdd: () -> Unit,
    onEntryClick: (DiaryEntryDto) -> Unit,
    onDone: (DiaryEntryDto) -> Unit,
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
                if (!readonly) IconButton(onClick = onAdd) { Icon(Icons.Filled.Add, contentDescription = "Добавить") }
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
                    EntryRow(e, readonly, onClick = { onEntryClick(e) }, onDone = { onDone(e) })
                }
            }
        }
    }
}

// ── День (делится на активные и выполненные) ──
@Composable
private fun DayList(viewModel: DiaryViewModel, diaryId: Long, onOpenEntry: (Long, Long, Long) -> Unit) {
    val active = viewModel.entriesFor(viewModel.cursor)
    val done = viewModel.dayViewDone
    if (active.isEmpty() && done.isEmpty()) {
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
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        if (active.isNotEmpty()) {
            item(key = "h-active") { DayGroupLabel("Активные") }
            items(active, key = { it.id }) { e ->
                Surface(shape = MaterialTheme.shapes.large, color = MaterialTheme.colorScheme.surfaceContainerLow, modifier = Modifier.fillMaxWidth()) {
                    EntryRow(e, viewModel.readonly, onClick = { onOpenEntry(diaryId, e.id, 0L) }, onDone = { viewModel.toggleDone(e, true) }, padded = true)
                }
            }
        }
        if (done.isNotEmpty()) {
            item(key = "h-done") { DayGroupLabel("Выполнено") }
            items(done, key = { it.id }) { e ->
                Surface(shape = MaterialTheme.shapes.large, color = MaterialTheme.colorScheme.surfaceContainerLow, modifier = Modifier.fillMaxWidth()) {
                    EntryRow(e, viewModel.readonly, onClick = { onOpenEntry(diaryId, e.id, 0L) }, onDone = { viewModel.toggleDone(e, false) }, padded = true, done = true)
                }
            }
        }
    }
}

// ── Архив (сгруппирован по дням) ──
@Composable
private fun ArchiveList(viewModel: DiaryViewModel, diaryId: Long, onOpenEntry: (Long, Long, Long) -> Unit) {
    val items = viewModel.archive
    if (items.isEmpty()) {
        Box(modifier = Modifier.fillMaxSize().padding(32.dp), contentAlignment = Alignment.Center) {
            Text(
                "Архив пуст — выполненные записи появятся здесь",
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
            )
        }
        return
    }
    val groups = groupByDay(items)
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        groups.forEach { (date, dayItems) ->
            item(key = "h-$date") {
                Text(
                    date.format(DAY_FULL).replaceFirstChar { it.titlecase(RU) },
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 8.dp, bottom = 2.dp),
                )
            }
            items(dayItems, key = { it.id }) { e ->
                Surface(
                    onClick = { onOpenEntry(diaryId, e.id, 0L) },
                    shape = MaterialTheme.shapes.large,
                    color = MaterialTheme.colorScheme.surfaceContainerLow,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 12.dp),
                    ) {
                        Icon(
                            Icons.Filled.CheckCircle,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.primary,
                            modifier = Modifier.padding(end = 12.dp),
                        )
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                e.title,
                                style = MaterialTheme.typography.bodyLarge,
                                fontWeight = FontWeight.Medium,
                                textDecoration = TextDecoration.LineThrough,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                            val t = entryTime(e)
                            if (t.isNotBlank()) {
                                Text(t, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        }
                        if (!viewModel.readonly) {
                            IconButton(onClick = { viewModel.toggleDone(e, false) }) {
                                Icon(Icons.AutoMirrored.Filled.Undo, contentDescription = "Вернуть в активные")
                            }
                        }
                    }
                }
            }
        }
    }
}

// ── Модалка дня: активные + выполненные (архив этого дня) ──
@Composable
private fun DiaryDayDialog(viewModel: DiaryViewModel, diaryId: Long, onOpenEntry: (Long, Long, Long) -> Unit) {
    val date = viewModel.dayDialogDate ?: return
    val active = viewModel.entriesFor(date)
    val done = viewModel.dayDone
    AlertDialog(
        onDismissRequest = { viewModel.closeDayDialog() },
        title = { Text(date.format(DAY_HEADER).replaceFirstChar { it.titlecase(RU) }) },
        text = {
            if (active.isEmpty() && done.isEmpty()) {
                Text("На этот день записей нет", color = MaterialTheme.colorScheme.onSurfaceVariant)
            } else {
                Column(
                    modifier = Modifier.fillMaxWidth().heightIn(max = 420.dp).verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(4.dp),
                ) {
                    if (active.isNotEmpty()) {
                        DayGroupLabel("Активные")
                        active.forEach { e ->
                            DayDialogRow(e, done = false, readonly = viewModel.readonly,
                                onOpen = { onOpenEntry(diaryId, e.id, 0L) },
                                onToggle = { viewModel.toggleDone(e, true) })
                        }
                    }
                    if (done.isNotEmpty()) {
                        DayGroupLabel("Выполнено")
                        done.forEach { e ->
                            DayDialogRow(e, done = true, readonly = viewModel.readonly,
                                onOpen = { onOpenEntry(diaryId, e.id, 0L) },
                                onToggle = { viewModel.toggleDone(e, false) })
                        }
                    }
                }
            }
        },
        confirmButton = {
            if (!viewModel.readonly) {
                // Модалку дня НЕ закрываем: уходим на экран записи, а при возврате
                // диалог дня покажется снова (состояние держит VM) с обновлёнными
                // активными.
                TextButton(onClick = {
                    onOpenEntry(diaryId, 0L, viewModel.defaultDateMillis(date))
                }) { Text("Добавить") }
            }
        },
        dismissButton = { TextButton(onClick = { viewModel.closeDayDialog() }) { Text("Закрыть") } },
    )
}

@Composable
private fun DayGroupLabel(text: String) {
    Text(
        text.uppercase(RU),
        style = MaterialTheme.typography.labelSmall,
        fontWeight = FontWeight.Bold,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(top = 6.dp, bottom = 2.dp),
    )
}

@Composable
private fun DayDialogRow(
    entry: DiaryEntryDto,
    done: Boolean,
    readonly: Boolean,
    onOpen: () -> Unit,
    onToggle: () -> Unit,
) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier.fillMaxWidth().clickable { onOpen() }.padding(vertical = 4.dp),
    ) {
        if (!readonly) {
            IconButton(onClick = onToggle, modifier = Modifier.size(36.dp)) {
                Icon(
                    if (done) Icons.Filled.CheckCircle else Icons.Outlined.RadioButtonUnchecked,
                    contentDescription = if (done) "Вернуть в активные" else "Выполнено",
                    tint = if (done) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
        val time = entryTime(entry)
        if (time.isNotBlank()) {
            Text(
                time,
                style = MaterialTheme.typography.labelLarge,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier.padding(start = if (readonly) 4.dp else 0.dp, end = 10.dp),
            )
        }
        Text(
            entry.title,
            style = MaterialTheme.typography.bodyLarge,
            color = if (done) MaterialTheme.colorScheme.onSurfaceVariant else MaterialTheme.colorScheme.onSurface,
            textDecoration = if (done) TextDecoration.LineThrough else null,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
    }
}

@Composable
private fun EntryRow(
    entry: DiaryEntryDto,
    readonly: Boolean,
    onClick: () -> Unit,
    onDone: () -> Unit,
    padded: Boolean = false,
    done: Boolean = false,
) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onClick() }
            .padding(horizontal = if (padded) 14.dp else 0.dp, vertical = if (padded) 12.dp else 6.dp),
    ) {
        val time = entryTime(entry)
        Text(
            time.ifBlank { "—" },
            style = MaterialTheme.typography.titleSmall,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.primary,
            modifier = Modifier.padding(end = 12.dp),
        )
        Column(modifier = Modifier.weight(1f)) {
            Text(
                entry.title,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Medium,
                textDecoration = if (done) TextDecoration.LineThrough else null,
                color = if (done) MaterialTheme.colorScheme.onSurfaceVariant else MaterialTheme.colorScheme.onSurface,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            if (entry.description.isNotBlank()) {
                Text(
                    entry.description,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
        if (!readonly) {
            IconButton(onClick = onDone) {
                Icon(
                    if (done) Icons.Filled.CheckCircle else Icons.Outlined.RadioButtonUnchecked,
                    contentDescription = if (done) "Вернуть в активные" else "Выполнено",
                    tint = if (done) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

private fun periodLabel(viewModel: DiaryViewModel): String = when (viewModel.view) {
    DiaryViewMode.DAY -> viewModel.cursor.format(DAY_FULL).replaceFirstChar { it.titlecase(RU) }
    DiaryViewMode.WEEK -> {
        val start = viewModel.rangeStart()
        val end = start.plusDays(6)
        "${start.format(DM)} – ${end.format(DM)} ${end.year}"
    }
    DiaryViewMode.MONTH -> viewModel.cursor.format(MONTH_YEAR).replaceFirstChar { it.titlecase(RU) }
}

private val DAY_FULL = DateTimeFormatter.ofPattern("EEEE, d MMMM yyyy", RU)
private val DAY_HEADER = DateTimeFormatter.ofPattern("EEEE, d MMMM", RU)
private val MONTH_YEAR = DateTimeFormatter.ofPattern("LLLL yyyy", RU)
private val DM = DateTimeFormatter.ofPattern("d MMM", RU)
