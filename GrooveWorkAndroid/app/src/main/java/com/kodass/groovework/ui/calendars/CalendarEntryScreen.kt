package com.kodass.groovework.ui.calendars

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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
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
import com.kodass.groovework.data.dto.CalendarFieldDto
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.registries.RegistryFieldInput
import com.kodass.groovework.ui.registries.RegistryFieldValue
import java.time.Instant
import java.time.LocalTime
import java.time.ZoneId
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CalendarEntryScreen(
    container: AppContainer,
    calendarId: Long,
    entryId: Long,
    dateMillis: Long,
    onBack: () -> Unit,
) {
    val viewModel: CalendarEntryViewModel = viewModel {
        CalendarEntryViewModel(calendarId, entryId, dateMillis, container.calendarsRepo, container.json)
    }
    val context = LocalContext.current

    LaunchedEffect(viewModel.message) {
        viewModel.message?.let {
            Toast.makeText(context, it, Toast.LENGTH_SHORT).show()
            viewModel.consumeMessage()
        }
    }

    val calendar = viewModel.calendar
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
            )
        },
        bottomBar = {
            if (calendar != null && !viewModel.loading) {
                Surface(tonalElevation = 3.dp) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .navigationBarsPadding()
                            .imePadding()
                            .padding(horizontal = 16.dp, vertical = 10.dp),
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        if (viewModel.editing) {
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
                                } else {
                                    Text(if (viewModel.isNew) "Создать" else "Сохранить")
                                }
                            }
                        } else {
                            Button(onClick = { viewModel.startEdit() }, modifier = Modifier.fillMaxWidth()) {
                                Icon(Icons.Filled.Edit, contentDescription = null, modifier = Modifier.size(18.dp))
                                Text("Редактировать", modifier = Modifier.padding(start = 8.dp))
                            }
                        }
                    }
                }
            }
        },
    ) { padding ->
        Box(modifier = Modifier.fillMaxSize().padding(padding)) {
            when {
                viewModel.loading -> CenteredLoading()
                viewModel.error != null -> ErrorState(viewModel.error!!, onRetry = { viewModel.load() })
                calendar == null -> ErrorState("Календарь не найден")
                else -> LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(16.dp),
                    verticalArrangement = Arrangement.spacedBy(16.dp),
                ) {
                    // Встроенное обязательное поле — дата и время.
                    item(key = "event_at") {
                        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                            Text(
                                "Дата и время",
                                style = MaterialTheme.typography.labelMedium,
                                fontWeight = FontWeight.SemiBold,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                            if (viewModel.editing) {
                                EventAtField(
                                    millis = viewModel.eventAtMillis,
                                    onChange = viewModel::setEventAt,
                                )
                            } else {
                                Text(
                                    formatEventAt(viewModel.eventAtMillis),
                                    style = MaterialTheme.typography.bodyLarge,
                                )
                            }
                        }
                    }

                    items(viewModel.visibleFields(), key = { it.id }) { field ->
                        FieldBlock(container, viewModel, field)
                    }
                }
            }
        }
    }
}

@Composable
private fun FieldBlock(
    container: AppContainer,
    viewModel: CalendarEntryViewModel,
    field: CalendarFieldDto,
) {
    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(
            field.label,
            style = MaterialTheme.typography.labelMedium,
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        if (viewModel.editing) {
            RegistryFieldInput(
                field = field.asRegistryField(),
                value = viewModel.value(field.key),
                uploading = viewModel.uploading[field.key] == true,
                onChange = { viewModel.setValue(field.key, it) },
                onUpload = { name, mime, bytes -> viewModel.uploadFile(field.key, name, mime, bytes) },
            )
        } else {
            RegistryFieldValue(
                container = container,
                field = field.asRegistryField(),
                value = viewModel.value(field.key),
            )
        }
    }
}

// Встроенный редактор даты+времени (без секунд). Зеркало DateTimeInput реестров.
@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun EventAtField(millis: Long?, onChange: (Long) -> Unit) {
    val zone = ZoneId.systemDefault()
    val current = millis?.let { Instant.ofEpochMilli(it).atZone(zone) }
    var showDate by remember { mutableStateOf(false) }
    var showTime by remember { mutableStateOf(false) }
    var pickedDateMillis by remember { mutableStateOf<Long?>(null) }

    Row(verticalAlignment = Alignment.CenterVertically) {
        OutlinedButton(onClick = { showDate = true }, modifier = Modifier.fillMaxWidth()) {
            Icon(Icons.Filled.CalendarMonth, contentDescription = null, modifier = Modifier.size(18.dp))
            Text(formatEventAt(millis), modifier = Modifier.padding(start = 8.dp))
        }
    }

    if (showDate) {
        val dateState = rememberDatePickerState(initialSelectedDateMillis = millis)
        DatePickerDialog(
            onDismissRequest = { showDate = false },
            confirmButton = {
                TextButton(onClick = {
                    pickedDateMillis = dateState.selectedDateMillis
                    showDate = false
                    showTime = true
                }) { Text("Далее") }
            },
            dismissButton = { TextButton(onClick = { showDate = false }) { Text("Отмена") } },
        ) {
            DatePicker(state = dateState)
        }
    }

    if (showTime) {
        val timeState = rememberTimePickerState(
            initialHour = current?.hour ?: 9,
            initialMinute = current?.minute ?: 0,
            is24Hour = true,
        )
        Dialog(onDismissRequest = { showTime = false }) {
            Box(
                modifier = Modifier
                    .clip(RoundedCornerShape(24.dp))
                    .background(MaterialTheme.colorScheme.surfaceContainerHigh)
                    .padding(20.dp),
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text("Время", style = MaterialTheme.typography.titleMedium, modifier = Modifier.padding(bottom = 12.dp))
                    TimePicker(state = timeState)
                    Row(modifier = Modifier.fillMaxWidth().padding(top = 8.dp), horizontalArrangement = Arrangement.End) {
                        TextButton(onClick = { showTime = false }) { Text("Отмена") }
                        TextButton(onClick = {
                            showTime = false
                            // Дата выбрана в UTC-таймлайне DatePicker'а — берём её
                            // календарную часть и склеиваем с локальным временем.
                            val dateMs = pickedDateMillis ?: millis ?: System.currentTimeMillis()
                            val date = Instant.ofEpochMilli(dateMs).atZone(ZoneOffset.UTC).toLocalDate()
                            val instant = date.atTime(LocalTime.of(timeState.hour, timeState.minute))
                                .atZone(zone).toInstant()
                            onChange(instant.toEpochMilli())
                        }) { Text("Готово") }
                    }
                }
            }
        }
    }
}

private val EVENT_FMT = DateTimeFormatter.ofPattern("dd.MM.yyyy HH:mm")

private fun formatEventAt(millis: Long?): String {
    if (millis == null) return "Не выбрано"
    return Instant.ofEpochMilli(millis).atZone(ZoneId.systemDefault()).format(EVENT_FMT)
}
