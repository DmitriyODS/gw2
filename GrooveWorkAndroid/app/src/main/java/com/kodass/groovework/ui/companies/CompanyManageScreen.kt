package com.kodass.groovework.ui.companies

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
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.ArrowDropDown
import androidx.compose.material.icons.filled.AutoAwesome
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.LockReset
import androidx.compose.material.icons.filled.PersonAdd
import androidx.compose.material.icons.filled.Pets
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Sync
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.PrimaryTabRow
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.api.CompaniesApi
import com.kodass.groovework.data.dto.AddMemberRequest
import com.kodass.groovework.data.dto.CompanyDto
import com.kodass.groovework.data.dto.CreateCompanyUserRequest
import com.kodass.groovework.data.dto.CreateInviteRequest
import com.kodass.groovework.data.dto.RoleIdRequest
import com.kodass.groovework.data.dto.RoleRef
import com.kodass.groovework.data.dto.UserDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.ConfirmDialog
import com.kodass.groovework.ui.common.ConfirmSpec
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.UserAvatar
import com.kodass.groovework.ui.common.generatePassword
import com.kodass.groovework.ui.common.rememberClipboardCopy
import com.kodass.groovework.ui.settings.SwitchRow
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonObject

data class CompanyFlags(
    val usesStages: Boolean = false,
    val usesYougile: Boolean = false,
    val usesCalls: Boolean = true,
)

