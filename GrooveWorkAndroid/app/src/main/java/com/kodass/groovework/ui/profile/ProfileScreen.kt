package com.kodass.groovework.ui.profile

import android.net.Uri
import android.widget.Toast
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Domain
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.PhotoCamera
import androidx.compose.material.icons.filled.SwapHoriz
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ListItem
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.graphics.Color
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.api.AuthApi
import com.kodass.groovework.data.api.StatsApi
import com.kodass.groovework.data.dto.ProfileStatsDto
import com.kodass.groovework.data.dto.UpdateMeRequest
import com.kodass.groovework.data.dto.UserDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.ui.common.RefreshOnResume
import com.kodass.groovework.ui.common.UserAvatar
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.MultipartBody
import okhttp3.RequestBody.Companion.toRequestBody
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.util.Locale

class ProfileViewModel(
    private val session: SessionManager,
    private val authApi: AuthApi,
    private val statsApi: StatsApi,
    private val messengerRepo: com.kodass.groovework.data.repo.MessengerRepository,
    private val json: Json,
) : ViewModel() {
    var fio by mutableStateOf("")
    var login by mutableStateOf("")
    var post by mutableStateOf("")
    var phone by mutableStateOf("")
    var email by mutableStateOf("")
    var profileError by mutableStateOf<String?>(null)
    var profileSaving by mutableStateOf(false)
        private set

    var currentPass by mutableStateOf("")
    var newPass by mutableStateOf("")
    var confirmPass by mutableStateOf("")
    var passwordError by mutableStateOf<String?>(null)
    var passwordSaving by mutableStateOf(false)
        private set

    var avatarBusy by mutableStateOf(false)
        private set
    var message by mutableStateOf<String?>(null)

    var stats by mutableStateOf<ProfileStatsDto?>(null)
        private set
    var switching by mutableStateOf(false)
        private set
    var loggingOut by mutableStateOf(false)
        private set

    private var formInit = false

    init {
        viewModelScope.launch { session.loadMe() }
        viewModelScope.launch {
            session.me.collect { me ->
                if (me != null && !formInit) {
                    syncForm(me)
                    formInit = true
                }
            }
        }
        loadStats()
    }

    private fun syncForm(me: UserDto) {
        fio = me.fio
        login = me.login ?: ""
        post = me.post ?: ""
        phone = me.phone ?: ""
        email = me.email ?: ""
    }

    fun saveProfile() {
        profileError = null
        if (fio.isBlank() || login.isBlank()) {
            profileError = "ФИО и логин обязательны"
            return
        }
        viewModelScope.launch {
            profileSaving = true
            try {
                apiCall(json) {
                    authApi.updateMe(
                        UpdateMeRequest(
                            fio = fio.trim(),
                            login = login.trim(),
                            post = post.trim(),
                            phone = phone.trim().ifBlank { null },
                            email = email.trim().ifBlank { null },
                        )
                    )
                }
                session.loadMe()
                message = "Профиль обновлён"
            } catch (e: ApiException) {
                profileError = e.message
            } finally {
                profileSaving = false
            }
        }
    }

    fun changePassword(onSuccess: () -> Unit = {}) {
        passwordError = null
        if (currentPass.isBlank()) {
            passwordError = "Введите текущий пароль"
            return
        }
        if (newPass.length < 8) {
            passwordError = "Пароль должен быть не короче 8 символов"
            return
        }
        if (newPass != confirmPass) {
            passwordError = "Пароли не совпадают"
            return
        }
        viewModelScope.launch {
            passwordSaving = true
            try {
                apiCall(json) {
                    authApi.updateMe(
                        UpdateMeRequest(
                            currentPassword = currentPass,
                            newPassword = newPass,
                            confirmPassword = confirmPass,
                        )
                    )
                }
                message = "Пароль изменён"
                resetPasswordForm()
                onSuccess()
            } catch (e: ApiException) {
                passwordError = e.message
            } finally {
                passwordSaving = false
            }
        }
    }

    fun resetPasswordForm() {
        currentPass = ""
        newPass = ""
        confirmPass = ""
        passwordError = null
    }

    // Загрузка уже обрезанного/сжатого изображения (из конструктора аватара).
    fun uploadAvatarBytes(bytes: ByteArray) {
        viewModelScope.launch {
            avatarBusy = true
            try {
                val body = bytes.toRequestBody("image/jpeg".toMediaTypeOrNull())
                val part = MultipartBody.Part.createFormData("file", "avatar.jpg", body)
                apiCall(json) { authApi.uploadAvatar(part) }
                session.loadMe()
                message = "Аватар обновлён"
            } catch (e: ApiException) {
                message = e.message
            } finally {
                avatarBusy = false
            }
        }
    }

    fun deleteAvatar() {
        viewModelScope.launch {
            avatarBusy = true
            try {
                apiCall(json) { authApi.deleteAvatar() }
                session.loadMe()
                message = "Аватар удалён"
            } catch (e: ApiException) {
                message = e.message
            } finally {
                avatarBusy = false
            }
        }
    }

    fun loadStats() {
        viewModelScope.launch {
            try {
                val today = LocalDate.now()
                val monday = today.with(DayOfWeek.MONDAY)
                val fmt = DateTimeFormatter.ofPattern("yyyy-MM-dd")
                stats = apiCall(json) { statsApi.profile(monday.format(fmt), today.format(fmt)) }
            } catch (_: Exception) {
            }
        }
    }

    // Обновление раздела при входе/возврате.
    fun reload() {
        viewModelScope.launch { session.loadMe() }
        loadStats()
    }

    fun switchCompany(companyId: Long) {
        viewModelScope.launch {
            switching = true
            try {
                session.switchCompany(companyId)
                formInit = false
                session.loadMe()
                loadStats()
                // Смена компании = новые данные во всех разделах: проактивно
                // обновляем общий кэш чатов/presence (бейдж непрочитанных и т.п.).
                runCatching { messengerRepo.refreshConversations() }
                runCatching { messengerRepo.refreshPresence() }
                message = "Компания переключена"
            } catch (e: ApiException) {
                message = e.message
            } finally {
                switching = false
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

    fun consumeMessage() {
        message = null
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ProfileScreen(container: AppContainer) {
    val viewModel: ProfileViewModel = viewModel {
        ProfileViewModel(
            container.sessionManager, container.authApi, container.statsApi,
            container.messengerRepo, container.json,
        )
    }
    val context = LocalContext.current
    RefreshOnResume { viewModel.reload() }
    val me by container.sessionManager.me.collectAsStateWithLifecycle()
    val authState by container.sessionManager.authState.collectAsStateWithLifecycle()
    val companies by container.sessionManager.companies.collectAsStateWithLifecycle()
    val claims = (authState as? AuthState.LoggedIn)?.claims

    var showCompanySheet by remember { mutableStateOf(false) }
    var showPasswordSheet by remember { mutableStateOf(false) }
    var cropUri by remember { mutableStateOf<Uri?>(null) }
    val picker = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        if (uri != null) cropUri = uri
    }
    cropUri?.let { uri ->
        AvatarCropDialog(
            uri = uri,
            onCancel = { cropUri = null },
            onCropped = { bytes ->
                cropUri = null
                viewModel.uploadAvatarBytes(bytes)
            },
        )
    }

    LaunchedEffect(viewModel.message) {
        viewModel.message?.let {
            Toast.makeText(context, it, Toast.LENGTH_SHORT).show()
            viewModel.consumeMessage()
        }
    }

    Scaffold(topBar = { TopAppBar(title = { Text("Профиль") }) }) { padding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    modifier = Modifier.fillMaxWidth().padding(vertical = 8.dp),
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        UserAvatar(userId = me?.id, name = me?.fio, avatarPath = me?.avatarPath, size = 96.dp)
                        if (viewModel.avatarBusy) {
                            CircularProgressIndicator(modifier = Modifier.size(96.dp), strokeWidth = 2.dp)
                        }
                    }
                    Text(
                        text = me?.fio ?: "…",
                        style = MaterialTheme.typography.headlineSmall,
                        fontWeight = FontWeight.SemiBold,
                        modifier = Modifier.padding(top = 12.dp),
                    )
                    me?.login?.let { Text("@$it", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant) }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.padding(top = 6.dp)) {
                        me?.role?.name?.let { AssistChip(onClick = {}, enabled = false, label = { Text(it) }) }
                        claims?.companyName?.let { AssistChip(onClick = {}, enabled = false, label = { Text(it) }) }
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.padding(top = 12.dp)) {
                        OutlinedButton(onClick = { picker.launch("image/*") }, enabled = !viewModel.avatarBusy) {
                            Icon(Icons.Filled.PhotoCamera, contentDescription = null, modifier = Modifier.size(18.dp))
                            Text("Загрузить", modifier = Modifier.padding(start = 6.dp))
                        }
                        if (me?.avatarPath != null) {
                            OutlinedButton(onClick = { viewModel.deleteAvatar() }, enabled = !viewModel.avatarBusy) {
                                Icon(Icons.Filled.Delete, contentDescription = null, modifier = Modifier.size(18.dp))
                                Text("Удалить", modifier = Modifier.padding(start = 6.dp))
                            }
                        }
                    }
                }
            }

            viewModel.stats?.let { stats ->
                item { StatsCard(stats) }
            }

            item {
                SectionCard(title = "Редактирование профиля") {
                    OutlinedTextField(
                        value = viewModel.fio,
                        onValueChange = { viewModel.fio = it },
                        label = { Text("ФИО") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = viewModel.login,
                        onValueChange = { viewModel.login = it },
                        label = { Text("Логин") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                    )
                    OutlinedTextField(
                        value = viewModel.post,
                        onValueChange = { viewModel.post = it },
                        label = { Text("Должность") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                    )
                    OutlinedTextField(
                        value = viewModel.phone,
                        onValueChange = { viewModel.phone = it },
                        label = { Text("Телефон") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                    )
                    OutlinedTextField(
                        value = viewModel.email,
                        onValueChange = { viewModel.email = it },
                        label = { Text("Email") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                    )
                    viewModel.profileError?.let {
                        Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall, modifier = Modifier.padding(top = 8.dp))
                    }
                    Button(
                        onClick = { viewModel.saveProfile() },
                        enabled = !viewModel.profileSaving,
                        modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                    ) {
                        Text(if (viewModel.profileSaving) "Сохраняю…" else "Сохранить")
                    }
                }
            }

            item {
                SectionCard(title = "Безопасность") {
                    Text(
                        text = "Пароль для входа в аккаунт",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    OutlinedButton(
                        onClick = {
                            viewModel.resetPasswordForm()
                            showPasswordSheet = true
                        },
                        modifier = Modifier.fillMaxWidth().padding(top = 10.dp),
                    ) {
                        Icon(Icons.Filled.Lock, contentDescription = null, modifier = Modifier.size(18.dp))
                        Text("Сменить пароль", modifier = Modifier.padding(start = 8.dp))
                    }
                }
            }

            // Переключение компании — для многокомпанийных (кроме Администратора
            // системы). Выбор вынесен в модалку: список не растёт в карточке профиля.
            if (companies.size > 1 && claims?.isRootAdmin != true) {
                item {
                    SectionCard(title = "Компания") {
                        Text(
                            text = claims?.companyName ?: "Не выбрана",
                            style = MaterialTheme.typography.titleMedium,
                            modifier = Modifier.padding(top = 2.dp),
                        )
                        OutlinedButton(
                            onClick = { showCompanySheet = true },
                            enabled = !viewModel.switching,
                            modifier = Modifier.fillMaxWidth().padding(top = 10.dp),
                        ) {
                            Icon(Icons.Filled.SwapHoriz, contentDescription = null, modifier = Modifier.size(18.dp))
                            Text("Сменить компанию", modifier = Modifier.padding(start = 8.dp))
                        }
                    }
                }
            }

            item {
                OutlinedButton(
                    onClick = { viewModel.logout() },
                    enabled = !viewModel.loggingOut,
                    colors = ButtonDefaults.outlinedButtonColors(
                        contentColor = MaterialTheme.colorScheme.error,
                    ),
                    modifier = Modifier.fillMaxWidth().padding(top = 4.dp),
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

    if (showPasswordSheet) {
        ChangePasswordSheet(
            viewModel = viewModel,
            onDismiss = {
                viewModel.resetPasswordForm()
                showPasswordSheet = false
            },
        )
    }

    if (showCompanySheet) {
        val sheetState = rememberModalBottomSheetState()
        ModalBottomSheet(onDismissRequest = { showCompanySheet = false }, sheetState = sheetState) {
            Column(modifier = Modifier.fillMaxWidth().navigationBarsPadding()) {
                Text(
                    text = "Выберите компанию",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    modifier = Modifier.padding(start = 20.dp, end = 20.dp, bottom = 8.dp),
                )
                companies.forEach { company ->
                    val active = company.companyId == claims?.companyId
                    ListItem(
                        headlineContent = { Text(company.companyName) },
                        supportingContent = if (!company.isActive) { { Text("отключена") } } else null,
                        leadingContent = { Icon(Icons.Filled.Domain, contentDescription = null) },
                        trailingContent = if (active) {
                            { Icon(Icons.Filled.Check, contentDescription = "Активна", tint = MaterialTheme.colorScheme.primary) }
                        } else null,
                        colors = ListItemDefaults.colors(containerColor = Color.Transparent),
                        modifier = Modifier
                            .fillMaxWidth()
                            .then(
                                if (!active && company.isActive && !viewModel.switching) {
                                    Modifier.clickable {
                                        viewModel.switchCompany(company.companyId)
                                        showCompanySheet = false
                                    }
                                } else Modifier,
                            ),
                    )
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ChangePasswordSheet(viewModel: ProfileViewModel, onDismiss: () -> Unit) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .imePadding()
                .padding(horizontal = 24.dp)
                .padding(bottom = 24.dp),
        ) {
            Text(
                text = "Смена пароля",
                style = MaterialTheme.typography.titleLarge,
                modifier = Modifier.padding(bottom = 16.dp),
            )
            OutlinedTextField(
                value = viewModel.currentPass,
                onValueChange = { viewModel.currentPass = it },
                label = { Text("Текущий пароль") },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Next),
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = viewModel.newPass,
                onValueChange = { viewModel.newPass = it },
                label = { Text("Новый пароль") },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Next),
                modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
            )
            OutlinedTextField(
                value = viewModel.confirmPass,
                onValueChange = { viewModel.confirmPass = it },
                label = { Text("Подтверждение пароля") },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Done),
                keyboardActions = KeyboardActions(onDone = { viewModel.changePassword(onSuccess = onDismiss) }),
                modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
            )
            viewModel.passwordError?.let {
                Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall, modifier = Modifier.padding(top = 8.dp))
            }
            Button(
                onClick = { viewModel.changePassword(onSuccess = onDismiss) },
                enabled = !viewModel.passwordSaving,
                modifier = Modifier.fillMaxWidth().padding(top = 20.dp).height(52.dp),
            ) {
                if (viewModel.passwordSaving) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(22.dp),
                        color = MaterialTheme.colorScheme.onPrimary,
                        strokeWidth = 2.dp,
                    )
                } else {
                    Text("Изменить пароль")
                }
            }
        }
    }
}

@Composable
private fun StatsCard(stats: ProfileStatsDto) {
    SectionCard(title = "Моя неделя") {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceAround) {
            StatMetric(value = formatHours(stats.totalHours), label = "часов")
            StatMetric(value = stats.tasksCount.toString(), label = "задач")
        }
        val maxHours = (stats.byUnitTypes.maxOfOrNull { it.hours } ?: 0.0).coerceAtLeast(0.001)
        stats.byUnitTypes.forEach { type ->
            Column(modifier = Modifier.fillMaxWidth().padding(top = 10.dp)) {
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text(type.name, style = MaterialTheme.typography.bodyMedium)
                    Text(formatHours(type.hours) + " ч", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(8.dp)
                        .padding(top = 4.dp)
                        .clipBar(),
                ) {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth((type.hours / maxHours).toFloat().coerceIn(0.04f, 1f))
                            .height(8.dp)
                            .barFill(),
                    )
                }
            }
        }
    }
}

@Composable
private fun StatMetric(value: String, label: String) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Text(value, style = MaterialTheme.typography.headlineSmall, color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.Bold)
        Text(label, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@Composable
private fun Modifier.clipBar(): Modifier =
    this.then(
        Modifier
            .padding(0.dp)
            .clip(RoundedCornerShape(4.dp))
            .background(MaterialTheme.colorScheme.surfaceContainerHighest)
    )

@Composable
private fun Modifier.barFill(): Modifier =
    this.then(
        Modifier
            .clip(RoundedCornerShape(4.dp))
            .background(MaterialTheme.colorScheme.primary)
    )

@Composable
private fun SectionCard(title: String, content: @Composable () -> Unit) {
    Card(
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Text(title, style = MaterialTheme.typography.titleMedium, modifier = Modifier.padding(bottom = 8.dp))
            content()
        }
    }
}

private val ruLocale = Locale.forLanguageTag("ru")

private fun formatHours(hours: Double): String =
    if (hours == hours.toLong().toDouble()) hours.toLong().toString()
    else String.format(ruLocale, "%.1f", hours)
