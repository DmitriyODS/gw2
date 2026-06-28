package com.kodass.groovework.ui.tasks

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.scaleIn
import androidx.compose.animation.scaleOut
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.ExperimentalFoundationApi
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
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.Block
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.filled.Unarchive
import androidx.compose.material.icons.outlined.StarBorder
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledIconButton
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.PrimaryTabRow
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.CommentDto
import com.kodass.groovework.data.dto.UnitDto
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.ConfirmDialog
import com.kodass.groovework.ui.common.ConfirmSpec
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.LinkifiedText
import com.kodass.groovework.ui.common.UserAvatar
import com.kodass.groovework.ui.common.formatDate
import com.kodass.groovework.ui.common.formatDateTime
import com.kodass.groovework.ui.units.EditUnitSheet
import com.kodass.groovework.ui.units.StartUnitSheet
import com.kodass.groovework.ui.units.UnitRow
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.ZoneOffset

private val detailTabs = listOf("Задача", "Юниты", "Комментарии")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TaskDetailScreen(container: AppContainer, taskId: Long, onBack: () -> Unit) {
    val viewModel: TaskDetailViewModel = viewModel(key = "task-$taskId") {
        TaskDetailViewModel(
            container.tasksRepo,
            container.unitsRepo,
            container.authApi,
            container.sessionManager,
            container.gateway,
            container.json,
            taskId,
        )
    }
    val task = viewModel.task
    val activeUnit by container.unitManager.activeUnit.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()
    val pagerState = rememberPagerState { detailTabs.size }

    var menuOpen by remember { mutableStateOf(false) }
    var showRename by remember { mutableStateOf(false) }
    var showStagePicker by remember { mutableStateOf(false) }
    var showResponsiblePicker by remember { mutableStateOf(false) }
    var showDeadlinePicker by remember { mutableStateOf(false) }
    var showStartUnit by remember { mutableStateOf(false) }
    var showComplete by remember { mutableStateOf(false) }

    // FAB «Начать юнит»: задача открыта, не в архиве, активного юнита нет,
    // и мы не на вкладке комментариев (там своё поле ввода).
    val showStartFab = task != null && !task.isArchived && activeUnit == null && pagerState.currentPage != 2

    Scaffold(
        topBar = {
            TopAppBar(
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Назад")
                    }
                },
                title = { Text(if (task?.isArchived == true) "Задача · Архив" else "Задача") },
                actions = {
                    if (task != null) {
                        IconButton(onClick = { viewModel.toggleFavorite() }) {
                            Icon(
                                imageVector = if (task.isFavorite) Icons.Filled.Star else Icons.Outlined.StarBorder,
                                contentDescription = "Избранное",
                                tint = if (task.isFavorite) MaterialTheme.colorScheme.tertiary
                                else MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                        // Завершение неархивной задачи — кнопкой во вкладке «Задача»
                        // (раздел ниже). В меню остаётся только восстановление из архива.
                        if (task.isArchived) {
                            IconButton(onClick = { menuOpen = true }) {
                                Icon(Icons.Filled.MoreVert, contentDescription = "Меню")
                            }
                            DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                                DropdownMenuItem(
                                    text = { Text("Восстановить") },
                                    leadingIcon = { Icon(Icons.Filled.Unarchive, contentDescription = null) },
                                    onClick = {
                                        menuOpen = false
                                        viewModel.restore()
                                    },
                                )
                            }
                        }
                    }
                },
            )
        },
        floatingActionButton = {
            AnimatedVisibility(
                visible = showStartFab,
                enter = scaleIn() + fadeIn(),
                exit = scaleOut() + fadeOut(),
            ) {
                ExtendedFloatingActionButton(
                    onClick = { showStartUnit = true },
                    icon = { Icon(Icons.Filled.PlayArrow, contentDescription = null) },
                    text = { Text("Начать юнит") },
                )
            }
        },
    ) { padding ->
        Box(modifier = Modifier.fillMaxSize().padding(padding)) {
            when {
                viewModel.loading -> CenteredLoading()
                viewModel.error != null || task == null ->
                    ErrorState(viewModel.error ?: "Задача не найдена", onRetry = { viewModel.load() })
                else -> Column(modifier = Modifier.fillMaxSize()) {
                    PrimaryTabRow(selectedTabIndex = pagerState.currentPage) {
                        detailTabs.forEachIndexed { index, label ->
                            Tab(
                                selected = pagerState.currentPage == index,
                                onClick = { scope.launch { pagerState.animateScrollToPage(index) } },
                                text = { Text(label) },
                            )
                        }
                    }
                    HorizontalPager(
                        state = pagerState,
                        modifier = Modifier.weight(1f).fillMaxWidth(),
                    ) { page ->
                        when (page) {
                            0 -> TaskInfoTab(
                                viewModel = viewModel,
                                onRename = { showRename = true },
                                onStage = { showStagePicker = true },
                                onResponsible = {
                                    viewModel.loadDirectory()
                                    showResponsiblePicker = true
                                },
                                onDeadline = { showDeadlinePicker = true },
                                onComplete = { showComplete = true },
                            )
                            1 -> UnitsTab(viewModel = viewModel, container = container)
                            else -> CommentsTab(viewModel = viewModel)
                        }
                    }
                }
            }
        }
    }

    if (showStartUnit && task != null) {
        StartUnitSheet(
            container = container,
            taskId = taskId,
            onDismiss = { showStartUnit = false },
            onStarted = {
                showStartUnit = false
                viewModel.loadUnits()
                scope.launch { pagerState.animateScrollToPage(1) }
            },
        )
    }

    if (showComplete && task != null) {
        ConfirmDialog(
            ConfirmSpec(
                title = "Завершить задачу",
                text = "Задача отправится в архив и будет закрыта. Продолжить?",
                confirmLabel = "Завершить",
                destructive = true,
                action = { viewModel.complete(onBack) },
            ),
            onDismiss = { showComplete = false },
        )
    }

    if (showRename && task != null) {
        RenameDialog(
            initial = task.name,
            onDismiss = { showRename = false },
            onConfirm = { name ->
                showRename = false
                if (name.isNotBlank() && name != task.name) viewModel.rename(name)
            },
        )
    }

    if (showStagePicker) {
        ModalBottomSheet(onDismissRequest = { showStagePicker = false }) {
            Column(modifier = Modifier.padding(horizontal = 16.dp).navigationBarsPadding()) {
                Text("Этап", style = MaterialTheme.typography.titleLarge, modifier = Modifier.padding(bottom = 8.dp))
                Text(
                    text = "Без этапа",
                    style = MaterialTheme.typography.bodyLarge,
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable {
                            showStagePicker = false
                            viewModel.setStage(null)
                        }
                        .padding(vertical = 12.dp),
                )
                viewModel.stages.forEach { stage ->
                    Text(
                        text = stage.name,
                        style = MaterialTheme.typography.bodyLarge,
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable {
                                showStagePicker = false
                                viewModel.setStage(stage.id)
                            }
                            .padding(vertical = 12.dp),
                    )
                }
            }
        }
    }

    if (showResponsiblePicker) {
        ModalBottomSheet(onDismissRequest = { showResponsiblePicker = false }) {
            Column(modifier = Modifier.padding(horizontal = 16.dp)) {
                Text(
                    "Ответственный",
                    style = MaterialTheme.typography.titleLarge,
                    modifier = Modifier.padding(bottom = 8.dp),
                )
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable {
                            showResponsiblePicker = false
                            viewModel.setResponsible(null)
                        }
                        .padding(vertical = 10.dp),
                ) {
                    Icon(Icons.Filled.Block, contentDescription = null, tint = MaterialTheme.colorScheme.onSurfaceVariant)
                    Text("Снять ответственного", modifier = Modifier.padding(start = 12.dp))
                }
                LazyColumn(modifier = Modifier.fillMaxWidth().padding(bottom = 16.dp)) {
                    items(viewModel.directory, key = { it.id }) { user ->
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
                                    showResponsiblePicker = false
                                    viewModel.setResponsible(user.id)
                                }
                                .padding(vertical = 8.dp),
                        ) {
                            UserAvatar(userId = user.id, name = user.fio, avatarPath = user.avatarPath, size = 40.dp)
                            Text(user.fio, modifier = Modifier.padding(start = 12.dp))
                        }
                    }
                }
            }
        }
    }

    if (showDeadlinePicker) {
        val datePickerState = rememberDatePickerState()
        DatePickerDialog(
            onDismissRequest = { showDeadlinePicker = false },
            confirmButton = {
                TextButton(onClick = {
                    datePickerState.selectedDateMillis?.let { millis ->
                        val date = Instant.ofEpochMilli(millis).atZone(ZoneOffset.UTC).toLocalDate()
                        viewModel.setDeadline(date.toString())
                    }
                    showDeadlinePicker = false
                }) { Text("Готово") }
            },
            dismissButton = {
                TextButton(onClick = {
                    viewModel.setDeadline(null)
                    showDeadlinePicker = false
                }) { Text("Убрать") }
            },
        ) {
            DatePicker(state = datePickerState)
        }
    }
}

