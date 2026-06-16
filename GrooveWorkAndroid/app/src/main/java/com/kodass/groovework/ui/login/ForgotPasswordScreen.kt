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
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.MarkEmailRead
import androidx.compose.material.icons.outlined.AlternateEmail
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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.session.SessionManager
import kotlinx.coroutines.launch

class ForgotPasswordViewModel(private val session: SessionManager) : ViewModel() {
    var email by mutableStateOf("")
    var loading by mutableStateOf(false)
        private set
    // Ответ всегда ok (наличие аккаунта не раскрываем) — показываем «проверьте почту».
    var sent by mutableStateOf(false)
        private set

    val canSubmit: Boolean
        get() = !loading && email.isNotBlank()

    fun submit() {
        if (!canSubmit) return
        viewModelScope.launch {
            loading = true
            try {
                runCatching { session.forgotPassword(email.trim()) }
                sent = true
            } finally {
                loading = false
            }
        }
    }
}

@Composable
fun ForgotPasswordScreen(container: AppContainer, onBack: () -> Unit) {
    val viewModel: ForgotPasswordViewModel = viewModel { ForgotPasswordViewModel(container.sessionManager) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .imePadding()
            .padding(horizontal = 32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        if (viewModel.sent) {
            Icon(
                Icons.Filled.MarkEmailRead,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.primary,
                modifier = Modifier.size(56.dp),
            )
            Text(
                text = "Проверьте почту",
                style = MaterialTheme.typography.headlineSmall,
                modifier = Modifier.padding(top = 16.dp),
            )
            Text(
                text = "Если аккаунт с таким email существует, мы отправили на него ссылку для сброса пароля.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(top = 8.dp),
            )
            Button(onClick = onBack, modifier = Modifier.fillMaxWidth().padding(top = 24.dp).height(52.dp)) {
                Text("Вернуться ко входу")
            }
            return@Column
        }

        Text(
            text = "Восстановление пароля",
            style = MaterialTheme.typography.headlineSmall,
        )
        Text(
            text = "Укажите email аккаунта — пришлём ссылку для установки нового пароля.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(top = 8.dp, bottom = 24.dp),
        )
        OutlinedTextField(
            value = viewModel.email,
            onValueChange = { viewModel.email = it },
            label = { Text("Email") },
            leadingIcon = { Icon(Icons.Outlined.AlternateEmail, contentDescription = null) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email, imeAction = ImeAction.Done),
            keyboardActions = KeyboardActions(onDone = { viewModel.submit() }),
            modifier = Modifier.fillMaxWidth(),
        )
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
                Text("Отправить ссылку")
            }
        }
        TextButton(onClick = onBack, modifier = Modifier.padding(top = 8.dp)) {
            Text("Назад ко входу")
        }
    }
}
