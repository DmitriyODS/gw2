package com.kodass.groovework.data.network

import okhttp3.Authenticator
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.Interceptor
import okhttp3.Request
import okhttp3.Response
import okhttp3.Route
import java.io.IOException

// Подменяет хост плейсхолдера на адрес сервера, выбранный пользователем.
class HostSelectionInterceptor : Interceptor {
    @Volatile
    var baseUrl: HttpUrl? = null

    fun setServer(url: String) {
        baseUrl = normalizeServerUrl(url).toHttpUrlOrNull()
    }

    override fun intercept(chain: Interceptor.Chain): Response {
        val base = baseUrl ?: throw IOException("Адрес сервера не задан")
        val request = chain.request()
        val newUrl = request.url.newBuilder()
            .scheme(base.scheme)
            .host(base.host)
            .port(base.port)
            .build()
        return chain.proceed(request.newBuilder().url(newUrl).build())
    }
}

fun normalizeServerUrl(raw: String): String {
    var url = raw.trim().trimEnd('/')
    if (url.isNotEmpty() && !url.startsWith("http://") && !url.startsWith("https://")) {
        url = "https://$url"
    }
    return url
}

class AuthHeaderInterceptor(private val tokenProvider: () -> String?) : Interceptor {
    override fun intercept(chain: Interceptor.Chain): Response {
        val token = tokenProvider()
        val request = if (token != null && chain.request().header("Authorization") == null) {
            chain.request().newBuilder().header("Authorization", "Bearer $token").build()
        } else {
            chain.request()
        }
        return chain.proceed(request)
    }
}

// 401 → один синхронный refresh (как очередь запросов во фронтовом client.js) → повтор запроса.
class TokenAuthenticator(private val refresh: (staleToken: String?) -> String?) : Authenticator {
    override fun authenticate(route: Route?, response: Response): Request? {
        val path = response.request.url.encodedPath
        if (path.endsWith("/api/auth/refresh") || path.endsWith("/api/auth/login")) return null
        if (responseCount(response) >= 2) return null
        val stale = response.request.header("Authorization")?.removePrefix("Bearer ")
        val newToken = refresh(stale) ?: return null
        if (newToken == stale) return null
        return response.request.newBuilder()
            .header("Authorization", "Bearer $newToken")
            .build()
    }

    private fun responseCount(response: Response): Int {
        var count = 1
        var prior = response.priorResponse
        while (prior != null) {
            count++
            prior = prior.priorResponse
        }
        return count
    }
}