@Composable
private fun TaskInfoTab(
    viewModel: TaskDetailViewModel,
    onRename: () -> Unit,
    onStage: () -> Unit,
    onResponsible: () -> Unit,
    onDeadline: () -> Unit,
    onComplete: () -> Unit,
) {
    val task = viewModel.task ?: return
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
    ) {
        item {
            Row(verticalAlignment = Alignment.Top) {
                Text(
                    text = task.name,
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.SemiBold,
                    modifier = Modifier.weight(1f),
                )
                IconButton(onClick = onRename) {
                    Icon(Icons.Filled.Edit, contentDescription = "Переименовать")
                }
            }
        }
        item { ColorRow(selected = task.color, onSelect = { viewModel.setColor(it) }) }
        item {
            viewModel.actionError?.let { error ->
                Text(
                    text = error,
                    color = MaterialTheme.colorScheme.error,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.padding(vertical = 4.dp),
                )
            }
        }
        item {
            MetaRow("Этап", task.stage?.name ?: "Не задан", onStage)
            MetaRow("Ответственный", task.responsible?.fio ?: "Не назначен", onResponsible)
            MetaRow("Дедлайн", task.deadline?.let { formatDate(it) } ?: "Не задан", onDeadline)
            MetaRow("Отдел", task.department?.name ?: "—", onClick = null)
            MetaRow("Автор", task.author?.fio ?: "—", onClick = null)
            MetaRow("Получена", task.receivedAt?.let { formatDate(it) } ?: "—", onClick = null)
        }
        if (!task.isArchived) {
            item {
                FilledTonalButton(
                    onClick = onComplete,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = 24.dp, bottom = 80.dp),
                ) {
                    Icon(Icons.Filled.CheckCircle, contentDescription = null, modifier = Modifier.size(18.dp))
                    Text("Завершить задачу", modifier = Modifier.padding(start = 8.dp))
                }
            }
        }
    }
}

