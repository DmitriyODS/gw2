package com.kodass.groovework.ui.registries

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.dto.RegistryDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.repo.RegistriesRepository
import com.kodass.groovework.data.session.AuthState
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.longField
import kotlinx.coroutines.launch

// Первый уровень раздела «Реестры» — просто список реестров компании.
class RegistriesListViewModel(
    private val repo: RegistriesRepository,
    private val session: SessionManager,
    gateway: GatewayClient,
) : ViewModel() {

    var registries by mutableStateOf<List<RegistryDto>>(emptyList())
        private set
    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set

    init {
        load(initial = true)
        viewModelScope.launch {
            gateway.events.collect { event ->
                when (event.event) {
                    "registry:created", "registry:updated", "registry:deleted" ->
                        if (companyMatches(session, event.data.longField("company_id"))) load(initial = false)
                }
            }
        }
    }

    fun load(initial: Boolean) {
        viewModelScope.launch {
            if (initial) loading = true
            error = null
            try {
                registries = repo.registries()
            } catch (e: ApiException) {
                if (registries.isEmpty()) error = e.message
            } finally {
                loading = false
            }
        }
    }
}

// Событие нашей активной компании? (null company_id у клиента → пропускаем все).
internal fun companyMatches(session: SessionManager, eventCompanyId: Long?): Boolean {
    val mine = (session.authState.value as? AuthState.LoggedIn)?.claims?.companyId ?: return true
    return eventCompanyId == null || eventCompanyId == mine
}
