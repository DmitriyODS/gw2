package com.kodass.groovework.ui.settings

import android.widget.Toast
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowDropDown
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.YougileConnectRequest
import com.kodass.groovework.data.dto.YougileLoginRequest
import com.kodass.groovework.data.dto.YougileNamedDto
import com.kodass.groovework.data.dto.YougileRotateRequest
import com.kodass.groovework.data.dto.YougileSettingsDto
import com.kodass.groovework.data.dto.YougileStatusDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.ConfirmDialog
import com.kodass.groovework.ui.common.ConfirmSpec
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

// ── Личный коннект YouGile ──────────────────────────────────────────────────
@Composable
fun YougileUserSettingsScreen(container: AppContainer, onBack: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var status by remember { mutableStateOf<YougileStatusDto?>(null) }
    var loading by remember { mutableStateOf(true) }
    var login by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var showRotate by remember { mutableStateOf(false) }
    var rotatePass by remember { mutableStateOf("") }

    suspend fun refresh() {
        status = runCatching { apiCall(container.json) { container.yougileApi.status() } }.getOrNull()
    }
    LaunchedEffect(Unit) { refresh(); loading = false }

    if (showRotate) {
        AlertDialog(
            onDismissRequest = { showRotate = false },
            title = { Text("Перевыпуск ключа") },
            text = {
                OutlinedTextField(
                    value = rotatePass,
                    onValueChange = { rotatePass = it },
                    label = { Text("Пароль YouGile") },
                    singleLine = true,
                    visualTransformation = PasswordVisualTransformation(),
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        busy = true
                        try {
                            apiCall(container.json) { container.yougileApi.rotate(YougileRotateRequest(rotatePass)) }
                            refresh()
                            Toast.makeText(context, "Ключ перевыпущен", Toast.LENGTH_SHORT).show()
                            showRotate = false; rotatePass = ""
                        } catch (e: ApiException) {
                            Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
                        } finally { busy = false }
                    }
                }) { Text("Перевыпустить") }
            },
            dismissButton = { TextButton(onClick = { showRotate = false }) { Text("Отмена") } },
        )
    }

    SettingsSubScaffold(title = "YouGile", onBack = onBack) { padding ->
        if (loading) { CenteredLoading(Modifier.padding(padding)); return@SettingsSubScaffold }
        Column(modifier = Modifier.fillMaxSize().padding(padding).padding(16.dp).verticalScroll(rememberScrollState())) {
            val s = status
            if (s?.connected == true) {
                Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text("Подключено", style = MaterialTheme.typography.titleMedium, color = MaterialTheme.colorScheme.primary)
                        s.ygLogin?.let { Text("Аккаунт: $it", style = MaterialTheme.typography.bodyMedium, modifier = Modifier.padding(top = 6.dp)) }
                        s.keyFingerprint?.let { Text("Ключ: $it", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant) }
                    }
                }
                OutlinedButton(onClick = { showRotate = true }, enabled = !busy, modifier = Modifier.fillMaxWidth().padding(top = 12.dp)) {
                    Text("Перевыпустить ключ")
                }
                OutlinedButton(
                    onClick = {
                        scope.launch {
                            busy = true
                            try {
                                apiCall(container.json) { container.yougileApi.disconnect() }
                                refresh()
                                Toast.makeText(context, "YouGile отвязан", Toast.LENGTH_SHORT).show()
                            } catch (e: ApiException) {
                                Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
                            } finally { busy = false }
                        }
                    },
                    enabled = !busy,
                    modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                ) { Text("Отвязать аккаунт") }
            } else {
                Text(
                    "Подключите личный аккаунт YouGile, чтобы импортировать и экспортировать задачи.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                OutlinedTextField(value = login, onValueChange = { login = it }, label = { Text("Логин YouGile") }, singleLine = true, modifier = Modifier.fillMaxWidth().padding(top = 16.dp))
                OutlinedTextField(value = password, onValueChange = { password = it }, label = { Text("Пароль") }, singleLine = true, visualTransformation = PasswordVisualTransformation(), modifier = Modifier.fillMaxWidth().padding(top = 8.dp))
                Button(
                    onClick = {
                        scope.launch {
                            busy = true
                            try {
                                apiCall(container.json) { container.yougileApi.connect(YougileConnectRequest(login.trim(), password)) }
                                password = ""
                                refresh()
                                Toast.makeText(context, "YouGile подключён", Toast.LENGTH_SHORT).show()
                            } catch (e: ApiException) {
                                Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
                            } finally { busy = false }
                        }
                    },
                    enabled = !busy && login.isNotBlank() && password.isNotEmpty(),
                    modifier = Modifier.fillMaxWidth().padding(top = 16.dp),
                ) { Text(if (busy) "Подключаю…" else "Подключить") }
            }
        }
    }
}

