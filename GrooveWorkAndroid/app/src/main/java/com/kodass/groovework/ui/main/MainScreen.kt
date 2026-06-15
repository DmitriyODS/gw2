package com.kodass.groovework.ui.main

import android.Manifest
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.provider.Settings
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.background
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.Chat
import androidx.compose.material.icons.automirrored.filled.Chat
import androidx.compose.material.icons.filled.BarChart
import androidx.compose.material.icons.filled.Groups
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.TaskAlt
import androidx.compose.material.icons.outlined.BarChart
import androidx.compose.material.icons.outlined.Groups
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.Settings
import androidx.compose.material.icons.outlined.TaskAlt
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.core.content.ContextCompat
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.kodass.groovework.AppContainer
import com.kodass.groovework.ui.about.AboutScreen
import com.kodass.groovework.ui.chats.ChatScreen
import com.kodass.groovework.ui.chats.ChatsScreen
import com.kodass.groovework.ui.employees.EmployeesScreen
import com.kodass.groovework.ui.profile.ProfileScreen
import com.kodass.groovework.ui.settings.AiSettingsScreen
import com.kodass.groovework.ui.settings.BackupSettingsScreen
import com.kodass.groovework.ui.settings.GrooveSettingsScreen
import com.kodass.groovework.ui.settings.InviteSettingsScreen
import com.kodass.groovework.ui.settings.SettingsScreen
import com.kodass.groovework.ui.settings.WeekendSettingsScreen
import com.kodass.groovework.ui.settings.YougileCompanySettingsScreen
import com.kodass.groovework.ui.settings.YougileUserSettingsScreen
import com.kodass.groovework.ui.stats.StatsScreen
import com.kodass.groovework.ui.tasks.TaskDetailScreen
import com.kodass.groovework.ui.tasks.TasksScreen
import com.kodass.groovework.ui.units.UnitSheet

private data class TopLevelDestination(
    val route: String,
    val label: String,
    val icon: ImageVector,
    val selectedIcon: ImageVector,
)

// Порядок и состав пунктов нижней панели (она прокручивается — все сразу не влезают).
private val topLevelDestinations = listOf(
    TopLevelDestination("tasks", "Задачи", Icons.Outlined.TaskAlt, Icons.Filled.TaskAlt),
    TopLevelDestination("chats", "Чат", Icons.AutoMirrored.Outlined.Chat, Icons.AutoMirrored.Filled.Chat),
    TopLevelDestination("employees", "Сотрудники", Icons.Outlined.Groups, Icons.Filled.Groups),
    TopLevelDestination("stats", "Статистика", Icons.Outlined.BarChart, Icons.Filled.BarChart),
    TopLevelDestination("profile", "Профиль", Icons.Outlined.Person, Icons.Filled.Person),
    TopLevelDestination("settings", "Настройки", Icons.Outlined.Settings, Icons.Filled.Settings),
)

