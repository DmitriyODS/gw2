package com.kodass.groovework.ui.units

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowDropDown
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Stop
import androidx.compose.material.icons.filled.Timer
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.UnitDto
import com.kodass.groovework.data.dto.UnitTypeDto
import com.kodass.groovework.data.units.unitStartMillis
import com.kodass.groovework.ui.common.formatDateTime
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

// Живой отсчёт «ЧЧ:ММ:СС» от старта юнита (тикает раз в секунду).
@Composable
fun rememberElapsedText(startMillis: Long): String {
    var now by remember { mutableLongStateOf(System.currentTimeMillis()) }
    LaunchedEffect(startMillis) {
        while (true) {
            now = System.currentTimeMillis()
            delay(1000)
        }
    }
    return formatElapsed(((now - startMillis) / 1000).coerceAtLeast(0))
}

fun formatElapsed(totalSec: Long): String {
    val h = totalSec / 3600
    val m = (totalSec % 3600) / 60
    val s = totalSec % 60
    return if (h > 0) "%d:%02d:%02d".format(h, m, s) else "%02d:%02d".format(m, s)
}

// Длительность завершённого юнита: «X ч Y мин» / «Y мин» (как на вебе).
private fun formatDuration(totalSec: Long): String {
    val totalMin = (totalSec / 60).coerceAtLeast(0)
    val h = totalMin / 60
    val m = totalMin % 60
    return if (h > 0) "$h ч $m мин" else "$totalMin мин"
}

// Плашка «Текущий юнит» поверх приложения (как баннер возврата к звонку).
@Composable
fun UnitBanner(container: AppContainer, modifier: Modifier = Modifier) {
    val unit by container.unitManager.activeUnit.collectAsStateWithLifecycle()
    val u = unit ?: return
    val elapsed = rememberElapsedText(unitStartMillis(u))
    Surface(
        color = MaterialTheme.colorScheme.secondaryContainer,
        contentColor = MaterialTheme.colorScheme.onSecondaryContainer,
        shape = RoundedCornerShape(24.dp),
        modifier = modifier
            .padding(top = 8.dp, start = 12.dp, end = 12.dp)
            .clickable { container.unitManager.requestShowSheet() },
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
        ) {
            Icon(Icons.Filled.Timer, contentDescription = null, modifier = Modifier.size(18.dp))
            Column(modifier = Modifier.padding(start = 8.dp).weight(1f, fill = false)) {
                Text(
                    text = "Текущий юнит · ${u.name}",
                    style = MaterialTheme.typography.labelLarge,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Text(
                text = elapsed,
                style = MaterialTheme.typography.labelLarge,
                fontWeight = FontWeight.SemiBold,
                modifier = Modifier.padding(start = 10.dp),
            )
        }
    }
}

// Модалка текущего юнита: название, тип, задача, отсчёт, «Завершить».
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun UnitSheet(
    container: AppContainer,
    onOpenTask: (Long) -> Unit,
    onDismiss: () -> Unit,
) {
    val unit by container.unitManager.activeUnit.collectAsStateWithLifecycle()
    val u = unit
    if (u == null) {
        LaunchedEffect(Unit) { onDismiss() }
        return
    }
    var taskName by remember(u.taskId) { mutableStateOf<String?>(null) }
    LaunchedEffect(u.taskId) {
        runCatching { container.tasksRepo.task(u.taskId) }.getOrNull()?.let { taskName = it.name }
    }
    val elapsed = rememberElapsedText(unitStartMillis(u))
    var stopping by remember { mutableStateOf(false) }

    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 20.dp)
                .navigationBarsPadding(),
        ) {
            Text("Текущий юнит", style = MaterialTheme.typography.labelLarge, color = MaterialTheme.colorScheme.primary)
            Text(
                text = u.name,
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.SemiBold,
                modifier = Modifier.padding(top = 4.dp),
            )
            Text(
                text = elapsed,
                style = MaterialTheme.typography.displaySmall,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.primary,
                modifier = Modifier.padding(top = 12.dp),
            )
            u.unitType?.name?.let {
                UnitMetaRow(label = "Тип", value = it)
            }
            UnitMetaRow(label = "Задача", value = taskName ?: "Задача №${u.taskId}")

            Button(
                onClick = {
                    stopping = true
                    container.unitManager.stopActiveUnit { onDismiss() }
                },
                enabled = !stopping,
                colors = ButtonDefaults.buttonColors(
                    containerColor = MaterialTheme.colorScheme.error,
                    contentColor = MaterialTheme.colorScheme.onError,
                ),
                modifier = Modifier.fillMaxWidth().padding(top = 20.dp),
            ) {
                Icon(Icons.Filled.Stop, contentDescription = null, modifier = Modifier.size(18.dp))
                Text(if (stopping) "Завершаю…" else "Завершить", modifier = Modifier.padding(start = 8.dp))
            }
            OutlinedButton(
                onClick = { onOpenTask(u.taskId); onDismiss() },
                modifier = Modifier.fillMaxWidth().padding(top = 8.dp, bottom = 12.dp),
            ) {
                Text("Открыть задачу")
            }
        }
    }
}

