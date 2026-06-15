package com.kodass.groovework.data.calls

import android.content.Context
import android.content.Intent
import android.media.AudioManager
import android.media.MediaPlayer
import android.media.ToneGenerator
import com.kodass.groovework.data.api.CallsApi
import com.kodass.groovework.data.dto.CallDto
import com.kodass.groovework.data.dto.LivekitInfoDto
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.GatewayEvent
import com.kodass.groovework.data.ws.longField
import com.kodass.groovework.data.ws.objField
import com.kodass.groovework.CallActivity
import com.kodass.groovework.notifications.Notifier
import com.kodass.groovework.service.CallService
import com.kodass.groovework.service.Ringer
import com.twilio.audioswitch.AudioDevice
import io.livekit.android.AudioOptions
import io.livekit.android.LiveKit
import io.livekit.android.LiveKitOverrides
import io.livekit.android.RoomOptions
import io.livekit.android.audio.AudioSwitchHandler
import io.livekit.android.events.DisconnectReason
import io.livekit.android.events.RoomEvent
import io.livekit.android.events.collect
import io.livekit.android.room.Room
import io.livekit.android.room.track.AudioTrack
import io.livekit.android.room.track.LocalVideoTrack
import io.livekit.android.room.track.Track
import io.livekit.android.room.track.VideoTrack
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonArray

sealed interface CallPhase {
    data object Idle : CallPhase
    data class Incoming(val call: CallDto) : CallPhase
    data class Outgoing(val call: CallDto) : CallPhase
    data class Active(val call: CallDto) : CallPhase
}

// Запрос на ответ (из шторки или с экрана входящего): шлюз разрешений в UI
// подхватывает его, спрашивает доступ к микрофону/камере и зовёт answerCall.
data class PendingAnswer(val callId: Long, val video: Boolean, val fio: String?)

private const val OUTGOING_TIMEOUT_MS = 45_000L
// Страховка для входящего, поднятого пушем офлайн: если событие отмены не
// дойдёт (были офлайн), звонилка не должна звенеть вечно.
private const val INCOMING_TIMEOUT_MS = 60_000L

