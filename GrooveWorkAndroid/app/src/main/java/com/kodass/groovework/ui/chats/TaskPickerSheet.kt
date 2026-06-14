package com.kodass.groovework.ui.chats

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.background
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.TaskDto
import com.kodass.groovework.data.repo.TasksRepository
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.EmptyState
import com.kodass.groovework.ui.tasks.taskAccentColor
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

// Пикер задачи для прикрепления к сообщению (#8). Источник — активные задачи
// пользователя с поиском, как на вебе (MessageInput.pickTask).
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TaskPickerSheet(
    container: AppContainer,
    onDismiss: () -> Unit,
    onPick: (TaskDto) -> Unit,
) {
    val viewModel: TaskPickerViewModel = viewModel { TaskPickerViewModel(container.tasksRepo) }
    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp)) {
            Text(
                text = "Прикрепить задачу",
                style = MaterialTheme.typography.titleLarge,
                modifier = Modifier.padding(bottom = 12.dp),
            )
            OutlinedTextField(
                value = viewModel.query,
                onValueChange = { viewModel.onQuery(it) },
                label = { Text("Поиск задачи") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            when {
                viewModel.loading && viewModel.tasks.isEmpty() ->
                    Box(modifier = Modifier.fillMaxWidth().height(220.dp)) { CenteredLoading() }
                viewModel.tasks.isEmpty() ->
                    Box(modifier = Modifier.fillMaxWidth().height(220.dp)) {
                        EmptyState("Задачи не найдены")
                    }
                else -> LazyColumn(
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(max = 440.dp)
                        .navigationBarsPadding(),
                    contentPadding = androidx.compose.foundation.layout.PaddingValues(vertical = 8.dp),
                    verticalArrangement = Arrangement.spacedBy(2.dp),
                ) {
                    items(viewModel.tasks, key = { it.id }) { task ->
                        TaskPickerRow(task = task, onClick = { onPick(task); onDismiss() })
                    }
                }
            }
        }
    }
}

@Composable
private fun TaskPickerRow(task: TaskDto, onClick: () -> Unit) {
    val accent = taskAccentColor(task.color) ?: MaterialTheme.colorScheme.outline
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = 10.dp),
    ) {
        Box(
            modifier = Modifier
                .size(10.dp)
                .clip(CircleShape)
                .background(accent),
        )
        Column(modifier = Modifier.weight(1f).padding(start = 12.dp)) {
            Text(
                text = task.name.ifBlank { "Задача #${task.id}" },
                style = MaterialTheme.typography.bodyLarge,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            task.responsible?.fio?.takeIf { it.isNotBlank() }?.let { fio ->
                Text(
                    text = fio,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

class TaskPickerViewModel(private val repo: TasksRepository) : ViewModel() {
    var query by mutableStateOf("")
        private set
    var tasks by mutableStateOf<List<TaskDto>>(emptyList())
        private set
    var loading by mutableStateOf(true)
        private set

    private var searchJob: Job? = null

    init {
        load()
    }

    fun onQuery(value: String) {
        query = value
        searchJob?.cancel()
        searchJob = viewModelScope.launch {
            delay(350)
            load()
        }
    }

    private fun load() {
        viewModelScope.launch {
            loading = true
            try {
                tasks = repo.tasks(tab = "active", search = query, page = 1, perPage = 30).items
            } catch (_: Exception) {
                tasks = emptyList()
            } finally {
                loading = false
            }
        }
    }
}
