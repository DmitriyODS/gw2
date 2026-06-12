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
import androidx.compose.material.icons.outlined.Badge
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

class ChangeDefaultViewModel(private val session: SessionManager) : ViewModel() {
    var newLogin by mutableStateOf("")
    var newPassword by mutableStateOf("")
    var confirmPassword by mutableStateOf("")
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    val canSubmit: Boolean
        get() = !loading && newLogin.trim().length >= 3 && newPassword.length >= 8 && confirmPassword.isNotEmpty()

    fun submit() {
        if (!canSubmit) return
        if (newPassword != confirmPassword) {
            error = "Пароли не совпадают"
            return
        }
        viewModelScope.launch {
            loading = true
            error = null
            try {
                session.changeDefault(newLogin.trim(), newPassword, confirmPassword)
            } catch (e: ApiException) {
                error = when (e.code) {
                    "LOGIN_TAKEN" -> "Такой логин уже занят"
                    "PASSWORDS_MISMATCH" -> "Пароли не совпадают"
                    else -> e.message
                }
            } catch (_: Exception) {
                error = "Не удалось сменить пароль"
            } finally {
                loading = false
            }
        }
    }

    fun logout() {
        viewModelScope.launch { session.logout() }
    }
}

// Принудительная смена дефолтного пароля: все API отвечают 403 FORCE_PASSWORD_CHANGE,
// пока пользователь не задаст свои логин и пароль.
@Composable
fun ChangeDefaultScreen(container: AppContainer) {
    val viewModel: ChangeDefaultViewModel = viewModel { ChangeDefaultViewModel(container.sessionManager) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .imePadding()
            .padding(horizontal = 32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text(
            text = "Смена пароля",
            style = MaterialTheme.typography.headlineMedium,
        )
        Text(
            text = "Вы вошли с временным паролем. Придумайте собственный логин и пароль, чтобы продолжить.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(top = 8.dp, bottom = 24.dp),
        )

        OutlinedTextField(
            value = viewModel.newLogin,
            onValueChange = { viewModel.newLogin = it },
            label = { Text("Новый логин") },
            supportingText = { Text("Не менее 3 символов") },
            leadingIcon = { Icon(Icons.Outlined.Badge, contentDescription = null) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
            modifier = Modifier.fillMaxWidth(),
        )
        OutlinedTextField(
            value = viewModel.newPassword,
            onValueChange = { viewModel.newPassword = it },
            label = { Text("Новый пароль") },
            supportingText = { Text("Не менее 8 символов") },
            leadingIcon = { Icon(Icons.Outlined.Lock, contentDescription = null) },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Next),
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp),
        )
        OutlinedTextField(
            value = viewModel.confirmPassword,
            onValueChange = { viewModel.confirmPassword = it },
            label = { Text("Повторите пароль") },
            leadingIcon = { Icon(Icons.Outlined.Lock, contentDescription = null) },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Done),
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp),
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
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 24.dp)
                .height(52.dp),
        ) {
            if (viewModel.loading) {
                CircularProgressIndicator(
                    modifier = Modifier.size(22.dp),
                    color = MaterialTheme.colorScheme.onPrimary,
                    strokeWidth = 2.dp,
                )
            } else {
                Text("Сохранить")
            }
        }

        TextButton(onClick = { viewModel.logout() }, modifier = Modifier.padding(top = 8.dp)) {
            Text("Выйти")
        }
    }
}
