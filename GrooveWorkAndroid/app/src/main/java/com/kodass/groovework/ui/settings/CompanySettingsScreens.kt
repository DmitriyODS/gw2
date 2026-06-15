package com.kodass.groovework.ui.settings

import android.widget.Toast
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.Download
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Upload
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.AnnotatedString
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.GrooveSettingsDto
import com.kodass.groovework.data.dto.WeekendSettingsDto
import com.kodass.groovework.data.files.DownloadState
import com.kodass.groovework.data.files.openDownloadedFile
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.ConfirmDialog
import com.kodass.groovework.ui.common.ConfirmSpec
import kotlinx.coroutines.launch
import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.MultipartBody
import okhttp3.RequestBody.Companion.toRequestBody
import java.time.LocalDate
import java.time.format.DateTimeFormatter

@Composable
private fun companyId(container: AppContainer): Long? {
    val authState by container.sessionManager.authState.collectAsStateWithLifecycle()
    return (authState as? AuthState.LoggedIn)?.claims?.companyId
}

@Composable
internal fun NoCompanyHint() {
    Box(modifier = Modifier.fillMaxSize().padding(24.dp), contentAlignment = Alignment.Center) {
        Text(
            "Раздел доступен пользователю компании. Управление по всем компаниям — в веб-версии.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

// ── Выходные дни ────────────────────────────────────────────────────────────
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun WeekendSettingsScreen(container: AppContainer, onBack: () -> Unit) {
    val cid = companyId(container)
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var days by remember { mutableStateOf<Set<Int>>(emptySet()) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }

    LaunchedEffect(cid) {
        if (cid == null) { loading = false; return@LaunchedEffect }
        try {
            days = apiCall(container.json) { container.companiesApi.weekendSettings(cid) }.weekendDays.toSet()
        } catch (_: Exception) {} finally { loading = false }
    }

    SettingsSubScaffold(title = "Выходные дни", onBack = onBack) { padding ->
        if (cid == null) { NoCompanyHint(); return@SettingsSubScaffold }
        if (loading) { CenteredLoading(Modifier.padding(padding)); return@SettingsSubScaffold }
        Column(modifier = Modifier.fillMaxSize().padding(padding).padding(16.dp).verticalScroll(rememberScrollState())) {
            Text(
                "Отметьте выходные компании — в эти дни Грувик отдыхает и не болеет от простоя.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            run {
                val names = listOf("Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс")
                FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    modifier = Modifier.fillMaxWidth().padding(top = 16.dp),
                ) {
                    names.forEachIndexed { index, name ->
                        FilterChip(
                            selected = index in days,
                            onClick = { days = if (index in days) days - index else days + index },
                            label = { Text(name) },
                        )
                    }
                }
                Button(
                    onClick = {
                        scope.launch {
                            saving = true
                            try {
                                apiCall(container.json) {
                                    container.companiesApi.updateWeekendSettings(cid, WeekendSettingsDto(days.sorted()))
                                }
                                Toast.makeText(context, "Сохранено", Toast.LENGTH_SHORT).show()
                            } catch (e: ApiException) {
                                Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
                            } finally { saving = false }
                        }
                    },
                    enabled = !saving,
                    modifier = Modifier.fillMaxWidth().padding(top = 20.dp),
                ) { Text(if (saving) "Сохраняю…" else "Сохранить") }
            }
        }
    }
}

// ── Мой Groove ──────────────────────────────────────────────────────────────
@Composable
fun GrooveSettingsScreen(container: AppContainer, onBack: () -> Unit) {
    val cid = companyId(container)
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var enabled by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }

    LaunchedEffect(cid) {
        if (cid == null) { loading = false; return@LaunchedEffect }
        try {
            enabled = apiCall(container.json) { container.companiesApi.grooveSettings(cid) }.enabled
        } catch (_: Exception) {} finally { loading = false }
    }

    SettingsSubScaffold(title = "Мой Groove", onBack = onBack) { padding ->
        if (cid == null) { NoCompanyHint(); return@SettingsSubScaffold }
        if (loading) { CenteredLoading(Modifier.padding(padding)); return@SettingsSubScaffold }
        Column(modifier = Modifier.fillMaxSize().padding(padding).padding(16.dp)) {
            run {
                Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        SwitchRow(
                            title = "Геймификация «Мой Groove»",
                            subtitle = "Питомцы-Грувики, грувы, рейды и квесты для команды",
                            checked = enabled,
                            enabled = !saving,
                            onChange = { enabled = it },
                        )
                    }
                }
                Button(
                    onClick = {
                        scope.launch {
                            saving = true
                            try {
                                apiCall(container.json) {
                                    container.companiesApi.updateGrooveSettings(cid, GrooveSettingsDto(enabled))
                                }
                                Toast.makeText(context, "Сохранено", Toast.LENGTH_SHORT).show()
                            } catch (e: ApiException) {
                                Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
                            } finally { saving = false }
                        }
                    },
                    enabled = !saving,
                    modifier = Modifier.fillMaxWidth().padding(top = 16.dp),
                ) { Text(if (saving) "Сохраняю…" else "Сохранить") }
            }
        }
    }
}

