package com.kodass.groovework

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.kodass.groovework.ui.AppRoot
import com.kodass.groovework.ui.common.LocalServerUrl
import com.kodass.groovework.ui.theme.GrooveWorkTheme

class MainActivity : ComponentActivity() {
    private val container get() = (application as GrooveApp).container

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        handleIntent(intent)
        val container = container
        setContent {
            GrooveWorkTheme {
                val serverUrl by container.sessionManager.serverUrl.collectAsStateWithLifecycle()
                CompositionLocalProvider(LocalServerUrl provides serverUrl) {
                    AppRoot(container)
                }
            }
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        handleIntent(intent)
    }

    // Extras уведомлений: маршрут (чат/задача). Действия со звонком обрабатывает
    // отдельная CallActivity.
    private fun handleIntent(intent: Intent?) {
        intent ?: return
        intent.getStringExtra("route")?.let { route ->
            // «unit» — не маршрут навигации, а запрос открыть модалку текущего юнита.
            if (route == "unit") container.unitManager.requestShowSheet()
            else container.pendingRoute.value = route
        }
        // Тап по системному уведомлению FCM (приложение было в фоне/убито):
        // onMessageReceived не вызывается, data-поля приходят как extras
        // запускающего интента — навигируем к нужному чату/задаче по ним.
        when (intent.getStringExtra("type")) {
            "message" -> intent.getStringExtra("conversation_id")?.toLongOrNull()?.let {
                container.pendingRoute.value = "chat/$it"
            }
            "task" -> intent.getStringExtra("task_id")?.toLongOrNull()?.let {
                container.pendingRoute.value = "task/$it"
            }
        }
    }

    override fun onResume() {
        super.onResume()
        container.notificationCenter.appForeground.value = true
        container.gateway.setVisible(true)
    }

    override fun onPause() {
        container.notificationCenter.appForeground.value = false
        container.gateway.setVisible(false)
        super.onPause()
    }
}
