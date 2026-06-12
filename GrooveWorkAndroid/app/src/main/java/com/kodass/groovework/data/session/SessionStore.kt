package com.kodass.groovework.data.session

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.first

private val Context.sessionDataStore by preferencesDataStore(name = "session")

// Персистентная часть сессии: адрес сервера и refresh-токен (аналог HttpOnly-cookie веба).
class SessionStore(private val context: Context) {
    private val keyServerUrl = stringPreferencesKey("server_url")
    private val keyRefreshToken = stringPreferencesKey("refresh_token")

    suspend fun serverUrl(): String? = context.sessionDataStore.data.first()[keyServerUrl]

    suspend fun refreshToken(): String? = context.sessionDataStore.data.first()[keyRefreshToken]

    suspend fun setServerUrl(url: String) {
        context.sessionDataStore.edit { it[keyServerUrl] = url }
    }

    suspend fun setRefreshToken(token: String?) {
        context.sessionDataStore.edit {
            if (token == null) it.remove(keyRefreshToken) else it[keyRefreshToken] = token
        }
    }
}
