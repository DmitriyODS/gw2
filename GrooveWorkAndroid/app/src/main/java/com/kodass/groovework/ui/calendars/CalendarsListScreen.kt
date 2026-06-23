package com.kodass.groovework.ui.calendars

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.outlined.CalendarMonth
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.CalendarDto
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.EmptyState
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.RefreshOnResume

// Уровень 1 раздела «Календари»: список календарей. По тапу открывается уровень
// 2 — выбранный календарь с записями по дням/неделям/месяцам.
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CalendarsListScreen(
    container: AppContainer,
    onOpenCalendar: (calendarId: Long) -> Unit,
) {
    val viewModel: CalendarsListViewModel = viewModel {
        CalendarsListViewModel(container.calendarsRepo, container.sessionManager, container.gateway)
    }
    RefreshOnResume { viewModel.load(initial = false) }

    Scaffold(topBar = { TopAppBar(title = { Text("Календари") }) }) { padding ->
        Box(modifier = Modifier.fillMaxSize().padding(padding)) {
            when {
                viewModel.loading && viewModel.calendars.isEmpty() -> CenteredLoading()
                viewModel.error != null && viewModel.calendars.isEmpty() ->
                    ErrorState(viewModel.error!!, onRetry = { viewModel.load(initial = true) })
                viewModel.calendars.isEmpty() ->
                    EmptyState("Календарей пока нет", "Создайте календарь в веб-версии — он появится здесь.")
                else -> PullToRefreshBox(
                    isRefreshing = false,
                    onRefresh = { viewModel.load(initial = false) },
                ) {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(16.dp),
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        items(viewModel.calendars, key = { it.id }) { calendar ->
                            CalendarRow(calendar = calendar, onClick = { onOpenCalendar(calendar.id) })
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun CalendarRow(calendar: CalendarDto, onClick: () -> Unit) {
    Surface(
        onClick = onClick,
        shape = MaterialTheme.shapes.large,
        color = MaterialTheme.colorScheme.surfaceContainerLow,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 16.dp),
        ) {
            Icon(
                Icons.Outlined.CalendarMonth,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.primary,
            )
            Text(
                calendar.name,
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f).padding(start = 14.dp),
            )
            Icon(
                Icons.AutoMirrored.Filled.KeyboardArrowRight,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(24.dp),
            )
        }
    }
}
