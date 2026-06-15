package com.kodass.groovework.calls

import android.content.Context
import android.media.AudioManager
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
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout

// Сигналы медиа-сессии наружу (в CallController), по которым тот двигает машину
// состояний. Чисто медиа — без знания о ринг-фазе/сигналинге.
sealed interface SessionEvent {
    data object PeerJoined : SessionEvent          // собеседник в комнате
    data object RemoteAudio : SessionEvent         // первый аудиотрек собеседника — слышно
    data object PeerLeft : SessionEvent            // в комнате не осталось собеседников (p2p)
    data object MicUnavailable : SessionEvent      // микрофон не опубликовался
    data object CameraUnavailable : SessionEvent   // камера не опубликовалась
    data class Failed(val message: String) : SessionEvent // не удалось подключиться
    data class Ended(val duplicate: Boolean) : SessionEvent // Disconnected (duplicate = вошли с др. устройства)
    data class AudioFocus(val gained: Boolean) : SessionEvent // система отобрала/вернула фокус (сотовый звонок)
}

private const val CONNECT_TIMEOUT_MS = 15_000L

// Обёртка над комнатой LiveKit — ЕДИНОЛИЧНЫЙ владелец её жизненного цикла.
// Атомарность: монотонная epoch + один connect-Job; dispose() гасит и
// создаваемую, и созданную комнату, поэтому ownership не размазан по корутинам
// (исключает «фантом» с открытым микрофоном после отбоя).
class LiveKitSession(
    private val appContext: Context,
    private val scope: CoroutineScope,
) {
    private val _room = MutableStateFlow<Room?>(null)
    val room: StateFlow<Room?> = _room.asStateFlow()

    private val _connection = MutableStateFlow(CallConnection.Connecting)
    val connection: StateFlow<CallConnection> = _connection.asStateFlow()

    private val _localVideo = MutableStateFlow<VideoTrack?>(null)
    val localVideo: StateFlow<VideoTrack?> = _localVideo.asStateFlow()

    private val _remoteVideo = MutableStateFlow<VideoTrack?>(null)
    val remoteVideo: StateFlow<VideoTrack?> = _remoteVideo.asStateFlow()

    private val _remoteSpeaking = MutableStateFlow(false)
    val remoteSpeaking: StateFlow<Boolean> = _remoteSpeaking.asStateFlow()

    private val _events = MutableSharedFlow<SessionEvent>(extraBufferCapacity = 16)
    val events: SharedFlow<SessionEvent> = _events

    private var connectJob: Job? = null
    private var eventsJob: Job? = null

    // Желаемое состояние устройств — применяется и при тоггле на «Соединение…»
    // (до публикации), и сразу после connect.
    @Volatile private var wantMic = true
    @Volatile private var wantCamera = false

    // Эпоха сессии: инкрементится в connect() и dispose(); connect-корутина,
    // возобновившись, по расхождению понимает, что её комнату надо бросить.
    @Volatile private var epoch = 0L

    // Состояние комнаты, изменяемое ТОЛЬКО из events-коллектора (один поток).
    private val remoteIds = mutableSetOf<String>()
    private val videoById = mutableMapOf<String, VideoTrack>()
    private var audioArrived = false

    fun connect(url: String, token: String, video: Boolean, withMic: Boolean) {
        val myEpoch = ++epoch
        wantMic = withMic
        wantCamera = video
        connectJob?.cancel()
        connectJob = scope.launch {
            _connection.value = CallConnection.Connecting
            // Порядок устройств под тип звонка: у дефолта динамик выше уха, из-за
            // чего голосовой уходил на громкую связь. BT/проводная — всегда выше.
            val order = if (video) {
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
            val audioHandler = AudioSwitchHandler(appContext).apply {
                preferredDeviceList = order
                // LiveKit держит аудио-фокус (manageAudioFocus=true) и форвардит его
                // изменения сюда — ловим потерю фокуса при входящем сотовом/будильнике.
                onAudioFocusChangeListener = AudioManager.OnAudioFocusChangeListener { change ->
                    when (change) {
                        AudioManager.AUDIOFOCUS_LOSS,
                        AudioManager.AUDIOFOCUS_LOSS_TRANSIENT,
                        AudioManager.AUDIOFOCUS_LOSS_TRANSIENT_CAN_DUCK ->
                            _events.tryEmit(SessionEvent.AudioFocus(gained = false))
                        AudioManager.AUDIOFOCUS_GAIN ->
                            _events.tryEmit(SessionEvent.AudioFocus(gained = true))
                    }
                }
            }
            val room = LiveKit.create(
                appContext,
                options = RoomOptions(adaptiveStream = true, dynacast = true),
                overrides = LiveKitOverrides(audioOptions = AudioOptions(audioHandler = audioHandler)),
            )
            if (myEpoch != epoch) { runCatching { room.release() }; return@launch }
            // Присваиваем комнату ДО connect — тогда dispose() гарантированно её найдёт.
            _room.value = room
            eventsJob = launch { room.events.collect { onRoomEvent(it, room) } }
            try {
                withTimeout(CONNECT_TIMEOUT_MS) { room.connect(url, token) }
            } catch (_: Exception) {
                if (myEpoch == epoch) {
                    _events.tryEmit(SessionEvent.Failed("Не удалось подключиться к звонку"))
                }
                return@launch
            }
            if (myEpoch != epoch) return@launch
            _connection.value = CallConnection.Connected
            // Собеседник мог быть уже в комнате к моменту нашего входа
            // (ParticipantConnected для него не придёт) — засеваем presence.
            seedExisting(room)
            // Микрофон/камеру публикуем ВНЕ роняющего try: сбой устройства не рвёт
            // звонок (можно остаться слушателем) — паритет с вебом.
            applyMic(room)
            if (wantCamera) applyCamera(room)
            // Маршрут по умолчанию: видео — громкая связь, голос — ухо (гарнитура,
            // если есть, выигрывает внутри setSpeaker).
            setSpeaker(video)
        }
    }

    private fun seedExisting(room: Room) {
        var any = false
        room.remoteParticipants.keys.forEach { remoteIds += it.value; any = true }
        if (any) _events.tryEmit(SessionEvent.PeerJoined)
    }

    private fun onRoomEvent(event: RoomEvent, room: Room) {
        when (event) {
            is RoomEvent.Connected -> _connection.value = CallConnection.Connected
            is RoomEvent.Reconnecting -> _connection.value = CallConnection.Reconnecting
            is RoomEvent.Reconnected -> _connection.value = CallConnection.Connected
            is RoomEvent.ParticipantConnected -> {
                val id = event.participant.identity?.value ?: return
                remoteIds += id
                _events.tryEmit(SessionEvent.PeerJoined)
            }
            is RoomEvent.ParticipantDisconnected -> {
                val id = event.participant.identity?.value ?: return
                remoteIds -= id
                videoById -= id
                refreshRemoteVideo()
                if (remoteIds.isEmpty()) _events.tryEmit(SessionEvent.PeerLeft)
            }
            is RoomEvent.TrackSubscribed -> {
                val id = event.participant.identity?.value ?: return
                when (val track = event.track) {
                    is VideoTrack -> { videoById[id] = track; refreshRemoteVideo() }
                    is AudioTrack -> if (!audioArrived) {
                        audioArrived = true
                        _events.tryEmit(SessionEvent.RemoteAudio)
                    }
                    else -> {}
                }
            }
            is RoomEvent.TrackUnsubscribed -> {
                if (event.track is VideoTrack) {
                    val id = event.participant.identity?.value ?: return
                    videoById -= id
                    refreshRemoteVideo()
                }
            }
            is RoomEvent.ActiveSpeakersChanged -> {
                val localId = room.localParticipant.identity?.value
                _remoteSpeaking.value = event.speakers.any { it.identity?.value != null && it.identity?.value != localId }
            }
            is RoomEvent.Disconnected -> {
                _connection.value = CallConnection.Disconnected
                _events.tryEmit(SessionEvent.Ended(event.reason == DisconnectReason.DUPLICATE_IDENTITY))
            }
            else -> {}
        }
    }

    private fun refreshRemoteVideo() {
        _remoteVideo.value = videoById.values.firstOrNull()
    }

    // ── Управление устройствами (идемпотентно; до connect лишь запоминает интент;
    //    setMicrophoneEnabled/setCameraEnabled — suspend, поэтому fire-and-forget,
    //    сбой публикации сообщаем событием) ──

    fun setMic(enabled: Boolean) {
        wantMic = enabled
        val lp = _room.value?.localParticipant ?: return
        scope.launch { runCatching { lp.setMicrophoneEnabled(enabled) } }
    }

    fun setCamera(enabled: Boolean) {
        wantCamera = enabled
        val room = _room.value ?: return
        scope.launch {
            val ok = runCatching {
                room.localParticipant.setCameraEnabled(enabled)
                _localVideo.value = if (enabled) {
                    room.localParticipant.getTrackPublication(Track.Source.CAMERA)?.track as? VideoTrack
                } else null
            }.isSuccess
            if (!ok && enabled) _events.tryEmit(SessionEvent.CameraUnavailable)
        }
    }

    fun flipCamera() {
        val track = _room.value?.localParticipant
            ?.getTrackPublication(Track.Source.CAMERA)?.track as? LocalVideoTrack ?: return
        runCatching { track.switchCamera() }
    }

    // Выбор аудио-маршрута. Проводная/BT-гарнитура ВСЕГДА выигрывает (не прыгаем
    // на динамик); иначе on → громкая связь, off → ухо.
    fun setSpeaker(on: Boolean) {
        val handler = _room.value?.audioHandler as? AudioSwitchHandler ?: return
        val devices = handler.availableAudioDevices
        val headset = devices.firstOrNull {
            it is AudioDevice.BluetoothHeadset || it is AudioDevice.WiredHeadset
        }
        val target = when {
            headset != null -> headset
            on -> devices.firstOrNull { it is AudioDevice.Speakerphone }
            else -> devices.firstOrNull { it is AudioDevice.Earpiece }
                ?: devices.firstOrNull { it !is AudioDevice.Speakerphone }
        }
        if (target != null) runCatching { handler.selectDevice(target) }
    }

    private suspend fun applyMic(room: Room) {
        if (!wantMic) {
            runCatching { room.localParticipant.setMicrophoneEnabled(false) }
            return
        }
        val ok = runCatching { room.localParticipant.setMicrophoneEnabled(true) }.isSuccess
        val live = room.localParticipant.getTrackPublication(Track.Source.MICROPHONE)?.track != null
        if (!ok || !live) {
            val retried = runCatching { room.localParticipant.setMicrophoneEnabled(true) }.isSuccess &&
                room.localParticipant.getTrackPublication(Track.Source.MICROPHONE)?.track != null
            if (!retried) _events.tryEmit(SessionEvent.MicUnavailable)
        }
    }

    private suspend fun applyCamera(room: Room) {
        val ok = runCatching {
            room.localParticipant.setCameraEnabled(true)
            _localVideo.value = room.localParticipant.getTrackPublication(Track.Source.CAMERA)?.track as? VideoTrack
        }.isSuccess
        if (!ok) _events.tryEmit(SessionEvent.CameraUnavailable)
    }

    fun dispose() {
        epoch++ // инвалидируем in-flight connect
        connectJob?.cancel(); connectJob = null
        eventsJob?.cancel(); eventsJob = null
        val old = _room.value
        _room.value = null
        _connection.value = CallConnection.Connecting
        _localVideo.value = null
        _remoteVideo.value = null
        _remoteSpeaking.value = false
        remoteIds.clear()
        videoById.clear()
        audioArrived = false
        wantMic = true
        wantCamera = false
        if (old != null) scope.launch {
            runCatching { old.disconnect() }
            runCatching { old.release() }
        }
    }
}
