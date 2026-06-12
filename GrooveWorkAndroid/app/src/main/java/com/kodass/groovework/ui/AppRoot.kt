package com.kodass.groovework.ui

import android.content.Intent
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
import androidx.compose.material3.ExperimentalMaterial3ExpressiveApi
import androidx.compose.material3.Icon
import androidx.compose.material3.LoadingIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.clickable
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.kodass.groovework.AppContainer
import com.kodass.groovework.R
import com.kodass.groovework.data.calls.CallPhase
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.service.GatewayOnlineService
import com.kodass.groovework.ui.calls.CallScreen
import com.kodass.groovework.ui.calls.IncomingCallScreen
import com.kodass.groovework.ui.login.ChangeDefaultScreen
import com.kodass.groovework.ui.login.LoginScreen
import com.kodass.groovework.ui.main.MainScreen

@Composable
fun AppRoot(container: AppContainer) {
    val authState by container.sessionManager.authState.collectAsStateWithLifecycle()
    val context = LocalContext.current

    // WS живёт в foreground-сервисе: события и уведомления доходят и в фоне.
    val loggedIn = authState is AuthState.LoggedIn
    LaunchedEffect(loggedIn) {
        val intent = Intent(context, GatewayOnlineService::class.java)
        if (loggedIn) {
            context.startForegroundService(intent)
        } else {
            context.stopService(intent)
            container.gateway.stop()
        }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
            when (val state = authState) {
                AuthState.Loading -> SplashScreen()
                AuthState.LoggedOut -> LoginScreen(container)
                is AuthState.LoggedIn ->
                    if (state.claims.forceChange) {
                        ChangeDefaultScreen(container)
                    } else {
                        MainScreen(container)
                    }
            }
        }

        // Поверх всего — звонок: входящий, полноэкранный или баннер возврата.
        val callPhase by container.callManager.phase.collectAsStateWithLifecycle()
        val callUiVisible by container.callManager.callUiVisible.collectAsStateWithLifecycle()
        when (callPhase) {
            is CallPhase.Incoming -> IncomingCallScreen(container)
            is CallPhase.Outgoing, is CallPhase.Active -> {
                if (callUiVisible) {
                    CallScreen(container)
                } else {
                    ReturnToCallBanner(
                        onClick = { container.callManager.callUiVisible.value = true },
                        modifier = Modifier.align(Alignment.TopCenter),
                    )
                }
            }
            CallPhase.Idle -> {}
        }
    }
}

@Composable
private fun ReturnToCallBanner(onClick: () -> Unit, modifier: Modifier = Modifier) {
    Surface(
        color = MaterialTheme.colorScheme.primary,
        shape = RoundedCornerShape(24.dp),
        modifier = modifier
            .statusBarsPadding()
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
