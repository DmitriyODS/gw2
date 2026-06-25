package com.kodass.groovework.ui.diary

import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Undo
import androidx.compose.material.icons.filled.AddTask
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Clear
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.Link
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TimePicker
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.material3.rememberTimePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.ui.tasks.CreateTaskSheet
import com.kodass.groovework.ui.tasks.TasksViewModel
import java.time.LocalDate
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter
import java.util.Locale

private val RU = Locale("ru")
private val DATE_FMT = DateTimeFormatter.ofPattern("EEEE, d MMMM yyyy", RU)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DiaryEntryScreen(
    container: AppContainer,
    diaryId: Long,
    entryId: Long,
    dateMillis: Long,
    onBack: () -> Unit,
) {
    val viewModel: DiaryEntryViewModel = viewModel {
        DiaryEntryViewModel(diaryId, entryId, dateMillis, container.diariesRepo)
    }
    val context = LocalContext.current
    LaunchedEffect(viewModel.message) {
        viewModel.message?.let {
            Toast.makeText(context, it, Toast.LENGTH_SHORT).show()
            viewModel.consumeMessage()
        }
    }

    var confirmDelete by remember { mutableStateOf(false) }
    var showCreateTask by remember { mutableStateOf(false) }

    val title = when {
        viewModel.isNew -> "Новая запись"
        viewModel.editing -> "Редактирование"
        else -> "Запись"
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(title) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Назад")
                    }
                },
                actions = {
                    if (!viewModel.isNew && !viewModel.readonly && !viewModel.loading) {
                        IconButton(onClick = { confirmDelete = true }) {
                            Icon(Icons.Filled.Delete, contentDescription = "Удалить")
                        }
                    }
                },
            )
        },
        bottomBar = {
            if (!viewModel.loading && viewModel.error == null) {
                Surface(tonalElevation = 3.dp) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .navigationBarsPadding()
                            .imePadding()
                            .padding(horizontal = 16.dp, vertical = 10.dp),
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        when {
                            viewModel.editing -> {
                                OutlinedButton(
                                    onClick = { if (viewModel.isNew) onBack() else viewModel.cancelEdit() },
                                    enabled = !viewModel.saving,
                                    modifier = Modifier.weight(1f),
                                ) { Text("Отмена") }
                                Button(
                                    onClick = { viewModel.save(onSuccess = { if (viewModel.isNew) onBack() }) },
                                    enabled = !viewModel.saving,
                                    modifier = Modifier.weight(1f),
                                ) {
                                    if (viewModel.saving) {
                                        CircularProgressIndicator(
                                            modifier = Modifier.size(20.dp),
                                            strokeWidth = 2.dp,
                                            color = MaterialTheme.colorScheme.onPrimary,
                                        )
                                    } else Text("Сохранить")
                                }
                            }
                            !viewModel.readonly -> {
                                OutlinedButton(
                                    onClick = { viewModel.toggleDone(onSuccess = {}) },
                                    modifier = Modifier.weight(1f),
                                ) {
                                    Icon(
                                        if (viewModel.done) Icons.AutoMirrored.Filled.Undo else Icons.Filled.Check,
                                        contentDescription = null, modifier = Modifier.size(18.dp),
                                    )
                                    Text(if (viewModel.done) "В активные" else "Выполнено", modifier = Modifier.padding(start = 6.dp))
                                }
                                Button(onClick = { viewModel.startEdit() }, modifier = Modifier.weight(1f)) {
                                    Icon(Icons.Filled.Edit, contentDescription = null, modifier = Modifier.size(18.dp))
                                    Text("Изменить", modifier = Modifier.padding(start = 6.dp))
                                }
                            }
                        }
                    }
                }
            }
        },
    ) { padding ->
        Box(modifier = Modifier.fillMaxSize().padding(padding)) {
            when {
                viewModel.loading -> Box(Modifier.fillMaxSize(), Alignment.Center) { CircularProgressIndicator() }
                viewModel.error != null -> Box(Modifier.fillMaxSize().padding(24.dp), Alignment.Center) {
                    Text(viewModel.error!!, color = MaterialTheme.colorScheme.error)
                }
                viewModel.editing -> EditForm(viewModel)
                else -> ViewBody(viewModel, onCreateTask = { showCreateTask = true })
            }
        }
    }

    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text("Удалить запись?") },
            text = { Text("Запись будет удалена безвозвратно.") },
            confirmButton = {
                TextButton(onClick = {
                    confirmDelete = false
                    viewModel.delete(onSuccess = onBack)
                }) { Text("Удалить") }
            },
            dismissButton = { TextButton(onClick = { confirmDelete = false }) { Text("Отмена") } },
        )
    }

    // Создание задачи из записи (переиспользуем лист создания задачи). После
    // создания привязываем задачу к записи. Авто-старт юнита — веб-функция.
    if (showCreateTask) {
        val tasksVm: TasksViewModel = viewModel {
            TasksViewModel(container.tasksRepo, container.gateway, container.json)
        }
        LaunchedEffect(Unit) { tasksVm.loadDepartments() }
        CreateTaskSheet(
            viewModel = tasksVm,
            presetName = viewModel.title,
            onDismiss = { showCreateTask = false },
            onCreated = { task ->
                showCreateTask = false
                viewModel.linkTask(task.id)
            },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun EditForm(viewModel: DiaryEntryViewModel) {
    Column(
        modifier = Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        OutlinedTextField(
            value = viewModel.title,
            onValueChange = { viewModel.title = it },
            label = { Text("Название") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        OutlinedTextField(
            value = viewModel.description,
            onValueChange = { viewModel.description = it },
            label = { Text("Описание") },
            minLines = 3,
            modifier = Modifier.fillMaxWidth(),
        )

        FieldLabel("Дата")
        DateField(viewModel.date, viewModel::updateDate)

        Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                FieldLabel("Начало")
                TimeField(viewModel.startMin, onChange = viewModel::setStart)
            }
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                FieldLabel("Завершение")
                TimeField(viewModel.endMin, onChange = viewModel::setEnd)
            }
        }
    }
}

@Composable
private fun ViewBody(viewModel: DiaryEntryViewModel, onCreateTask: () -> Unit) {
    Column(
        modifier = Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Icon(Icons.Filled.CalendarMonth, contentDescription = null, tint = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.size(20.dp))
            val timeStr = timeRange(viewModel.startMin, viewModel.endMin)
            Text(
                viewModel.date.format(DATE_FMT).replaceFirstChar { it.titlecase(RU) } + if (timeStr.isNotBlank()) " · $timeStr" else "",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(start = 8.dp),
            )
        }
        Text(viewModel.title, style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
        if (viewModel.description.isNotBlank()) {
            Text(viewModel.description, style = MaterialTheme.typography.bodyLarge)
        }

        if (viewModel.linkedTaskId != null) {
            Surface(
                shape = MaterialTheme.shapes.medium,
                color = MaterialTheme.colorScheme.primaryContainer,
                modifier = Modifier.padding(top = 4.dp),
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier.padding(horizontal = 14.dp, vertical = 10.dp),
                ) {
                    Icon(Icons.Filled.Link, contentDescription = null, tint = MaterialTheme.colorScheme.onPrimaryContainer, modifier = Modifier.size(18.dp))
                    Text(
                        "К записи привязана задача",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onPrimaryContainer,
                        modifier = Modifier.padding(start = 8.dp),
                    )
                }
            }
        } else if (!viewModel.readonly) {
            OutlinedButton(onClick = onCreateTask, modifier = Modifier.padding(top = 4.dp)) {
                Icon(Icons.Filled.AddTask, contentDescription = null, modifier = Modifier.size(18.dp))
                Text("Создать задачу", modifier = Modifier.padding(start = 6.dp))
            }
        }
    }
}