@Composable
private fun UnitMetaRow(label: String, value: String) {
    Row(
        modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(
            value,
            style = MaterialTheme.typography.bodyLarge,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.padding(start = 16.dp),
        )
    }
}

// Модалка запуска юнита: название + выбор типа.
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StartUnitSheet(
    container: AppContainer,
    taskId: Long,
    onDismiss: () -> Unit,
    onStarted: () -> Unit,
) {
    var types by remember { mutableStateOf<List<UnitTypeDto>>(emptyList()) }
    var name by remember { mutableStateOf("") }
    var selectedType by remember { mutableStateOf<UnitTypeDto?>(null) }
    var typeMenuOpen by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var submitting by remember { mutableStateOf(false) }
    val scope = androidx.compose.runtime.rememberCoroutineScope()

    LaunchedEffect(Unit) {
        runCatching { container.unitsRepo.unitTypes() }.getOrNull()?.let { types = it }
    }

    ModalBottomSheet(onDismissRequest = { if (!submitting) onDismiss() }) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 20.dp)
                .navigationBarsPadding(),
        ) {
            Text("Начать юнит", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.SemiBold)
            Text(
                "Зафиксируйте время работы над задачей.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 4.dp),
            )
            OutlinedTextField(
                value = name,
                onValueChange = { name = it },
                label = { Text("Название юнита") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth().padding(top = 16.dp),
            )
            Box(modifier = Modifier.padding(top = 12.dp)) {
                OutlinedTextField(
                    value = selectedType?.name ?: "",
                    onValueChange = {},
                    readOnly = true,
                    enabled = false,
                    label = { Text("Тип юнита") },
                    trailingIcon = { Icon(Icons.Filled.ArrowDropDown, contentDescription = null) },
                    colors = androidx.compose.material3.OutlinedTextFieldDefaults.colors(
                        disabledTextColor = MaterialTheme.colorScheme.onSurface,
                        disabledBorderColor = MaterialTheme.colorScheme.outline,
                        disabledLabelColor = MaterialTheme.colorScheme.onSurfaceVariant,
                        disabledTrailingIconColor = MaterialTheme.colorScheme.onSurfaceVariant,
                    ),
                    modifier = Modifier.fillMaxWidth(),
                )
                Box(
                    modifier = Modifier
                        .matchParentSize()
                        .clickable { typeMenuOpen = true },
                )
                DropdownMenu(
                    expanded = typeMenuOpen,
                    onDismissRequest = { typeMenuOpen = false },
                ) {
                    if (types.isEmpty()) {
                        DropdownMenuItem(text = { Text("Нет типов юнитов") }, onClick = { typeMenuOpen = false })
                    }
                    types.forEach { type ->
                        DropdownMenuItem(
                            text = { Text(type.name) },
                            onClick = {
                                selectedType = type
                                typeMenuOpen = false
                            },
                        )
                    }
                }
            }
            error?.let {
                Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall, modifier = Modifier.padding(top = 8.dp))
            }
            Button(
                onClick = {
                    error = null
                    if (name.isBlank()) { error = "Введите название юнита"; return@Button }
                    val type = selectedType
                    if (type == null) { error = "Выберите тип юнита"; return@Button }
                    submitting = true
                    scope.launch {
                        val result = container.unitManager.startUnit(taskId, name.trim(), type.id)
                        submitting = false
                        result.onSuccess { onStarted() }
                            .onFailure { e ->
                                error = if ((e as? com.kodass.groovework.data.network.ApiException)?.status == 409)
                                    "У вас уже есть активный юнит"
                                else (e as? com.kodass.groovework.data.network.ApiException)?.message
                                    ?: "Не удалось запустить юнит"
                            }
                    }
                },
                enabled = !submitting,
                modifier = Modifier.fillMaxWidth().padding(top = 20.dp, bottom = 12.dp),
            ) {
                Icon(Icons.Filled.PlayArrow, contentDescription = null, modifier = Modifier.size(18.dp))
                Text(if (submitting) "Запускаю…" else "Начать юнит", modifier = Modifier.padding(start = 8.dp))
            }
        }
    }
}

