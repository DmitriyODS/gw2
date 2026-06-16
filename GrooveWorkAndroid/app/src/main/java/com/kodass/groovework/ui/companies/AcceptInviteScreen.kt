package com.kodass.groovework.ui.companies

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Apartment
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.InvitePreviewDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.ui.common.CenteredLoading
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json

class AcceptInviteViewModel(
    private val session: SessionManager,
    private val json: Json,
    private val token: String,
) : ViewModel() {
    var preview by mutableStateOf<InvitePreviewDto?>(null)
        private set
    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var accepting by mutableStateOf(false)
        private set

    init { load() }

    fun load() {
        viewModelScope.launch {
            loading = true
            error = null
            try {
                preview = apiCall(json) { session.companiesApi.invitePreview(token) }
            } catch (e: ApiException) {
                error = e.message
            } catch (_: Exception) {
                error = "Не удалось загрузить приглашение"
            } finally {
                loading = false
            }
        }
    }

    fun accept(onAccepted: () -> Unit) {
        viewModelScope.launch {
            accepting = true
            error = null
            try {
                session.acceptInvite(token)
                onAccepted()
            } catch (e: ApiException) {
                error = e.message
            } catch (_: Exception) {
                error = "Не удалось принять приглашение"
            } finally {
                accepting = false
            }
        }
    }
}

@Composable
fun AcceptInviteScreen(container: AppContainer, token: String, onAccepted: () -> Unit, onBack: () -> Unit) {
    val viewModel: AcceptInviteViewModel = viewModel {
        AcceptInviteViewModel(container.sessionManager, container.json, token)
    }

    Column(
        modifier = Modifier.fillMaxSize().padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        when {
            viewModel.loading -> CenteredLoading()
            viewModel.preview == null -> {
                Text(
                    text = viewModel.error ?: "Приглашение недействительно",
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center,
                )
                Button(onClick = onBack, modifier = Modifier.padding(top = 20.dp)) { Text("Назад") }
            }
            else -> {
                val preview = viewModel.preview!!
                Icon(
                    Icons.Filled.Apartment,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.size(56.dp),
                )
                Text(
                    text = "Приглашение в компанию",
                    style = MaterialTheme.typography.headlineSmall,
                    modifier = Modifier.padding(top = 16.dp),
                )
                Text(
                    text = "${preview.companyName} · роль «${preview.roleName}»",
                    style = MaterialTheme.typography.titleMedium,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.padding(top = 8.dp),
                )
                viewModel.error?.let {
                    Text(
                        it,
                        color = MaterialTheme.colorScheme.error,
                        style = MaterialTheme.typography.bodyMedium,
                        textAlign = TextAlign.Center,
                        modifier = Modifier.padding(top = 12.dp),
                    )
                }
                Button(
                    onClick = { viewModel.accept(onAccepted) },
                    enabled = !viewModel.accepting,
                    modifier = Modifier.fillMaxWidth().padding(top = 24.dp).height(52.dp),
                ) {
                    if (viewModel.accepting) {
                        CircularProgressIndicator(modifier = Modifier.size(22.dp), color = MaterialTheme.colorScheme.onPrimary, strokeWidth = 2.dp)
                    } else {
                        Text("Принять приглашение")
                    }
                }
                TextButton(onClick = onBack, modifier = Modifier.padding(top = 8.dp)) { Text("Не сейчас") }
            }
        }
    }
}
