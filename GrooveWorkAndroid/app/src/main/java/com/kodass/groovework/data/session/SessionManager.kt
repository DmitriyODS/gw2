package com.kodass.groovework.data.session

import com.kodass.groovework.data.api.AuthApi
import com.kodass.groovework.data.dto.ChangeDefaultRequest
import com.kodass.groovework.data.dto.LoginRequest
import com.kodass.groovework.data.dto.MembershipDto
import com.kodass.groovework.data.dto.SelectCompanyRequest
import com.kodass.groovework.data.dto.SessionResponse
import com.kodass.groovework.data.dto.SwitchCompanyRequest
import com.kodass.groovework.data.dto.UserDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.network.HostSelectionInterceptor
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.data.network.normalizeServerUrl
import com.kodass.groovework.data.network.parseApiError
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.serialization.json.Json
import retrofit2.Response

data class SessionClaims(
    val userId: Long,
    val forceChange: Boolean,
    val companyId: Long?,
    val companyName: String?,
    val roleLevel: Int?,
    val isRootAdmin: Boolean,
)

sealed interface AuthState {
    data object Loading : AuthState
    data object LoggedOut : AuthState
    // Есть сохранённая сессия, но возобновить не удалось из-за отсутствия сети —
    // НЕ разлогиниваем, показываем экран «нет подключения» с кнопкой «Повторить».
    data object Offline : AuthState
    data class LoggedIn(val claims: SessionClaims) : AuthState
}

// Результат login: либо сразу сессия, либо (для многокомпанийных) нужен выбор компании.
sealed interface LoginResult {
    data object Success : LoginResult
    data class NeedsCompany(val selectToken: String, val companies: List<MembershipDto>) : LoginResult
}

