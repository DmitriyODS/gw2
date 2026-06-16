package com.kodass.groovework.ui.companies

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Apartment
import androidx.compose.material.icons.filled.ChecklistRtl
import androidx.compose.material.icons.filled.Groups
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewModelScope
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.api.CompaniesApi
import com.kodass.groovework.data.dto.CompanyCreateRequest
import com.kodass.groovework.data.dto.CompanyDto
import com.kodass.groovework.data.network.ApiException
import com.kodass.groovework.data.network.apiCall
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.EmptyState
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.RefreshOnResume
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json

class CompaniesViewModel(
    private val api: CompaniesApi,
    private val json: Json,
) : ViewModel() {
    var companies by mutableStateOf<List<CompanyDto>>(emptyList())
        private set
    var loading by mutableStateOf(true)
        private set
    var error by mutableStateOf<String?>(null)
        private set
    var creating by mutableStateOf(false)
        private set
    var createError by mutableStateOf<String?>(null)

    fun load() {
        viewModelScope.launch {
            if (companies.isEmpty()) loading = true
            error = null
            try {
                companies = apiCall(json) { api.mine() }.items
            } catch (e: ApiException) {
                if (companies.isEmpty()) error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun create(name: String, description: String, onCreated: (Long) -> Unit) {
        if (name.isBlank()) {
            createError = "Введите название"
            return
        }
        viewModelScope.launch {
            creating = true
            createError = null
            try {
                val company = apiCall(json) {
                    api.create(CompanyCreateRequest(name.trim(), description.trim().ifBlank { null }))
                }
                load()
                onCreated(company.id)
            } catch (e: ApiException) {
                createError = e.message
            } finally {
                creating = false
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CompaniesScreen(
    container: AppContainer,
    onBack: () -> Unit,
    onOpenCompany: (Long) -> Unit,
) {
    val viewModel: CompaniesViewModel = viewModel { CompaniesViewModel(container.companiesApi, container.json) }
    RefreshOnResume { viewModel.load() }
    val claims = container.sessionManager.claimsOrNull()
    var showCreate by remember { mutableStateOf(false) }

    if (showCreate) {
        CreateCompanyDialog(
            viewModel = viewModel,
            onDismiss = { showCreate = false },
            onCreated = { id -> showCreate = false; onOpenCompany(id) },
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Мои компании") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Назад")
                    }
                },
            )
        },
        floatingActionButton = {
            ExtendedFloatingActionButton(
                onClick = { showCreate = true },
                icon = { Icon(Icons.Filled.Add, contentDescription = null) },
                text = { Text("Создать") },
            )
        },
    ) { padding ->
        when {
            viewModel.loading && viewModel.companies.isEmpty() -> CenteredLoading(Modifier.padding(padding))
            viewModel.error != null && viewModel.companies.isEmpty() ->
                ErrorState(viewModel.error ?: "", onRetry = { viewModel.load() }, modifier = Modifier.padding(padding))
            viewModel.companies.isEmpty() -> EmptyState(
                title = "Компаний нет",
                subtitle = "Создайте свою — вы станете её администратором.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.fillMaxSize().padding(padding),
                contentPadding = PaddingValues(16.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                items(viewModel.companies, key = { it.id }) { company ->
                    CompanyCard(
                        company = company,
                        isCreator = company.createdBy == claims?.userId,
                        onClick = { onOpenCompany(company.id) },
                    )
                }
            }
        }
    }
}

@Composable
private fun CompanyCard(company: CompanyDto, isCreator: Boolean, onClick: () -> Unit) {
    Card(
        onClick = onClick,
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Filled.Apartment, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                Text(
                    text = company.name,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    modifier = Modifier.weight(1f).padding(start = 10.dp),
                )
                RoleBadge(if (isCreator) "Создатель" else "Администратор", isCreator)
            }
            if (!company.isActive) {
                Text(
                    "Компания отключена",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error,
                    modifier = Modifier.padding(top = 6.dp),
                )
            }
            Row(
                modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
                horizontalArrangement = Arrangement.spacedBy(20.dp),
            ) {
                Counter(Icons.Filled.Groups, company.employeesCount, "сотрудников")
                Counter(Icons.Filled.ChecklistRtl, company.tasksCount, "задач")
            }
        }
    }
}

@Composable
private fun Counter(icon: androidx.compose.ui.graphics.vector.ImageVector, value: Int, label: String) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(icon, contentDescription = null, tint = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(end = 6.dp))
        Text("$value ", style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.Bold)
        Text(label, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@Composable
internal fun RoleBadge(text: String, highlight: Boolean) {
    val bg = if (highlight) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.secondaryContainer
    val fg = if (highlight) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSecondaryContainer
    Surface(color = bg, contentColor = fg, shape = RoundedCornerShape(8.dp)) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.SemiBold,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp),
        )
    }
}

@Composable
private fun CreateCompanyDialog(
    viewModel: CompaniesViewModel,
    onDismiss: () -> Unit,
    onCreated: (Long) -> Unit,
) {
    var name by remember { mutableStateOf("") }
    var description by remember { mutableStateOf("") }
    AlertDialog(
        onDismissRequest = { if (!viewModel.creating) onDismiss() },
        title = { Text("Новая компания") },
        text = {
            Column {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    label = { Text("Название") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = description,
                    onValueChange = { description = it },
                    label = { Text("Описание (необязательно)") },
                    modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                )
                viewModel.createError?.let {
                    Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall, modifier = Modifier.padding(top = 8.dp))
                }
            }
        },
        confirmButton = {
            TextButton(
                onClick = { viewModel.create(name, description, onCreated) },
                enabled = !viewModel.creating,
            ) { Text(if (viewModel.creating) "Создаю…" else "Создать") }
        },
        dismissButton = { TextButton(onClick = onDismiss, enabled = !viewModel.creating) { Text("Отмена") } },
    )
}
