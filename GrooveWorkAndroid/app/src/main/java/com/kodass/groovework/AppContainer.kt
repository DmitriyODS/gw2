package com.kodass.groovework

import android.app.Application
import com.kodass.groovework.data.api.AuthApi
import com.kodass.groovework.data.calls.CallManager
import com.kodass.groovework.data.api.MessengerApi
import com.kodass.groovework.data.api.MetaApi
import com.kodass.groovework.data.api.PushApi
import com.kodass.groovework.data.api.TasksApi
import com.kodass.groovework.data.calls.CallPhase
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.notifications.PushTokenManager
import com.kodass.groovework.data.network.AuthHeaderInterceptor
import com.kodass.groovework.data.network.HostSelectionInterceptor
import com.kodass.groovework.data.network.TokenAuthenticator
import com.kodass.groovework.data.repo.MessengerRepository
import com.kodass.groovework.data.repo.TasksRepository
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.session.SessionStore
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.notifications.NotificationCenter
import com.kodass.groovework.notifications.Notifier
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory
import java.util.concurrent.TimeUnit

// Ручной DI: один контейнер на процесс, без фреймворков.
class AppContainer(app: Application) {
    val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        coerceInputValues = true
    }

    val appScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    private val store = SessionStore(app)
    private val hostInterceptor = HostSelectionInterceptor()
    val sessionManager = SessionManager(store, hostInterceptor, json)

    val okHttp: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(8, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .addInterceptor(hostInterceptor)
        .addInterceptor(AuthHeaderInterceptor { sessionManager.accessToken })
        .authenticator(TokenAuthenticator { stale -> sessionManager.refreshBlocking(stale) })
        .build()

    private val retrofit = Retrofit.Builder()
        // Реальный хост подставляет HostSelectionInterceptor из настроек сессии.
        .baseUrl("https://gw.invalid/")
        .client(okHttp)
        .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
        .build()

    val authApi: AuthApi = retrofit.create(AuthApi::class.java)
    val messengerApi: MessengerApi = retrofit.create(MessengerApi::class.java)
    val tasksApi: TasksApi = retrofit.create(TasksApi::class.java)
    val metaApi: MetaApi = retrofit.create(MetaApi::class.java)
    val pushApi: PushApi = retrofit.create(PushApi::class.java)

    val gateway = GatewayClient(okHttp, sessionManager, json)
    val messengerRepo = MessengerRepository(messengerApi, gateway, sessionManager, json, appScope)
    val tasksRepo = TasksRepository(tasksApi, json)

    val notifier = Notifier(app)
    val notificationCenter = NotificationCenter(notifier, gateway, messengerRepo, sessionManager, json, appScope)
    val callManager = CallManager(app, gateway, sessionManager, json, notifier, appScope)
    val pushTokens = PushTokenManager(pushApi, appScope)

    // Маршрут из тапа по уведомлению — MainScreen подхватывает и навигирует.
    val pendingRoute = MutableStateFlow<String?>(null)

    init {
        sessionManager.authApi = authApi
        sessionManager.onLogout = { pushTokens.unregisterCurrentToken() }
        appScope.launch { sessionManager.bootstrap(BuildConfig.DEFAULT_SERVER_URL) }

        // FCM-токен регистрируем при входе (фоновая доставка пушей).
        appScope.launch {
            sessionManager.authState.collect { state ->
                if (state is AuthState.LoggedIn && !state.claims.forceChange) {
                    pushTokens.registerCurrentToken()
                }
            }
        }

        // WS держим живым только когда приложение на экране ИЛИ идёт звонок;
        // в остальное время фоновые уведомления доставляет FCM (экономит батарею).
        appScope.launch {
            combine(
                notificationCenter.appForeground,
                callManager.phase,
                sessionManager.authState,
            ) { foreground, phase, auth ->
                auth is AuthState.LoggedIn && (foreground || phase != CallPhase.Idle)
            }.distinctUntilChanged().collect { shouldRun ->
                if (shouldRun) gateway.start() else gateway.stop()
            }
        }
    }
}
