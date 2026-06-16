package com.kodass.groovework.ui.tasks

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.scaleOut
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListState
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.outlined.StarBorder
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.PrimaryTabRow
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.TaskDto
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.EmptyState
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.RefreshOnResume
import com.kodass.groovework.ui.common.SearchField
import com.kodass.groovework.ui.common.UserAvatar
import com.kodass.groovework.ui.common.formatChatStamp
import com.kodass.groovework.ui.common.parseIso
import com.kodass.groovework.ui.common.rememberIsScrollingUp
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.launch
import java.time.LocalDate

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TasksScreen(container: AppContainer, onOpenTask: (Long) -> Unit) {
    val viewModel: TasksViewModel = viewModel {
        TasksViewModel(container.tasksRepo, container.gateway, container.json)
    }
    var showCreate by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val initialPage = remember { taskTabs.indexOfFirst { it.first == viewModel.tab }.coerceAtLeast(0) }
    val pagerState = rememberPagerState(initialPage = initialPage) { taskTabs.size }
    // Свой LazyListState на вкладку: сохраняет позицию при свайпе и питает скрытие FAB.
    val listStates = listOf(rememberLazyListState(), rememberLazyListState(), rememberLazyListState())
    val fabVisible = listStates[pagerState.currentPage].rememberIsScrollingUp()

    // Свайп пейджера → выбор вкладки (её данные подгружаются при оседании).
    LaunchedEffect(pagerState) {
        snapshotFlow { pagerState.currentPage }
            .distinctUntilChanged()
            .collect { page -> viewModel.selectTab(taskTabs[page].first) }
    }

    // Живые обновления приходят по WebSocket; при входе/возврате и смене компании —
    // разовое обновление загруженных вкладок.
    RefreshOnResume { viewModel.backgroundRefresh() }

    Scaffold(
        topBar = { TopAppBar(title = { Text("Задачи") }) },
        floatingActionButton = {
            AnimatedVisibility(
                visible = fabVisible,
                enter = scaleIn() + fadeIn(),
                exit = scaleOut() + fadeOut(),
            ) {
                FloatingActionButton(onClick = {
                    showCreate = true
                    viewModel.loadDepartments()
                }) {
                    Icon(Icons.Filled.Add, contentDescription = "Новая задача")
                }
            }
        },
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            SearchField(
                value = viewModel.search,
                onValueChange = { viewModel.setSearchValue(it) },
                placeholder = "Поиск задач",
            )
            PrimaryTabRow(selectedTabIndex = pagerState.currentPage) {
                taskTabs.forEachIndexed { index, (_, label) ->
                    Tab(
                        selected = pagerState.currentPage == index,
                        onClick = { scope.launch { pagerState.animateScrollToPage(index) } },
                        text = { Text(label) },
                    )
                }
            }
            HorizontalPager(
                state = pagerState,
                key = { taskTabs[it].first },
                modifier = Modifier.weight(1f).fillMaxWidth(),
            ) { page ->
                val key = taskTabs[page].first
                LaunchedEffect(key) { viewModel.ensureLoaded(key) }
                TaskTabContent(
                    viewModel = viewModel,
                    tabKey = key,
                    listState = listStates[page],
                    onOpenTask = onOpenTask,
                )
            }
        }
    }

    if (showCreate) {
        CreateTaskSheet(
            viewModel = viewModel,
            onDismiss = { showCreate = false },
            onCreated = { task ->
                showCreate = false
                onOpenTask(task.id)
            },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun TaskTabContent(
    viewModel: TasksViewModel,
    tabKey: String,
    listState: LazyListState,
    onOpenTask: (Long) -> Unit,
) {
    LaunchedEffect(listState, tabKey) {
        snapshotFlow {
            val last = listState.layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: 0
            last >= listState.layoutInfo.totalItemsCount - 5
        }
            .distinctUntilChanged()
            .collect { nearEnd -> if (nearEnd) viewModel.loadMore(tabKey) }
    }
    PullToRefreshBox(
        isRefreshing = viewModel.isRefreshing(tabKey),
        onRefresh = { viewModel.pullRefresh(tabKey) },
        modifier = Modifier.fillMaxSize(),
    ) {
        val taskItems = viewModel.items(tabKey)
        when {
            viewModel.isLoading(tabKey) && taskItems.isEmpty() -> CenteredLoading()
            viewModel.errorOf(tabKey) != null && taskItems.isEmpty() -> LazyColumn(modifier = Modifier.fillMaxSize()) {
                item {
                    ErrorState(
                        viewModel.errorOf(tabKey) ?: "",
                        onRetry = { viewModel.reload(tabKey) },
                        modifier = Modifier.fillParentMaxSize(),
                    )
                }
            }
            taskItems.isEmpty() -> LazyColumn(modifier = Modifier.fillMaxSize()) {
                item {
                    EmptyState(
                        title = when (tabKey) {
                            "favorites" -> "Нет избранных задач"
                            "archive" -> "Архив пуст"
                            else -> "Задач пока нет"
                        },
                        subtitle = if (viewModel.search.isNotBlank()) "Попробуйте изменить запрос" else null,
                        modifier = Modifier.fillParentMaxSize(),
                    )
                }
            }
            else -> LazyColumn(
                state = listState,
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.fillMaxSize(),
            ) {
                items(taskItems, key = { it.id }) { task ->
                    TaskCard(
                        task = task,
                        onClick = { onOpenTask(task.id) },
                        onToggleFavorite = { viewModel.toggleFavorite(task) },
                    )
                }
                if (viewModel.isLoadingMore(tabKey)) {
                    item {
                        Box(modifier = Modifier.fillMaxWidth().padding(12.dp), contentAlignment = Alignment.Center) {
                            CircularProgressIndicator(modifier = Modifier.size(24.dp), strokeWidth = 2.dp)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun TaskCard(task: TaskDto, onClick: () -> Unit, onToggleFavorite: () -> Unit) {
    val tag = taskTagColors(task.color)
    Card(
        onClick = onClick,
        colors = CardDefaults.cardColors(
            containerColor = tag?.container ?: MaterialTheme.colorScheme.surfaceContainerLow,
        ),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(start = 16.dp, end = 4.dp, top = 8.dp, bottom = 12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = task.name,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f).padding(top = 4.dp),
                )
                IconButton(onClick = onToggleFavorite) {
                    Icon(
                        imageVector = if (task.isFavorite) Icons.Filled.Star else Icons.Outlined.StarBorder,
                        contentDescription = if (task.isFavorite) "Убрать из избранного" else "В избранное",
                        tint = if (task.isFavorite) MaterialTheme.colorScheme.tertiary
                        else MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.padding(top = 6.dp, end = 12.dp),
            ) {
                task.stage?.let { stage ->
                    Surface(
                        shape = RoundedCornerShape(8.dp),
                        color = MaterialTheme.colorScheme.secondaryContainer,
                    ) {
                        Text(
                            text = stage.name,
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSecondaryContainer,
                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp),
                        )
                    }
                }
                task.department?.let { dept ->
                    Text(
                        text = dept.name,
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                if (task.hasUnits) {
                    Box(
                        modifier = Modifier
                            .size(8.dp)
                            .clip(CircleShape)
                            .background(MaterialTheme.colorScheme.primary),
                    )
                }
            }
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.fillMaxWidth().padding(top = 8.dp, end = 12.dp),
            ) {
                task.responsible?.let { responsible ->
                    UserAvatar(userId = responsible.id, name = responsible.fio, avatarPath = responsible.avatarPath, size = 24.dp)
                    Text(
                        text = responsible.fio,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.padding(start = 6.dp).weight(1f, fill = false),
                    )
                }
                androidx.compose.foundation.layout.Spacer(modifier = Modifier.weight(1f))
                task.deadline?.let { deadline ->
                    val overdue = parseIso(deadline)?.toLocalDate()?.isBefore(LocalDate.now()) == true && !task.isArchived
                    Icon(
                        Icons.Filled.Schedule,
                        contentDescription = null,
                        tint = if (overdue) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.size(14.dp),
                    )
                    Text(
                        text = formatChatStamp(deadline),
                        style = MaterialTheme.typography.labelMedium,
                        color = if (overdue) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(start = 4.dp),
                    )
                }
            }
        }
    }
}
