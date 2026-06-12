package com.kodass.groovework.data.ws

import com.kodass.groovework.data.session.SessionManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener

data class GatewayEvent(val event: String, val data: JsonElement?)

// Тонкий клиент gatewaysvc: кадры {"event","data"}, auth первым кадром,
// heartbeat presence каждые 25с, реконнект 1с→5с (как front/src/socket/gateway.js).
class GatewayClient(
    private val okHttp: OkHttpClient,
    private val session: SessionManager,
    private val json: Json,
) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    val events = MutableSharedFlow<GatewayEvent>(extraBufferCapacity = 256)

    private val _connected = MutableStateFlow(false)
    val connected: StateFlow<Boolean> = _connected

    private var ws: WebSocket? = null
    private var loopJob: Job? = null
    private var heartbeatJob: Job? = null

    @Volatile
    private var visible = true

    // Честный presence: вкладка «видима», только когда приложение на экране.
    fun setVisible(value: Boolean) {
        visible = value
        if (_connected.value) {
            send("presence:visibility", buildJsonObject { put("visible", value) })
        }
    }

    @Synchronized
    fun start() {
        if (loopJob?.isActive == true) return
        loopJob = scope.launch { runLoop() }
    }

    @Synchronized
    fun stop() {
        loopJob?.cancel()
        loopJob = null
        heartbeatJob?.cancel()
        heartbeatJob = null
        ws?.close(1000, null)
        ws = null
        _connected.value = false
    }

    fun send(event: String, data: JsonElement) {
        val frame = buildJsonObject {
            put("event", event)
            put("data", data)
        }
        ws?.send(frame.toString())
    }

    private suspend fun runLoop() {
        var attempt = 0
        while (scope.isActive && loopJob?.isActive == true) {
            val token = session.accessToken
            if (token == null) {
                delay(1000)
                continue
            }
            val closed = kotlinx.coroutines.CompletableDeferred<Unit>()
            val url = session.serverUrl.value.trimEnd('/') + "/ws"
            val request = Request.Builder().url(url).build()
            val socket = okHttp.newWebSocket(request, object : WebSocketListener() {
                override fun onOpen(webSocket: WebSocket, response: Response) {
                    val frame = buildJsonObject {
                        put("event", "auth")
                        put("data", buildJsonObject { put("token", token) })
                    }
                    webSocket.send(frame.toString())
                }

                override fun onMessage(webSocket: WebSocket, text: String) {
                    handleFrame(text)
                }

                override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                    closed.complete(Unit)
                }

                override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                    closed.complete(Unit)
                }
            })
            ws = socket
            closed.await()
            _connected.value = false
            heartbeatJob?.cancel()
            ws = null
            attempt++
            delay(if (attempt <= 1) 1000 else 5000)
        }
    }

    private fun handleFrame(text: String) {
        val frame = try {
            json.parseToJsonElement(text).jsonObject
        } catch (_: Exception) {
            return
        }
        val event = frame["event"]?.jsonPrimitive?.content ?: return
        val data = frame["data"]
        when (event) {
            "_connected" -> {
                _connected.value = true
                startHeartbeat()
            }
            "_error" -> {
                // Скорее всего протух access-токен — обновим перед реконнектом.
                scope.launch { session.refreshBlocking(session.accessToken) }
            }
            else -> events.tryEmit(GatewayEvent(event, data))
        }
    }

    private fun startHeartbeat() {
        heartbeatJob?.cancel()
        heartbeatJob = scope.launch {
            send("presence:visibility", buildJsonObject { put("visible", visible) })
            while (isActive) {
                delay(25_000)
                send("presence:heartbeat", buildJsonObject { })
            }
        }
    }
}

// Удобный доступ к полю объекта события.
fun JsonElement?.objField(name: String): JsonElement? = (this as? JsonObject)?.get(name)

fun JsonElement?.longField(name: String): Long? =
    objField(name)?.jsonPrimitive?.content?.toLongOrNull()

fun JsonElement?.boolField(name: String): Boolean? =
    (objField(name) as? kotlinx.serialization.json.JsonPrimitive)?.boolean
