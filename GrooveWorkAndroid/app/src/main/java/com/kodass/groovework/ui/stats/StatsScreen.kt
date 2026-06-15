package com.kodass.groovework.ui.stats

import androidx.compose.foundation.background
import androidx.compose.foundation.horizontalScroll
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
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.DateRangePicker
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.PrimaryTabRow
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberDateRangePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.lerp
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.api.StatsApi
import com.kodass.groovework.data.dto.CalendarDayDto
import com.kodass.groovework.data.dto.StatsCommonDto
import com.kodass.groovework.data.dto.StatsExtendedDto
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.RefreshOnResume
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import java.time.DayOfWeek
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter
import java.util.Locale

private val ISO = DateTimeFormatter.ISO_LOCAL_DATE
private val ruLocale = Locale.forLanguageTag("ru")

class StatsViewModel(
    private val statsApi: StatsApi,
    private val json: Json,
) : ViewModel() {
    var from by mutableStateOf(LocalDate.now().with(DayOfWeek.MONDAY))
        private set
    var to by mutableStateOf(LocalDate.now())
        private set
    var preset by mutableStateOf("week")
        private set

    var common by mutableStateOf<StatsCommonDto?>(null)
        private set
    var extended by mutableStateOf<StatsExtendedDto?>(null)
        private set
    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    init { load() }

    fun applyPreset(p: String) {
        preset = p
        val today = LocalDate.now()
        when (p) {
            "week" -> { from = today.with(DayOfWeek.MONDAY); to = today }
            "month" -> { from = today.withDayOfMonth(1); to = today }
            "year" -> { from = today.withDayOfYear(1); to = today }
        }
        load()
    }

    fun setRange(f: LocalDate, t: LocalDate) {
        preset = "custom"
        from = f
        to = t
        load()
    }

    fun load() {
        viewModelScope.launch {
            loading = true
            error = null
            try {
                val f = from.format(ISO)
                val t = to.format(ISO)
                common = apiCall(json) { statsApi.common(f, t) }
                extended = apiCall(json) { statsApi.extended(f, t) }
            } catch (e: com.kodass.groovework.data.network.ApiException) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }
}

private val statsTabs = listOf("Общая", "Расширенная")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StatsScreen(container: AppContainer) {
    val viewModel: StatsViewModel = viewModel { StatsViewModel(container.statsApi, container.json) }
    RefreshOnResume { viewModel.load() }
    var tab by remember { mutableStateOf(0) }
    var showRange by remember { mutableStateOf(false) }

    Scaffold(topBar = { TopAppBar(title = { Text("Статистика") }) }) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            PeriodControl(
                preset = viewModel.preset,
                from = viewModel.from,
                to = viewModel.to,
                onPreset = { viewModel.applyPreset(it) },
                onCustom = { showRange = true },
            )
            PrimaryTabRow(selectedTabIndex = tab) {
                statsTabs.forEachIndexed { index, label ->
                    Tab(selected = tab == index, onClick = { tab = index }, text = { Text(label) })
                }
            }
            Box(modifier = Modifier.fillMaxSize()) {
                when {
                    viewModel.loading && viewModel.common == null -> CenteredLoading()
                    viewModel.error != null && viewModel.common == null ->
                        ErrorState(viewModel.error ?: "", onRetry = { viewModel.load() })
                    tab == 0 -> CommonTab(viewModel.common)
                    else -> ExtendedTab(viewModel.extended)
                }
            }
        }
    }

    if (showRange) {
        DateRangeDialog(
            initialFrom = viewModel.from,
            initialTo = viewModel.to,
            onDismiss = { showRange = false },
            onConfirm = { f, t ->
                showRange = false
                viewModel.setRange(f, t)
            },
        )
    }
}