@Composable
private fun UnitsTab(viewModel: TaskDetailViewModel, container: AppContainer) {
    LaunchedEffect(Unit) { viewModel.loadUnits() }
    val scope = rememberCoroutineScope()
    var editingUnit by remember { mutableStateOf<UnitDto?>(null) }
    var cloningUnit by remember { mutableStateOf<UnitDto?>(null) }
    Box(modifier = Modifier.fillMaxSize()) {
        when {
            viewModel.unitsLoading && viewModel.units.isEmpty() -> CenteredLoading()
            viewModel.units.isEmpty() -> Text(
                text = "Юнитов пока нет",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.align(Alignment.Center),
            )
            else -> LazyColumn(
                contentPadding = PaddingValues(16.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.fillMaxSize(),
            ) {
                items(viewModel.units, key = { it.id }) { unit ->
                    val canManage = viewModel.canManageUnit(unit)
                    UnitRow(
                        unit = unit,
                        canDelete = canManage,
                        onDelete = { viewModel.deleteUnit(unit) },
                        onEdit = if (canManage) ({ editingUnit = unit }) else null,
                        onClone = { cloningUnit = unit },
                    )
                }
            }
        }
    }
    editingUnit?.let { unit ->
        EditUnitSheet(
            container = container,
            viewModel = viewModel,
            unit = unit,
            onDismiss = { editingUnit = null },
        )
    }
    cloningUnit?.let { unit ->
        ConfirmDialog(
            ConfirmSpec(
                title = "Создать новый юнит",
                text = "Начать новый юнит «${unit.name}» с тем же типом? Учёт времени пойдёт заново.",
                confirmLabel = "Создать",
                destructive = false,
                action = {
                    scope.launch {
                        if (container.unitManager.cloneUnit(unit)) viewModel.loadUnits()
                    }
                },
            ),
            onDismiss = { cloningUnit = null },
        )
    }
}

@Composable
private fun CommentsTab(viewModel: TaskDetailViewModel) {
    Column(modifier = Modifier.fillMaxSize()) {
        LazyColumn(
            modifier = Modifier.weight(1f).fillMaxWidth(),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        ) {
            item {
                Text(
                    text = "Комментарии (${viewModel.comments.size})",
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.padding(bottom = 8.dp),
                )
            }
            items(viewModel.comments, key = { it.id }) { comment ->
                CommentRow(
                    comment = comment,
                    mine = comment.authorId == viewModel.myUserId,
                    onDelete = { viewModel.deleteComment(comment) },
                )
            }
            if (viewModel.comments.isEmpty()) {
                item {
                    Text(
                        text = "Пока нет комментариев",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
        CommentInputBar(viewModel)
    }
}

@Composable
private fun MetaRow(label: String, value: String, onClick: (() -> Unit)?) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier
            .fillMaxWidth()
            .let { if (onClick != null) it.clickable(onClick = onClick) else it }
            .padding(vertical = 10.dp),
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.weight(0.4f),
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyLarge,
            color = if (onClick != null) MaterialTheme.colorScheme.onSurface
            else MaterialTheme.colorScheme.onSurfaceVariant,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(0.6f),
        )
    }
}

@Composable
private fun ColorRow(selected: String?, onSelect: (String?) -> Unit) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(10.dp),
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier.padding(vertical = 8.dp),
    ) {
        TaskColorNames.forEach { name ->
            val colors = taskTagColors(name) ?: return@forEach
            Box(
                modifier = Modifier
                    .size(28.dp)
                    .clip(CircleShape)
                    .background(colors.accent)
                    .let {
                        if (selected == name) {
                            it.border(3.dp, MaterialTheme.colorScheme.onSurface, CircleShape)
                        } else {
                            it
                        }
                    }
                    .clickable { onSelect(if (selected == name) null else name) },
            )
        }
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun CommentRow(comment: CommentDto, mine: Boolean, onDelete: () -> Unit) {
    var menuOpen by remember { mutableStateOf(false) }
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .combinedClickable(onClick = {}, onLongClick = { if (mine) menuOpen = true })
            .padding(vertical = 8.dp),
    ) {
        UserAvatar(
            userId = comment.author?.id ?: comment.authorId,
            name = comment.author?.fio,
            avatarPath = comment.author?.avatarPath,
            size = 36.dp,
        )
        Column(modifier = Modifier.padding(start = 10.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = comment.author?.fio ?: "Пользователь",
                    style = MaterialTheme.typography.labelLarge,
                )
                Text(
                    text = formatDateTime(comment.createdAt),
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.outline,
                    modifier = Modifier.padding(start = 8.dp),
                )
            }
            LinkifiedText(
                text = comment.text,
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.padding(top = 2.dp),
            )
        }
        DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
            DropdownMenuItem(
                text = { Text("Удалить") },
                onClick = {
                    menuOpen = false
                    onDelete()
                },
            )
        }
    }
}