@Composable
fun MainScreen(container: AppContainer) {
    val navController = rememberNavController()
    val backStackEntry by navController.currentBackStackEntryAsState()
    val currentRoute = backStackEntry?.destination?.route
    val showBottomBar = topLevelDestinations.any { it.route == currentRoute }

    val conversations by container.messengerRepo.conversations.collectAsStateWithLifecycle()
    val totalUnread = conversations.sumOf { it.unreadCount }

    // Разрешение на уведомления — один раз при входе.
    val context = LocalContext.current
    val notifPermissionLauncher = rememberLauncherForActivityResult(
        ActivityResultContracts.RequestPermission()
    ) {}
    LaunchedEffect(Unit) {
        // Рантайм-разрешение POST_NOTIFICATIONS появилось в Android 13; на 12/12L
        // уведомления включены по умолчанию — спрашивать нечего.
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            val granted = ContextCompat.checkSelfPermission(
                context, Manifest.permission.POST_NOTIFICATIONS
            ) == PackageManager.PERMISSION_GRANTED
            if (!granted) notifPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
        }
    }

    // Полноэкранные уведомления (Android 14+): без них входящий звонок не
    // развернётся поверх заблокированного экрана. Системного runtime-диалога нет —
    // ведём пользователя в настройки. Спрашиваем один раз.
    var showFsiDialog by remember { mutableStateOf(false) }
    LaunchedEffect(Unit) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE &&
            !container.notifier.canUseFullScreenIntent()
        ) {
            val prefs = context.getSharedPreferences("calls_prefs", Context.MODE_PRIVATE)
            if (!prefs.getBoolean("fsi_asked", false)) showFsiDialog = true
        }
    }
    if (showFsiDialog) {
        AlertDialog(
            onDismissRequest = {
                context.getSharedPreferences("calls_prefs", Context.MODE_PRIVATE)
                    .edit().putBoolean("fsi_asked", true).apply()
                showFsiDialog = false
            },
            title = { Text("Звонки на заблокированном экране") },
            text = {
                Text(
                    "Чтобы видеть входящий вызов поверх заблокированного экрана, " +
                        "разрешите Groove Work показывать полноэкранные уведомления."
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    context.getSharedPreferences("calls_prefs", Context.MODE_PRIVATE)
                        .edit().putBoolean("fsi_asked", true).apply()
                    showFsiDialog = false
                    runCatching {
                        context.startActivity(
                            Intent(
                                Settings.ACTION_MANAGE_APP_USE_FULL_SCREEN_INTENT,
                                Uri.parse("package:${context.packageName}"),
                            )
                        )
                    }
                }) { Text("Разрешить") }
            },
            dismissButton = {
                TextButton(onClick = {
                    context.getSharedPreferences("calls_prefs", Context.MODE_PRIVATE)
                        .edit().putBoolean("fsi_asked", true).apply()
                    showFsiDialog = false
                }) { Text("Позже") }
            },
        )
    }

    // Тап по уведомлению → маршрут из MainActivity.handleIntent.
    LaunchedEffect(Unit) {
        container.pendingRoute.collect { route ->
            if (route != null) {
                container.pendingRoute.value = null
                navController.navigate(route) { launchSingleTop = true }
            }
        }
    }

    // Модалка текущего юнита: по тапу в плашке/уведомлении (флаг в UnitManager).
    val showUnitSheet by container.unitManager.showSheet.collectAsStateWithLifecycle()

    Scaffold(
        // Инсеты обрабатывают вложенные Scaffold экранов — иначе статус-бар учитывается дважды.
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        bottomBar = {
            if (showBottomBar) {
                ScrollableBottomBar(
                    destinations = topLevelDestinations,
                    currentRoute = currentRoute,
                    chatUnread = totalUnread,
                    onSelect = { route ->
                        navController.navigate(route) {
                            popUpTo(navController.graph.findStartDestination().id) { saveState = true }
                            launchSingleTop = true
                            restoreState = true
                        }
                    },
                )
            }
        },
    ) { innerPadding ->
        NavHost(
            navController = navController,
            startDestination = "tasks",
            modifier = Modifier.padding(innerPadding),
        ) {
            composable("chats") {
                ChatsScreen(
                    container = container,
                    onOpenChat = { id -> navController.navigate("chat/$id") },
                )
            }
            composable(
                route = "chat/{id}",
                arguments = listOf(navArgument("id") { type = NavType.LongType }),
            ) { entry ->
                ChatScreen(
                    container = container,
                    conversationId = entry.arguments?.getLong("id") ?: 0L,
                    onBack = { navController.popBackStack() },
                    onOpenTask = { id -> navController.navigate("task/$id") },
                )
            }
            composable("tasks") {
                TasksScreen(
                    container = container,
                    onOpenTask = { id -> navController.navigate("task/$id") },
                )
            }
            composable(
                route = "task/{id}",
                arguments = listOf(navArgument("id") { type = NavType.LongType }),
            ) { entry ->
                TaskDetailScreen(
                    container = container,
                    taskId = entry.arguments?.getLong("id") ?: 0L,
                    onBack = { navController.popBackStack() },
                )
            }
            composable("employees") {
                EmployeesScreen(
                    container = container,
                    onOpenChat = { id -> navController.navigate("chat/$id") },
                )
            }
            composable("stats") {
                StatsScreen(container = container)
            }
            composable("profile") {
                ProfileScreen(container = container)
            }
            composable("settings") {
                SettingsScreen(container = container, onOpen = { section ->
                    navController.navigate("settings/$section") { launchSingleTop = true }
                })
            }
            composable("settings/about") {
                AboutScreen(
                    container = container,
                    onBack = { navController.popBackStack() },
                    onOpenChat = { id -> navController.navigate("chat/$id") },
                )
            }
            composable("settings/weekends") {
                WeekendSettingsScreen(container = container, onBack = { navController.popBackStack() })
            }
            composable("settings/groove") {
                GrooveSettingsScreen(container = container, onBack = { navController.popBackStack() })
            }
            composable("settings/invite") {
                InviteSettingsScreen(container = container, onBack = { navController.popBackStack() })
            }
            composable("settings/ai") {
                AiSettingsScreen(container = container, onBack = { navController.popBackStack() })
            }
            composable("settings/yougile") {
                YougileUserSettingsScreen(container = container, onBack = { navController.popBackStack() })
            }
            composable("settings/yougile-company") {
                YougileCompanySettingsScreen(container = container, onBack = { navController.popBackStack() })
            }
            composable("settings/backup") {
                BackupSettingsScreen(container = container, onBack = { navController.popBackStack() })
            }
        }
    }

    if (showUnitSheet) {
        UnitSheet(
            container = container,
            onOpenTask = { id -> navController.navigate("task/$id") { launchSingleTop = true } },
            onDismiss = { container.unitManager.consumeShowSheet() },
        )
    }
}

