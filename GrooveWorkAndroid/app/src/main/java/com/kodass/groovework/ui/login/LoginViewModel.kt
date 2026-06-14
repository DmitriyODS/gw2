package com.kodass.groovework.ui.login

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.dto.MembershipDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.session.LoginResult
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

    // Многокомпанийный вход: после login показываем пикер компаний (см. submit).
    var companyChoices by mutableStateOf<List<MembershipDto>?>(null)
        private set
    private var selectToken: String = ""

    private var timerJob: Job? = null

    val canSubmit: Boolean
        get() = !loading && retrySeconds == 0 && login.isNotBlank() && password.isNotEmpty() && serverUrl.isNotBlank()

    fun submit() {
        if (!canSubmit) return
        viewModelScope.launch {
            loading = true
            error = null
            try {
                when (val result = session.login(serverUrl, login.trim(), password)) {
                    LoginResult.Success -> {}
                    is LoginResult.NeedsCompany -> {
                        selectToken = result.selectToken
                        companyChoices = result.companies
                    }
                }
            } catch (e: ApiException) {
                handleApiError(e)
            } catch (_: Exception) {
                error = "Не удалось войти"
            } finally {
                loading = false
            }
        }
    }

    fun selectCompany(companyId: Long) {
        if (loading) return
        viewModelScope.launch {
            loading = true
            error = null
            try {
                session.selectCompany(selectToken, companyId)
            } catch (e: ApiException) {
                // Истёкший select-токен — вернуть на форму входа.
                if (e.code == "INVALID_TOKEN") {
                    companyChoices = null
                    selectToken = ""
                }
                handleApiError(e)
            } catch (_: Exception) {
                error = "Не удалось войти"
            } finally {
                loading = false
            }
        }
    }

    fun cancelCompanySelection() {
        companyChoices = null
        selectToken = ""
        error = null
    }

    private fun handleApiError(e: ApiException) {
        when (e.code) {
            // Брутфорс-щит authsvc: 429 + retry_after_sec.
            "TOO_MANY_ATTEMPTS" -> startRetryTimer(e.retryAfterSec ?: 30)
            "NO_COMPANY_ACCESS" -> error = "Нет доступа ни к одной компании. Обратитесь к администратору."
            "COMPANY_DISABLED" -> error = "Компания отключена. Обратитесь к администратору."
            "NETWORK_ERROR" -> error = "Сервер недоступен. Проверьте адрес и подключение."
            else -> error = e.message
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