class CompanyManageViewModel(
    private val api: CompaniesApi,
    private val session: SessionManager,
    private val json: Json,
    val companyId: Long,
) : ViewModel() {
    var company by mutableStateOf<CompanyDto?>(null)
        private set
    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var members by mutableStateOf<List<UserDto>>(emptyList())
        private set
    var roles by mutableStateOf<List<RoleRef>>(emptyList())
        private set
    var inviteCode by mutableStateOf<String?>(null)
        private set

    var candidates by mutableStateOf<List<UserDto>>(emptyList())
        private set
    var message by mutableStateOf<String?>(null)
    var busy by mutableStateOf(false)
        private set

    // Фич-флаги компании (settings.uses_*); дефолты как на бэке: звонки on, остальное off.
    var flags by mutableStateOf(CompanyFlags())
        private set

    private val myUserId get() = session.claimsOrNull()?.userId
    val isCreator: Boolean get() = company?.createdBy != null && company?.createdBy == myUserId
    val canManageMembers: Boolean get() = isCreator || session.claimsOrNull()?.isRootAdmin == true

    private var candJob: Job? = null

    init {
        load()
        viewModelScope.launch { roles = runCatching { apiCall(json) { api.roles() } }.getOrDefault(emptyList()) }
    }

    fun load() {
        viewModelScope.launch {
            if (company == null) loading = true
            error = null
            try {
                company = apiCall(json) { api.company(companyId) }.also { flags = readFlags(it.settings) }
                members = runCatching { apiCall(json) { api.members(companyId) } }.getOrDefault(emptyList())
                inviteCode = runCatching { apiCall(json) { api.invite(companyId) }.code.ifBlank { null } }.getOrNull()
            } catch (e: ApiException) {
                if (company == null) error = e.message else message = e.message
            } finally {
                loading = false
            }
        }
    }

    fun searchCandidates(query: String) {
        candJob?.cancel()
        if (query.isBlank()) { candidates = emptyList(); return }
        candJob = viewModelScope.launch {
            delay(300)
            candidates = runCatching { apiCall(json) { api.candidates(companyId, query.trim()) } }.getOrDefault(emptyList())
        }
    }

    private fun employeeRoleId(): Long = roles.firstOrNull { it.level == 1 }?.id ?: 1L

    private fun mutate(block: suspend () -> Unit, success: String) {
        viewModelScope.launch {
            busy = true
            try {
                block()
                message = success
                load()
            } catch (e: ApiException) {
                message = e.message
            } finally {
                busy = false
            }
        }
    }

    fun addExisting(userId: Long) {
        candidates = emptyList()
        mutate({ apiCall(json) { api.addMember(companyId, AddMemberRequest(userId, employeeRoleId())) } }, "Сотрудник добавлен")
    }

    fun changeRole(userId: Long, roleId: Long) =
        mutate({ apiCall(json) { api.setMemberRole(companyId, userId, RoleIdRequest(roleId)) } }, "Роль обновлена")

    fun removeMember(userId: Long) =
        mutate({ apiCall(json) { api.removeMember(companyId, userId) } }, "Сотрудник убран")

    fun resetPassword(userId: Long) {
        viewModelScope.launch {
            busy = true
            try {
                apiCall(json) { api.resetCompanyMemberPassword(companyId, userId) }
                message = "Пароль сброшен"
            } catch (e: ApiException) {
                message = e.message
            } finally {
                busy = false
            }
        }
    }

    fun createEmployee(req: CreateCompanyUserRequest, onDone: () -> Unit) {
        viewModelScope.launch {
            busy = true
            try {
                apiCall(json) { api.createCompanyUser(companyId, req) }
                message = "Сотрудник создан"
                onDone()
                load()
            } catch (e: ApiException) {
                message = e.message
            } finally {
                busy = false
            }
        }
    }

    fun sendInvite(email: String, roleId: Long, onDone: () -> Unit) {
        viewModelScope.launch {
            busy = true
            try {
                apiCall(json) { api.createInvite(companyId, CreateInviteRequest(email.trim(), roleId)) }
                message = "Приглашение отправлено"
                onDone()
            } catch (e: ApiException) {
                message = e.message
            } finally {
                busy = false
            }
        }
    }

    fun regenerateInvite() {
        viewModelScope.launch {
            busy = true
            try {
                inviteCode = apiCall(json) { api.regenerateInvite(companyId) }.code
                message = "Ссылка обновлена"
            } catch (e: ApiException) {
                message = e.message
            } finally {
                busy = false
            }
        }
    }

    // Сохраняет фич-флаги (PATCH settings — бэк мёрджит, weekend_days/uses_groove не трогаем).
    // Оптимистично, с откатом при ошибке.
    fun saveFlags(new: CompanyFlags) {
        val prev = flags
        flags = new
        viewModelScope.launch {
            busy = true
            try {
                val body = buildJsonObject {
                    putJsonObject("settings") {
                        put("uses_stages", new.usesStages)
                        put("uses_yougile", new.usesYougile)
                        put("uses_calls", new.usesCalls)
                    }
                }
                company = apiCall(json) { api.update(companyId, body) }.also { flags = readFlags(it.settings) }
            } catch (e: ApiException) {
                flags = prev
                message = e.message
            } finally {
                busy = false
            }
        }
    }

    private fun readFlags(settings: JsonObject?): CompanyFlags = CompanyFlags(
        usesStages = settings.boolFlag("uses_stages", false),
        usesYougile = settings.boolFlag("uses_yougile", false),
        usesCalls = settings.boolFlag("uses_calls", true),
    )

    private fun JsonObject?.boolFlag(key: String, default: Boolean): Boolean =
        (this?.get(key) as? JsonPrimitive)?.booleanOrNull ?: default

    fun deleteCompany(onDeleted: () -> Unit) {
        viewModelScope.launch {
            busy = true
            try {
                apiCall(json) { api.delete(companyId) }
                onDeleted()
            } catch (e: ApiException) {
                message = e.message
            } finally {
                busy = false
            }
        }
    }

    fun consumeMessage() { message = null }
}

