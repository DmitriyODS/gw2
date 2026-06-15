package com.kodass.groovework.calls

import com.kodass.groovework.data.dto.CallDto
import io.livekit.android.room.track.VideoTrack

// ─────────────────────────────────────────────────────────────────────────────
// Единый источник истины подсистемы звонков. Весь UI ведётся ОДНИМ снимком
// CallUiState (lifecycle + media атомарно), поэтому экран никогда не увидит
// несогласованного промежуточного состояния (например, Active без комнаты).
// ─────────────────────────────────────────────────────────────────────────────

// Направление в ринг-фазе.
enum class CallDirection { Incoming, Outgoing }

// Состояние медиа-соединения с комнатой LiveKit (из RoomEvent).
enum class CallConnection { Connecting, Connected, Reconnecting, Disconnected }

// Жизненный цикл звонка. Меняется редко — на переходах фаз.
sealed interface CallState {
    // Звонок, к которому относится фаза (у Dialing его ещё нет).
    val call: CallDto? get() = null
    // Видеозвонок (намерение пользователя/тип звонка).
    val video: Boolean get() = false

    data object Idle : CallState

    // Набор номера: мы инициировали звонок и ждём ответа сервера (call:started).
    // До его прихода CallDto и livekit-токена ещё нет.
    data class Dialing(val targetUserId: Long, override val video: Boolean) : CallState

    // Ринг-фаза: входящий (нам звонят) либо исходящий (звоним, идут гудки).
    data class Ringing(
        val direction: CallDirection,
        override val call: CallDto,
        override val video: Boolean,
    ) : CallState

    // Подключаемся к комнате и ждём устойчивого соединения + первого аудио
    // собеседника. На экране — «Соединение…».
    data class Connecting(
        override val call: CallDto,
        override val video: Boolean,
    ) : CallState

    // Разговор идёт. activeSinceMs — момент старта отсчёта длительности
    // (выставляется на первом аудио собеседника, не на signaling-join).
    data class Active(
        override val call: CallDto,
        override val video: Boolean,
        val activeSinceMs: Long,
    ) : CallState
}

// Высокочастотное медиа-состояние (микрофон/камера/маршрут/треки): отделено в
// поля одного снимка, чтобы частые обновления треков не плодили лишние
// перерисовки lifecycle-частей UI (Compose пропускает неизменившиеся аргументы).
data class MediaState(
    val micEnabled: Boolean = true,
    val cameraEnabled: Boolean = false,
    val speakerOn: Boolean = false,
    val connection: CallConnection = CallConnection.Connecting,
    // Звонок на паузе — система отобрала аудио-фокус (входящий сотовый, будильник).
    val paused: Boolean = false,
    val localVideo: VideoTrack? = null,
    val remoteVideo: VideoTrack? = null,
    val remoteSpeaking: Boolean = false,
)

// Консистентный снимок для UI: lifecycle + media в одном неизменяемом объекте.
data class CallUiState(
    val state: CallState = CallState.Idle,
    val media: MediaState = MediaState(),
)

// Собеседник (для аватара/подписи). Для p2p — единственный «не я».
data class CallParty(
    val userId: Long?,
    val fio: String?,
    val avatarPath: String?,
)

// Намерение ответить на звонок: шлюз разрешений в UI подхватывает его, спрашивает
// доступ к микрофону/камере и зовёт CallController.answer. Работает и когда экран
// входящего ещё не показан (звонок поднят пушем из фона).
data class PendingAnswer(val callId: Long, val video: Boolean, val fio: String?)

// Собеседник звонка с точки зрения myUserId: первый участник «не я», иначе
// инициатор (для входящего, где список участников может быть неполным).
fun CallDto.peerParty(myUserId: Long?): CallParty {
    participants.firstOrNull { it.userId != myUserId }?.let {
        return CallParty(it.userId, it.fio.ifBlank { null }, it.avatarPath)
    }
    val initiator = participants.firstOrNull { it.userId == initiatorId }
    return CallParty(
        userId = initiatorId.takeIf { it != 0L } ?: initiator?.userId,
        fio = initiatorFio ?: initiator?.fio?.ifBlank { null },
        avatarPath = initiator?.avatarPath,
    )
}
