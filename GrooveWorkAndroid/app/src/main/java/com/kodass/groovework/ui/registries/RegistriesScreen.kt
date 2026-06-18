package com.kodass.groovework.ui.registries

import android.widget.Toast
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
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Sort
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.ArrowDownward
import androidx.compose.material.icons.filled.ArrowUpward
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.RegistryDto
import com.kodass.groovework.data.dto.RegistryFieldDto
import com.kodass.groovework.data.dto.RegistryRecordDto
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.ConfirmDialog
import com.kodass.groovework.ui.common.ConfirmSpec
import com.kodass.groovework.ui.common.EmptyState
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.RefreshOnResume
import com.kodass.groovework.ui.common.SearchField

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RegistriesScreen(
    container: AppContainer,
    onOpenRecord: (registryId: Long, recordId: Long) -> Unit,
) {
    val viewModel: RegistriesViewModel = viewModel {
        RegistriesViewModel(
            container.registriesRepo, container.sessionManager, container.gateway, container.json,
        )
    }
    val context = LocalContext.current
    RefreshOnResume { viewModel.refresh() }

    LaunchedEffect(viewModel.message) {
        viewModel.message?.let {
            Toast.makeText(context, it, Toast.LENGTH_SHORT).show()
            viewModel.consumeMessage()
        }
    }

    var confirm by remember { mutableStateOf<ConfirmSpec?>(null) }
    confirm?.let { ConfirmDialog(spec = it, onDismiss = { confirm = null }) }

    val selected = viewModel.selected

    Scaffold(
        topBar = { TopAppBar(title = { Text("Реестры") }) },
        floatingActionButton = {
            if (selected != null && selected.fields.isNotEmpty() && viewModel.selectedIds.isEmpty()) {
                ExtendedFloatingActionButton(
                    onClick = { onOpenRecord(selected.id, 0L) },
                    icon = { Icon(Icons.Filled.Add, contentDescription = null) },
                    text = { Text("Добавить") },
                )
            }
        },
    ) { padding ->
        Box(modifier = Modifier.fillMaxSize().padding(padding)) {
            when {
                viewModel.loadingRegistries && viewModel.registries.isEmpty() -> CenteredLoading()
                viewModel.registriesError != null && viewModel.registries.isEmpty() ->
                    ErrorState(viewModel.registriesError!!, onRetry = { viewModel.loadRegistries(initial = true) })
                viewModel.registries.isEmpty() ->
                    EmptyState("Реестров пока нет", "Создайте реестр в веб-версии — он появится здесь.")
                else -> Column(modifier = Modifier.fillMaxSize()) {
                    RegistryStrip(
                        registries = viewModel.registries,
                        selectedId = viewModel.selectedId,
                        onSelect = viewModel::select,
                    )
                    if (selected != null) {
                        SearchField(
                            value = viewModel.search,
                            onValueChange = viewModel::updateSearch,
                            placeholder = "Поиск по записям…",
                        )
                        if (selected.fields.isNotEmpty()) {
                            SortRow(
                                fields = selected.fields,
                                sort = viewModel.sort,
                                order = viewModel.order,
                                onSort = viewModel::setSortField,
                                onToggleOrder = viewModel::toggleOrder,
                            )
                        }
                        if (viewModel.selectedIds.isNotEmpty()) {
                            SelectionBar(
                                count = viewModel.selectedIds.size,
                                onDelete = {
                                    confirm = ConfirmSpec(
                                        title = "Удалить выбранные записи?",
                                        text = "Будет удалено записей: ${viewModel.selectedIds.size}. Действие необратимо.",
                                        action = { viewModel.bulkDelete() },
                                    )
                                },
                                onClear = viewModel::clearSelection,
                            )
                        }
                        RecordsList(
                            viewModel = viewModel,
                            registry = selected,
                            onOpenRecord = onOpenRecord,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun RegistryStrip(
    registries: List<RegistryDto>,
    selectedId: Long?,
    onSelect: (Long) -> Unit,
) {
    LazyRow(
        modifier = Modifier.fillMaxWidth(),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 6.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        items(registries, key = { it.id }) { r ->
            FilterChip(
                selected = r.id == selectedId,
                onClick = { onSelect(r.id) },
                label = { Text(r.name, maxLines = 1, overflow = TextOverflow.Ellipsis) },
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SortRow(
    fields: List<RegistryFieldDto>,
    sort: String,
    order: String,
    onSort: (String) -> Unit,
    onToggleOrder: () -> Unit,
) {
    val options = remember(fields) {
        buildList {
            add("" to "Дате создания")
            fields.filter { isSortable(it.type) }.forEach { add(it.id.toString() to it.label) }
        }
    }
    var expanded by remember { mutableStateOf(false) }
    val currentLabel = options.firstOrNull { it.first == sort }?.second ?: "Дате создания"

    Row(
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Icon(
            Icons.AutoMirrored.Filled.Sort,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Box(modifier = Modifier.weight(1f)) {
            OutlinedButton(onClick = { expanded = true }, modifier = Modifier.fillMaxWidth()) {
                Text(
                    "Сортировка: $currentLabel",
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f, fill = false),
                )
            }
            DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                options.forEach { (value, label) ->
                    DropdownMenuItem(
                        text = { Text(label) },
                        onClick = {
                            expanded = false
                            onSort(value)
                        },
                    )
                }
            }
        }
        IconButton(onClick = onToggleOrder) {
            Icon(
                if (order == "asc") Icons.Filled.ArrowUpward else Icons.Filled.ArrowDownward,
                contentDescription = if (order == "asc") "По возрастанию" else "По убыванию",
            )
        }
    }
}

@Composable
private fun SelectionBar(count: Int, onDelete: () -> Unit, onClear: () -> Unit) {
    Surface(color = MaterialTheme.colorScheme.primaryContainer) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(
                "Выбрано: $count",
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.SemiBold,
                color = MaterialTheme.colorScheme.onPrimaryContainer,
                modifier = Modifier.weight(1f),
            )
            AssistChip(
                onClick = onDelete,
                leadingIcon = { Icon(Icons.Filled.Delete, contentDescription = null, modifier = Modifier.size(18.dp)) },
                label = { Text("Удалить") },
            )
            AssistChip(onClick = onClear, label = { Text("Сбросить") })
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun RecordsList(
    viewModel: RegistriesViewModel,
    registry: RegistryDto,
    onOpenRecord: (Long, Long) -> Unit,
) {
    val listState = rememberLazyListState()
    // Подгрузка следующей страницы при подходе к концу списка.
    val atEnd by remember {
        derivedStateOf {
            val last = listState.layoutInfo.visibleItemsInfo.lastOrNull()?.index ?: 0
            last >= viewModel.records.size - 4
        }
    }
    LaunchedEffect(atEnd, viewModel.hasMore) {
        if (atEnd && viewModel.hasMore) viewModel.loadMore()
    }

    val shownFields = remember(registry) {
        registry.fields.filter { it.showInTable }.ifEmpty { registry.fields.take(4) }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        PullToRefreshBox(
            isRefreshing = viewModel.refreshing,
            onRefresh = viewModel::refresh,
        ) {
            LazyColumn(
                state = listState,
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(16.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                items(viewModel.records, key = { it.id }) { rec ->
                    RecordCard(
                        record = rec,
                        fields = shownFields,
                        selected = rec.id in viewModel.selectedIds,
                        onClick = { onOpenRecord(registry.id, rec.id) },
                        onToggleSelect = { viewModel.toggleRow(rec.id) },
                    )
                }
                if (viewModel.loadingMore) {
                    item {
                        Box(modifier = Modifier.fillMaxWidth().padding(16.dp), contentAlignment = Alignment.Center) {
                            CircularProgressIndicator(modifier = Modifier.size(28.dp), strokeWidth = 2.dp)
                        }
                    }
                }
            }
        }

        when {
            viewModel.loadingRecords && viewModel.records.isEmpty() -> CenteredLoading()
            viewModel.recordsError != null && viewModel.records.isEmpty() ->
                ErrorState(viewModel.recordsError!!, onRetry = { viewModel.refresh() })
            viewModel.records.isEmpty() && !viewModel.refreshing -> EmptyState(
                if (viewModel.search.isNotBlank()) "Ничего не найдено" else "Записей пока нет",
            )
        }
    }
}

@Composable
private fun RecordCard(
    record: RegistryRecordDto,
    fields: List<RegistryFieldDto>,
    selected: Boolean,
    onClick: () -> Unit,
    onToggleSelect: () -> Unit,
) {
    val titleField = fields.firstOrNull()
    val title = titleField?.let { textValue(it, record.data[it.key]) }?.takeIf { it.isNotBlank() }
        ?: "Запись #${record.id}"
    val bodyFields = fields.drop(1)

    Surface(
        onClick = onClick,
        shape = MaterialTheme.shapes.large,
        color = if (selected) MaterialTheme.colorScheme.primaryContainer
        else MaterialTheme.colorScheme.surfaceContainerLow,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(start = 4.dp, top = 8.dp, end = 14.dp, bottom = 12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Checkbox(checked = selected, onCheckedChange = { onToggleSelect() })
                Text(
                    title,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.weight(1f),
                )
                Icon(
                    Icons.Filled.ChevronRight,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            bodyFields.forEach { f ->
                val v = textValue(f, record.data[f.key]).ifBlank { "—" }
                Row(
                    modifier = Modifier.fillMaxWidth().padding(start = 12.dp, top = 4.dp),
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    Text(
                        f.label,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(0.4f),
                    )
                    Text(
                        v,
                        style = MaterialTheme.typography.bodyMedium,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(0.6f),
                    )
                }
            }
        }
    }
}