private val tabTitles = listOf("Обзор", "Участники", "Настройки", "Опасная зона")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CompanyManageScreen(
    container: AppContainer,
    companyId: Long,
    onBack: () -> Unit,
    onOpenSettings: (String) -> Unit,
) {
    val viewModel: CompanyManageViewModel = viewModel {
        CompanyManageViewModel(container.companiesApi, container.sessionManager, container.json, companyId)
    }
    val context = LocalContext.current
    val authState by container.sessionManager.authState.collectAsStateWithLifecycle()
    val activeCompanyId = (authState as? AuthState.LoggedIn)?.claims?.companyId
    var tab by remember { mutableIntStateOf(0) }

    LaunchedEffect(viewModel.message) {
        viewModel.message?.let {
            Toast.makeText(context, it, Toast.LENGTH_SHORT).show()
            viewModel.consumeMessage()
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(viewModel.company?.name ?: "Компания") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Назад")
                    }
                },
            )
        },
    ) { padding ->
        when {
            viewModel.loading && viewModel.company == null -> CenteredLoading(Modifier.padding(padding))
            viewModel.error != null && viewModel.company == null ->
                ErrorState(viewModel.error ?: "", onRetry = { viewModel.load() }, modifier = Modifier.padding(padding))
            else -> {
                val company = viewModel.company ?: return@Scaffold
                Column(modifier = Modifier.fillMaxSize().padding(padding)) {
                    PrimaryTabRow(selectedTabIndex = tab) {
                        tabTitles.forEachIndexed { i, title ->
                            Tab(selected = tab == i, onClick = { tab = i }, text = { Text(title) })
                        }
                    }
                    when (tab) {
                        0 -> OverviewTab(viewModel, company, container)
                        1 -> MembersTab(viewModel, container)
                        2 -> SettingsTab(
                            companyId = companyId,
                            isActiveCompany = activeCompanyId == companyId,
                            flags = viewModel.flags,
                            busy = viewModel.busy,
                            onFlagsChange = viewModel::saveFlags,
                            onOpenSettings = onOpenSettings,
                            onSwitch = { container.appScope.launch { runCatching { container.sessionManager.switchCompany(companyId) } } },
                        )
                        else -> DangerTab(viewModel, company, onDeleted = onBack)
                    }
                }
            }
        }
    }
}

