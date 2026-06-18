package com.kodass.groovework.ui.registries

import android.widget.Toast
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
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Edit
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.ErrorState

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RegistryRecordScreen(
    container: AppContainer,
    registryId: Long,
    recordId: Long,
    onBack: () -> Unit,
) {
    val viewModel: RegistryRecordViewModel = viewModel {
        RegistryRecordViewModel(registryId, recordId, container.registriesRepo, container.json)
    }
    val context = LocalContext.current

    LaunchedEffect(viewModel.message) {
        viewModel.message?.let {
            Toast.makeText(context, it, Toast.LENGTH_SHORT).show()
            viewModel.consumeMessage()
        }
    }

    val registry = viewModel.registry
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
            if (registry != null && !viewModel.loading) {
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
                                enabled = !viewModel.saving && registry.fields.isNotEmpty(),
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
                            Button(
                                onClick = { viewModel.startEdit() },
                                modifier = Modifier.fillMaxWidth(),
                            ) {
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
                registry == null -> ErrorState("Реестр не найден")
                registry.fields.isEmpty() -> Box(
                    modifier = Modifier.fillMaxSize().padding(32.dp),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        "В этом реестре пока нет полей.",
                        style = MaterialTheme.typography.bodyLarge,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                else -> LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(16.dp),
                    verticalArrangement = Arrangement.spacedBy(16.dp),
                ) {
                    items(registry.fields, key = { it.id }) { field ->
                        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                            Text(
                                field.label,
                                style = MaterialTheme.typography.labelMedium,
                                fontWeight = FontWeight.SemiBold,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                            if (viewModel.editing) {
                                RegistryFieldInput(
                                    field = field,
                                    value = viewModel.value(field.key),
                                    uploading = viewModel.uploading[field.key] == true,
                                    onChange = { viewModel.setValue(field.key, it) },
                                    onUpload = { name, mime, bytes ->
                                        viewModel.uploadFile(field.key, name, mime, bytes)
                                    },
                                )
                            } else {
                                RegistryFieldValue(
                                    container = container,
                                    field = field,
                                    value = viewModel.value(field.key),
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