// ── Ссылка-приглашение ──────────────────────────────────────────────────────
@Composable
fun InviteSettingsScreen(container: AppContainer, onBack: () -> Unit) {
    val cid = companyId(container)
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val clipboard = LocalClipboardManager.current
    val serverUrl by container.sessionManager.serverUrl.collectAsStateWithLifecycle()
    var code by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var busy by remember { mutableStateOf(false) }

    LaunchedEffect(cid) {
        if (cid == null) { loading = false; return@LaunchedEffect }
        try {
            code = apiCall(container.json) { container.companiesApi.invite(cid) }.code.ifBlank { null }
        } catch (_: Exception) {} finally { loading = false }
    }

    val link = code?.let { "${serverUrl.trimEnd('/')}/join/$it" }

    SettingsSubScaffold(title = "Ссылка-приглашение", onBack = onBack) { padding ->
        if (cid == null) { NoCompanyHint(); return@SettingsSubScaffold }
        if (loading) { CenteredLoading(Modifier.padding(padding)); return@SettingsSubScaffold }
        Column(modifier = Modifier.fillMaxSize().padding(padding).padding(16.dp)) {
            Text(
                "Поделитесь ссылкой, чтобы пригласить сотрудника в компанию. Перевыпуск делает старую ссылку недействительной.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            run {
                Card(
                    colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
                    modifier = Modifier.fillMaxWidth().padding(top = 16.dp),
                ) {
                    Text(
                        text = link ?: "Ссылка ещё не создана",
                        style = MaterialTheme.typography.bodyMedium,
                        modifier = Modifier.padding(16.dp),
                    )
                }
                if (link != null) {
                    OutlinedButton(
                        onClick = {
                            clipboard.setText(AnnotatedString(link))
                            Toast.makeText(context, "Ссылка скопирована", Toast.LENGTH_SHORT).show()
                        },
                        modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                    ) {
                        Icon(Icons.Filled.ContentCopy, contentDescription = null, modifier = Modifier.size(18.dp))
                        Text("Скопировать", modifier = Modifier.padding(start = 8.dp))
                    }
                }
                Button(
                    onClick = {
                        scope.launch {
                            busy = true
                            try {
                                code = apiCall(container.json) { container.companiesApi.regenerateInvite(cid) }.code
                                Toast.makeText(context, "Ссылка обновлена", Toast.LENGTH_SHORT).show()
                            } catch (e: ApiException) {
                                Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
                            } finally { busy = false }
                        }
                    },
                    enabled = !busy,
                    modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                ) {
                    Icon(Icons.Filled.Refresh, contentDescription = null, modifier = Modifier.size(18.dp))
                    Text(if (code == null) "Создать ссылку" else "Перевыпустить", modifier = Modifier.padding(start = 8.dp))
                }
            }
        }
    }
}

