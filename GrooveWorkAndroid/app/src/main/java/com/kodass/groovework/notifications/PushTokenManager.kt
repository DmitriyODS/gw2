package com.kodass.groovework.notifications

import android.util.Log
import com.google.firebase.messaging.FirebaseMessaging
import com.kodass.groovework.data.api.DeviceTokenRequest
import com.kodass.groovework.data.api.PushApi
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

// Регистрация FCM-токена на сервере (pushsvc). Токен берём у Firebase; при
// логине регистрируем текущий, при logout — снимаем (пока есть авторизация),
// onNewToken шлёт обновлённый.
class PushTokenManager(
    private val pushApi: PushApi,
    private val scope: CoroutineScope,
) {
    fun registerCurrentToken() {
        FirebaseMessaging.getInstance().token
            .addOnSuccessListener { token -> register(token) }
            .addOnFailureListener { e -> Log.w(TAG, "getToken failed", e) }
    }

    fun register(token: String) {
        if (token.isBlank()) return
        scope.launch {
            runCatching { pushApi.register(DeviceTokenRequest(token)) }
                .onFailure { Log.w(TAG, "register failed", it) }
        }
    }

    // Снять текущий токен с сервера (вызывать ДО очистки сессии — нужен токен
    // доступа). suspend: logout дожидается завершения.
    suspend fun unregisterCurrentToken() {
        val token = runCatching { currentToken() }.getOrNull() ?: return
        runCatching { pushApi.unregister(DeviceTokenRequest(token)) }
            .onFailure { Log.w(TAG, "unregister failed", it) }
    }

    private suspend fun currentToken(): String =
        kotlinx.coroutines.suspendCancellableCoroutine { cont ->
            FirebaseMessaging.getInstance().token
                .addOnSuccessListener { cont.resumeWith(Result.success(it)) }
                .addOnFailureListener { cont.resumeWith(Result.failure(it)) }
        }

    private companion object {
        const val TAG = "PushTokenManager"
    }
}
