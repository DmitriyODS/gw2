package com.kodass.groovework.ui

import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Call
import androidx.compose.material.icons.filled.CloudOff
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.Icon
import androidx.compose.material3.LoadingIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import android.widget.Toast
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
import kotlinx.coroutines.launch
import androidx.compose.ui.Alignment
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.clickable
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.kodass.groovework.AppContainer
import com.kodass.groovework.R
import com.kodass.groovework.calls.CallDirection
import com.kodass.groovework.calls.CallState
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.ui.login.AuthFlow
import com.kodass.groovework.ui.login.ChangeDefaultScreen
import com.kodass.groovework.ui.main.MainScreen

@Composable
fun AppRoot(container: AppContainer) {
    val authState by container.sessionManager.authState.collectAsStateWithLifecycle()

    // Жизненный цикл WebSocket ведёт AppContainer (foreground/звонок); в фоне
    // уведомления доставляет FCM — постоянный foreground-сервис больше не нужен.

    // Ошибки звонка — пользователю (иначе сбой соединения был незаметен).
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    LaunchedEffect(Unit) {
        container.callController.errors.collect { message ->
            Toast.makeText(context, message, Toast.LENGTH_SHORT).show()
        }
    }
    // Ошибки операций с юнитом (старт/клон/стоп) — пользователю.
    LaunchedEffect(Unit) {
        container.unitManager.errors.collect { message ->
            Toast.makeText(context, message, Toast.LENGTH_SHORT).show()
        }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
            when (val state = authState) {
                AuthState.Loading -> SplashScreen()
                AuthState.LoggedOut -> AuthFlow(container)
                AuthState.Offline -> OfflineScreen(
                    onRetry = { scope.launch { container.sessionManager.retryBootstrap() } },
                )
                is AuthState.LoggedIn ->
                    if (state.claims.forceChange) {
                        ChangeDefaultScreen(container)
                    } else {
                        MainScreen(container)
                    }
            }
        }

        // Сам экран звонка живёт в отдельной CallActivity (поверх локскрина).
        // Здесь — только баннеры поверх приложения: возврат к свёрнутому звонку
        // и плашка текущего юнита (с отсчётом времени).
        val callUi by container.callController.ui.collectAsStateWithLifecycle()
        val callUiVisible by container.callController.callUiVisible.collectAsStateWithLifecycle()
        val callState = callUi.state
        val callActive = callState is CallState.Dialing || callState is CallState.Connecting ||
            callState is CallState.Active ||
            (callState is CallState.Ringing && callState.direction == CallDirection.Outgoing)
        val authed = authState is AuthState.LoggedIn
        Column(
            modifier = Modifier.align(Alignment.TopCenter).statusBarsPadding(),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            if (callActive && !callUiVisible) {
                ReturnToCallBanner(onClick = { container.callController.showCallUi() })
            }
            if (authed) {
                com.kodass.groovework.ui.units.UnitBanner(container = container)
            }
        }
    }
}

@Composable
private fun ReturnToCallBanner(onClick: () -> Unit, modifier: Modifier = Modifier) {
    Surface(
        color = MaterialTheme.colorScheme.primary,
        shape = RoundedCornerShape(24.dp),
        modifier = modifier
            .padding(top = 8.dp)
            .clickable(onClick = onClick),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
        ) {
            Icon(
                Icons.Filled.Call,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onPrimary,
                modifier = Modifier.size(18.dp),
            )
            Text(
                text = "Вернуться к звонку",
                color = MaterialTheme.colorScheme.onPrimary,
                style = MaterialTheme.typography.labelLarge,
                modifier = Modifier.padding(start = 8.dp),
            )
        }
    }
}

@Composable
private fun OfflineScreen(onRetry: () -> Unit) {
    Column(
        modifier = Modifier.fillMaxSize().padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Icon(
            imageVector = Icons.Filled.CloudOff,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.size(56.dp),
        )
        Text(
            text = "Нет подключения к интернету",
            style = MaterialTheme.typography.titleMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(top = 16.dp),
        )
        Text(
            text = "Проверьте соединение и попробуйте снова.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.outline,
            modifier = Modifier.padding(top = 4.dp),
        )
        Button(onClick = onRetry, modifier = Modifier.padding(top = 20.dp)) {
            Text("Повторить")
        }
    }
}

@OptIn(ExperimentalMaterial3ExpressiveApi::class)
@Composable
private fun SplashScreen() {
    Column(
        modifier = Modifier.fillMaxSize(),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Image(
            painter = painterResource(R.drawable.logo_groove),
            contentDescription = null,
            modifier = Modifier.size(96.dp),
        )
        Text(
            text = "Groove Work",
            style = MaterialTheme.typography.headlineMedium,
            modifier = Modifier.padding(top = 16.dp),
        )
        LoadingIndicator(modifier = Modifier.padding(top = 24.dp))
    }
}
