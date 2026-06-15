package com.kodass.groovework.ui.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.AiIndexingDto
import com.kodass.groovework.data.dto.AiSettingsUpdate
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.ui.common.CenteredLoading
import kotlinx.coroutines.launch

@Composable
fun AiSettingsScreen(container: AppContainer, onBack: () -> Unit) {
    val authState by container.sessionManager.authState.collectAsStateWithLifecycle()
    val cid = (authState as? AuthState.LoggedIn)?.claims?.companyId
    val scope = rememberCoroutineScope()

    var enabled by remember { mutableStateOf(false) }
    var modelChat by remember { mutableStateOf("gpt-4o-mini") }
    var modelEmbedding by remember { mutableStateOf("text-embedding-3-small") }
    var apiKey by remember { mutableStateOf("") }
    var hasKey by remember { mutableStateOf(false) }
    var keyHint by remember { mutableStateOf<String?>(null) }
    var showKey by remember { mutableStateOf(false) }

    var initEnabled by remember { mutableStateOf(false) }
    var initChat by remember { mutableStateOf("gpt-4o-mini") }
    var initEmbedding by remember { mutableStateOf("text-embedding-3-small") }

    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var testing by remember { mutableStateOf(false) }
    var testResult by remember { mutableStateOf<Pair<Boolean, String>?>(null) }
    var indexing by remember { mutableStateOf<AiIndexingDto?>(null) }
    var reindexing by remember { mutableStateOf(false) }

    val dirty = enabled != initEnabled || modelChat != initChat ||
        modelEmbedding != initEmbedding || apiKey.trim().isNotEmpty()

    suspend fun loadIndexing() {
        indexing = if (cid != null && hasKey && enabled) {
            runCatching { apiCall(container.json) { container.aiApi.indexing(cid) } }.getOrNull()
        } else null
    }

    suspend fun load() {
        if (cid == null) { loading = false; return }
        try {
            val s = apiCall(container.json) { container.aiApi.settings(cid) }
            enabled = s.enabled; initEnabled = s.enabled
            modelChat = s.modelChat; initChat = s.modelChat
            modelEmbedding = s.modelEmbedding; initEmbedding = s.modelEmbedding
            hasKey = s.hasKey; keyHint = s.keyHint; apiKey = ""
            loadIndexing()
        } catch (_: Exception) {} finally { loading = false }
    }

    LaunchedEffect(cid) { load() }

    SettingsSubScaffold(title = "Нейро-функции", onBack = onBack) { padding ->
        if (cid == null) { NoCompanyHint(); return@SettingsSubScaffold }
        if (loading) { CenteredLoading(Modifier.padding(padding)); return@SettingsSubScaffold }
        Column(
            modifier = Modifier.fillMaxSize().padding(padding).padding(16.dp).verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("ИИ-функции через ProxyAPI", style = MaterialTheme.typography.titleMedium)
                    Text(
                        "Ключ хранится в зашифрованном виде. Без действующего ключа функции остаются выключенными.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 4.dp, bottom = 8.dp),
                    )
                    SwitchRow(
                        title = "Включить ИИ для компании",
                        subtitle = "Факт дня в ТВ и семантический поиск задач",
                        checked = enabled,
                        onChange = { enabled = it },
                    )
                    OutlinedTextField(
                        value = apiKey,
                        onValueChange = { apiKey = it },
                        label = { Text(if (hasKey) "Новый ключ (необязательно)" else "Ключ ProxyAPI") },
                        placeholder = { Text(if (hasKey) "Текущий: ${keyHint ?: "••••"}" else "sk-…") },
                        singleLine = true,
                        visualTransformation = if (showKey) VisualTransformation.None else PasswordVisualTransformation(),
                        trailingIcon = {
                            IconButton(onClick = { showKey = !showKey }) {
                                Icon(
                                    if (showKey) Icons.Filled.VisibilityOff else Icons.Filled.Visibility,
                                    contentDescription = null,
                                )
                            }
                        },
                        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                    )
                    if (hasKey) {
                        Text(
                            "Оставьте поле пустым, чтобы не менять текущий ключ.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(top = 4.dp),
                        )
                    }
                    OutlinedTextField(
                        value = modelChat,
                        onValueChange = { modelChat = it },
                        label = { Text("Модель чата") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                    )
                    OutlinedTextField(
                        value = modelEmbedding,
                        onValueChange = { modelEmbedding = it },
                        label = { Text("Модель эмбеддингов") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                    )

                    testResult?.let { (ok, text) ->
                        Text(
                            text = text,
                            color = if (ok) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.error,
                            style = MaterialTheme.typography.bodySmall,
                            modifier = Modifier.padding(top = 8.dp),
                        )
                    }

                    Row(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                    ) {
                        OutlinedButton(
                            onClick = {
                                scope.launch {
                                    testing = true; testResult = null
                                    try {
                                        val r = apiCall(container.json) { container.aiApi.test(cid) }
                                        val ok = r.chat && r.embedding
                                        testResult = ok to (
                                            if (ok) "Связь установлена (${r.latencyMs ?: 0} мс)"
                                            else "Ошибка: ${r.error ?: "модель не ответила"}"
                                        )
                                    } catch (e: ApiException) {
                                        testResult = false to (e.message)
                                    } finally { testing = false }
                                }
                            },
                            enabled = !testing && hasKey && !dirty,
                            modifier = Modifier.weight(1f),
                        ) { Text(if (testing) "Проверяю…" else "Проверить") }
                        Button(
                            onClick = {
                                scope.launch {
                                    saving = true; testResult = null
                                    try {
                                        val key = apiKey.trim()
                                        val s = apiCall(container.json) {
                                            container.aiApi.updateSettings(
                                                cid,
                                                AiSettingsUpdate(
                                                    enabled = enabled,
                                                    modelChat = modelChat.trim().ifBlank { "gpt-4o-mini" },
                                                    modelEmbedding = modelEmbedding.trim().ifBlank { "text-embedding-3-small" },
                                                    apiKey = key.ifBlank { null },
                                                ),
                                            )
                                        }
                                        enabled = s.enabled; initEnabled = s.enabled
                                        modelChat = s.modelChat; initChat = s.modelChat
                                        modelEmbedding = s.modelEmbedding; initEmbedding = s.modelEmbedding
                                        hasKey = s.hasKey; keyHint = s.keyHint; apiKey = ""
                                        loadIndexing()
                                    } catch (e: ApiException) {
                                        testResult = false to (e.message)
                                    } finally { saving = false }
                                }
                            },
                            enabled = !saving && dirty,
                            modifier = Modifier.weight(1f),
                        ) { Text(if (saving) "Сохраняю…" else "Сохранить") }
                    }

                    if (hasKey) {
                        OutlinedButton(
                            onClick = {
                                scope.launch {
                                    saving = true
                                    try {
                                        val s = apiCall(container.json) {
                                            container.aiApi.updateSettings(cid, AiSettingsUpdate(clearKey = true))
                                        }
                                        hasKey = s.hasKey; keyHint = s.keyHint
                                        enabled = s.enabled; initEnabled = s.enabled
                                        loadIndexing()
                                    } catch (_: Exception) {} finally { saving = false }
                                }
                            },
                            enabled = !saving,
                            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                        ) { Text("Удалить ключ") }
                    }
                }
            }

            indexing?.let { idx ->
                Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text("Индексация задач", style = MaterialTheme.typography.titleMedium)
                        Text(
                            "Проиндексировано ${idx.indexed} / ${idx.totalTasks}" +
                                (if (idx.pending > 0) " · осталось ${idx.pending}" else ""),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(top = 4.dp),
                        )
                        Button(
                            onClick = {
                                scope.launch {
                                    reindexing = true
                                    try {
                                        apiCall(container.json) { container.aiApi.reindex(cid) }
                                        loadIndexing()
                                    } catch (_: Exception) {} finally { reindexing = false }
                                }
                            },
                            enabled = !reindexing && idx.pending > 0,
                            modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                        ) {
                            if (reindexing) CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                            Text(
                                if (reindexing) "Запускаю…" else "Переиндексировать",
                                modifier = Modifier.padding(start = if (reindexing) 8.dp else 0.dp),
                            )
                        }
                    }
                }
            }
        }
    }
}