@Composable
private fun PeriodControl(
    preset: String,
    from: LocalDate,
    to: LocalDate,
    onPreset: (String) -> Unit,
    onCustom: () -> Unit,
) {
    val rangeFmt = DateTimeFormatter.ofPattern("d MMM", ruLocale)
    Column(modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            FilterChip(selected = preset == "week", onClick = { onPreset("week") }, label = { Text("Неделя") })
            FilterChip(selected = preset == "month", onClick = { onPreset("month") }, label = { Text("Месяц") })
            FilterChip(selected = preset == "year", onClick = { onPreset("year") }, label = { Text("Год") })
            AssistChip(onClick = onCustom, label = { Text("Период…") })
        }
        Text(
            text = "${from.format(rangeFmt)} — ${to.format(rangeFmt)}",
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(top = 6.dp),
        )
    }
}

@Composable
private fun CommonTab(data: StatsCommonDto?) {
    data ?: return
    LazyColumn(
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
        modifier = Modifier.fillMaxSize(),
    ) {
        item {
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                MetricCard("Получено", data.tasks.received, MaterialTheme.colorScheme.primaryContainer, Modifier.weight(1f))
                MetricCard("Закрыто", data.tasks.closed, MaterialTheme.colorScheme.secondaryContainer, Modifier.weight(1f))
            }
        }
        item {
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                MetricCard("Осталось", data.tasks.remaining, MaterialTheme.colorScheme.tertiaryContainer, Modifier.weight(1f))
                MetricCard("Долг", data.tasks.debt, MaterialTheme.colorScheme.errorContainer, Modifier.weight(1f))
            }
        }
        if (data.tasksByEmployees.isNotEmpty()) {
            item { SectionTitle("По сотрудникам") }
            items(data.tasksByEmployees.size) { i ->
                val e = data.tasksByEmployees[i]
                ListRow(title = e.fio, value = "${formatHours(e.totalHours)} ч", subtitle = "${e.tasksCount} задач")
            }
        }
        if (data.tasksByHours.isNotEmpty()) {
            item { SectionTitle("По задачам (часы)") }
            items(data.tasksByHours.size) { i ->
                val t = data.tasksByHours[i]
                ListRow(title = t.name, value = "${formatHours(t.totalHours)} ч", subtitle = null)
            }
        }
    }
}

@Composable
private fun ExtendedTab(data: StatsExtendedDto?) {
    data ?: return
    LazyColumn(
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
        modifier = Modifier.fillMaxSize(),
    ) {
        if (data.byUnitTypes.isNotEmpty()) {
            item { SectionTitle("По типам юнитов") }
            val maxHours = data.byUnitTypes.maxOf { it.totalHours }.coerceAtLeast(0.001)
            items(data.byUnitTypes.size) { i ->
                val t = data.byUnitTypes[i]
                Column(modifier = Modifier.fillMaxWidth()) {
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        Text(t.name, style = MaterialTheme.typography.bodyMedium)
                        Text("${formatHours(t.totalHours)} ч", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    Box(
                        modifier = Modifier.fillMaxWidth().height(8.dp).padding(top = 4.dp)
                            .clip(RoundedCornerShape(4.dp))
                            .background(MaterialTheme.colorScheme.surfaceContainerHighest),
                    ) {
                        Box(
                            modifier = Modifier
                                .fillMaxWidth((t.totalHours / maxHours).toFloat().coerceIn(0.04f, 1f))
                                .height(8.dp)
                                .clip(RoundedCornerShape(4.dp))
                                .background(MaterialTheme.colorScheme.primary),
                        )
                    }
                }
            }
        }
        if (data.byDepartments.isNotEmpty()) {
            item { SectionTitle("По отделам") }
            items(data.byDepartments.size) { i ->
                val d = data.byDepartments[i]
                ListRow(title = d.name, value = "${d.tasksCount}", subtitle = null)
            }
        }
        if (data.calendar.isNotEmpty()) {
            item { SectionTitle("Календарь активности") }
            item { CalendarHeatmap(data.calendar) }
        }
    }
}

@Composable
private fun MetricCard(label: String, value: Int, color: androidx.compose.ui.graphics.Color, modifier: Modifier = Modifier) {
    Card(colors = CardDefaults.cardColors(containerColor = color), modifier = modifier) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text(value.toString(), style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Bold)
            Text(label, style = MaterialTheme.typography.bodyMedium)
        }
    }
}

