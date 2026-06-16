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
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.outlined.Lock
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.session.SessionManager
import kotlinx.coroutines.launch

class ResetPasswordViewModel(
    private val session: SessionManager,
    private val token: String,
) : ViewModel() {
    var password by mutableStateOf("")
    var confirm by mutableStateOf("")
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var done by mutableStateOf(false)
        private set

    val validToken: Boolean get() = token.isNotBlank()

    val canSubmit: Boolean
        get() = !loading && validToken && password.length >= 8 && confirm.isNotEmpty()

    fun submit() {
        error = null
        if (password.length < 8) {
            error = "Пароль должен содержать не менее 8 символов"
            return
        }
        if (password != confirm) {
            error = "Пароли не совпадают"
            return
        }
        viewModelScope.launch {
            loading = true
            try {
                session.resetPassword(token, password)
                done = true
            } catch (e: ApiException) {
                error = if (e.code == "INVALID_RESET") "Ссылка недействительна или истекла" else e.message
            } catch (_: Exception) {
                error = "Не удалось сменить пароль"
            } finally {
                loading = false
            }
        }
    }
}

@Composable
fun ResetPasswordScreen(container: AppContainer, token: String, onDone: () -> Unit) {
    val viewModel: ResetPasswordViewModel = viewModel {
        ResetPasswordViewModel(container.sessionManager, token)
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
        if (viewModel.done) {
            Icon(
                Icons.Filled.CheckCircle,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.primary,
                modifier = Modifier.size(56.dp),
            )
            Text(
                text = "Пароль изменён",
                style = MaterialTheme.typography.headlineSmall,
                modifier = Modifier.padding(top = 16.dp),
            )
            Text(
                text = "Войдите с новым паролем.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 8.dp),
            )
            Button(onClick = onDone, modifier = Modifier.fillMaxWidth().padding(top = 24.dp).height(52.dp)) {
                Text("Войти")
            }
            return@Column
        }

        Text(
            text = "Новый пароль",
            style = MaterialTheme.typography.headlineSmall,
        )
        Text(
            text = if (viewModel.validToken) "Придумайте новый пароль для входа."
            else "Ссылка сброса недействительна. Запросите новую на экране входа.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(top = 8.dp, bottom = 24.dp),
        )

        if (viewModel.validToken) {
            OutlinedTextField(
                value = viewModel.password,
                onValueChange = { viewModel.password = it },
                label = { Text("Новый пароль") },
                leadingIcon = { Icon(Icons.Outlined.Lock, contentDescription = null) },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Next),
                supportingText = { Text("Не менее 8 символов") },
                modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                value = viewModel.confirm,
                onValueChange = { viewModel.confirm = it },
                label = { Text("Повторите пароль") },
                leadingIcon = { Icon(Icons.Outlined.Lock, contentDescription = null) },
                singleLine = true,
                visualTransformation = PasswordVisualTransformation(),
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Done),
                modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
            )
        }

        viewModel.error?.let { error ->
            Text(
                text = error,
                color = MaterialTheme.colorScheme.error,
                style = MaterialTheme.typography.bodyMedium,
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(top = 16.dp),
            )
        }

        if (viewModel.validToken) {
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
                    Text("Сменить пароль")
                }
            }
        }

        TextButton(onClick = onDone, modifier = Modifier.padding(top = 8.dp)) {
            Text("Ко входу")
        }
    }
}
