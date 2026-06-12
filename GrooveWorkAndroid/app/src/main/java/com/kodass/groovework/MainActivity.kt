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

    // Extras уведомлений: маршрут (чат/задача) или действие со звонком.
    private fun handleIntent(intent: Intent?) {
        intent ?: return
        intent.getStringExtra("route")?.let { container.pendingRoute.value = it }
        if (intent.hasExtra("answer_call_id")) {
            container.callManager.autoAcceptRequested.value = true
            container.callManager.callUiVisible.value = true
            intent.removeExtra("answer_call_id")
        }
        if (intent.getBooleanExtra("open_call", false)) {
            container.callManager.callUiVisible.value = true
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