// ── Интеграция YouGile для компании (визард) ────────────────────────────────
@Composable
fun YougileCompanySettingsScreen(container: AppContainer, onBack: () -> Unit) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var status by remember { mutableStateOf<YougileStatusDto?>(null) }
    var settings by remember { mutableStateOf<YougileSettingsDto?>(null) }
    var companies by remember { mutableStateOf<List<com.kodass.groovework.data.dto.YougileRefDto>>(emptyList()) }
    var projects by remember { mutableStateOf<List<YougileNamedDto>>(emptyList()) }
    var boards by remember { mutableStateOf<List<YougileNamedDto>>(emptyList()) }
    var columns by remember { mutableStateOf<List<YougileNamedDto>>(emptyList()) }
    var adminLogin by remember { mutableStateOf("") }
    var adminPassword by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var busy by remember { mutableStateOf(false) }
    var confirm by remember { mutableStateOf<ConfirmSpec?>(null) }

    confirm?.let { ConfirmDialog(it, onDismiss = { confirm = null }) }

    suspend fun refreshStatus() {
        status = runCatching { apiCall(container.json) { container.yougileApi.status() } }.getOrNull()
    }
    suspend fun update(obj: JsonObject) {
        settings = apiCall(container.json) { container.yougileApi.updateCompanySettings(obj) }
        refreshStatus()
    }
    suspend fun loadProjects() {
        projects = runCatching { apiCall(container.json) { container.yougileApi.projects() } }.getOrDefault(emptyList())
    }
    suspend fun loadBoards(projectId: String) {
        boards = runCatching { apiCall(container.json) { container.yougileApi.boards(projectId) } }.getOrDefault(emptyList())
    }
    suspend fun loadColumns(boardId: String) {
        columns = runCatching { apiCall(container.json) { container.yougileApi.columns(boardId) } }.getOrDefault(emptyList())
    }

    LaunchedEffect(Unit) {
        runCatching {
            refreshStatus()
            settings = apiCall(container.json) { container.yougileApi.companySettings() }
        }
        val st = status
        val cfg = settings
        if (st?.connected == true) {
            loadProjects()
            cfg?.ygProjectId?.let { pid ->
                loadBoards(pid)
                cfg.ygBoardId?.let { loadColumns(it) }
            }
        }
        loading = false
    }

    SettingsSubScaffold(title = "Интеграция YouGile", onBack = onBack) { padding ->
        if (loading) { CenteredLoading(Modifier.padding(padding)); return@SettingsSubScaffold }
        Column(
            modifier = Modifier.fillMaxSize().padding(padding).padding(16.dp).verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            val cfg = settings
            val connected = status?.connected == true

            // Шаг 1 — подключение компании.
            if (!connected || cfg?.ygCompanyId == null) {
                Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text("Шаг 1. Вход администратора YouGile", style = MaterialTheme.typography.titleMedium)
                        OutlinedTextField(value = adminLogin, onValueChange = { adminLogin = it }, label = { Text("Логин (email)") }, singleLine = true, modifier = Modifier.fillMaxWidth().padding(top = 8.dp))
                        OutlinedTextField(value = adminPassword, onValueChange = { adminPassword = it }, label = { Text("Пароль") }, singleLine = true, visualTransformation = PasswordVisualTransformation(), modifier = Modifier.fillMaxWidth().padding(top = 8.dp))
                        Button(
                            onClick = {
                                scope.launch {
                                    busy = true
                                    try {
                                        companies = apiCall(container.json) {
                                            container.yougileApi.lookupCompanies(YougileLoginRequest(adminLogin.trim(), adminPassword))
                                        }
                                    } catch (e: ApiException) {
                                        Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
                                    } finally { busy = false }
                                }
                            },
                            enabled = !busy && adminLogin.isNotBlank() && adminPassword.isNotEmpty(),
                            modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                        ) { Text(if (busy) "Загружаю…" else "Получить список компаний") }

                        companies.forEach { c ->
                            OutlinedButton(
                                onClick = {
                                    scope.launch {
                                        busy = true
                                        try {
                                            apiCall(container.json) {
                                                container.yougileApi.connect(YougileConnectRequest(adminLogin.trim(), adminPassword, c.id))
                                            }
                                            update(buildJsonObject {
                                                put("yg_company_id", c.id)
                                                put("yg_company_name", c.name)
                                            })
                                            adminPassword = ""
                                            loadProjects()
                                            Toast.makeText(context, "Компания выбрана", Toast.LENGTH_SHORT).show()
                                        } catch (e: ApiException) {
                                            Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
                                        } finally { busy = false }
                                    }
                                },
                                enabled = !busy,
                                modifier = Modifier.fillMaxWidth().padding(top = 6.dp),
                            ) { Text(c.name) }
                        }
                    }
                }
            }

            // Шаги 2-4 — проект/доска/колонка.
            if (connected && cfg?.ygCompanyId != null) {
                Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text("Компания: ${cfg.ygCompanyName ?: cfg.ygCompanyId}", style = MaterialTheme.typography.titleMedium)

                        PickerField(
                            label = "Проект",
                            current = cfg.ygProjectTitle,
                            options = projects,
                            enabled = !busy,
                            onPick = { picked ->
                                scope.launch {
                                    busy = true
                                    try {
                                        update(buildJsonObject {
                                            put("yg_project_id", picked.id)
                                            put("yg_project_title", picked.title)
                                            put("yg_board_id", JsonNull)
                                            put("yg_board_title", JsonNull)
                                            put("yg_completed_column_id", JsonNull)
                                        })
                                        boards = emptyList(); columns = emptyList()
                                        loadBoards(picked.id)
                                    } catch (e: ApiException) {
                                        Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
                                    } finally { busy = false }
                                }
                            },
                        )

                        if (cfg.ygProjectId != null) {
                            PickerField(
                                label = "Доска",
                                current = cfg.ygBoardTitle,
                                options = boards,
                                enabled = !busy,
                                modifier = Modifier.padding(top = 8.dp),
                                onPick = { picked ->
                                    scope.launch {
                                        busy = true
                                        try {
                                            update(buildJsonObject {
                                                put("yg_board_id", picked.id)
                                                put("yg_board_title", picked.title)
                                                put("yg_completed_column_id", JsonNull)
                                            })
                                            columns = emptyList()
                                            loadColumns(picked.id)
                                        } catch (e: ApiException) {
                                            Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
                                        } finally { busy = false }
                                    }
                                },
                            )
                        }

                        if (cfg.ygBoardId != null) {
                            val completedTitle = columns.firstOrNull { it.id == cfg.ygCompletedColumnId }?.title
                            PickerField(
                                label = "Колонка «Выполнено» (необязательно)",
                                current = completedTitle,
                                options = columns,
                                enabled = !busy,
                                modifier = Modifier.padding(top = 8.dp),
                                onPick = { picked ->
                                    scope.launch {
                                        busy = true
                                        try {
                                            update(buildJsonObject { put("yg_completed_column_id", picked.id) })
                                        } catch (e: ApiException) {
                                            Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
                                        } finally { busy = false }
                                    }
                                },
                            )
                        }
                    }
                }

                val canEnable = cfg.ygCompanyId != null && cfg.ygProjectId != null && cfg.ygBoardId != null
                Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        SwitchRow(
                            title = "Включить интеграцию",
                            subtitle = if (canEnable) "Двусторонняя синхронизация задач" else "Сначала выберите проект и доску",
                            checked = cfg.enabled,
                            enabled = !busy && (canEnable || cfg.enabled),
                            onChange = { v ->
                                scope.launch {
                                    busy = true
                                    try {
                                        update(buildJsonObject { put("enabled", v) })
                                        Toast.makeText(context, if (v) "Интеграция включена" else "Интеграция выключена", Toast.LENGTH_SHORT).show()
                                    } catch (e: ApiException) {
                                        Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
                                    } finally { busy = false }
                                }
                            },
                        )
                    }
                }

                OutlinedButton(
                    onClick = {
                        confirm = ConfirmSpec(
                            title = "Сбросить интеграцию",
                            text = "Снять привязку, отключить вебхук и начать настройку заново?",
                            confirmLabel = "Сбросить",
                            destructive = true,
                            action = {
                                scope.launch {
                                    busy = true
                                    try {
                                        settings = apiCall(container.json) { container.yougileApi.reset() }
                                        refreshStatus()
                                        companies = emptyList(); projects = emptyList(); boards = emptyList(); columns = emptyList()
                                        adminLogin = ""; adminPassword = ""
                                        Toast.makeText(context, "Интеграция сброшена", Toast.LENGTH_SHORT).show()
                                    } catch (e: ApiException) {
                                        Toast.makeText(context, e.message, Toast.LENGTH_SHORT).show()
                                    } finally { busy = false }
                                }
                            },
                        )
                    },
                    enabled = !busy,
                    modifier = Modifier.fillMaxWidth(),
                ) { Text("Сбросить интеграцию") }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun PickerField(
    label: String,
    current: String?,
    options: List<YougileNamedDto>,
    enabled: Boolean,
    modifier: Modifier = Modifier,
    onPick: (YougileNamedDto) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    Box(modifier = modifier.fillMaxWidth()) {
        OutlinedTextField(
            value = current ?: "Не выбрано",
            onValueChange = {},
            label = { Text(label) },
            readOnly = true,
            enabled = false,
            trailingIcon = { Icon(Icons.Filled.ArrowDropDown, contentDescription = null) },
            modifier = Modifier
                .fillMaxWidth()
                .clickable(enabled = enabled && options.isNotEmpty()) { expanded = true },
        )
        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            options.forEach { option ->
                DropdownMenuItem(
                    text = { Text(option.title) },
                    onClick = { expanded = false; onPick(option) },
                )
            }
        }
    }
}