@Composable
private fun OverviewTab(viewModel: CompanyManageViewModel, company: CompanyDto, container: AppContainer) {
    val context = LocalContext.current
    val copyToClipboard = rememberClipboardCopy()
    val serverUrl by container.sessionManager.serverUrl.collectAsStateWithLifecycle()
    val link = viewModel.inviteCode?.let { "${serverUrl.trimEnd('/')}/join/$it" }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item {
            Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                Row(modifier = Modifier.fillMaxWidth().padding(16.dp), horizontalArrangement = Arrangement.SpaceAround) {
                    StatBlock(company.employeesCount, "сотрудников")
                    StatBlock(company.tasksCount, "задач")
                }
            }
        }
        company.description?.takeIf { it.isNotBlank() }?.let { desc ->
            item {
                Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                    Text(desc, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.padding(16.dp))
                }
            }
        }
        if (viewModel.canManageMembers) {
            item {
                Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text("Ссылка-приглашение", style = MaterialTheme.typography.titleSmall)
                        Text(
                            "Любой по этой ссылке вступит в компанию сотрудником. Перевыпуск делает старую недействительной.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier.padding(top = 4.dp),
                        )
                        Text(
                            link ?: "Ссылка ещё не создана",
                            style = MaterialTheme.typography.bodyMedium,
                            modifier = Modifier.padding(top = 8.dp),
                        )
                        Row(modifier = Modifier.padding(top = 8.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            if (link != null) {
                                OutlinedButton(onClick = {
                                    copyToClipboard(link)
                                    Toast.makeText(context, "Скопировано", Toast.LENGTH_SHORT).show()
                                }) {
                                    Icon(Icons.Filled.ContentCopy, contentDescription = null, modifier = Modifier.size(18.dp))
                                    Text("Копировать", modifier = Modifier.padding(start = 6.dp))
                                }
                            }
                            Button(onClick = { viewModel.regenerateInvite() }, enabled = !viewModel.busy) {
                                Icon(Icons.Filled.Refresh, contentDescription = null, modifier = Modifier.size(18.dp))
                                Text(if (viewModel.inviteCode == null) "Создать" else "Перевыпустить", modifier = Modifier.padding(start = 6.dp))
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun StatBlock(value: Int, label: String) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Text("$value", style = MaterialTheme.typography.headlineSmall, color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.Bold)
        Text(label, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@Composable
private fun MembersTab(viewModel: CompanyManageViewModel, container: AppContainer) {
    var showCreate by remember { mutableStateOf(false) }
    var candQuery by remember { mutableStateOf("") }
    var inviteEmail by remember { mutableStateOf("") }
    var inviteRole by remember { mutableStateOf<RoleRef?>(null) }
    var confirm by remember { mutableStateOf<ConfirmSpec?>(null) }

    confirm?.let { ConfirmDialog(it, onDismiss = { confirm = null }) }
    if (showCreate) {
        CreateEmployeeDialog(viewModel, onDismiss = { showCreate = false })
    }

    val manage = viewModel.canManageMembers
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        if (!manage) {
            item {
                Text(
                    "Управлять участниками может только создатель компании. Вам доступен просмотр и настройки.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
        items(viewModel.members, key = { it.id }) { member ->
            MemberRow(
                member = member,
                roles = viewModel.roles,
                canManage = manage,
                busy = viewModel.busy,
                onChangeRole = { roleId -> viewModel.changeRole(member.id, roleId) },
                onResetPassword = {
                    confirm = ConfirmSpec(
                        title = "Сбросить пароль",
                        text = "Пароль ${member.fio} станет временным (логин + 123). Продолжить?",
                        confirmLabel = "Сбросить",
                        destructive = false,
                        action = { viewModel.resetPassword(member.id) },
                    )
                },
                onRemove = {
                    confirm = ConfirmSpec(
                        title = "Убрать из компании",
                        text = "Убрать ${member.fio} из компании?",
                        confirmLabel = "Убрать",
                        action = { viewModel.removeMember(member.id) },
                    )
                },
            )
        }

        if (manage) {
            item {
                Button(onClick = { showCreate = true }, modifier = Modifier.fillMaxWidth().padding(top = 8.dp)) {
                    Icon(Icons.Filled.PersonAdd, contentDescription = null, modifier = Modifier.size(18.dp))
                    Text("Создать сотрудника", modifier = Modifier.padding(start = 8.dp))
                }
            }
            item {
                Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                    Column(modifier = Modifier.padding(12.dp)) {
                        Text("Добавить существующего", style = MaterialTheme.typography.titleSmall)
                        OutlinedTextField(
                            value = candQuery,
                            onValueChange = { candQuery = it; viewModel.searchCandidates(it) },
                            placeholder = { Text("Имя или логин…") },
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                        )
                        viewModel.candidates.forEach { c ->
                            Row(
                                verticalAlignment = Alignment.CenterVertically,
                                modifier = Modifier.fillMaxWidth().clickable {
                                    candQuery = ""; viewModel.addExisting(c.id)
                                }.padding(vertical = 8.dp),
                            ) {
                                UserAvatar(userId = c.id, name = c.fio, avatarPath = c.avatarPath, size = 36.dp)
                                Column(modifier = Modifier.weight(1f).padding(start = 10.dp)) {
                                    Text(c.fio, style = MaterialTheme.typography.bodyMedium)
                                    Text("@${c.login ?: ""}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                }
                                Icon(Icons.Filled.Add, contentDescription = "Добавить")
                            }
                        }
                    }
                }
            }
            item {
                Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                    Column(modifier = Modifier.padding(12.dp)) {
                        Text("Пригласить по email", style = MaterialTheme.typography.titleSmall)
                        OutlinedTextField(
                            value = inviteEmail,
                            onValueChange = { inviteEmail = it },
                            placeholder = { Text("name@example.com") },
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                        )
                        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 8.dp)) {
                            RoleSelector(
                                roles = viewModel.roles,
                                selected = inviteRole ?: viewModel.roles.firstOrNull { it.level == 1 },
                                onSelect = { inviteRole = it },
                                modifier = Modifier.weight(1f),
                            )
                            Button(
                                onClick = {
                                    val role = inviteRole ?: viewModel.roles.firstOrNull { it.level == 1 }
                                    if (role != null && inviteEmail.isNotBlank()) {
                                        viewModel.sendInvite(inviteEmail, role.id) { inviteEmail = "" }
                                    }
                                },
                                enabled = !viewModel.busy && inviteEmail.isNotBlank(),
                                modifier = Modifier.padding(start = 8.dp),
                            ) {
                                Icon(Icons.AutoMirrored.Filled.Send, contentDescription = null, modifier = Modifier.size(18.dp))
                                Text("Отправить", modifier = Modifier.padding(start = 6.dp))
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun MemberRow(
    member: UserDto,
    roles: List<RoleRef>,
    canManage: Boolean,
    busy: Boolean,
    onChangeRole: (Long) -> Unit,
    onResetPassword: () -> Unit,
    onRemove: () -> Unit,
) {
    Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(12.dp)) {
            UserAvatar(userId = member.id, name = member.fio, avatarPath = member.avatarPath, size = 44.dp)
            Column(modifier = Modifier.weight(1f).padding(horizontal = 10.dp)) {
                Text(member.fio, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold)
                Text(
                    "@${member.login ?: ""}" + (member.post?.takeIf { it.isNotBlank() }?.let { " · $it" } ?: ""),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            if (canManage) {
                RoleSelector(
                    roles = roles,
                    selected = member.role?.let { r -> roles.firstOrNull { it.id == r.id } ?: RoleRef(r.id, r.name, r.level) },
                    onSelect = { if (it.id != member.role?.id) onChangeRole(it.id) },
                    enabled = !busy,
                )
                IconButton(onClick = onResetPassword, enabled = !busy) {
                    Icon(Icons.Filled.LockReset, contentDescription = "Сбросить пароль")
                }
                IconButton(onClick = onRemove, enabled = !busy) {
                    Icon(Icons.Filled.Delete, contentDescription = "Убрать", tint = MaterialTheme.colorScheme.error)
                }
            } else {
                member.role?.let { RoleBadge(it.name, it.level >= 3) }
            }
        }
    }
}

// Селектор роли: DropdownMenu поверх текста (ExposedDropdownMenu недоступен в alpha-BOM).
@Composable
private fun RoleSelector(
    roles: List<RoleRef>,
    selected: RoleRef?,
    onSelect: (RoleRef) -> Unit,
    enabled: Boolean = true,
    modifier: Modifier = Modifier,
) {
    var expanded by remember { mutableStateOf(false) }
    val options = roles.filter { it.level in 1..3 }.ifEmpty { listOfNotNull(selected) }
    Box(modifier = modifier) {
        OutlinedButton(onClick = { if (enabled) expanded = true }, enabled = enabled) {
            Text(selected?.name ?: "Роль")
            Icon(Icons.Filled.ArrowDropDown, contentDescription = null)
        }
        DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            options.forEach { role ->
                DropdownMenuItem(text = { Text(role.name) }, onClick = { expanded = false; onSelect(role) })
            }
        }
    }
}

@Composable
private fun CreateEmployeeDialog(viewModel: CompanyManageViewModel, onDismiss: () -> Unit) {
    var fio by remember { mutableStateOf("") }
    var login by remember { mutableStateOf("") }
    var post by remember { mutableStateOf("") }
    var email by remember { mutableStateOf("") }
    var role by remember { mutableStateOf<RoleRef?>(null) }
    val password = remember { generatePassword() }

    AlertDialog(
        onDismissRequest = { if (!viewModel.busy) onDismiss() },
        title = { Text("Новый сотрудник") },
        text = {
            Column {
                OutlinedTextField(value = fio, onValueChange = { fio = it }, label = { Text("ФИО") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(value = login, onValueChange = { login = it }, label = { Text("Логин") }, singleLine = true, modifier = Modifier.fillMaxWidth().padding(top = 8.dp))
                OutlinedTextField(value = post, onValueChange = { post = it }, label = { Text("Должность (необязательно)") }, singleLine = true, modifier = Modifier.fillMaxWidth().padding(top = 8.dp))
                OutlinedTextField(value = email, onValueChange = { email = it }, label = { Text("Email (необязательно)") }, singleLine = true, modifier = Modifier.fillMaxWidth().padding(top = 8.dp))
                Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 8.dp)) {
                    Text("Роль:", style = MaterialTheme.typography.bodyMedium)
                    RoleSelector(
                        roles = viewModel.roles,
                        selected = role ?: viewModel.roles.firstOrNull { it.level == 1 },
                        onSelect = { role = it },
                        modifier = Modifier.padding(start = 8.dp),
                    )
                }
                Text(
                    "Временный пароль: $password (сотрудник сменит при первом входе).",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 8.dp),
                )
            }
        },
        confirmButton = {
            TextButton(
                onClick = {
                    val r = role ?: viewModel.roles.firstOrNull { it.level == 1 }
                    if (fio.isNotBlank() && login.isNotBlank() && r != null) {
                        viewModel.createEmployee(
                            CreateCompanyUserRequest(
                                fio = fio.trim(), login = login.trim(),
                                post = post.trim().ifBlank { null }, roleId = r.id,
                                email = email.trim().ifBlank { null }, password = password,
                            ),
                            onDone = onDismiss,
                        )
                    }
                },
                enabled = !viewModel.busy,
            ) { Text(if (viewModel.busy) "Создаю…" else "Создать") }
        },
        dismissButton = { TextButton(onClick = onDismiss, enabled = !viewModel.busy) { Text("Отмена") } },
    )
}

@Composable
private fun SettingsTab(
    companyId: Long,
    isActiveCompany: Boolean,
    flags: CompanyFlags,
    busy: Boolean,
    onFlagsChange: (CompanyFlags) -> Unit,
    onOpenSettings: (String) -> Unit,
    onSwitch: () -> Unit,
) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        item {
            Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Возможности", style = MaterialTheme.typography.titleSmall)
                    SwitchRow(
                        title = "Этапы задач",
                        subtitle = "Канбан-режим и теги этапов",
                        checked = flags.usesStages,
                        enabled = !busy,
                        onChange = { onFlagsChange(flags.copy(usesStages = it)) },
                    )
                    SwitchRow(
                        title = "Интеграция с YouGile",
                        subtitle = "Импорт и экспорт карточек",
                        checked = flags.usesYougile,
                        enabled = !busy,
                        onChange = { onFlagsChange(flags.copy(usesYougile = it)) },
                    )
                    SwitchRow(
                        title = "Аудио- и видеозвонки",
                        subtitle = "Кнопки звонка в мессенджере",
                        checked = flags.usesCalls,
                        enabled = !busy,
                        onChange = { onFlagsChange(flags.copy(usesCalls = it)) },
                    )
                }
            }
        }
        item { SettingNavRow(Icons.Filled.CalendarMonth, "Выходные дни", "Когда Грувик отдыхает") { onOpenSettings("settings/weekends?companyId=$companyId") } }
        item { SettingNavRow(Icons.Filled.Pets, "Мой Groove", "Геймификация и питомцы") { onOpenSettings("settings/groove?companyId=$companyId") } }
        if (isActiveCompany) {
            item { SettingNavRow(Icons.Filled.AutoAwesome, "Нейро-функции", "ИИ через ProxyAPI") { onOpenSettings("settings/ai") } }
            item { SettingNavRow(Icons.Filled.Sync, "Интеграция YouGile", "Синхронизация задач компании") { onOpenSettings("settings/yougile-company") } }
        } else {
            item {
                Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow)) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text(
                            "Нейро-функции и YouGile настраиваются для активной компании. Переключитесь на эту компанию, чтобы открыть их.",
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        OutlinedButton(onClick = onSwitch, modifier = Modifier.padding(top = 12.dp)) {
                            Text("Переключиться на эту компанию")
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun SettingNavRow(icon: androidx.compose.ui.graphics.vector.ImageVector, title: String, subtitle: String, onClick: () -> Unit) {
    Card(
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
        modifier = Modifier.fillMaxWidth().clickable(onClick = onClick),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(14.dp)) {
            Icon(icon, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
            Column(modifier = Modifier.weight(1f).padding(start = 14.dp)) {
                Text(title, style = MaterialTheme.typography.titleSmall)
                Text(subtitle, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null, tint = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
private fun DangerTab(viewModel: CompanyManageViewModel, company: CompanyDto, onDeleted: () -> Unit) {
    var confirm by remember { mutableStateOf<ConfirmSpec?>(null) }
    confirm?.let { ConfirmDialog(it, onDismiss = { confirm = null }) }

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        if (!viewModel.isCreator) {
            Text(
                "Удалить компанию может только её создатель.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            return@Column
        }
        Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.errorContainer)) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("Удаление компании", style = MaterialTheme.typography.titleMedium, color = MaterialTheme.colorScheme.onErrorContainer)
                Text(
                    "Все данные компании (задачи, статистика, участники) будут удалены безвозвратно.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onErrorContainer,
                    modifier = Modifier.padding(top = 4.dp),
                )
                Button(
                    onClick = {
                        confirm = ConfirmSpec(
                            title = "Удалить компанию",
                            text = "Удалить «${company.name}» со всеми данными? Действие необратимо.",
                            confirmLabel = "Удалить",
                            action = { viewModel.deleteCompany(onDeleted) },
                        )
                    },
                    enabled = !viewModel.busy,
                    modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                ) {
                    Icon(Icons.Filled.Delete, contentDescription = null, modifier = Modifier.size(18.dp))
                    Text("Удалить компанию", modifier = Modifier.padding(start = 8.dp))
                }
            }
        }
    }
}
