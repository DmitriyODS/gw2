package com.kodass.groovework

import android.app.KeyguardManager
import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.kodass.groovework.data.calls.CallPhase
import com.kodass.groovework.ui.calls.AnswerCallGate
import com.kodass.groovework.ui.calls.CallScreen
import com.kodass.groovework.ui.calls.IncomingCallScreen
import com.kodass.groovework.ui.common.LocalServerUrl
import com.kodass.groovework.ui.theme.GrooveWorkTheme

// Выделенная активность звонка: показывается ПОВЕРХ локскрина (входящий из
// пуша), но не открывает доступ ко всему приложению (фикс #3). Хостит весь UI
// звонка; закрывается, когда звонок завершён или свёрнут.
class CallActivity : ComponentActivity() {
    private val container get() = (application as GrooveApp).container

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setShowWhenLocked(true)
        setTurnScreenOn(true)
        enableEdgeToEdge()
        container.callManager.callUiVisible.value = true
        handleIntent(intent)
        setContent {
            GrooveWorkTheme {
                val serverUrl by container.sessionManager.serverUrl.collectAsStateWithLifecycle()
                CompositionLocalProvider(LocalServerUrl provides serverUrl) {
                    CallHost(container) { finish() }
                }
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        handleIntent(intent)
    }

    private fun handleIntent(intent: Intent?) {
        intent ?: return
        container.callManager.callUiVisible.value = true
        if (intent.hasExtra("answer_call_id")) {
            // Ответ с кнопки уведомления: снимаем локскрин и просим разрешения.
            getSystemService(KeyguardManager::class.java)?.requestDismissKeyguard(this, null)
            container.callManager.requestAnswer(
                callId = intent.getLongExtra("answer_call_id", 0),
                video = intent.getBooleanExtra("answer_call_video", false),
                fio = intent.getStringExtra("answer_call_fio"),
            )
            intent.removeExtra("answer_call_id")
        }
    }
}

@Composable
private fun CallHost(container: AppContainer, onFinished: () -> Unit) {
    val phase by container.callManager.phase.collectAsStateWithLifecycle()
    val callUiVisible by container.callManager.callUiVisible.collectAsStateWithLifecycle()
    val pendingAnswer by container.callManager.pendingAnswer.collectAsStateWithLifecycle()

    // Закрыть активность, когда звонок завершён или свёрнут (баннер «вернуться»
    // в приложении поднимет её снова).
    LaunchedEffect(phase, callUiVisible) {
        val active = phase is CallPhase.Outgoing || phase is CallPhase.Active
        if (phase is CallPhase.Idle || (active && !callUiVisible)) onFinished()
    }

    Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
        when {
            pendingAnswer != null -> AnswerCallGate(container, pendingAnswer!!)
            phase is CallPhase.Incoming -> IncomingCallScreen(container)
            phase is CallPhase.Outgoing || phase is CallPhase.Active -> CallScreen(container)
            else -> {}
        }
    }
}