// Ринг-фаза через WS-команды call:* (gatewaysvc → gRPC callsvc), медиа — LiveKit.
class CallManager(
    private val appContext: Context,
    private val gateway: GatewayClient,
    private val session: SessionManager,
    private val json: Json,
    private val notifier: Notifier,
    private val callsApi: CallsApi,
    private val scope: CoroutineScope,
) {
    private val _phase = MutableStateFlow<CallPhase>(CallPhase.Idle)
    val phase: StateFlow<CallPhase> = _phase

    // Полноэкранный UI звонка показан (false при активном звонке = баннер «вернуться»).
    val callUiVisible = MutableStateFlow(false)
    // Запрос ответа на звонок — обрабатывает шлюз разрешений в AppRoot.
    val pendingAnswer = MutableStateFlow<PendingAnswer?>(null)

    val micEnabled = MutableStateFlow(true)
    val cameraEnabled = MutableStateFlow(false)
    val speakerOn = MutableStateFlow(false)
    val localVideoTrack = MutableStateFlow<VideoTrack?>(null)
    val remoteVideoTracks = MutableStateFlow<Map<String, VideoTrack>>(emptyMap())
    val activeSpeakers = MutableStateFlow<Set<String>>(emptySet())
    val connectedIdentities = MutableStateFlow<Set<String>>(emptySet())
    val activeSince = MutableStateFlow<Long?>(null)

    private val _errors = MutableSharedFlow<String>(extraBufferCapacity = 8)
    val errors: SharedFlow<String> = _errors

    // Комната LiveKit — наблюдаемая (StateFlow), чтобы экран звонка перерисовался
    // ровно тогда, когда комната появилась/исчезла, а не по случайному ререндеру.
    private val _room = MutableStateFlow<Room?>(null)
    val room: StateFlow<Room?> = _room

    // Идёт установка соединения с медиа-комнатой (ответ на звонок до connect()).
    // Снимается, как только LiveKit реально подключился, — иначе «Соединение…»
    // висело, пока разговор уже шёл (room выставлялся, но UI не реагировал).
    private val _connecting = MutableStateFlow(false)
    val connecting: StateFlow<Boolean> = _connecting

    private var roomEventsJob: Job? = null
    private var outgoingTimeoutJob: Job? = null
    private var incomingTimeoutJob: Job? = null
    private val ringer = Ringer(appContext)
    private var ringbackPlayer: MediaPlayer? = null
    private var ringbackTone: ToneGenerator? = null
    private var wasIncoming = false
    private var accepted = false

    // Монотонная «эпоха» звонка: cleanup() её увеличивает, поэтому любая
    // отложенная корутина (тянем токен / connect к LiveKit), возобновившись
    // после завершения/смены звонка, увидит расхождение и не воскресит звонок
    // (не откроет микрофон в осиротевшую комнату — иначе нас слышно после отбоя).
    @Volatile
    private var callEpoch = 0L

    // Single-flight для resync состояния звонка после реконнекта WS.
    @Volatile
    private var resyncing = false

    private val myUserId: Long?
        get() = (session.authState.value as? AuthState.LoggedIn)?.claims?.userId

    init {
        scope.launch { gateway.events.collect { handle(it) } }
        // После (ре)коннекта WS сверяем состояние звонка с сервером: пока сокет
        // лежал, мы могли пропустить call:ended (отмена/отклон) — без этого
        // входящий/исходящий «висел» бы до локального таймаута.
        scope.launch { gateway.connected.collect { up -> if (up) resyncFromServer() } }
    }

    val currentCall: CallDto?
        get() = when (val p = _phase.value) {
            is CallPhase.Incoming -> p.call
            is CallPhase.Outgoing -> p.call
            is CallPhase.Active -> p.call
            CallPhase.Idle -> null
        }

    // Собеседник для p2p / первый «не я» для группы.
    val peer
        get() = currentCall?.participants?.firstOrNull { it.userId != myUserId }

    // Входящий из FCM-пуша (приложение в фоне/убито, WS не подключён): полная
    // «звонилка» — рингтон+вибрация (Ringer) и full-screen-уведомление через
    // foreground-сервис. Экран звонка поднимет full-screen intent.
    fun onIncomingFromPush(call: CallDto) {
        onIncoming(call, launchUi = false)
    }

    // Входящий из пуша, доставленного НЕ high-priority (FCM понизил приоритет):
    // старт foreground-сервиса из фона тогда кинул бы исключение, поэтому показываем
    // полноэкранное уведомление напрямую, без FGS.
    fun onIncomingFromPushNoFgs(call: CallDto) {
        onIncoming(call, launchUi = false, useFgs = false)
    }

    // Общий вход входящего (из пуша и из WS). launchUi — поднять CallActivity
    // самим (true только из foreground/WS; из фона активити не стартуем —
    // развернёт full-screen intent уведомления). useFgs — поднимать ли
    // foreground-сервис (false при пониженном приоритете пуша; тогда уведомление
    // постим напрямую).
    private fun onIncoming(call: CallDto, launchUi: Boolean, useFgs: Boolean = true) {
        if (_phase.value != CallPhase.Idle) return
        wasIncoming = true
        accepted = false
        cameraEnabled.value = false
        _phase.value = CallPhase.Incoming(call)
        callUiVisible.value = true
        ringer.start()
        // Если FGS не стартовал (понижен приоритет/окно фона закрылось) — full-screen
        // уведомление всё равно показываем сами, иначе входящий потеряется.
        val started = if (useFgs) startCallService(CallService.MODE_INCOMING) else false
        if (!started) notifier.showIncomingCallStandalone(call)
        if (launchUi) launchCallUi()
        armIncomingTimeout(call.id)
    }

    private fun armIncomingTimeout(callId: Long) {
        incomingTimeoutJob?.cancel()
        incomingTimeoutJob = scope.launch {
            delay(INCOMING_TIMEOUT_MS)
            val current = (_phase.value as? CallPhase.Incoming)?.call
            if (current != null && current.id == callId) {
                val fio = current.initiatorFio
                cleanup()
                if (fio != null) notifier.showMissedCall(fio)
            }
        }
    }

    // Открыть полноэкранный экран звонка (CallActivity). Из фона стартовать
    // активити нельзя — там полагаемся на full-screen intent уведомления.
    private fun launchCallUi() {
        runCatching {
            appContext.startActivity(
                Intent(appContext, CallActivity::class.java)
                    .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            )
        }
    }

    // Вернуться к звёрнутому звонку (баннер в приложении).
    fun showCallUi() {
        callUiVisible.value = true
        launchCallUi()
    }

    // Тап по плашке живого звонка в чате: если это наш текущий звонок — просто
    // разворачиваем экран; иначе (мы не в звонке) — присоединяемся по REST-токену.
    fun returnOrJoinCall(callId: Long, video: Boolean) {
        if (currentCall?.id == callId && _phase.value !is CallPhase.Idle) {
            showCallUi()
            return
        }
        if (_phase.value !is CallPhase.Idle) {
            _errors.tryEmit("Вы уже в звонке")
            return
        }
        answerCall(callId, video)
    }

    fun startCall(userId: Long, video: Boolean) {
        if (_phase.value != CallPhase.Idle) {
            _errors.tryEmit("Вы уже в звонке")
            return
        }
        cameraEnabled.value = video
        gateway.send("call:start", buildJsonObject {
            putJsonArray("user_ids") { add(kotlinx.serialization.json.JsonPrimitive(userId)) }
            put("media", if (video) "video" else "audio")
        })
    }

    // Просьба ответить на звонок — выставляет pendingAnswer, который шлюз
    // разрешений в UI подхватит и вызовет answerCall (работает и когда экран
    // входящего ещё не показан — звонок поднят пушем из фона).
    fun requestAnswer(callId: Long, video: Boolean, fio: String? = null) {
        if (_phase.value is CallPhase.Active && currentCall?.id == callId) {
            callUiVisible.value = true
            return
        }
        pendingAnswer.value = PendingAnswer(callId, video, fio ?: currentCall?.initiatorFio)
        callUiVisible.value = true
    }

    // Ответ на входящий: токен входа берём REST-ом (POST /api/calls/{id}/token),
    // не полагаясь на WS-цикл call:accept→call:accepted. Сразу подключаемся к
    // LiveKit — инициатор узнаёт о нас по ParticipantConnected.
    fun answerCall(callId: Long, video: Boolean, fio: String? = null) {
        if (_phase.value is CallPhase.Active && currentCall?.id == callId) {
            callUiVisible.value = true
            return
        }
        accepted = true
        wasIncoming = true
        ringer.stop()
        incomingTimeoutJob?.cancel()
        cameraEnabled.value = video
        callUiVisible.value = true
        _connecting.value = true
        // Оптимистично показываем экран звонка («Соединение…»), пока тянем токен.
        val known = currentCall?.takeIf { it.id == callId }
        _phase.value = CallPhase.Active(
            known ?: CallDto(id = callId, media = if (video) "video" else "audio", initiatorFio = fio)
        )
        // Сразу переводим сервис в активный режим (мы на переднем плане —
        // mic/camera-FGS разрешён): убирает входящее full-screen-уведомление с
        // кнопками на время «Соединение…».
        startCallService(CallService.MODE_ONGOING)
        val epoch = callEpoch
        scope.launch {
            val token = runCatching { callsApi.token(callId) }.getOrNull()
            // Пока тянули токен, звонок мог завершиться/смениться (call:ended,
            // Disconnected, hangup из шторки) — не воскрешаем разорванный звонок.
            if (epoch != callEpoch || _phase.value !is CallPhase.Active || currentCall?.id != callId) {
                return@launch
            }
            if (token == null) {
                _errors.tryEmit("Звонок уже завершён")
                cleanup()
                return@launch
            }
            val call = token.call ?: known ?: return@launch
            _phase.value = CallPhase.Active(call)
            // activeSince НЕ выставляем здесь — таймер разговора пускаем только когда
            // реально пошло аудио собеседника (первый remote audio в onRoomEvent),
            // иначе секунды бегут все 15-20с установки медиа, пока тишина.
            connect(token.livekit, video = call.media == "video" && cameraEnabled.value)
        }
    }

    // Отклонение из шторки/экрана входящего: уведомление гасим всегда (иначе оно
    // зависало), call:decline шлём best-effort — если WS поднят.
    fun declineFromNotification(callId: Long) {
        ringer.stop()
        notifier.cancelCall()
        gateway.send("call:decline", buildJsonObject { put("call_id", callId) })
        val incoming = (_phase.value as? CallPhase.Incoming)?.call
        if (incoming != null && incoming.id == callId) cleanup()
        else pendingAnswer.value = null
    }

    fun hangup() {
        val call = currentCall ?: return
        when (_phase.value) {
            // Отмена исходящего — call:end (как веб-клиент), выход из активного — call:leave.
            is CallPhase.Outgoing -> gateway.send("call:end", buildJsonObject { put("call_id", call.id) })
            else -> gateway.send("call:leave", buildJsonObject { put("call_id", call.id) })
        }
        cleanup()
    }

    fun toggleMic() {
        val enabled = !micEnabled.value
        micEnabled.value = enabled
        scope.launch {
            val ok = runCatching { _room.value?.localParticipant?.setMicrophoneEnabled(enabled) }.isSuccess
            if (!ok) {
                micEnabled.value = !enabled
                _errors.tryEmit("Не удалось переключить микрофон")
            }
        }
    }

    fun toggleCamera() {
        val enabled = !cameraEnabled.value
        cameraEnabled.value = enabled
        scope.launch {
            val ok = runCatching {
                _room.value?.localParticipant?.setCameraEnabled(enabled)
                localVideoTrack.value = if (enabled) {
                    _room.value?.localParticipant?.getTrackPublication(Track.Source.CAMERA)?.track as? VideoTrack
                } else {
                    null
                }
            }.isSuccess
            if (!ok) {
                cameraEnabled.value = !enabled
                _errors.tryEmit("Не удалось переключить камеру")
            }
        }
    }

    fun flipCamera() {
        val track = _room.value?.localParticipant
            ?.getTrackPublication(Track.Source.CAMERA)?.track as? LocalVideoTrack ?: return
        runCatching { track.switchCamera() }
    }

    fun setSpeaker(on: Boolean) {
        speakerOn.value = on
        val handler = _room.value?.audioHandler as? AudioSwitchHandler ?: return
        val devices = handler.availableAudioDevices
        val target = if (on) {
            devices.firstOrNull { it is AudioDevice.Speakerphone }
        } else {
            devices.firstOrNull { it is AudioDevice.Earpiece }
                ?: devices.firstOrNull { it !is AudioDevice.Speakerphone }
        }
        if (target != null) runCatching { handler.selectDevice(target) }
    }

    private fun handle(event: GatewayEvent) {
        when (event.event) {
            "call:incoming" -> {
                // WS работает только когда приложение на экране → можно поднять UI сами.
                val call = decodeCall(event.data) ?: return
                onIncoming(call, launchUi = true)
            }
            "call:started" -> {
                val call = decodeCall(event.data.objField("call")) ?: return
                val livekit = decodeLivekit(event.data.objField("livekit")) ?: return
                wasIncoming = false
                _phase.value = CallPhase.Outgoing(call)
                callUiVisible.value = true
                launchCallUi()
                startRingback()
                connect(livekit, video = call.media == "video")
                armOutgoingTimeout()
            }
            "call:invited" -> {
                val call = decodeCall(event.data.objField("call")) ?: return
                _phase.value = when (val p = _phase.value) {
                    is CallPhase.Active -> CallPhase.Active(call)
                    is CallPhase.Outgoing -> CallPhase.Outgoing(call)
                    else -> return
                }
            }
            "call:ended" -> {
                val callId = event.data.longField("call_id")
                val current = currentCall ?: return
                if (callId != null && callId != current.id) return
                val status = event.data.objField("status")
                    ?.let { (it as? kotlinx.serialization.json.JsonPrimitive)?.content }
                val missed = wasIncoming && !accepted &&
                    (status == "missed" || status == "cancelled")
                val initiatorFio = current.initiatorFio
                cleanup()
                if (missed && initiatorFio != null) notifier.showMissedCall(initiatorFio)
            }
            "call:error" -> {
                val code = (event.data.objField("code") as? kotlinx.serialization.json.JsonPrimitive)?.content
                val message = when (code) {
                    "BUSY" -> "Вы уже в звонке"
                    "INVITEE_BUSY" -> "Собеседник сейчас разговаривает"
                    "CALLS_UNAVAILABLE" -> "Звонки временно недоступны"
                    else -> "Не удалось выполнить действие со звонком"
                }
                _errors.tryEmit(message)
                if (_phase.value is CallPhase.Outgoing || _phase.value is CallPhase.Incoming) cleanup()
            }
        }
    }

    private fun connect(info: LivekitInfoDto, video: Boolean) {
        val url = resolveLivekitUrl(info.url)
        val epoch = callEpoch
        _connecting.value = true
        scope.launch {
            // Свой AudioSwitchHandler с порядком устройств под тип звонка: у дефолтного
            // динамик стоит ВЫШЕ уха, из-за чего голосовой звонок уходил на громкую связь.
            val audioHandler = AudioSwitchHandler(appContext).apply {
                preferredDeviceList = if (video) {
                    listOf(
                        AudioDevice.BluetoothHeadset::class.java,
                        AudioDevice.WiredHeadset::class.java,
                        AudioDevice.Speakerphone::class.java,
                        AudioDevice.Earpiece::class.java,
                    )
                } else {
                    listOf(
                        AudioDevice.BluetoothHeadset::class.java,
                        AudioDevice.WiredHeadset::class.java,
                        AudioDevice.Earpiece::class.java,
                        AudioDevice.Speakerphone::class.java,
                    )
                }
            }
            val newRoom = LiveKit.create(
                appContext,
                options = RoomOptions(adaptiveStream = true, dynacast = true),
                overrides = LiveKitOverrides(audioOptions = AudioOptions(audioHandler = audioHandler)),
            )
            // Звонок мог быть свёрнут/завершён, пока создавали комнату.
            if (epoch != callEpoch) {
                runCatching { newRoom.release() }
                return@launch
            }
            // Присваиваем _room ДО connect — тогда cleanup() гарантированно найдёт
            // эту комнату и отключит её, а не оставит «фантом» с открытым микрофоном.
            _room.value = newRoom
            val eventsJob = scope.launch { newRoom.events.collect { onRoomEvent(it) } }
            roomEventsJob?.cancel()
            roomEventsJob = eventsJob
            try {
                kotlinx.coroutines.withTimeout(15_000) {
                    newRoom.connect(url, info.token)
                }
            } catch (e: Exception) {
                if (epoch == callEpoch) {
                    _errors.tryEmit("Не удалось подключиться к звонку")
                    hangup()
                } else {
                    eventsJob.cancel()
                    runCatching { newRoom.disconnect() }
                    runCatching { newRoom.release() }
                }
                return@launch
            }
            // Пока шёл connect, звонок мог завершиться (hangup из шторки, call:ended) —
            // НЕ публикуем микрофон в осиротевшую комнату.
            if (epoch != callEpoch || _phase.value !is CallPhase.Active) {
                eventsJob.cancel()
                runCatching { newRoom.disconnect() }
                runCatching { newRoom.release() }
                return@launch
            }
            // Подключились к комнате. «Соединение…» снимется на первом аудио собеседника.
            _connecting.value = false
            // Микрофон/камеру публикуем ПОСЛЕ connect и ВНЕ роняющего try: сбой
            // устройства не рвёт звонок (паритет с вебом) — можно остаться «слушателем».
            // Уважаем текущее намерение (юзер мог замьютить на «Соединение…»).
            val micWanted = micEnabled.value
            val micOk = runCatching { newRoom.localParticipant.setMicrophoneEnabled(micWanted) }.isSuccess
            if (micWanted) {
                val micLive = newRoom.localParticipant.getTrackPublication(Track.Source.MICROPHONE)?.track != null
                if (!micOk || !micLive) {
                    // Одна повторная попытка только если трека реально нет (двойной
                    // вызов setMicrophoneEnabled может опубликовать дубль-трек).
                    val retried = runCatching { newRoom.localParticipant.setMicrophoneEnabled(true) }.isSuccess &&
                        newRoom.localParticipant.getTrackPublication(Track.Source.MICROPHONE)?.track != null
                    if (!retried) {
                        micEnabled.value = false
                        _errors.tryEmit("Микрофон недоступен — вас не слышно")
                    }
                }
            }
            if (video) {
                val camOk = runCatching {
                    newRoom.localParticipant.setCameraEnabled(true)
                    localVideoTrack.value =
                        newRoom.localParticipant.getTrackPublication(Track.Source.CAMERA)?.track as? VideoTrack
                }.isSuccess
                cameraEnabled.value = camOk
                if (!camOk) _errors.tryEmit("Камера недоступна — звонок без видео")
            }
            // Видеозвонок — сразу громкая связь, голосовой — динамик у уха.
            setSpeaker(video)
            startCallService(CallService.MODE_ONGOING)
        }
    }

    private fun onRoomEvent(event: RoomEvent) {
        when (event) {
            is RoomEvent.ParticipantConnected -> {
                connectedIdentities.value =
                    connectedIdentities.value + (event.participant.identity?.value ?: return)
                // Собеседник вошёл в комнату — исходящий стал активным (как на вебе).
                // Таймер разговора НЕ пускаем здесь: signaling-join ещё не значит,
                // что слышно — отсчёт начнём на первом аудио (TrackSubscribed ниже).
                (_phase.value as? CallPhase.Outgoing)?.let { outgoing ->
                    outgoingTimeoutJob?.cancel()
                    stopRingback()
                    _phase.value = CallPhase.Active(outgoing.call)
                }
            }
            is RoomEvent.ParticipantDisconnected -> {
                val identity = event.participant.identity?.value ?: return
                connectedIdentities.value = connectedIdentities.value - identity
                remoteVideoTracks.value = remoteVideoTracks.value - identity
            }
            is RoomEvent.TrackSubscribed -> {
                val identity = event.participant.identity?.value ?: return
                when (val track = event.track) {
                    is VideoTrack ->
                        remoteVideoTracks.value = remoteVideoTracks.value + (identity to track)
                    is AudioTrack -> {
                        // Первый удалённый аудиотрек = собеседника реально слышно →
                        // только теперь пускаем таймер и снимаем «Соединение…».
                        if (_phase.value is CallPhase.Active) {
                            if (activeSince.value == null) activeSince.value = System.currentTimeMillis()
                            _connecting.value = false
                            // Устройства аудио уже перечислены — навязываем выбранный маршрут.
                            setSpeaker(speakerOn.value)
                        }
                    }
                    else -> {}
                }
            }
            is RoomEvent.TrackUnsubscribed -> {
                val identity = event.participant.identity?.value ?: return
                if (event.track is VideoTrack) {
                    remoteVideoTracks.value = remoteVideoTracks.value - identity
                }
            }
            is RoomEvent.ActiveSpeakersChanged -> {
                activeSpeakers.value = event.speakers.mapNotNull { it.identity?.value }.toSet()
            }
            is RoomEvent.Disconnected -> {
                if (_phase.value == CallPhase.Idle) return
                // Этим identity вошли с другого устройства — LiveKit вышиб ЭТО
                // соединение; собеседнику звонок не рвём (cleanup не шлёт call:leave).
                if (event.reason == DisconnectReason.DUPLICATE_IDENTITY) {
                    _errors.tryEmit("Звонок продолжен на другом устройстве")
                }
                cleanup()
            }
            else -> {}
        }
    }

    private fun armOutgoingTimeout() {
        outgoingTimeoutJob?.cancel()
        outgoingTimeoutJob = scope.launch {
            delay(OUTGOING_TIMEOUT_MS)
            if (_phase.value is CallPhase.Outgoing) hangup()
        }
    }

    // url бэкенда может быть относительным (/livekit) — достраиваем от адреса сервера.
    private fun resolveLivekitUrl(raw: String): String {
        if (raw.startsWith("ws://") || raw.startsWith("wss://")) return raw
        val base = session.serverUrl.value
        val wsBase = base
            .replaceFirst("https://", "wss://")
            .replaceFirst("http://", "ws://")
            .trimEnd('/')
        return if (raw.startsWith("/")) wsBase + raw else "$wsBase/$raw"
    }

    // Гудок исходящего: res/raw/ringback.* если положили свой звук,
    // иначе системный гудок ToneGenerator (TONE_SUP_RINGTONE, сам зациклен).
    private fun startRingback() {
        stopRingback()
        val resId = appContext.resources.getIdentifier("ringback", "raw", appContext.packageName)
        if (resId != 0) {
            runCatching {
                ringbackPlayer = MediaPlayer.create(appContext, resId)?.apply {
                    isLooping = true
                    start()
                }
            }
        } else {
            runCatching {
                ringbackTone = ToneGenerator(AudioManager.STREAM_VOICE_CALL, 70).also {
                    it.startTone(ToneGenerator.TONE_SUP_RINGTONE)
                }
            }
        }
    }

    private fun stopRingback() {
        runCatching {
            ringbackPlayer?.stop()
            ringbackPlayer?.release()
        }
        ringbackPlayer = null
        runCatching {
            ringbackTone?.stopTone()
            ringbackTone?.release()
        }
        ringbackTone = null
    }

    // Возвращает true, если старт сервиса прошёл без исключения. false — повод
    // показать full-screen уведомление напрямую (старт FGS из фона запрещён).
    private fun startCallService(mode: String): Boolean = runCatching {
        appContext.startForegroundService(
            Intent(appContext, CallService::class.java).putExtra(CallService.EXTRA_MODE, mode)
        )
    }.isSuccess

    // Сверка состояния с сервером после реконнекта WS: чистит зависшую ринг-фазу
    // (входящий/исходящий без живой комнаты), если сервер уже не видит звонок
    // живым. Активный звонок с комнатой не трогаем — им заведует RoomEvent.Disconnected.
    private fun resyncFromServer() {
        if (resyncing) return
        val phase = _phase.value
        if (_room.value != null) return
        if (phase !is CallPhase.Incoming && phase !is CallPhase.Outgoing) return
        val callId = currentCall?.id ?: return
        resyncing = true
        scope.launch {
            try {
                val live = runCatching { callsApi.active() }.getOrNull()?.call
                val stillLive = live != null && live.id == callId &&
                    live.status != "ended" && live.status != "missed"
                // Сверяем ещё раз: пока ходили на сервер, состояние могло измениться.
                if (!stillLive && _room.value == null &&
                    _phase.value !is CallPhase.Active && currentCall?.id == callId) {
                    cleanup()
                }
            } finally {
                resyncing = false
            }
        }
    }

    private fun cleanup() {
        // Инвалидируем любые отложенные connect/answer-корутины этого звонка.
        callEpoch++
        ringer.stop()
        stopRingback()
        outgoingTimeoutJob?.cancel()
        outgoingTimeoutJob = null
        incomingTimeoutJob?.cancel()
        incomingTimeoutJob = null
        roomEventsJob?.cancel()
        roomEventsJob = null
        _connecting.value = false
        // UI сбрасываем мгновенно; отключение LiveKit может быть долгим — в фон.
        val oldRoom = _room.value
        _room.value = null
        localVideoTrack.value = null
        remoteVideoTracks.value = emptyMap()
        activeSpeakers.value = emptySet()
        connectedIdentities.value = emptySet()
        activeSince.value = null
        micEnabled.value = true
        cameraEnabled.value = false
        speakerOn.value = false
        accepted = false
        wasIncoming = false
        pendingAnswer.value = null
        callUiVisible.value = false
        _phase.value = CallPhase.Idle
        notifier.cancelCall()
        appContext.stopService(Intent(appContext, CallService::class.java))
        if (oldRoom != null) {
            scope.launch {
                runCatching { oldRoom.disconnect() }
                runCatching { oldRoom.release() }
            }
        }
    }

    private fun decodeCall(element: kotlinx.serialization.json.JsonElement?): CallDto? =
        element?.let { runCatching { json.decodeFromJsonElement<CallDto>(it) }.getOrNull() }

    private fun decodeLivekit(element: kotlinx.serialization.json.JsonElement?): LivekitInfoDto? =
        element?.let { runCatching { json.decodeFromJsonElement<LivekitInfoDto>(it) }.getOrNull() }
}
