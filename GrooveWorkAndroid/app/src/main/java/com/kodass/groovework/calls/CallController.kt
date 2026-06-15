package com.kodass.groovework.calls

import android.content.Context
import android.content.Intent
import android.media.AudioManager
import android.media.MediaPlayer
import android.media.ToneGenerator
import android.os.PowerManager
import com.kodass.groovework.data.api.CallsApi
import com.kodass.groovework.data.dto.CallDto
import com.kodass.groovework.data.dto.LivekitInfoDto
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.ws.GatewayClient
import io.livekit.android.room.Room
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.distinctUntilChanged
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json

private const val DIALING_TIMEOUT_MS = 15_000L   // сервер молчит после call:start
private const val OUTGOING_TIMEOUT_MS = 45_000L  // собеседник не берёт трубку
private const val INCOMING_TIMEOUT_MS = 60_000L  // страховка для входящего из пуша
private const val CONNECTING_GRACE_MS = 8_000L    // peer вошёл, но аудио не пошло → всё равно Active
private const val EARLY_RESYNC_MS = 3_500L        // ранний REST-resync входящего из пуша

// Оркестратор подсистемы звонков — единственный источник истины (StateFlow<CallUiState>).
// Связывает сигналинг (CallSignaling) и медиа (LiveKitSession), ведёт машину
// состояний, тайминги, аудио-фокус (авто-пауза), proximity-lock, гудок и команды
// foreground-сервису. Все мутации состояния сериализованы на главном потоке.
class CallController(
    private val appContext: Context,
    gateway: GatewayClient,
    private val session: SessionManager,
    json: Json,
    callsApi: CallsApi,
    private val scope: CoroutineScope,
) {
    private val signaling = CallSignaling(gateway, callsApi, json)
    val notifications = CallNotifications(appContext)
    private val lk = LiveKitSession(appContext, scope)
    private val ringer = IncomingRinger(appContext)

    private val _ui = MutableStateFlow(CallUiState())
    val ui: StateFlow<CallUiState> = _ui

    val room: StateFlow<Room?> get() = lk.room

    private val _errors = MutableSharedFlow<String>(extraBufferCapacity = 8)
    val errors: SharedFlow<String> = _errors

    // Запрос ответа на звонок — шлюз разрешений в UI подхватывает и зовёт answer().
    val pendingAnswer = MutableStateFlow<PendingAnswer?>(null)

    // Полноэкранный UI звонка показан (false при активном звонке = баннер «вернуться»).
    val callUiVisible = MutableStateFlow(false)

    private var dialingTimeoutJob: Job? = null
    private var outgoingTimeoutJob: Job? = null
    private var incomingTimeoutJob: Job? = null
    private var connectingTimeoutJob: Job? = null
    private var earlyResyncJob: Job? = null

    private var ringbackPlayer: MediaPlayer? = null
    private var ringbackTone: ToneGenerator? = null

    private var wasIncoming = false
    private var accepted = false

    // Монотонная эпоха звонка — инвалидирует отложенные корутины (token-fetch).
    @Volatile private var callEpoch = 0L
    // Single-flight на ответ (тап на экране + кнопка уведомления одновременно).
    @Volatile private var answering = false
    @Volatile private var resyncing = false

    private val powerManager by lazy { appContext.getSystemService(Context.POWER_SERVICE) as PowerManager }
    private val proximityLock: PowerManager.WakeLock? by lazy {
        if (powerManager.isWakeLockLevelSupported(PowerManager.PROXIMITY_SCREEN_OFF_WAKE_LOCK)) {
            powerManager.newWakeLock(PowerManager.PROXIMITY_SCREEN_OFF_WAKE_LOCK, "groovework:call_proximity")
                .apply { setReferenceCounted(false) }
        } else null
    }

    private val myUserId: Long?
        get() = (session.authState.value as? AuthState.LoggedIn)?.claims?.userId

    val currentCall: CallDto? get() = _ui.value.state.call
    val peer: CallParty? get() = currentCall?.peerParty(myUserId)

    init {
        val main = Dispatchers.Main.immediate
        // Сериализуем мутации машины состояний на главном потоке.
        scope.launch(main) { signaling.signals.collect { onSignal(it) } }
        scope.launch(main) { lk.events.collect { onSessionEvent(it) } }
        // После (ре)коннекта WS сверяем зависшую ринг-фазу с сервером.
        scope.launch(main) { signaling.connected.collect { up -> if (up) resyncFromServer() } }
        // Зеркалим медиа-потоки сессии в единый снимок.
        scope.launch(main) { lk.localVideo.collect { v -> setMedia { it.copy(localVideo = v) } } }
        scope.launch(main) { lk.remoteVideo.collect { v -> setMedia { it.copy(remoteVideo = v) } } }
        scope.launch(main) { lk.remoteSpeaking.collect { s -> setMedia { it.copy(remoteSpeaking = s) } } }
        scope.launch(main) { lk.connection.collect { c -> setMedia { it.copy(connection = c) } } }
        // Датчик приближения гасит экран у уха: только в активном голосовом разговоре
        // с устойчивым соединением (не на «Соединение…», не громкая связь, не видео).
        scope.launch(main) {
            _ui.map { u ->
                u.state is CallState.Active && u.media.connection == CallConnection.Connected &&
                    !u.media.speakerOn && !u.media.cameraEnabled && !u.media.paused
            }.distinctUntilChanged().collect { nearEar ->
                if (nearEar) acquireProximityLock() else releaseProximityLock()
            }
        }
    }

    // ── Публичный API (UI / push / уведомления) ────────────────────────────────

    fun startCall(userId: Long, video: Boolean) {
        if (_ui.value.state != CallState.Idle) { _errors.tryEmit("Вы уже в звонке"); return }
        wasIncoming = false; accepted = false
        resetUi(CallState.Dialing(userId, video), MediaState(micEnabled = true, cameraEnabled = video, speakerOn = video))
        callUiVisible.value = true
        launchCallUi()
        signaling.startCall(listOf(userId), video)
        armDialingTimeout()
    }

    // Входящий из FCM-пуша (приложение в фоне/убито, WS не подключён): пробуем FGS;
    // решение standalone — по фактическому результату старта. + ранний REST-resync.
    fun onIncomingFromPush(call: CallDto) {
        onIncoming(call, launchUi = false)
        scheduleEarlyResync(call.id)
    }

    // Просьба ответить (с экрана входящего или кнопки уведомления): шлюз разрешений
    // в UI подхватит pendingAnswer и вызовет answer(). Работает в любой фазе.
    fun requestAnswer(callId: Long, video: Boolean, fio: String? = null) {
        if (_ui.value.state is CallState.Active && currentCall?.id == callId) {
            callUiVisible.value = true
            return
        }
        pendingAnswer.value = PendingAnswer(callId, video, fio ?: currentCall?.initiatorFio)
        callUiVisible.value = true
    }

    // Ответ: токен входа берём REST-ом (POST /api/calls/{id}/token — серверный
    // RejoinToken помечает участника joined). Работает и из убитого приложения.
    fun answer(callId: Long, video: Boolean, fio: String? = null) {
        if (_ui.value.state is CallState.Active && currentCall?.id == callId) {
            callUiVisible.value = true; return
        }
        if (answering) return
        answering = true
        accepted = true; wasIncoming = true
        ringer.stop()
        incomingTimeoutJob?.cancel()
        earlyResyncJob?.cancel()
        callUiVisible.value = true
        val known = currentCall?.takeIf { it.id == callId }
        val call = known ?: CallDto(id = callId, media = if (video) "video" else "audio", initiatorFio = fio)
        resetUi(CallState.Connecting(call, video), _ui.value.media.copy(cameraEnabled = video, speakerOn = video))
        // FGS в активный режим (без media-типов — повысим на Active, уже из foreground).
        startService(CallForegroundService.MODE_ONGOING)
        val epoch = callEpoch
        scope.launch {
            val token = signaling.fetchJoinToken(callId)
            if (epoch != callEpoch || _ui.value.state !is CallState.Connecting || currentCall?.id != callId) {
                answering = false; return@launch
            }
            if (token == null) { _errors.tryEmit("Звонок уже завершён"); cleanup(); return@launch }
            token.call?.let { setState(CallState.Connecting(it, video)) }
            armConnectingTimeout()
            lk.connect(resolveLivekitUrl(token.livekit.url), token.livekit.token, video, withMic = _ui.value.media.micEnabled)
            answering = false
        }
    }

    // Тап по плашке живого звонка в чате: наш текущий — разворачиваем; иначе — вход.
    fun returnOrJoinCall(callId: Long, video: Boolean) {
        if (currentCall?.id == callId && _ui.value.state != CallState.Idle) { showCallUi(); return }
        if (_ui.value.state != CallState.Idle) { _errors.tryEmit("Вы уже в звонке"); return }
        answer(callId, video)
    }

    fun declineFromNotification(callId: Long) {
        ringer.stop()
        notifications.cancelCall()
        signaling.decline(callId)
        val st = _ui.value.state
        if (st is CallState.Ringing && st.direction == CallDirection.Incoming && st.call.id == callId) cleanup()
        else pendingAnswer.value = null
    }

    fun hangup() {
        val call = currentCall
        when (val st = _ui.value.state) {
            is CallState.Ringing ->
                if (st.direction == CallDirection.Outgoing) call?.let { signaling.end(it.id) }
                else call?.let { signaling.decline(it.id) }
            is CallState.Connecting, is CallState.Active -> call?.let { signaling.leave(it.id) }
            else -> {} // Dialing (нет id) / Idle — гасим локально
        }
        cleanup()
    }

    fun showCallUi() {
        callUiVisible.value = true
        launchCallUi()
    }

    // Оптимистично; сбой реальной публикации скорректирует событие
    // MicUnavailable/CameraUnavailable. До connect — лишь запоминается интент.
    fun toggleMic() {
        val enabled = !_ui.value.media.micEnabled
        setMedia { it.copy(micEnabled = enabled) }
        lk.setMic(enabled)
    }

    fun toggleCamera() {
        val enabled = !_ui.value.media.cameraEnabled
        setMedia { it.copy(cameraEnabled = enabled) }
        lk.setCamera(enabled)
    }

    fun flipCamera() = lk.flipCamera()

    fun setSpeaker(on: Boolean) {
        setMedia { it.copy(speakerOn = on) }
        lk.setSpeaker(on)
    }

    // ── Сигналы сервера ─────────────────────────────────────────────────────────

    private fun onSignal(signal: CallSignal) {
        when (signal) {
            is CallSignal.Incoming -> onIncoming(signal.call, launchUi = true)
            is CallSignal.Started -> onStarted(signal.call, signal.livekit)
            is CallSignal.Invited -> onInvited(signal.call)
            is CallSignal.Ended -> onEnded(signal.callId, signal.status)
            is CallSignal.Error -> onError(signal.code)
        }
    }

    private fun onIncoming(call: CallDto, launchUi: Boolean) {
        if (_ui.value.state != CallState.Idle) {
            // Заняты: авто-отклон второго входящего + уведомление о пропущенном.
            signaling.decline(call.id)
            call.initiatorFio?.let { notifications.showMissedCall(it) }
            return
        }
        wasIncoming = true; accepted = false
        val video = call.media == "video"
        resetUi(CallState.Ringing(CallDirection.Incoming, call, video), MediaState())
        callUiVisible.value = true
        ringer.start()
        val started = startService(CallForegroundService.MODE_INCOMING)
        // FGS не стартовал (из фона/Doze) — full-screen уведомление напрямую.
        if (!started) {
            if (notifications.canPost()) notifications.showIncomingCallStandalone(call)
            else launchCallUi() // нет даже разрешения на уведомления — пробуем поднять экран
        }
        if (launchUi) launchCallUi()
        armIncomingTimeout(call.id)
    }

    private fun onStarted(call: CallDto, livekit: LivekitInfoDto) {
        if (_ui.value.state !is CallState.Dialing) {
            // Набор уже отменён (hangup до call:started) — гасим серверный звонок.
            signaling.end(call.id)
            return
        }
        dialingTimeoutJob?.cancel()
        wasIncoming = false
        val video = call.media == "video"
        setState(CallState.Ringing(CallDirection.Outgoing, call, video))
        callUiVisible.value = true
        startRingback()
        startService(CallForegroundService.MODE_ONGOING)
        lk.connect(resolveLivekitUrl(livekit.url), livekit.token, video, withMic = _ui.value.media.micEnabled)
        armOutgoingTimeout()
    }

    private fun onInvited(call: CallDto) {
        val st = _ui.value.state
        if (st.call?.id == call.id) when (st) {
            is CallState.Active -> setState(st.copy(call = call))
            is CallState.Connecting -> setState(st.copy(call = call))
            is CallState.Ringing -> setState(st.copy(call = call))
            else -> {}
        }
    }

    private fun onEnded(callId: Long?, status: String?) {
        val call = currentCall ?: return
        if (callId != null && callId != call.id) return
        val missed = wasIncoming && !accepted && (status == "missed" || status == "cancelled")
        val fio = call.initiatorFio
        cleanup()
        if (missed && fio != null) notifications.showMissedCall(fio)
    }

    private fun onError(code: String?) {
        _errors.tryEmit(
            when (code) {
                "BUSY" -> "Вы уже в звонке"
                "INVITEE_BUSY" -> "Собеседник сейчас разговаривает"
                "CALLS_UNAVAILABLE" -> "Звонки временно недоступны"
                else -> "Не удалось выполнить действие со звонком"
            }
        )
        val st = _ui.value.state
        if (st is CallState.Dialing || (st is CallState.Ringing && st.direction == CallDirection.Outgoing)) cleanup()
    }

    // ── События медиа-сессии ─────────────────────────────────────────────────────

    private fun onSessionEvent(event: SessionEvent) {
        when (event) {
            SessionEvent.PeerJoined -> {
                val st = _ui.value.state
                if (st is CallState.Ringing && st.direction == CallDirection.Outgoing) {
                    outgoingTimeoutJob?.cancel(); stopRingback()
                    setState(CallState.Connecting(st.call, st.video))
                    armConnectingTimeout()
                }
            }
            SessionEvent.RemoteAudio -> goActive()
            SessionEvent.PeerLeft -> {
                val st = _ui.value.state
                if (st is CallState.Active || st is CallState.Connecting) cleanup()
            }
            SessionEvent.MicUnavailable -> {
                setMedia { it.copy(micEnabled = false) }
                _errors.tryEmit("Микрофон недоступен — вас не слышно")
            }
            SessionEvent.CameraUnavailable -> {
                setMedia { it.copy(cameraEnabled = false) }
                _errors.tryEmit("Камера недоступна — звонок без видео")
            }
            is SessionEvent.Failed -> {
                _errors.tryEmit(event.message)
                hangup()
            }
            is SessionEvent.Ended -> {
                if (event.duplicate) _errors.tryEmit("Звонок продолжен на другом устройстве")
                cleanup()
            }
            is SessionEvent.AudioFocus -> {
                val st = _ui.value.state
                if (st is CallState.Active || st is CallState.Connecting) {
                    if (event.gained) {
                        lk.setMic(_ui.value.media.micEnabled)
                        setMedia { it.copy(paused = false) }
                    } else {
                        lk.setMic(false)
                        setMedia { it.copy(paused = true) }
                    }
                }
            }
        }
    }

    // Перевод в активный разговор: первый аудиотрек собеседника (слышно) либо
    // grace-таймаут (peer вошёл, но молчит/замьючен). Таймер пускаем отсюда.
    private fun goActive() {
        val st = _ui.value.state
        val call = st.call ?: return
        val fromRing = st is CallState.Ringing && st.direction == CallDirection.Outgoing
        if (st !is CallState.Connecting && !fromRing) return
        outgoingTimeoutJob?.cancel()
        connectingTimeoutJob?.cancel()
        stopRingback()
        setState(CallState.Active(call, st.video, System.currentTimeMillis()))
        // Повышаем тип FGS до microphone/camera — теперь мы активны и на переднем
        // плане (иначе SecurityException при промоушне из фона на Android 14+).
        startService(CallForegroundService.MODE_ONGOING_MEDIA)
        lk.setSpeaker(_ui.value.media.speakerOn)
    }

    // ── Тайминги ─────────────────────────────────────────────────────────────────

    private fun armDialingTimeout() {
        dialingTimeoutJob?.cancel()
        dialingTimeoutJob = scope.launch(Dispatchers.Main.immediate) {
            delay(DIALING_TIMEOUT_MS)
            if (_ui.value.state is CallState.Dialing) {
                _errors.tryEmit("Не удалось начать звонок")
                cleanup()
            }
        }
    }

    private fun armOutgoingTimeout() {
        outgoingTimeoutJob?.cancel()
        outgoingTimeoutJob = scope.launch(Dispatchers.Main.immediate) {
            delay(OUTGOING_TIMEOUT_MS)
            val st = _ui.value.state
            if (st is CallState.Ringing && st.direction == CallDirection.Outgoing) hangup()
        }
    }

    private fun armIncomingTimeout(callId: Long) {
        incomingTimeoutJob?.cancel()
        incomingTimeoutJob = scope.launch(Dispatchers.Main.immediate) {
            delay(INCOMING_TIMEOUT_MS)
            val st = _ui.value.state
            if (st is CallState.Ringing && st.direction == CallDirection.Incoming && st.call.id == callId) {
                val fio = st.call.initiatorFio
                cleanup()
                if (fio != null) notifications.showMissedCall(fio)
            }
        }
    }

    // Peer вошёл, но аудио так и не пошло (замьючен/баг) — всё равно идём в Active,
    // чтобы UI не висел на «Соединение…» бесконечно.
    private fun armConnectingTimeout() {
        connectingTimeoutJob?.cancel()
        connectingTimeoutJob = scope.launch(Dispatchers.Main.immediate) {
            delay(CONNECTING_GRACE_MS)
            if (_ui.value.state is CallState.Connecting) goActive()
        }
    }

    private fun scheduleEarlyResync(callId: Long) {
        earlyResyncJob?.cancel()
        earlyResyncJob = scope.launch(Dispatchers.Main.immediate) {
            delay(EARLY_RESYNC_MS)
            val st = _ui.value.state
            if (st is CallState.Ringing && st.direction == CallDirection.Incoming && st.call.id == callId) {
                resyncFromServer()
            }
        }
    }

    // Сверка зависшей ринг-фазы с сервером (после реконнекта WS / из пуша).
    private fun resyncFromServer() {
        if (resyncing || lk.room.value != null) return
        val st = _ui.value.state
        if (st !is CallState.Ringing) return
        val callId = st.call.id
        resyncing = true
        scope.launch {
            try {
                val live = signaling.resyncActive()
                val stillLive = live != null && live.id == callId &&
                    live.status != "ended" && live.status != "missed"
                val cur = _ui.value.state
                if (!stillLive && lk.room.value == null && cur is CallState.Ringing && cur.call.id == callId) {
                    cleanup()
                }
            } finally { resyncing = false }
        }
    }

    // ── Инфраструктура ────────────────────────────────────────────────────────────

    private fun setState(state: CallState) = _ui.update { it.copy(state = state) }
    private fun setMedia(f: (MediaState) -> MediaState) = _ui.update { it.copy(media = f(it.media)) }
    private fun resetUi(state: CallState, media: MediaState) = _ui.update { CallUiState(state, media) }

    private fun launchCallUi() {
        runCatching {
            appContext.startActivity(
                Intent(appContext, com.kodass.groovework.CallActivity::class.java)
                    .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            )
        }
    }

    private fun startService(mode: String): Boolean = runCatching {
        appContext.startForegroundService(
            Intent(appContext, CallForegroundService::class.java)
                .putExtra(CallForegroundService.EXTRA_MODE, mode)
        )
    }.isSuccess

    private fun resolveLivekitUrl(raw: String): String {
        if (raw.startsWith("ws://") || raw.startsWith("wss://")) return raw
        val wsBase = session.serverUrl.value
            .replaceFirst("https://", "wss://")
            .replaceFirst("http://", "ws://")
            .trimEnd('/')
        return if (raw.startsWith("/")) wsBase + raw else "$wsBase/$raw"
    }

    private fun startRingback() {
        stopRingback()
        val resId = appContext.resources.getIdentifier("ringback", "raw", appContext.packageName)
        if (resId != 0) {
            runCatching { ringbackPlayer = MediaPlayer.create(appContext, resId)?.apply { isLooping = true; start() } }
        } else {
            runCatching {
                ringbackTone = ToneGenerator(AudioManager.STREAM_VOICE_CALL, 70).also {
                    it.startTone(ToneGenerator.TONE_SUP_RINGTONE)
                }
            }
        }
    }

    private fun stopRingback() {
        runCatching { ringbackPlayer?.stop(); ringbackPlayer?.release() }
        ringbackPlayer = null
        runCatching { ringbackTone?.stopTone(); ringbackTone?.release() }
        ringbackTone = null
    }

    private fun acquireProximityLock() {
        val lock = proximityLock ?: return
        runCatching { if (!lock.isHeld) lock.acquire() }
    }

    private fun releaseProximityLock() {
        val lock = proximityLock ?: return
        runCatching { if (lock.isHeld) lock.release(PowerManager.RELEASE_FLAG_WAIT_FOR_NO_PROXIMITY) }
    }

    private fun cancelTimeouts() {
        dialingTimeoutJob?.cancel(); dialingTimeoutJob = null
        outgoingTimeoutJob?.cancel(); outgoingTimeoutJob = null
        incomingTimeoutJob?.cancel(); incomingTimeoutJob = null
        connectingTimeoutJob?.cancel(); connectingTimeoutJob = null
        earlyResyncJob?.cancel(); earlyResyncJob = null
    }

    private fun cleanup() {
        callEpoch++ // инвалидируем отложенные token-fetch/connect
        answering = false
        releaseProximityLock()
        ringer.stop()
        stopRingback()
        cancelTimeouts()
        lk.dispose()
        wasIncoming = false
        accepted = false
        resetUi(CallState.Idle, MediaState())
        pendingAnswer.value = null
        callUiVisible.value = false
        notifications.cancelCall()
        appContext.stopService(Intent(appContext, CallForegroundService::class.java))
    }
}
