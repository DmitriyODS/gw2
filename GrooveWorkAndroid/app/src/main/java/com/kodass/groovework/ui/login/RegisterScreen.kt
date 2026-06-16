package com.kodass.groovework.ui.login

import android.widget.Toast
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
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
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Visibility
import androidx.compose.material.icons.filled.VisibilityOff
import androidx.compose.material.icons.outlined.AlternateEmail
import androidx.compose.material.icons.outlined.Badge
import androidx.compose.material.icons.outlined.Lock
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.R
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.ui.common.generatePassword
import com.kodass.groovework.ui.common.rememberClipboardCopy
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

class RegisterViewModel(private val session: SessionManager) : ViewModel() {
    var fio by mutableStateOf("")
        private set
    var email by mutableStateOf("")
    var login by mutableStateOf("")
        private set
    var password by mutableStateOf(generatePassword())
    var loading by mutableStateOf(false)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    private var loginTouched = false
    private var suggestJob: Job? = null

    val canSubmit: Boolean
        get() = !loading && fio.isNotBlank() && email.isNotBlank() && password.length >= 8

    fun onFioChange(value: String) {
        fio = value
        if (loginTouched) return
        suggestJob?.cancel()
        suggestJob = viewModelScope.launch {
            delay(400)
            val f = value.trim()
            if (loginTouched || f.isEmpty()) return@launch
            runCatching { session.suggestLogin(f) }.getOrNull()?.let {
                if (!loginTouched && it.isNotBlank()) login = it
            }
        }
    }

    fun onLoginChange(value: String) {
        login = value
        loginTouched = true
    }

    fun regeneratePassword() {
        password = generatePassword()
    }

    fun submit(onRegistered: (String) -> Unit) {
        error = null
        if (login.isNotBlank() && login.trim().length < 3) {
            error = "Логин должен содержать не менее 3 символов"
            return
        }
        if (password.length < 8) {
            error = "Пароль должен содержать не менее 8 символов"
            return
        }
        viewModelScope.launch {
            loading = true
            try {
                val resultEmail = session.register(
                    serverUrl = session.serverUrl.value,
                    fio = fio.trim(),
                    email = email.trim(),
                    login = login.trim(),
                    password = password,
                )
                onRegistered(resultEmail)
            } catch (e: ApiException) {
                error = e.message
            } catch (_: Exception) {
                error = "Не удалось зарегистрироваться"
            } finally {
                loading = false
            }
        }
    }
}

@Composable
fun RegisterScreen(container: AppContainer, onBack: () -> Unit, onRegistered: (String) -> Unit) {
    val viewModel: RegisterViewModel = viewModel { RegisterViewModel(container.sessionManager) }
    val context = LocalContext.current
    val copyToClipboard = rememberClipboardCopy()
    var passwordVisible by remember { mutableStateOf(true) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .imePadding()
            .padding(horizontal = 32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Image(
            painter = painterResource(R.drawable.logo_groove),
            contentDescription = null,
            modifier = Modifier.size(72.dp).padding(top = 16.dp),
        )
        Text(
            text = "Создать аккаунт",
            style = MaterialTheme.typography.headlineSmall,
            modifier = Modifier.padding(top = 16.dp),
        )
        Text(
            text = "Заполните ФИО и почту — логин и пароль подставятся автоматически, при желании поправьте.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(top = 4.dp, bottom = 20.dp),
        )

        OutlinedTextField(
            value = viewModel.fio,
            onValueChange = { viewModel.onFioChange(it) },
            label = { Text("ФИО") },
            placeholder = { Text("Фамилия Имя Отчество") },
            leadingIcon = { Icon(Icons.Outlined.Person, contentDescription = null) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
            modifier = Modifier.fillMaxWidth(),
        )
        OutlinedTextField(
            value = viewModel.email,
            onValueChange = { viewModel.email = it },
            label = { Text("Email") },
            placeholder = { Text("name@example.com") },
            leadingIcon = { Icon(Icons.Outlined.AlternateEmail, contentDescription = null) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email, imeAction = ImeAction.Next),
            modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
        )
        OutlinedTextField(
            value = viewModel.login,
            onValueChange = { viewModel.onLoginChange(it) },
            label = { Text("Логин") },
            placeholder = { Text("Сгенерируется из ФИО") },
            leadingIcon = { Icon(Icons.Outlined.Badge, contentDescription = null) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
            modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
        )
        OutlinedTextField(
            value = viewModel.password,
            onValueChange = { viewModel.password = it },
            label = { Text("Пароль") },
            leadingIcon = { Icon(Icons.Outlined.Lock, contentDescription = null) },
            trailingIcon = {
                Row {
                    IconButton(onClick = { viewModel.regeneratePassword() }) {
                        Icon(Icons.Filled.Refresh, contentDescription = "Сгенерировать новый")
                    }
                    IconButton(onClick = {
                        copyToClipboard(viewModel.password)
                        Toast.makeText(context, "Пароль скопирован", Toast.LENGTH_SHORT).show()
                    }) {
                        Icon(Icons.Filled.ContentCopy, contentDescription = "Скопировать")
                    }
                    IconButton(onClick = { passwordVisible = !passwordVisible }) {
                        Icon(
                            if (passwordVisible) Icons.Filled.VisibilityOff else Icons.Filled.Visibility,
                            contentDescription = if (passwordVisible) "Скрыть" else "Показать",
                        )
                    }
                }
            },
            singleLine = true,
            visualTransformation = if (passwordVisible) VisualTransformation.None else PasswordVisualTransformation(),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = ImeAction.Done),
            supportingText = { Text("Сохраните пароль — он понадобится для входа") },
            modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
        )

        viewModel.error?.let { error ->
            Text(
                text = error,
                color = MaterialTheme.colorScheme.error,
                style = MaterialTheme.typography.bodyMedium,
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(top = 12.dp),
            )
        }

        Button(
            onClick = { viewModel.submit(onRegistered) },
            enabled = viewModel.canSubmit,
            modifier = Modifier.fillMaxWidth().padding(top = 20.dp).height(52.dp),
        ) {
            if (viewModel.loading) {
                CircularProgressIndicator(
                    modifier = Modifier.size(22.dp),
                    color = MaterialTheme.colorScheme.onPrimary,
                    strokeWidth = 2.dp,
                )
            } else {
                Text("Зарегистрироваться")
            }
        }

        TextButton(onClick = onBack, modifier = Modifier.padding(top = 8.dp)) {
            Text("Уже есть аккаунт? Войти")
        }
    }
}
