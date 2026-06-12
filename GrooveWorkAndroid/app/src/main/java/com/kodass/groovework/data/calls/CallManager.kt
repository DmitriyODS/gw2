package com.kodass.groovework.data.calls

import android.content.Context
import android.content.Intent
import android.media.AudioAttributes
import android.media.AudioManager
import android.media.MediaPlayer
import android.media.RingtoneManager
import android.media.ToneGenerator
import com.kodass.groovework.data.dto.CallDto
import com.kodass.groovework.data.dto.LivekitInfoDto
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.GatewayEvent
import com.kodass.groovework.data.ws.longField
import com.kodass.groovework.data.ws.objField
import com.kodass.groovework.notifications.Notifier
import com.kodass.groovework.service.CallService
import com.twilio.audioswitch.AudioDevice
import io.livekit.android.LiveKit
import io.livekit.android.audio.AudioSwitchHandler
import io.livekit.android.events.RoomEvent
import io.livekit.android.events.collect
import io.livekit.android.room.Room
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

private const val OUTGOING_TIMEOUT_MS = 45_000L

// Ринг-фаза через WS-команды call:* (gatewaysvc → gRPC callsvc), медиа — LiveKit.
class CallManager(
    private val appContext: Context,
    private val gateway: GatewayClient,
    private val session: SessionManager,
    private val json: Json,
    private val notifier: Notifier,
    private val scope: CoroutineScope,
) {
    private val _phase = MutableStateFlow<CallPhase>(CallPhase.Idle)
    val phase: StateFlow<CallPhase> = _phase

    // Полноэкранный UI звонка показан (false при активном звонке = баннер «вернуться»).
    val callUiVisible = MutableStateFlow(false)
    // Просьба авто-принять звонок (кнопка «Ответить» из шторки).
    val autoAcceptRequested = MutableStateFlow(false)

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

    private var room: Room? = null

    // Для видео-рендеров на экране звонка.
    val roomOrNull: Room?
        get() = room

    private var roomEventsJob: Job? = null
    private var outgoingTimeoutJob: Job? = null
    private var ringtone: MediaPlayer? = null
    private var ringbackPlayer: MediaPlayer? = null
    private var ringbackTone: ToneGenerator? = null
    private var wasIncoming = false
    private var accepted = false

    private val myUserId: Long?
        get() = (session.authState.value as? AuthState.LoggedIn)?.claims?.userId

    init {
        scope.launch { gateway.events.collect { handle(it) } }
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

    fun accept() {
        val call = (_phase.value as? CallPhase.Incoming)?.call ?: return
        accepted = true
        stopRingtone()
        gateway.send("call:accept", buildJsonObject { put("call_id", call.id) })
    }

    fun decline() {
        val call = (_phase.value as? CallPhase.Incoming)?.call ?: return
        gateway.send("call:decline", buildJsonObject { put("call_id", call.id) })
        cleanup()
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
            runCatching { room?.localParticipant?.setMicrophoneEnabled(enabled) }
        }
    }

    fun toggleCamera() {
        val enabled = !cameraEnabled.value
        cameraEnabled.value = enabled
        scope.launch {
            runCatching {
                room?.localParticipant?.setCameraEnabled(enabled)
                localVideoTrack.value = if (enabled) {
                    room?.localParticipant?.getTrackPublication(Track.Source.CAMERA)?.track as? VideoTrack
                } else {
                    null
                }
            }
        }
    }

    fun flipCamera() {
        val track = room?.localParticipant
            ?.getTrackPublication(Track.Source.CAMERA)?.track as? LocalVideoTrack ?: return
        runCatching { track.switchCamera() }
    }

    fun setSpeaker(on: Boolean) {
        speakerOn.value = on
        val handler = room?.audioHandler as? AudioSwitchHandler ?: return
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
                val call = decodeCall(event.data) ?: return
                if (_phase.value != CallPhase.Idle) return
                wasIncoming = true
                accepted = false
                cameraEnabled.value = false
                _phase.value = CallPhase.Incoming(call)
                callUiVisible.value = true
                startRingtone()
                notifier.showIncomingCall(call)
            }
            "call:started" -> {
                val call = decodeCall(event.data.objField("call")) ?: return
                val livekit = decodeLivekit(event.data.objField("livekit")) ?: return
                wasIncoming = false
                _phase.value = CallPhase.Outgoing(call)
                callUiVisible.value = true
                startRingback()
                connect(livekit, video = call.media == "video")
                armOutgoingTimeout()
            }
            "call:accepted" -> {
                val call = decodeCall(event.data.objField("call")) ?: return
                val livekit = decodeLivekit(event.data.objField("livekit")) ?: return
                if ((_phase.value as? CallPhase.Incoming)?.call?.id != call.id) return
                notifier.cancelCall()
                _phase.value = CallPhase.Active(call)
                activeSince.value = System.currentTimeMillis()
                callUiVisible.value = true
                connect(livekit, video = call.media == "video" && cameraEnabled.value)
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
        scope.launch {
            try {
                val newRoom = LiveKit.create(appContext)
                room = newRoom
                roomEventsJob?.cancel()
                roomEventsJob = scope.launch {
                    newRoom.events.collect { onRoomEvent(it) }
                }
                kotlinx.coroutines.withTimeout(15_000) {
                    newRoom.connect(url, info.token)
                }
                micEnabled.value = true
                newRoom.localParticipant.setMicrophoneEnabled(true)
                if (video) {
                    cameraEnabled.value = true
                    newRoom.localParticipant.setCameraEnabled(true)
                    localVideoTrack.value =
                        newRoom.localParticipant.getTrackPublication(Track.Source.CAMERA)?.track as? VideoTrack
                }
                // Видеозвонок — сразу громкая связь, голосовой — динамик у уха.
                setSpeaker(video)
                startCallService()
            } catch (e: Exception) {
                _errors.tryEmit("Не удалось подключиться к звонку")
                hangup()
            }
        }
    }

    private fun onRoomEvent(event: RoomEvent) {
        when (event) {
            is RoomEvent.ParticipantConnected -> {
                connectedIdentities.value =
                    connectedIdentities.value + (event.participant.identity?.value ?: return)
                // Собеседник вошёл в комнату — исходящий стал активным (как на вебе).
                (_phase.value as? CallPhase.Outgoing)?.let { outgoing ->
                    outgoingTimeoutJob?.cancel()
                    stopRingback()
                    _phase.value = CallPhase.Active(outgoing.call)
                    activeSince.value = System.currentTimeMillis()
                }
            }
            is RoomEvent.ParticipantDisconnected -> {
                val identity = event.participant.identity?.value ?: return
                connectedIdentities.value = connectedIdentities.value - identity
                remoteVideoTracks.value = remoteVideoTracks.value - identity
            }
            is RoomEvent.TrackSubscribed -> {
                val identity = event.participant.identity?.value ?: return
                val track = event.track
                if (track is VideoTrack) {
                    remoteVideoTracks.value = remoteVideoTracks.value + (identity to track)
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
                if (_phase.value != CallPhase.Idle) cleanup()
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

    private fun startRingtone() {
        stopRingtone()
        runCatching {
            val uri = RingtoneManager.getDefaultUri(RingtoneManager.TYPE_RINGTONE) ?: return
            ringtone = MediaPlayer().apply {
                setDataSource(appContext, uri)
                setAudioAttributes(
                    AudioAttributes.Builder()
                        .setUsage(AudioAttributes.USAGE_NOTIFICATION_RINGTONE)
                        .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                        .build()
                )
                isLooping = true
                prepare()
                start()
            }
        }
    }

    private fun stopRingtone() {
        runCatching {
            ringtone?.stop()
            ringtone?.release()
        }
        ringtone = null
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

    private fun startCallService() {
        runCatching {
            appContext.startForegroundService(Intent(appContext, CallService::class.java))
        }
    }

    private fun cleanup() {
        stopRingtone()
        stopRingback()
        outgoingTimeoutJob?.cancel()
        outgoingTimeoutJob = null
        roomEventsJob?.cancel()
        roomEventsJob = null
        // UI сбрасываем мгновенно; отключение LiveKit может быть долгим — в фон.
        val oldRoom = room
        room = null
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
        autoAcceptRequested.value = false
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