// Строка юнита в списке вкладки «Юниты».
@Composable
fun UnitRow(unit: UnitDto, canDelete: Boolean, onDelete: () -> Unit) {
    var expanded by remember { mutableStateOf(false) }
    val startMillis = unitStartMillis(unit)
    Surface(
        color = MaterialTheme.colorScheme.surfaceContainerLow,
        shape = RoundedCornerShape(12.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable { expanded = !expanded }
                    .padding(start = 14.dp, end = 12.dp, top = 12.dp, bottom = 12.dp),
            ) {
                Box(
                    modifier = Modifier
                        .size(8.dp)
                        .clip(CircleShape)
                        .background(
                            if (unit.isActive) MaterialTheme.colorScheme.primary
                            else MaterialTheme.colorScheme.outlineVariant
                        ),
                )
                Column(modifier = Modifier.padding(start = 12.dp).weight(1f)) {
                    Text(
                        unit.name,
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                    unit.unitType?.name?.let {
                        Text(it, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
                if (unit.isActive) {
                    Text(
                        rememberElapsedText(startMillis),
                        style = MaterialTheme.typography.labelLarge,
                        color = MaterialTheme.colorScheme.primary,
                        fontWeight = FontWeight.SemiBold,
                    )
                } else {
                    val endMillis = com.kodass.groovework.ui.common.parseIso(unit.datetimeEnd)
                        ?.toInstant()?.toEpochMilli() ?: startMillis
                    Text(
                        formatDuration((endMillis - startMillis) / 1000),
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            AnimatedVisibility(
                visible = expanded,
                enter = fadeIn() + slideInVertically(),
                exit = fadeOut() + slideOutVertically(),
            ) {
                Column(modifier = Modifier.fillMaxWidth().padding(start = 34.dp, end = 12.dp, bottom = 12.dp)) {
                    unit.user?.fio?.let {
                        Text(it, style = MaterialTheme.typography.bodySmall, fontWeight = FontWeight.SemiBold)
                    }
                    Text(
                        "Начат: ${formatDateTime(unit.datetimeStart)}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Text(
                        if (unit.datetimeEnd != null) "Окончен: ${formatDateTime(unit.datetimeEnd)}" else "В работе",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    if (canDelete) {
                        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) {
                            IconButton(onClick = onDelete) {
                                Icon(
                                    Icons.Outlined.Delete,
                                    contentDescription = "Удалить юнит",
                                    tint = MaterialTheme.colorScheme.error,
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
