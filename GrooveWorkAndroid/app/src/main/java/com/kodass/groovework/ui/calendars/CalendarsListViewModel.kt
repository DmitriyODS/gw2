package com.kodass.groovework.ui.calendars

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.kodass.groovework.data.dto.CalendarDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.repo.CalendarsRepository
import com.kodass.groovework.data.session.SessionManager
import com.kodass.groovework.data.ws.GatewayClient
import com.kodass.groovework.data.ws.longField
import com.kodass.groovework.ui.registries.companyMatches
import kotlinx.coroutines.launch

// Первый уровень раздела «Календари» — список календарей компании.
class CalendarsListViewModel(
    private val repo: CalendarsRepository,
    private val session: SessionManager,
    gateway: GatewayClient,
) : ViewModel() {

    var calendars by mutableStateOf<List<CalendarDto>>(emptyList())
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
                    "calendar:created", "calendar:updated", "calendar:deleted" ->
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
                calendars = repo.calendars()
            } catch (e: ApiException) {
                if (calendars.isEmpty()) error = e.message
            } finally {
                loading = false
            }
        }
    }
}
