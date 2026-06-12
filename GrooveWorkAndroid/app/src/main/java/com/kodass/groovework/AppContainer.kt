package com.kodass.groovework

import android.app.Application
import com.kodass.groovework.data.api.AuthApi
import com.kodass.groovework.data.calls.CallManager
import com.kodass.groovework.data.api.MessengerApi
import com.kodass.groovework.data.api.MetaApi
import com.kodass.groovework.data.api.TasksApi
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

    val gateway = GatewayClient(okHttp, sessionManager, json)
    val messengerRepo = MessengerRepository(messengerApi, gateway, sessionManager, json, appScope)
    val tasksRepo = TasksRepository(tasksApi, json)

    val notifier = Notifier(app)
    val notificationCenter = NotificationCenter(notifier, gateway, messengerRepo, sessionManager, json, appScope)
    val callManager = CallManager(app, gateway, sessionManager, json, notifier, appScope)

    // Маршрут из тапа по уведомлению — MainScreen подхватывает и навигирует.
    val pendingRoute = MutableStateFlow<String?>(null)

    init {
        sessionManager.authApi = authApi
        appScope.launch { sessionManager.bootstrap(BuildConfig.DEFAULT_SERVER_URL) }
    }
}
