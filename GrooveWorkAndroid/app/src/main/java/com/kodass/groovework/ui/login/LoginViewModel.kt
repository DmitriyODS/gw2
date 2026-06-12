package com.kodass.groovework.ui.login

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.session.SessionManager
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

class LoginViewModel(private val session: SessionManager) : ViewModel() {
    var serverUrl by mutableStateOf(session.serverUrl.value)
    var login by mutableStateOf("")
    var password by mutableStateOf("")
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var retrySeconds by mutableIntStateOf(0)
        private set

    private var timerJob: Job? = null

    val canSubmit: Boolean
        get() = !loading && retrySeconds == 0 && login.isNotBlank() && password.isNotEmpty() && serverUrl.isNotBlank()

    fun submit() {
        if (!canSubmit) return
        viewModelScope.launch {
            loading = true
            error = null
            try {
                session.login(serverUrl, login.trim(), password)
            } catch (e: ApiException) {
                when (e.code) {
                    // Брутфорс-щит authsvc: 429 + retry_after_sec.
                    "TOO_MANY_ATTEMPTS" -> startRetryTimer(e.retryAfterSec ?: 30)
                    "COMPANY_DISABLED" -> error = "Компания отключена. Обратитесь к администратору."
                    "NETWORK_ERROR" -> error = "Сервер недоступен. Проверьте адрес и подключение."
                    else -> error = e.message
                }
            } catch (_: Exception) {
                error = "Не удалось войти"
            } finally {
                loading = false
            }
        }
    }

    private fun startRetryTimer(seconds: Int) {
        timerJob?.cancel()
        retrySeconds = seconds
        error = null
        timerJob = viewModelScope.launch {
            while (retrySeconds > 0) {
                delay(1000)
                retrySeconds -= 1
            }
        }
    }
}
