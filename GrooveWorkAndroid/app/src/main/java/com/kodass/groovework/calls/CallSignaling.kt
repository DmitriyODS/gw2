package com.kodass.groovework.calls

import com.kodass.groovework.data.api.CallsApi
import com.kodass.groovework.data.dto.CallDto
import com.kodass.groovework.data.dto.CallTokenDto
import com.kodass.groovework.data.dto.LivekitInfoDto
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.GatewayEvent
import com.kodass.groovework.data.ws.longField
import com.kodass.groovework.data.ws.objField
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.mapNotNull
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonArray

// Сигналы звонка от сервера (после разбора WS-кадров call:*).
sealed interface CallSignal {
    data class Incoming(val call: CallDto) : CallSignal
    data class Started(val call: CallDto, val livekit: LivekitInfoDto) : CallSignal
    data class Invited(val call: CallDto) : CallSignal
    data class Ended(val callId: Long?, val status: String?) : CallSignal
    data class Error(val code: String?) : CallSignal
}

// Тонкий слой сигналинга: исходящие команды call:* через WS-шлюз, разбор
// входящих событий в типизированный поток, REST для токена входа и resync.
// Контракт совпадает с веб-фронтом и gatewaysvc (бэкенд не меняется).
class CallSignaling(
    private val gateway: GatewayClient,
    private val callsApi: CallsApi,
    private val json: Json,
) {
    val signals: Flow<CallSignal> = gateway.events.mapNotNull { parse(it) }
    val connected: StateFlow<Boolean> get() = gateway.connected

    fun startCall(userIds: List<Long>, video: Boolean) {
        gateway.send("call:start", buildJsonObject {
            putJsonArray("user_ids") { userIds.forEach { add(JsonPrimitive(it)) } }
            put("media", if (video) "video" else "audio")
        })
    }

    fun decline(callId: Long) = send("call:decline", callId)
    fun leave(callId: Long) = send("call:leave", callId)
    fun end(callId: Long) = send("call:end", callId)

    private fun send(event: String, callId: Long) =
        gateway.send(event, buildJsonObject { put("call_id", callId) })

    // Токен входа/ответа: серверный RejoinToken помечает участника joined — это
    // полноценный ответ на звонок, не зависящий от живого WS (работает из пуша).
    suspend fun fetchJoinToken(callId: Long): CallTokenDto? =
        runCatching { callsApi.token(callId) }.getOrNull()

    // Живой звонок пользователя по REST (resync после реконнекта/из пуша).
    suspend fun resyncActive(): CallDto? =
        runCatching { callsApi.active() }.getOrNull()?.call

    private fun parse(e: GatewayEvent): CallSignal? = when (e.event) {
        "call:incoming" -> decodeCall(e.data)?.let { CallSignal.Incoming(it) }
        "call:started" -> {
            val call = decodeCall(e.data.objField("call"))
            val lk = decodeLivekit(e.data.objField("livekit"))
            if (call != null && lk != null) CallSignal.Started(call, lk) else null
        }
        "call:invited" -> decodeCall(e.data.objField("call"))?.let { CallSignal.Invited(it) }
        "call:ended" -> CallSignal.Ended(e.data.longField("call_id"), e.data.stringField("status"))
        "call:error" -> CallSignal.Error(e.data.stringField("code"))
        else -> null
    }

    private fun decodeCall(element: JsonElement?): CallDto? =
        element?.let { runCatching { json.decodeFromJsonElement<CallDto>(it) }.getOrNull() }

    private fun decodeLivekit(element: JsonElement?): LivekitInfoDto? =
        element?.let { runCatching { json.decodeFromJsonElement<LivekitInfoDto>(it) }.getOrNull() }

    private fun JsonElement?.stringField(name: String): String? =
        (objField(name) as? JsonPrimitive)?.content
}
