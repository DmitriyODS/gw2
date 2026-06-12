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
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
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
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.R
import com.kodass.groovework.data.api.MetaApi
import com.kodass.groovework.data.dto.ChangelogVersionDto
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.ui.common.UserAvatar
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json

class AboutViewModel(
    private val session: SessionManager,
    private val metaApi: MetaApi,
    private val json: Json,
) : ViewModel() {
    var versions by mutableStateOf<List<ChangelogVersionDto>>(emptyList())
        private set
    var changelogError by mutableStateOf(false)
        private set
    var refreshing by mutableStateOf(false)
        private set
    var loggingOut by mutableStateOf(false)
        private set

    init {
        viewModelScope.launch { session.loadMe() }
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
                session.loadMe()
                loadChangelog()
            } finally {
                refreshing = false
            }
        }
    }

    fun logout() {
        if (loggingOut) return
        viewModelScope.launch {
            loggingOut = true
            try {
                session.logout()
            } finally {
                loggingOut = false
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AboutScreen(container: AppContainer) {
    val viewModel: AboutViewModel = viewModel {
        AboutViewModel(container.sessionManager, container.metaApi, container.json)
    }
    val context = LocalContext.current
    val appVersion = remember(context) {
        runCatching {
            context.packageManager.getPackageInfo(context.packageName, 0).versionName
        }.getOrNull() ?: "1.0"
    }

    Scaffold(
        topBar = { TopAppBar(title = { Text("О приложении") }) },
    ) { padding ->
        PullToRefreshBox(
            isRefreshing = viewModel.refreshing,
            onRefresh = { viewModel.pullRefresh() },
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            AboutContent(viewModel = viewModel, container = container, appVersion = appVersion)
        }
    }
}

@Composable
private fun AboutContent(viewModel: AboutViewModel, container: AppContainer, appVersion: String) {
    val me by container.sessionManager.me.collectAsStateWithLifecycle()
    val serverUrl by container.sessionManager.serverUrl.collectAsStateWithLifecycle()
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
                        text = "Платформа учёта времени, задач и общения команды",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 4.dp),
                    )
                }
            }
            item {
                Card(
                    colors = CardDefaults.cardColors(
                        containerColor = MaterialTheme.colorScheme.surfaceContainerLow,
                    ),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            UserAvatar(
                                userId = me?.id,
                                name = me?.fio,
                                avatarPath = me?.avatarPath,
                                size = 52.dp,
                            )
                            Column(modifier = Modifier.padding(start = 12.dp)) {
                                Text(
                                    text = me?.fio ?: "…",
                                    style = MaterialTheme.typography.titleMedium,
                                )
                                val subtitle = listOfNotNull(me?.role?.name, me?.post?.takeIf { it.isNotBlank() })
                                    .joinToString(" · ")
                                if (subtitle.isNotEmpty()) {
                                    Text(
                                        text = subtitle,
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                }
                            }
                        }
                        Text(
                            text = "Сервер: $serverUrl",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.outline,
                            modifier = Modifier.padding(top = 12.dp),
                        )
                        OutlinedButton(
                            onClick = { viewModel.logout() },
                            enabled = !viewModel.loggingOut,
                            modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                        ) {
                            Icon(
                                Icons.AutoMirrored.Filled.Logout,
                                contentDescription = null,
                                modifier = Modifier.size(18.dp),
                            )
                            Text("Выйти из аккаунта", modifier = Modifier.padding(start = 8.dp))
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

@Composable
private fun ChangelogCard(version: ChangelogVersionDto) {
    Card(
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerLow,
        ),
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
