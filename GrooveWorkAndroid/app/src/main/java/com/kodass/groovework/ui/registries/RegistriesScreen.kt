package com.kodass.groovework.ui.registries

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.outlined.TableChart
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kodass.groovework.AppContainer
import com.kodass.groovework.data.dto.RegistryDto
import com.kodass.groovework.ui.common.CenteredLoading
import com.kodass.groovework.ui.common.EmptyState
import com.kodass.groovework.ui.common.ErrorState
import com.kodass.groovework.ui.common.RefreshOnResume

// Уровень 1 раздела «Реестры»: список реестров сверху вниз. По тапу открывается
// уровень 2 с записями выбранного реестра.
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RegistriesScreen(
    container: AppContainer,
    onOpenRegistry: (registryId: Long) -> Unit,
) {
    val viewModel: RegistriesListViewModel = viewModel {
        RegistriesListViewModel(container.registriesRepo, container.sessionManager, container.gateway)
    }
    RefreshOnResume { viewModel.load(initial = false) }

    Scaffold(topBar = { TopAppBar(title = { Text("Реестры") }) }) { padding ->
        Box(modifier = Modifier.fillMaxSize().padding(padding)) {
            when {
                viewModel.loading && viewModel.registries.isEmpty() -> CenteredLoading()
                viewModel.error != null && viewModel.registries.isEmpty() ->
                    ErrorState(viewModel.error!!, onRetry = { viewModel.load(initial = true) })
                viewModel.registries.isEmpty() ->
                    EmptyState("Реестров пока нет", "Создайте реестр в веб-версии — он появится здесь.")
                else -> PullToRefreshBox(
                    isRefreshing = false,
                    onRefresh = { viewModel.load(initial = false) },
                ) {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(16.dp),
                        verticalArrangement = Arrangement.spacedBy(10.dp),
                    ) {
                        items(viewModel.registries, key = { it.id }) { registry ->
                            RegistryRow(registry = registry, onClick = { onOpenRegistry(registry.id) })
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun RegistryRow(registry: RegistryDto, onClick: () -> Unit) {
    Surface(
        onClick = onClick,
        shape = MaterialTheme.shapes.large,
        color = MaterialTheme.colorScheme.surfaceContainerLow,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 16.dp),
        ) {
            Icon(
                Icons.Outlined.TableChart,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.primary,
            )
            Text(
                registry.name,
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.SemiBold,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f).padding(start = 14.dp),
            )
            Icon(
                Icons.AutoMirrored.Filled.KeyboardArrowRight,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.size(24.dp),
            )
        }
    }
}