// ── Резервная копия ─────────────────────────────────────────────────────────
@Composable
fun BackupSettingsScreen(container: AppContainer, onBack: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val serverUrl by container.sessionManager.serverUrl.collectAsStateWithLifecycle()
    var exportState by remember { mutableStateOf<DownloadState>(DownloadState.Idle) }
    var importing by remember { mutableStateOf(false) }
    var confirm by remember { mutableStateOf<ConfirmSpec?>(null) }

    confirm?.let { ConfirmDialog(it, onDismiss = { confirm = null }) }

    fun doImport(uri: android.net.Uri) {
        scope.launch {
            importing = true
            try {
                val bytes = withIO { context.contentResolver.openInputStream(uri)?.use { it.readBytes() } }
                if (bytes == null) {
                    Toast.makeText(context, "Не удалось прочитать файл", Toast.LENGTH_SHORT).show()
                    return@launch
                }
                val body = bytes.toRequestBody("application/zip".toMediaTypeOrNull())
                val part = MultipartBody.Part.createFormData("file", "backup.zip", body)
                apiCall(container.json) { container.backupApi.import(part) }
                Toast.makeText(context, "Резервная копия восстановлена", Toast.LENGTH_LONG).show()
            } catch (e: ApiException) {
                Toast.makeText(context, e.message, Toast.LENGTH_LONG).show()
            } catch (_: Exception) {
                Toast.makeText(context, "Не удалось восстановить", Toast.LENGTH_LONG).show()
            } finally { importing = false }
        }
    }

    val picker = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        if (uri != null) {
            confirm = ConfirmSpec(
                title = "Восстановить из копии",
                text = "ВНИМАНИЕ: импорт ПОЛНОСТЬЮ заменит все текущие данные и необратим. Продолжить?",
                confirmLabel = "Восстановить",
                destructive = true,
                action = { doImport(uri) },
            )
        }
    }

    SettingsSubScaffold(title = "Резервная копия", onBack = onBack) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding).padding(16.dp)) {
            Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow), modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Создать резервную копию", style = MaterialTheme.typography.titleMedium)
                    Text(
                        "Скачивает ZIP-архив всех данных в папку «Загрузки».",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 4.dp),
                    )
                    val state = exportState
                    Button(
                        onClick = {
                            if (state is DownloadState.Running) return@Button
                            exportState = DownloadState.Running(-1f)
                            scope.launch {
                                try {
                                    val date = LocalDate.now().format(DateTimeFormatter.ofPattern("yyyy-MM-dd"))
                                    val uri = container.downloader.download(
                                        url = "${serverUrl.trimEnd('/')}/api/backup/export",
                                        fileName = "backup_$date.zip",
                                        mime = "application/zip",
                                        toImages = false,
                                    ) { p -> exportState = DownloadState.Running(p) }
                                    exportState = DownloadState.Done(uri, "application/zip")
                                    Toast.makeText(context, "Копия сохранена", Toast.LENGTH_SHORT).show()
                                } catch (_: Exception) {
                                    exportState = DownloadState.Idle
                                    Toast.makeText(context, "Не удалось создать копию", Toast.LENGTH_SHORT).show()
                                }
                            }
                        },
                        enabled = state !is DownloadState.Running,
                        modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                    ) {
                        if (state is DownloadState.Running) {
                            CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                        } else {
                            Icon(Icons.Filled.Download, contentDescription = null, modifier = Modifier.size(18.dp))
                        }
                        Text(
                            text = when (state) {
                                is DownloadState.Running -> "Скачиваю…"
                                is DownloadState.Done -> "Открыть архив"
                                else -> "Создать копию"
                            },
                            modifier = Modifier.padding(start = 8.dp),
                        )
                    }
                    if (state is DownloadState.Done) {
                        OutlinedButton(
                            onClick = { openDownloadedFile(context, state.uri, "application/zip") },
                            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                        ) { Text("Открыть архив") }
                    }
                }
            }

            Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow), modifier = Modifier.fillMaxWidth().padding(top = 12.dp)) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Восстановление", style = MaterialTheme.typography.titleMedium)
                    Text(
                        "Загрузка ZIP полностью заменит текущие данные. Действие необратимо!",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                        modifier = Modifier.padding(top = 4.dp),
                    )
                    Button(
                        onClick = { picker.launch("application/zip") },
                        enabled = !importing,
                        modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                    ) {
                        if (importing) {
                            CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                        } else {
                            Icon(Icons.Filled.Upload, contentDescription = null, modifier = Modifier.size(18.dp))
                        }
                        Text(if (importing) "Восстанавливаю…" else "Выбрать архив", modifier = Modifier.padding(start = 8.dp))
                    }
                }
            }
        }
    }
}

private suspend fun <T> withIO(block: () -> T): T =
    kotlinx.coroutines.withContext(kotlinx.coroutines.Dispatchers.IO) { block() }
