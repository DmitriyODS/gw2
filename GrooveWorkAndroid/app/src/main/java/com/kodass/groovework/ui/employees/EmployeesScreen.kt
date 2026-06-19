package com.kodass.groovework.ui.employees

import android.Manifest
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
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
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
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
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.api.AuthApi
import com.kodass.groovework.data.dto.UserDto
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.data.repo.MessengerRepository
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.EmptyState
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.RefreshOnResume
import com.kodass.groovework.ui.common.SearchField
import com.kodass.groovework.ui.common.UserAvatar
import com.kodass.groovework.ui.common.UserInfoSheet
import com.kodass.groovework.ui.common.formatLastSeen
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json

class EmployeesViewModel(
    private val authApi: AuthApi,
    private val messengerRepo: MessengerRepository,
    private val json: Json,
) : ViewModel() {
    var users by mutableStateOf<List<UserDto>>(emptyList())
        private set
    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var search by mutableStateOf("")
    var roleFilter by mutableStateOf<Int?>(null)

    init {
        load()
        viewModelScope.launch { runCatching { messengerRepo.refreshPresence() } }
    }

    fun load() {
        viewModelScope.launch {
            if (users.isEmpty()) loading = true
            error = null
            try {
                users = apiCall(json) { authApi.directory() }
            } catch (e: com.kodass.groovework.data.network.ApiException) {
                if (users.isEmpty()) error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun reload() {
        load()
        viewModelScope.launch { runCatching { messengerRepo.refreshPresence() } }
    }

    fun filtered(): List<UserDto> {
        val q = search.trim().lowercase()
        return users.filter { u ->
            (roleFilter == null || u.role?.level == roleFilter) &&
                (q.isEmpty() ||
                    u.fio.lowercase().contains(q) ||
                    (u.login ?: "").lowercase().contains(q) ||
                    (u.post ?: "").lowercase().contains(q))
        }
    }

    // Уровни ролей, присутствующие в списке, с количеством — для чипов-фильтров.
    fun roleCounts(): List<Pair<Int, Int>> =
        users.mapNotNull { it.role?.level }.groupingBy { it }.eachCount()
            .toList().sortedBy { it.first }
}

private val roleNames = mapOf(
    1 to "Сотрудники", 2 to "Менеджеры", 3 to "Администраторы",
)

@OptIn(androidx.compose.material3.ExperimentalMaterial3Api::class)
@Composable
fun EmployeesScreen(container: AppContainer, onOpenChat: (Long) -> Unit) {
    val viewModel: EmployeesViewModel = viewModel {
        EmployeesViewModel(container.authApi, container.messengerRepo, container.json)
    }
    RefreshOnResume { viewModel.reload() }
    val online by container.messengerRepo.onlineUsers.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()

    var selected by remember { mutableStateOf<UserDto?>(null) }

    // Звонок сотруднику — после выдачи разрешений.
    var pendingCall by remember { mutableStateOf<Pair<Long, Boolean>?>(null) }
    val callPermLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { result ->
        val (uid, video) = pendingCall ?: return@rememberLauncherForActivityResult
        pendingCall = null
        val micOk = result[Manifest.permission.RECORD_AUDIO] == true
        val camOk = !video || result[Manifest.permission.CAMERA] == true
        if (micOk && camOk) container.callController.startCall(uid, video)
    }
    val requestCall: (Long, Boolean) -> Unit = { uid, video ->
        pendingCall = uid to video
        callPermLauncher.launch(
            if (video) arrayOf(Manifest.permission.RECORD_AUDIO, Manifest.permission.CAMERA)
            else arrayOf(Manifest.permission.RECORD_AUDIO)
        )
    }

    Scaffold(topBar = { TopAppBar(title = { Text("Сотрудники") }) }) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding)) {
            SearchField(
                value = viewModel.search,
                onValueChange = { viewModel.search = it },
                placeholder = "Поиск по ФИО, логину, должности",
            )

            val counts = viewModel.roleCounts()
            if (counts.size > 1) {
                Row(
                    modifier = Modifier.fillMaxWidth().horizontalScroll(rememberScrollState())
                        .padding(horizontal = 16.dp, vertical = 4.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    FilterChip(
                        selected = viewModel.roleFilter == null,
                        onClick = { viewModel.roleFilter = null },
                        label = { Text("Все (${viewModel.users.size})") },
                    )
                    counts.forEach { (level, count) ->
                        FilterChip(
                            selected = viewModel.roleFilter == level,
                            onClick = { viewModel.roleFilter = if (viewModel.roleFilter == level) null else level },
                            label = { Text("${roleNames[level] ?: "Ур. $level"} ($count)") },
                        )
                    }
                }
            }

            val list = viewModel.filtered()
            when {
                viewModel.loading && viewModel.users.isEmpty() -> CenteredLoading()
                viewModel.error != null && viewModel.users.isEmpty() ->
                    ErrorState(viewModel.error ?: "", onRetry = { viewModel.load() })
                list.isEmpty() -> EmptyState(
                    title = if (viewModel.search.isNotBlank()) "Никого не нашли" else "Сотрудников нет",
                    subtitle = if (viewModel.search.isNotBlank()) "Попробуйте изменить запрос" else null,
                )
                else -> LazyColumn(
                    contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                    modifier = Modifier.fillMaxSize(),
                ) {
                    items(list, key = { it.id }) { user ->
                        EmployeeCard(
                            user = user,
                            online = user.id in online,
                            onClick = { selected = user },
                        )
                    }
                }
            }
        }
    }

    selected?.let { user ->
        UserInfoSheet(
            container = container,
            userId = user.id,
            fallback = user,
            online = user.id in online,
            canCall = true,
            onAudioCall = { selected = null; requestCall(user.id, false) },
            onVideoCall = { selected = null; requestCall(user.id, true) },
            onWrite = {
                selected = null
                scope.launch {
                    runCatching { container.messengerRepo.openConversation(user.id) }
                        .getOrNull()?.let { onOpenChat(it.id) }
                }
            },
            onDismiss = { selected = null },
        )
    }
}

@Composable
private fun EmployeeCard(user: UserDto, online: Boolean, onClick: () -> Unit) {
    Card(
        onClick = onClick,
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(12.dp),
        ) {
            Box {
                UserAvatar(userId = user.id, name = user.fio, avatarPath = user.avatarPath, size = 48.dp)
                if (online) {
                    Box(
                        modifier = Modifier
                            .align(Alignment.BottomEnd)
                            .size(13.dp)
                            .clip(CircleShape)
                            .background(MaterialTheme.colorScheme.surfaceContainerLow)
                            .padding(2.dp)
                            .clip(CircleShape)
                            .background(MaterialTheme.colorScheme.primary),
                    )
                }
            }
            Column(modifier = Modifier.padding(start = 12.dp).weight(1f)) {
                Text(
                    user.fio,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                user.post?.takeIf { it.isNotBlank() }?.let {
                    Text(
                        it,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                Text(
                    text = if (online) "в сети" else formatLastSeen(user.lastSeenAt),
                    style = MaterialTheme.typography.labelMedium,
                    color = if (online) MaterialTheme.colorScheme.primary
                    else MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            user.role?.let { RolePill(level = it.level, name = it.name) }
        }
    }
}

@Composable
private fun RolePill(level: Int, name: String) {
    val (bg, fg) = when (level) {
        2 -> MaterialTheme.colorScheme.secondaryContainer to MaterialTheme.colorScheme.onSecondaryContainer
        3 -> MaterialTheme.colorScheme.tertiaryContainer to MaterialTheme.colorScheme.onTertiaryContainer
        4 -> MaterialTheme.colorScheme.primaryContainer to MaterialTheme.colorScheme.onPrimaryContainer
        else -> MaterialTheme.colorScheme.surfaceContainerHighest to MaterialTheme.colorScheme.onSurfaceVariant
    }
    Surface(color = bg, contentColor = fg, shape = RoundedCornerShape(8.dp)) {
        Text(
            text = name,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.SemiBold,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp),
        )
    }
}