// Жизненный цикл сессии: access-токен в памяти (TTL 15 мин), refresh — в DataStore.
class SessionManager(
    private val store: SessionStore,
    private val hostInterceptor: HostSelectionInterceptor,
    private val json: Json,
) {
    lateinit var authApi: AuthApi

    @Volatile
    var accessToken: String? = null
        private set

    private val refreshMutex = Mutex()

    private val _authState = kotlinx.coroutines.flow.MutableStateFlow<AuthState>(AuthState.Loading)
    val authState: kotlinx.coroutines.flow.StateFlow<AuthState> = _authState

    private val _serverUrl = kotlinx.coroutines.flow.MutableStateFlow("")
    val serverUrl: kotlinx.coroutines.flow.StateFlow<String> = _serverUrl

    private val _me = kotlinx.coroutines.flow.MutableStateFlow<UserDto?>(null)
    val me: kotlinx.coroutines.flow.StateFlow<UserDto?> = _me

    // Хук перед logout (пока есть авторизация) — снятие FCM-токена с сервера.
    var onLogout: (suspend () -> Unit)? = null

    // Компании пользователя (для переключателя активной компании); пусто у Администратора системы.
    private val _companies = kotlinx.coroutines.flow.MutableStateFlow<List<MembershipDto>>(emptyList())
    val companies: kotlinx.coroutines.flow.StateFlow<List<MembershipDto>> = _companies

    suspend fun bootstrap(defaultServerUrl: String) {
        val server = store.serverUrl() ?: defaultServerUrl
        applyServer(server)
        attemptResume()
    }

    // Повторная попытка возобновить сессию (кнопка «Повторить» на offline-экране).
    suspend fun retryBootstrap() {
        _authState.value = AuthState.Loading
        attemptResume()
    }

    private suspend fun attemptResume() {
        val refresh = store.refreshToken()
        if (refresh == null) {
            _authState.value = AuthState.LoggedOut
            return
        }
        try {
            val resp = apiCall(json) { authApi.refresh("refresh_token=$refresh") }
            applySession(unwrap(resp))
        } catch (e: ApiException) {
            if (e.status == 401 || e.status == 403) {
                // Сессия недействительна — выходим на экран входа.
                store.setRefreshToken(null)
                _authState.value = AuthState.LoggedOut
            } else {
                // Нет сети / сервер недоступен — сессию сохраняем, показываем offline-экран.
                _authState.value = AuthState.Offline
            }
        }
    }

    suspend fun login(serverUrl: String, login: String, password: String): LoginResult {
        applyServer(serverUrl)
        store.setServerUrl(_serverUrl.value)
        val resp = apiCall(json) { authApi.login(LoginRequest(login, password)) }
        val body = unwrap(resp)
        // Многокомпанийный аккаунт: access-токена ещё нет — сначала выбор компании.
        if (body.needsCompanySelection) {
            return LoginResult.NeedsCompany(body.selectToken.orEmpty(), body.companies)
        }
        applySession(body)
        return LoginResult.Success
    }

    suspend fun selectCompany(selectToken: String, companyId: Long) {
        val resp = apiCall(json) { authApi.selectCompany(SelectCompanyRequest(selectToken, companyId)) }
        applySession(unwrap(resp))
    }

    suspend fun switchCompany(companyId: Long) {
        val resp = apiCall(json) { authApi.switchCompany(SwitchCompanyRequest(companyId)) }
        applySession(unwrap(resp))
    }

    suspend fun changeDefault(newLogin: String, newPassword: String, confirmPassword: String) {
        val resp = apiCall(json) {
            authApi.changeDefault(ChangeDefaultRequest(newLogin, newPassword, confirmPassword))
        }
        applySession(unwrap(resp))
    }

    suspend fun logout() {
        // Снимаем FCM-токен, пока ещё авторизованы (нужен access-токен).
        runCatching { onLogout?.invoke() }
        try {
            authApi.logout()
        } catch (_: Exception) {
            // Выходим локально даже без связи с сервером.
        }
        clear()
    }

    suspend fun loadMe() {
        try {
            _me.value = apiCall(json) { authApi.me() }
        } catch (_: Exception) {
        }
    }

    // Смена адреса сервера: токены принадлежат старому серверу, поэтому это выход —
    // снимаем FCM и завершаем сессию на СТАРОМ сервере, затем переключаем хост и
    // чистим сессию (экран входа откроется с новым адресом).
    suspend fun changeServer(url: String) {
        runCatching { onLogout?.invoke() }
        runCatching { authApi.logout() }
        applyServer(url)
        store.setServerUrl(_serverUrl.value)
        clear()
    }

    // Вызывается из OkHttp Authenticator (поток OkHttp) — runBlocking namеренно.
    fun refreshBlocking(staleToken: String?): String? = runBlocking {
        refreshMutex.withLock {
            val current = accessToken
            if (current != null && current != staleToken) return@withLock current
            val refresh = store.refreshToken() ?: return@withLock null
            try {
                val resp = authApi.refresh("refresh_token=$refresh")
                if (!resp.isSuccessful) {
                    if (resp.code() == 401) {
                        store.setRefreshToken(null)
                        clearState()
                    }
                    return@withLock null
                }
                val session = resp.body() ?: return@withLock null
                captureRefreshCookie(resp)
                applySessionInternal(session)
                session.accessToken
            } catch (_: Exception) {
                null
            }
        }
    }

    private suspend fun unwrap(resp: Response<SessionResponse>): SessionResponse {
        if (!resp.isSuccessful) {
            throw parseApiError(json, resp.code(), resp.errorBody()?.string())
        }
        captureRefreshCookie(resp)
        return resp.body() ?: throw ApiException("EMPTY_BODY", "Пустой ответ сервера", resp.code())
    }

    private suspend fun captureRefreshCookie(resp: Response<SessionResponse>) {
        val cookie = resp.headers().values("Set-Cookie")
            .firstOrNull { it.startsWith("refresh_token=") } ?: return
        val value = cookie.substringAfter("refresh_token=").substringBefore(';')
        if (value.isNotEmpty()) store.setRefreshToken(value)
    }

    private fun applySession(session: SessionResponse) {
        applySessionInternal(session)
    }

    private fun applySessionInternal(session: SessionResponse) {
        accessToken = session.accessToken
        _companies.value = session.companies
        _authState.value = AuthState.LoggedIn(
            SessionClaims(
                userId = session.userId,
                forceChange = session.forceChange,
                companyId = session.companyId,
                companyName = session.companyName,
                roleLevel = session.roleLevel,
                isRootAdmin = session.isRootAdmin,
            )
        )
    }

    private fun applyServer(url: String) {
        val normalized = normalizeServerUrl(url)
        _serverUrl.value = normalized
        hostInterceptor.setServer(normalized)
    }

    private suspend fun clear() {
        store.setRefreshToken(null)
        clearState()
    }

    private fun clearState() {
        accessToken = null
        _me.value = null
        _companies.value = emptyList()
        _authState.value = AuthState.LoggedOut
    }
}
