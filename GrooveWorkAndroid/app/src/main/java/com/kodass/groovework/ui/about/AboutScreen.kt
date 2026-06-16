package com.kodass.groovework.ui.about

import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Dns
import androidx.compose.material.icons.filled.SupportAgent
import androidx.compose.material.icons.filled.SystemUpdate
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.R
import com.kodass.groovework.data.api.MetaApi
import com.kodass.groovework.data.dto.ChangelogVersionDto
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.data.network.normalizeServerUrl
import com.kodass.groovework.data.repo.MessengerRepository
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.update.AppUpdater
import com.kodass.groovework.data.update.UpdateState
import com.kodass.groovework.ui.common.ConfirmDialog
import com.kodass.groovework.ui.common.ConfirmSpec
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json

class AboutViewModel(
    private val session: SessionManager,
    private val messengerRepo: MessengerRepository,
    private val metaApi: MetaApi,
    private val json: Json,
) : ViewModel() {
    var versions by mutableStateOf<List<ChangelogVersionDto>>(emptyList())
        private set
    var changelogError by mutableStateOf(false)
        private set
    var refreshing by mutableStateOf(false)
        private set
    var openingSupport by mutableStateOf(false)
        private set
    var changingServer by mutableStateOf(false)
        private set

    init {
        viewModelScope.launch { loadChangelog() }
    }

    private suspend fun loadChangelog() {
        try {
            versions = apiCall(json) { metaApi.changelog() }.versions
            changelogError = false
        } catch (_: Exception) {
            changelogError = versions.isEmpty()
        }
    }

    fun pullRefresh() {
        if (refreshing) return
        viewModelScope.launch {
            refreshing = true
            try {
                loadChangelog()
            } finally {
                refreshing = false
            }
        }
    }

    fun openSupport(onOpened: (Long) -> Unit) {
        if (openingSupport) return
        viewModelScope.launch {
            openingSupport = true
            try {
                val chat = messengerRepo.openDevChat()
                runCatching { messengerRepo.refreshConversations() }
                onOpened(chat.id)
            } catch (_: Exception) {
            } finally {
                openingSupport = false
            }
        }
    }

    fun changeServer(url: String) {
        if (changingServer) return
        viewModelScope.launch {
            changingServer = true
            try {
                session.changeServer(url)
            } finally {
                changingServer = false
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AboutScreen(container: AppContainer, onOpenChat: (Long) -> Unit) {
    val viewModel: AboutViewModel = viewModel {
        AboutViewModel(container.sessionManager, container.messengerRepo, container.metaApi, container.json)
    }
    val context = LocalContext.current
    val appVersion = remember(context) {
        runCatching {
            context.packageManager.getPackageInfo(context.packageName, 0).versionName
        }.getOrNull() ?: "1.0"
    }
    val serverUrl by container.sessionManager.serverUrl.collectAsStateWithLifecycle()
    var serverInput by remember(serverUrl) { mutableStateOf(serverUrl) }
    var confirmServer by remember { mutableStateOf(false) }

    if (confirmServer) {
        ConfirmDialog(
            ConfirmSpec(
                title = "Сменить сервер",
                text = "Приложение выйдет из аккаунта и переключится на «${normalizeServerUrl(serverInput)}». Продолжить?",
                confirmLabel = "Сменить и выйти",
                destructive = true,
                action = { viewModel.changeServer(serverInput) },
            ),
            onDismiss = { confirmServer = false },
        )
    }

    Scaffold(topBar = { TopAppBar(title = { Text("О приложении") }) }) { padding ->
        PullToRefreshBox(
            isRefreshing = viewModel.refreshing,
            onRefresh = { viewModel.pullRefresh() },
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                item {
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally,
                        modifier = Modifier.fillMaxWidth().padding(vertical = 12.dp),
                    ) {
                        Image(
                            painter = painterResource(R.drawable.logo_groove),
                            contentDescription = null,
                            modifier = Modifier.size(80.dp),
                        )
                        Text(
                            text = "Groove Work",
                            style = MaterialTheme.typography.headlineSmall,
                            fontWeight = FontWeight.SemiBold,
                            modifier = Modifier.padding(top = 12.dp),
                        )
                        Text(
                            text = "Версия приложения $appVersion",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        Text(
                            text = "Сборка ${container.appUpdater.currentBuild}",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        Text(
                            text = "Платформа учёта времени, задач и общения команды",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(top = 4.dp),
                        )
                    }
                }
                item { UpdateCard(container.appUpdater) }
                item {
                    Card(
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Column(modifier = Modifier.padding(16.dp)) {
                            Button(
                                onClick = { viewModel.openSupport(onOpenChat) },
                                enabled = !viewModel.openingSupport,
                                modifier = Modifier.fillMaxWidth(),
                            ) {
                                Icon(Icons.Filled.SupportAgent, contentDescription = null, modifier = Modifier.size(18.dp))
                                Text("Чат с техподдержкой", modifier = Modifier.padding(start = 8.dp))
                            }
                        }
                    }
                }
                item {
                    Card(
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Column(modifier = Modifier.padding(16.dp)) {
                            Text("Сервер", style = MaterialTheme.typography.titleMedium)
                            Text(
                                text = "Смена адреса выполнит выход из аккаунта.",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(top = 2.dp, bottom = 8.dp),
                            )
                            OutlinedTextField(
                                value = serverInput,
                                onValueChange = { serverInput = it },
                                label = { Text("Адрес сервера") },
                                singleLine = true,
                                enabled = !viewModel.changingServer,
                                modifier = Modifier.fillMaxWidth(),
                            )
                            Button(
                                onClick = { confirmServer = true },
                                enabled = !viewModel.changingServer &&
                                    serverInput.isNotBlank() &&
                                    normalizeServerUrl(serverInput) != serverUrl,
                                modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                            ) {
                                Icon(Icons.Filled.Dns, contentDescription = null, modifier = Modifier.size(18.dp))
                                Text(
                                    if (viewModel.changingServer) "Меняю…" else "Сменить сервер",
                                    modifier = Modifier.padding(start = 8.dp),
                                )
                            }
                        }
                    }
                }
                item {
                    Text(
                        text = "Что нового",
                        style = MaterialTheme.typography.titleMedium,
                        modifier = Modifier.padding(top = 8.dp),
                    )
                    if (viewModel.changelogError) {
                        Text(
                            text = "Не удалось загрузить список изменений",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(top = 8.dp),
                        )
                    }
                }
                items(viewModel.versions, key = { it.version }) { version ->
                    ChangelogCard(version)
                }
            }
        }
    }
}

@Composable
private fun UpdateCard(updater: AppUpdater) {
    val state by updater.state.collectAsStateWithLifecycle()
    Card(
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text("Обновления", style = MaterialTheme.typography.titleMedium)

            val status: String? = when (val s = state) {
                is UpdateState.UpToDate -> "Установлена последняя версия"
                is UpdateState.Available -> "Доступна новая сборка ${s.build}"
                is UpdateState.ReadyToInstall -> "Обновление загружено — нажмите, чтобы установить"
                is UpdateState.Failed -> s.message
                else -> null
            }
            status?.let {
                Text(
                    text = it,
                    style = MaterialTheme.typography.bodySmall,
                    color = if (state is UpdateState.Failed) MaterialTheme.colorScheme.error
                    else MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 2.dp, bottom = 8.dp),
                )
            }

            val downloading = state as? UpdateState.Downloading
            if (downloading != null) {
                val progress = downloading.progress
                Text(
                    text = if (progress >= 0f) "Скачивание… ${(progress * 100).toInt()}%" else "Скачивание…",
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.padding(top = 8.dp),
                )
                if (progress >= 0f) {
                    LinearProgressIndicator(
                        progress = { progress },
                        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                    )
                } else {
                    LinearProgressIndicator(modifier = Modifier.fillMaxWidth().padding(top = 8.dp))
                }
            } else {
                Button(
                    onClick = {
                        when (state) {
                            is UpdateState.Available -> updater.download()
                            is UpdateState.ReadyToInstall -> updater.install()
                            else -> updater.check()
                        }
                    },
                    enabled = state !is UpdateState.Checking,
                    modifier = Modifier.fillMaxWidth().padding(top = if (status != null) 0.dp else 8.dp),
                ) {
                    when (state) {
                        is UpdateState.Checking -> {
                            CircularProgressIndicator(
                                strokeWidth = 2.dp,
                                color = MaterialTheme.colorScheme.onPrimary,
                                modifier = Modifier.size(18.dp),
                            )
                            Text("Проверяю…", modifier = Modifier.padding(start = 8.dp))
                        }
                        is UpdateState.Available -> {
                            Icon(Icons.Filled.SystemUpdate, contentDescription = null, modifier = Modifier.size(18.dp))
                            Text("Скачать обновление", modifier = Modifier.padding(start = 8.dp))
                        }
                        is UpdateState.ReadyToInstall -> {
                            Icon(Icons.Filled.CheckCircle, contentDescription = null, modifier = Modifier.size(18.dp))
                            Text("Установить обновление", modifier = Modifier.padding(start = 8.dp))
                        }
                        else -> {
                            Icon(Icons.Filled.SystemUpdate, contentDescription = null, modifier = Modifier.size(18.dp))
                            Text("Проверка обновлений", modifier = Modifier.padding(start = 8.dp))
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ChangelogCard(version: ChangelogVersionDto) {
    Card(
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = version.version,
                    style = MaterialTheme.typography.titleSmall,
                    color = MaterialTheme.colorScheme.primary,
                    fontWeight = FontWeight.Bold,
                )
                version.date?.let { date ->
                    Text(
                        text = date,
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.outline,
                        modifier = Modifier.padding(start = 8.dp),
                    )
                }
            }
            version.title?.takeIf { it.isNotBlank() }?.let { title ->
                Text(
                    text = title,
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.padding(top = 4.dp),
                )
            }
            version.description?.takeIf { it.isNotBlank() }?.let { description ->
                Text(
                    text = description,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 6.dp),
                )
            }
            ChangelogSection("Новое", version.added)
            ChangelogSection("Улучшено", version.improved)
            ChangelogSection("Исправлено", version.fixed)
        }
    }
}

@Composable
private fun ChangelogSection(title: String, items: List<String>) {
    if (items.isEmpty()) return
    Text(
        text = title,
        style = MaterialTheme.typography.labelLarge,
        color = MaterialTheme.colorScheme.primary,
        modifier = Modifier.padding(top = 10.dp),
    )
    items.forEach { item ->
        Text(
            text = "•  $item",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(top = 4.dp),
        )
    }
}
