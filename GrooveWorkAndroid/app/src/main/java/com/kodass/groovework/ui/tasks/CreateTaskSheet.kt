package com.kodass.groovework.ui.tasks

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.kodass.groovework.data.dto.DeptRef
import com.kodass.groovework.data.dto.TaskDto
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneOffset

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CreateTaskSheet(
    viewModel: TasksViewModel,
    onDismiss: () -> Unit,
    onCreated: (TaskDto) -> Unit,
    presetName: String = "",
) {
    var name by remember { mutableStateOf(presetName) }
    var department by remember { mutableStateOf<DeptRef?>(null) }
    var deptExpanded by remember { mutableStateOf(false) }
    var deadline by remember { mutableStateOf<LocalDate?>(null) }
    var showDatePicker by remember { mutableStateOf(false) }

    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .imePadding()
                .padding(horizontal = 24.dp),
        ) {
            Text(
                text = "Новая задача",
                style = MaterialTheme.typography.titleLarge,
                modifier = Modifier.padding(bottom = 16.dp),
            )
            OutlinedTextField(
                value = name,
                onValueChange = { name = it },
                label = { Text("Название") },
                modifier = Modifier.fillMaxWidth(),
            )
            Box(modifier = Modifier.padding(top = 12.dp)) {
                OutlinedTextField(
                    value = department?.name ?: "",
                    onValueChange = {},
                    readOnly = true,
                    enabled = false,
                    label = { Text("Отдел") },
                    colors = OutlinedTextFieldDefaults.colors(
                        disabledTextColor = MaterialTheme.colorScheme.onSurface,
                        disabledBorderColor = MaterialTheme.colorScheme.outline,
                        disabledLabelColor = MaterialTheme.colorScheme.onSurfaceVariant,
                    ),
                    modifier = Modifier.fillMaxWidth(),
                )
                // Прозрачный кликабельный слой поверх отключённого поля.
                Box(
                    modifier = Modifier
                        .matchParentSize()
                        .clickable { deptExpanded = true },
                )
                DropdownMenu(
                    expanded = deptExpanded,
                    onDismissRequest = { deptExpanded = false },
                ) {
                    viewModel.departments.forEach { dept ->
                        DropdownMenuItem(
                            text = { Text(dept.name) },
                            onClick = {
                                department = dept
                                deptExpanded = false
                            },
                        )
                    }
                }
            }
            OutlinedTextField(
                value = deadline?.toString() ?: "",
                onValueChange = {},
                readOnly = true,
                label = { Text("Дедлайн (необязательно)") },
                trailingIcon = {
                    TextButton(onClick = { showDatePicker = true }) { Text("Выбрать") }
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 12.dp),
            )
            viewModel.createError?.let { error ->
                Text(
                    text = error,
                    color = MaterialTheme.colorScheme.error,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.padding(top = 12.dp),
                )
            }
            Button(
                onClick = {
                    val dept = department ?: return@Button
                    viewModel.create(name.trim(), dept.id, deadline?.toString(), onCreated)
                },
                enabled = !viewModel.creating && name.isNotBlank() && department != null,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 20.dp, bottom = 24.dp)
                    .height(52.dp),
            ) {
                if (viewModel.creating) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(22.dp),
                        color = MaterialTheme.colorScheme.onPrimary,
                        strokeWidth = 2.dp,
                    )
                } else {
                    Text("Создать")
                }
            }
        }
    }

    if (showDatePicker) {
        val datePickerState = rememberDatePickerState()
        DatePickerDialog(
            onDismissRequest = { showDatePicker = false },
            confirmButton = {
                TextButton(onClick = {
                    datePickerState.selectedDateMillis?.let { millis ->
                        deadline = Instant.ofEpochMilli(millis).atZone(ZoneOffset.UTC).toLocalDate()
                    }
                    showDatePicker = false
                }) { Text("Готово") }
            },
            dismissButton = {
                TextButton(onClick = { showDatePicker = false }) { Text("Отмена") }
            },
        ) {
            DatePicker(state = datePickerState)
        }
    }
}
