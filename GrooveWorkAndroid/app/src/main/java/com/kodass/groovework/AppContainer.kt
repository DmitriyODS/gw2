package com.kodass.groovework

import android.app.Application
import com.kodass.groovework.data.api.AuthApi
import com.kodass.groovework.data.api.CallsApi
import com.kodass.groovework.calls.CallController
import com.kodass.groovework.data.api.MessengerApi
import com.kodass.groovework.data.api.MetaApi
import com.kodass.groovework.data.api.PushApi
import com.kodass.groovework.data.api.TasksApi
import com.kodass.groovework.calls.CallState
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

// Deep link из писем (verify-email / reset-password / invite). Парсится из
// Intent.ACTION_VIEW в MainActivity; потребляется в зависимости от состояния входа.
sealed interface DeepLink {
    data class VerifyEmail(val token: String) : DeepLink
    data class ResetPassword(val token: String) : DeepLink
    data class Invite(val token: String) : DeepLink

    companion object {
        // Разбор пути ссылки приложения: /verify-email?token=, /reset-password?token=,
        // /invite/<token>. Возвращает null, если ссылка не наша.
        fun parse(path: String?, token: String?): DeepLink? {
            if (path == null) return null
            return when {
                path.startsWith("/verify-email") && !token.isNullOrBlank() -> VerifyEmail(token)
                path.startsWith("/reset-password") && !token.isNullOrBlank() -> ResetPassword(token)
                path.startsWith("/invite/") -> path.removePrefix("/invite/").trim('/')
                    .takeIf { it.isNotBlank() }?.let { Invite(it) }
                else -> null
            }
        }
    }
}

// Ручной DI: один контейнер на процесс, без фреймворков.
class AppContainer(app: Application) {
    val json = Json {
        ignoreUnknownKeys = true
        explicitNulls = false
        coerceInputValues = true
    }

    val appScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    // Персонализация оформления (режим + палитра) — локально на устройстве.
    val theme = com.kodass.groovework.ui.theme.ThemeController(app, appScope)

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
    val callsApi: CallsApi = retrofit.create(CallsApi::class.java)
    val statsApi: com.kodass.groovework.data.api.StatsApi =
        retrofit.create(com.kodass.groovework.data.api.StatsApi::class.java)
    val companiesApi: com.kodass.groovework.data.api.CompaniesApi =
        retrofit.create(com.kodass.groovework.data.api.CompaniesApi::class.java)
    val aiApi: com.kodass.groovework.data.api.AiApi =
        retrofit.create(com.kodass.groovework.data.api.AiApi::class.java)
    // YouGile-ручки под капотом ходят в ru.yougile.com (несколько запросов с
    // ретраями) — 30с клиента не хватает, поднимаем планку как на фронте (60с).
    private val yougileRetrofit = retrofit.newBuilder()
        .client(okHttp.newBuilder().readTimeout(60, TimeUnit.SECONDS).callTimeout(70, TimeUnit.SECONDS).build())
        .build()
    val yougileApi: com.kodass.groovework.data.api.YougileApi =
        yougileRetrofit.create(com.kodass.groovework.data.api.YougileApi::class.java)
    val backupApi: com.kodass.groovework.data.api.BackupApi =
        retrofit.create(com.kodass.groovework.data.api.BackupApi::class.java)

    // Отдельный клиент для скачивания файлов: без read-таймаута (большие файлы),
    // те же интерсепторы (хост/токен) — путь /uploads на том же сервере.
    val downloadHttp: OkHttpClient = okHttp.newBuilder()
        .readTimeout(0, TimeUnit.SECONDS)
        .build()
    val downloader = com.kodass.groovework.data.files.Downloader(app, downloadHttp)

    // Обновление приложения «по воздуху» (проверка версии + скачивание APK).
    // Один на процесс: фаза «готово к установке» переживает уход с экрана.
    val appUpdater = com.kodass.groovework.data.update.AppUpdater(
        app, metaApi, downloadHttp, sessionManager, json, appScope,
    )

    val unitsApi: com.kodass.groovework.data.api.UnitsApi =
        retrofit.create(com.kodass.groovework.data.api.UnitsApi::class.java)

    val registriesApi: com.kodass.groovework.data.api.RegistriesApi =
        retrofit.create(com.kodass.groovework.data.api.RegistriesApi::class.java)

    val gateway = GatewayClient(okHttp, sessionManager, json)
    val messengerRepo = MessengerRepository(messengerApi, gateway, sessionManager, json, appScope)
    val tasksRepo = TasksRepository(tasksApi, json)
    val unitsRepo = com.kodass.groovework.data.repo.UnitsRepository(unitsApi, json)
    val registriesRepo = com.kodass.groovework.data.repo.RegistriesRepository(registriesApi, json)

    val notifier = Notifier(app)
    val notificationCenter = NotificationCenter(notifier, gateway, messengerRepo, sessionManager, json, appScope)
    val callController = CallController(app, gateway, sessionManager, json, callsApi, appScope)
    val unitManager = com.kodass.groovework.data.units.UnitManager(
        unitsRepo, sessionManager, gateway, json, notifier, appScope,
    )
    val pushTokens = PushTokenManager(pushApi, appScope)

    // Маршрут из тапа по уведомлению — MainScreen подхватывает и навигирует.
    val pendingRoute = MutableStateFlow<String?>(null)

    // Deep link из письма (App Links на gw.kodass.ru) — потребляют AuthFlow
    // (verify/reset до входа) и MainScreen (invite после входа).
    val pendingDeepLink = MutableStateFlow<DeepLink?>(null)

    init {
        sessionManager.authApi = authApi
        sessionManager.companiesApi = companiesApi
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
                callController.ui,
                sessionManager.authState,
            ) { foreground, ui, auth ->
                auth is AuthState.LoggedIn && (foreground || ui.state != CallState.Idle)
            }.distinctUntilChanged().collect { shouldRun ->
                if (shouldRun) gateway.start() else gateway.stop()
            }
        }
    }
}
