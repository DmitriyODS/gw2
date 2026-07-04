package com.kodass.groovework.ui.diary

import android.widget.Toast
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
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.outlined.Book
import androidx.compose.material.icons.outlined.FolderShared
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.PrimaryTabRow
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.DiaryDto
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.EmptyState
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.RefreshOnResume

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DiariesListScreen(
    container: AppContainer,
    onOpenDiary: (diaryId: Long) -> Unit,
) {
    val viewModel: DiariesListViewModel = viewModel {
        DiariesListViewModel(container.diariesRepo, container.gateway)
    }
    RefreshOnResume { viewModel.load(initial = false) }

    val context = LocalContext.current
    LaunchedEffect(viewModel.message) {
        viewModel.message?.let {
            Toast.makeText(context, it, Toast.LENGTH_SHORT).show()
            viewModel.consumeMessage()
        }
    }

    var showCreate by remember { mutableStateOf(false) }

    Scaffold(
        topBar = { TopAppBar(title = { Text("Ежедневник") }) },
        floatingActionButton = {
            if (viewModel.tab == DiaryTab.MINE) {
                FloatingActionButton(onClick = { showCreate = true }) {
                    Icon(Icons.Filled.Add, contentDescription = "Новый ежедневник")
                }
            }
        },
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            PrimaryTabRow(selectedTabIndex = viewModel.tab.ordinal) {
                Tab(
                    selected = viewModel.tab == DiaryTab.MINE,
                    onClick = { viewModel.selectTab(DiaryTab.MINE) },
                    text = { Text("Мои") },
                )
                Tab(
                    selected = viewModel.tab == DiaryTab.SHARED,
                    onClick = { viewModel.selectTab(DiaryTab.SHARED) },
                    text = { Text("Поделились") },
                )
            }
            Box(
                modifier = Modifier.fillMaxSize().swipeTabs(
                    onPrev = { viewModel.selectTab(DiaryTab.MINE) },
                    onNext = { viewModel.selectTab(DiaryTab.SHARED) },
                ),
            ) {
                when {
                    viewModel.loading && viewModel.diaries.isEmpty() -> CenteredLoading()
                    viewModel.error != null && viewModel.diaries.isEmpty() ->
                        ErrorState(viewModel.error!!, onRetry = { viewModel.load(initial = true) })
                    viewModel.diaries.isEmpty() -> EmptyState(
                        if (viewModel.tab == DiaryTab.MINE) "Ежедневников пока нет" else "С вами пока не делились",
                        if (viewModel.tab == DiaryTab.MINE) "Создайте первый ежедневник кнопкой «+»." else "Здесь появятся ежедневники, которыми с вами поделились.",
                    )
                    else -> PullToRefreshBox(
                        isRefreshing = false,
                        onRefresh = { viewModel.load(initial = false) },
                    ) {
                        LazyColumn(
                            modifier = Modifier.fillMaxSize(),
                            contentPadding = PaddingValues(16.dp),
                            verticalArrangement = Arrangement.spacedBy(10.dp),
                        ) {
                            items(viewModel.diaries, key = { it.id }) { diary ->
                                DiaryRow(diary, onClick = { onOpenDiary(diary.id) })
                            }
                        }
                    }
                }
            }
        }
    }

    if (showCreate) {
        var name by remember { mutableStateOf("") }
        AlertDialog(
            onDismissRequest = { showCreate = false },
            title = { Text("Новый ежедневник") },
            text = {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    singleLine = true,
                    label = { Text("Название") },
                    modifier = Modifier.fillMaxWidth(),
                )
            },
            confirmButton = {
                TextButton(
                    enabled = name.isNotBlank() && !viewModel.creating,
                    onClick = {
                        viewModel.createDiary(name) { id ->
                            showCreate = false
                            onOpenDiary(id)
                        }
                    },
                ) { Text("Создать") }
            },
            dismissButton = { TextButton(onClick = { showCreate = false }) { Text("Отмена") } },
        )
    }
}

@Composable
private fun DiaryRow(diary: DiaryDto, onClick: () -> Unit) {
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
                if (diary.shared) Icons.Outlined.FolderShared else Icons.Outlined.Book,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.primary,
            )
            Column(modifier = Modifier.weight(1f).padding(start = 14.dp)) {
                Text(
                    diary.name,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                if (diary.shared && !diary.ownerName.isNullOrBlank()) {
                    Text(
                        diary.ownerName,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                // Прогресс выполнения: показываем только когда записи вообще есть.
                val total = diary.activeCount + diary.doneCount
                if (total > 0) {
                    LinearProgressIndicator(
                        progress = { diary.doneCount.toFloat() / total },
                        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                    )
                    Text(
                        "${diary.doneCount} из $total",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 4.dp),
                    )
                }
            }
            Icon(
                Icons.AutoMirrored.Filled.KeyboardArrowRight,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(24.dp),
            )
        }
    }
}