@Composable
private fun CommentInputBar(viewModel: TaskDetailViewModel) {
    Surface(color = MaterialTheme.colorScheme.surfaceContainer) {
        Row(
            verticalAlignment = Alignment.Bottom,
            modifier = Modifier
                .fillMaxWidth()
                .navigationBarsPadding()
                .imePadding()
                .padding(horizontal = 8.dp, vertical = 6.dp),
        ) {
            TextField(
                value = viewModel.commentInput,
                onValueChange = { viewModel.commentInput = it },
                placeholder = { Text("Комментарий…") },
                maxLines = 4,
                colors = TextFieldDefaults.colors(
                    focusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHighest,
                    unfocusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHighest,
                    focusedIndicatorColor = androidx.compose.ui.graphics.Color.Transparent,
                    unfocusedIndicatorColor = androidx.compose.ui.graphics.Color.Transparent,
                ),
                shape = RoundedCornerShape(24.dp),
                modifier = Modifier.weight(1f),
            )
            FilledIconButton(
                onClick = { viewModel.addComment() },
                enabled = viewModel.commentInput.isNotBlank() && !viewModel.sendingComment,
                modifier = Modifier.padding(start = 4.dp),
            ) {
                Icon(Icons.AutoMirrored.Filled.Send, contentDescription = "Отправить")
            }
        }
    }
}

@Composable
private fun RenameDialog(initial: String, onDismiss: () -> Unit, onConfirm: (String) -> Unit) {
    var value by remember { mutableStateOf(initial) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Название задачи") },
        text = {
            OutlinedTextField(
                value = value,
                onValueChange = { value = it },
                modifier = Modifier.fillMaxWidth(),
            )
        },
        confirmButton = {
            TextButton(onClick = { onConfirm(value.trim()) }, enabled = value.isNotBlank()) {
                Text("Сохранить")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("Отмена") }
        },
    )
}