@Composable
private fun FieldLabel(text: String) {
    Text(
        text,
        style = MaterialTheme.typography.labelMedium,
        fontWeight = FontWeight.SemiBold,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DateField(date: LocalDate, onChange: (LocalDate) -> Unit) {
    var show by remember { mutableStateOf(false) }
    OutlinedButton(onClick = { show = true }, modifier = Modifier.fillMaxWidth()) {
        Icon(Icons.Filled.CalendarMonth, contentDescription = null, modifier = Modifier.size(18.dp))
        Text(date.format(DATE_FMT).replaceFirstChar { it.titlecase(RU) }, modifier = Modifier.padding(start = 8.dp))
    }
    if (show) {
        val state = rememberDatePickerState(
            initialSelectedDateMillis = date.atStartOfDay(ZoneOffset.UTC).toInstant().toEpochMilli(),
        )
        DatePickerDialog(
            onDismissRequest = { show = false },
            confirmButton = {
                TextButton(onClick = {
                    state.selectedDateMillis?.let {
                        onChange(java.time.Instant.ofEpochMilli(it).atZone(ZoneOffset.UTC).toLocalDate())
                    }
                    show = false
                }) { Text("Готово") }
            },
            dismissButton = { TextButton(onClick = { show = false }) { Text("Отмена") } },
        ) { DatePicker(state = state) }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun TimeField(min: Int?, onChange: (Int?) -> Unit) {
    var show by remember { mutableStateOf(false) }
    Row(verticalAlignment = Alignment.CenterVertically) {
        OutlinedButton(onClick = { show = true }, modifier = Modifier.weight(1f)) {
            Icon(Icons.Filled.Schedule, contentDescription = null, modifier = Modifier.size(18.dp))
            Text(min?.let { "%02d:%02d".format(it / 60, it % 60) } ?: "—", modifier = Modifier.padding(start = 6.dp))
        }
        if (min != null) {
            IconButton(onClick = { onChange(null) }) {
                Icon(Icons.Filled.Clear, contentDescription = "Очистить")
            }
        }
    }
    if (show) {
        val state = rememberTimePickerState(
            initialHour = min?.let { it / 60 } ?: 9,
            initialMinute = min?.let { it % 60 } ?: 0,
            is24Hour = true,
        )
        Dialog(onDismissRequest = { show = false }) {
            Box(
                modifier = Modifier
                    .clip(RoundedCornerShape(24.dp))
                    .background(MaterialTheme.colorScheme.surfaceContainerHigh)
                    .padding(20.dp),
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text("Время", style = MaterialTheme.typography.titleMedium, modifier = Modifier.padding(bottom = 12.dp))
                    TimePicker(state = state)
                    Row(modifier = Modifier.fillMaxWidth().padding(top = 8.dp), horizontalArrangement = Arrangement.End) {
                        TextButton(onClick = { show = false }) { Text("Отмена") }
                        TextButton(onClick = {
                            onChange(state.hour * 60 + state.minute)
                            show = false
                        }) { Text("Готово") }
                    }
                }
            }
        }
    }
}

private fun timeRange(start: Int?, end: Int?): String {
    fun f(m: Int) = "%02d:%02d".format(m / 60, m % 60)
    if (start == null) return ""
    return if (end == null) f(start) else "${f(start)}–${f(end)}"
}