@Composable
private fun SectionTitle(title: String) {
    Text(title, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold, modifier = Modifier.padding(top = 4.dp))
}

@Composable
private fun ListRow(title: String, value: String, subtitle: String?) {
    Row(
        modifier = Modifier.fillMaxWidth().padding(vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.bodyLarge, maxLines = 1, overflow = TextOverflow.Ellipsis)
            subtitle?.let { Text(it, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
        }
        Text(value, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, modifier = Modifier.padding(start = 12.dp))
    }
}

// Heatmap по неделям: строки — недели (Пн→Вс), цвет ячейки — интенсивность часов.
@Composable
private fun CalendarHeatmap(days: List<CalendarDayDto>) {
    val byDate = days.associate { it.date to it }
    val parsed = days.mapNotNull { runCatching { LocalDate.parse(it.date) }.getOrNull() }
    if (parsed.isEmpty()) return
    val minDate = parsed.min()
    val maxDate = parsed.max()
    val maxHours = days.maxOf { it.totalHours }.coerceAtLeast(0.001)
    val start = minDate.with(DayOfWeek.MONDAY)
    val base = MaterialTheme.colorScheme.surfaceContainerHighest
    val accent = MaterialTheme.colorScheme.primary

    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
        var weekStart = start
        while (!weekStart.isAfter(maxDate)) {
            Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                for (i in 0..6) {
                    val date = weekStart.plusDays(i.toLong())
                    val inRange = !date.isBefore(minDate) && !date.isAfter(maxDate)
                    val hours = byDate[date.format(ISO)]?.totalHours ?: 0.0
                    val fraction = (hours / maxHours).toFloat().coerceIn(0f, 1f)
                    val cell = if (!inRange) androidx.compose.ui.graphics.Color.Transparent
                    else lerp(base, accent, fraction)
                    Box(
                        modifier = Modifier
                            .size(28.dp)
                            .clip(RoundedCornerShape(6.dp))
                            .background(cell),
                        contentAlignment = Alignment.Center,
                    ) {
                        if (inRange) {
                            Text(
                                date.dayOfMonth.toString(),
                                style = MaterialTheme.typography.labelSmall,
                                color = if (fraction > 0.5f) MaterialTheme.colorScheme.onPrimary
                                else MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
            }
            weekStart = weekStart.plusWeeks(1)
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DateRangeDialog(
    initialFrom: LocalDate,
    initialTo: LocalDate,
    onDismiss: () -> Unit,
    onConfirm: (LocalDate, LocalDate) -> Unit,
) {
    val state = rememberDateRangePickerState(
        initialSelectedStartDateMillis = initialFrom.toEpochMillisUtc(),
        initialSelectedEndDateMillis = initialTo.toEpochMillisUtc(),
    )
    DatePickerDialog(
        onDismissRequest = onDismiss,
        confirmButton = {
            TextButton(onClick = {
                val s = state.selectedStartDateMillis
                val e = state.selectedEndDateMillis
                if (s != null && e != null) onConfirm(millisToLocalDate(s), millisToLocalDate(e))
                else onDismiss()
            }) { Text("Готово") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Отмена") } },
    ) {
        DateRangePicker(state = state, modifier = Modifier.height(480.dp))
    }
}

private fun LocalDate.toEpochMillisUtc(): Long =
    this.atStartOfDay(ZoneOffset.UTC).toInstant().toEpochMilli()

private fun millisToLocalDate(millis: Long): LocalDate =
    Instant.ofEpochMilli(millis).atZone(ZoneOffset.UTC).toLocalDate()

private fun formatHours(hours: Double): String =
    if (hours == hours.toLong().toDouble()) hours.toLong().toString()
    else String.format(ruLocale, "%.1f", hours)
