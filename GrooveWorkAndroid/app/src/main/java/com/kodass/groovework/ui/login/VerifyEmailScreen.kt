package com.kodass.groovework.ui.login

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.MarkEmailRead
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.session.SessionManager
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

class VerifyEmailViewModel(
    private val session: SessionManager,
    val email: String,
    token: String,
) : ViewModel() {
    var code by mutableStateOf("")
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var resendSeconds by mutableIntStateOf(0)
        private set

    private var timerJob: Job? = null

    val canSubmit: Boolean
        get() = !loading && code.trim().length == 6

    init {
        // Переход по ссылке из письма — подтверждаем токеном автоматически.
        if (token.isNotBlank()) verifyToken(token)
    }

    private fun verifyToken(token: String) {
        viewModelScope.launch {
            loading = true
            error = null
            try {
                session.verifyEmail(token = token)
            } catch (e: ApiException) {
                error = mapError(e)
            } catch (_: Exception) {
                error = "Не удалось подтвердить"
            } finally {
                loading = false
            }
        }
    }

    fun submit() {
        if (!canSubmit) return
        viewModelScope.launch {
            loading = true
            error = null
            try {
                session.verifyEmail(email = email, code = code.trim())
            } catch (e: ApiException) {
                error = mapError(e)
            } catch (_: Exception) {
                error = "Не удалось подтвердить"
            } finally {
                loading = false
            }
        }
    }

    fun resend() {
        if (resendSeconds > 0 || email.isBlank()) return
        viewModelScope.launch {
            error = null
            runCatching { session.resendVerification(email) }
            startTimer(60)
        }
    }

    private fun mapError(e: ApiException): String = when (e.code) {
        "INVALID_VERIFICATION" -> "Неверный код подтверждения"
        "VERIFICATION_EXPIRED" -> "Код истёк — запросите новый"
        "TOO_MANY_ATTEMPTS" -> "Слишком много попыток — запросите новый код"
        else -> e.message
    }

    private fun startTimer(seconds: Int) {
        timerJob?.cancel()
        resendSeconds = seconds
        timerJob = viewModelScope.launch {
            while (resendSeconds > 0) {
                delay(1000)
                resendSeconds -= 1
            }
        }
    }
}

@Composable
fun VerifyEmailScreen(container: AppContainer, email: String, token: String, onBack: () -> Unit) {
    val viewModel: VerifyEmailViewModel = viewModel {
        VerifyEmailViewModel(container.sessionManager, email, token)
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .imePadding()
            .padding(horizontal = 32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Icon(
            Icons.Filled.MarkEmailRead,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.primary,
            modifier = Modifier.size(56.dp),
        )
        Text(
            text = "Подтвердите email",
            style = MaterialTheme.typography.headlineSmall,
            modifier = Modifier.padding(top = 16.dp),
        )
        Text(
            text = if (viewModel.email.isNotBlank())
                "Мы отправили 6-значный код на ${viewModel.email}. Введите его, чтобы завершить регистрацию."
            else "Введите код подтверждения из письма.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(top = 8.dp, bottom = 24.dp),
        )

        OutlinedTextField(
            value = viewModel.code,
            onValueChange = { v -> viewModel.code = v.filter { it.isDigit() }.take(6) },
            label = { Text("Код из письма") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.NumberPassword, imeAction = ImeAction.Done),
            modifier = Modifier.fillMaxWidth(),
        )

        viewModel.error?.let { error ->
            Text(
                text = error,
                color = MaterialTheme.colorScheme.error,
                style = MaterialTheme.typography.bodyMedium,
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(top = 16.dp),
            )
        }

        Button(
            onClick = { viewModel.submit() },
            enabled = viewModel.canSubmit,
            modifier = Modifier.fillMaxWidth().padding(top = 24.dp).height(52.dp),
        ) {
            if (viewModel.loading) {
                CircularProgressIndicator(
                    modifier = Modifier.size(22.dp),
                    color = MaterialTheme.colorScheme.onPrimary,
                    strokeWidth = 2.dp,
                )
            } else {
                Text("Подтвердить")
            }
        }

        TextButton(
            onClick = { viewModel.resend() },
            enabled = viewModel.resendSeconds == 0 && !viewModel.loading,
            modifier = Modifier.padding(top = 8.dp),
        ) {
            Text(
                if (viewModel.resendSeconds > 0) "Отправить ещё раз через ${viewModel.resendSeconds} с"
                else "Отправить код ещё раз"
            )
        }

        TextButton(onClick = onBack) { Text("Назад") }
    }
}