// Прокручиваемая нижняя панель (пунктов больше, чем влезает по ширине).
@Composable
private fun ScrollableBottomBar(
    destinations: List<TopLevelDestination>,
    currentRoute: String?,
    chatUnread: Int,
    onSelect: (String) -> Unit,
) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceContainer,
        tonalElevation = 3.dp,
    ) {
        androidx.compose.foundation.layout.Row(
            modifier = Modifier
                .fillMaxWidth()
                .horizontalScroll(rememberScrollState())
                .navigationBarsPadding()
                .height(72.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            destinations.forEach { destination ->
                BottomBarItem(
                    destination = destination,
                    selected = currentRoute == destination.route,
                    badge = if (destination.route == "chats") chatUnread else 0,
                    onClick = { onSelect(destination.route) },
                )
            }
        }
    }
}

@Composable
private fun BottomBarItem(
    destination: TopLevelDestination,
    selected: Boolean,
    badge: Int,
    onClick: () -> Unit,
) {
    Column(
        modifier = Modifier
            .width(84.dp)
            .fillMaxHeight()
            .selectable(selected = selected, role = Role.Tab, onClick = onClick),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = androidx.compose.foundation.layout.Arrangement.Center,
    ) {
        val icon = if (selected) destination.selectedIcon else destination.icon
        Box(
            modifier = Modifier
                .height(32.dp)
                .width(56.dp)
                .clip(RoundedCornerShape(16.dp))
                .then(
                    if (selected) Modifier.background(MaterialTheme.colorScheme.secondaryContainer)
                    else Modifier
                ),
            contentAlignment = Alignment.Center,
        ) {
            val tint = if (selected) MaterialTheme.colorScheme.onSecondaryContainer
            else MaterialTheme.colorScheme.onSurfaceVariant
            if (badge > 0) {
                BadgedBox(badge = { Badge { Text(badge.toString()) } }) {
                    Icon(icon, contentDescription = destination.label, tint = tint)
                }
            } else {
                Icon(icon, contentDescription = destination.label, tint = tint)
            }
        }
        Text(
            text = destination.label,
            style = MaterialTheme.typography.labelMedium,
            color = if (selected) MaterialTheme.colorScheme.onSurface
            else MaterialTheme.colorScheme.onSurfaceVariant,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.padding(top = 4.dp),
        )
    }
}
