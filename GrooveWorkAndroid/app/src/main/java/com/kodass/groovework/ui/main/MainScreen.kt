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
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.Chat
import androidx.compose.material.icons.automirrored.filled.Chat
import androidx.compose.material.icons.filled.BarChart
import androidx.compose.material.icons.filled.Groups
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.MoreHoriz
import androidx.compose.material.icons.filled.Palette
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.outlined.BarChart
import androidx.compose.material.icons.outlined.Groups
import androidx.compose.material.icons.outlined.Info
import androidx.compose.material.icons.outlined.MoreHoriz
import androidx.compose.material.icons.outlined.Palette
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material.icons.outlined.TableChart
import androidx.compose.material.icons.filled.TableChart
import androidx.compose.material.icons.outlined.TaskAlt
import androidx.compose.material.icons.filled.TaskAlt
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.ListItem
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.core.content.ContextCompat
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavController
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.kodass.groovework.AppContainer
import com.kodass.groovework.DeepLink
import com.kodass.groovework.ui.about.AboutScreen
import com.kodass.groovework.ui.appearance.AppearanceScreen
import com.kodass.groovework.ui.chats.ChatScreen
import com.kodass.groovework.ui.chats.ChatsScreen
import com.kodass.groovework.ui.companies.AcceptInviteScreen
import com.kodass.groovework.ui.employees.EmployeesScreen
import com.kodass.groovework.ui.profile.ProfileScreen
import com.kodass.groovework.ui.registries.RegistriesScreen
import com.kodass.groovework.ui.registries.RegistryRecordScreen
import com.kodass.groovework.ui.registries.RegistryRecordsScreen
import com.kodass.groovework.ui.stats.StatsScreen
import com.kodass.groovework.ui.tasks.TaskDetailScreen
import com.kodass.groovework.ui.tasks.TasksScreen
import com.kodass.groovework.ui.units.UnitSheet
import kotlinx.coroutines.launch

private data class TopLevelDestination(
    val route: String,
    val label: String,
    val icon: ImageVector,
    val selectedIcon: ImageVector,
)

// Постоянно видимые пункты нижней панели.
private val primaryDestinations = listOf(
    TopLevelDestination("tasks", "Задачи", Icons.Outlined.TaskAlt, Icons.Filled.TaskAlt),
    TopLevelDestination("chats", "Чат", Icons.AutoMirrored.Outlined.Chat, Icons.AutoMirrored.Filled.Chat),
    TopLevelDestination("profile", "Профиль", Icons.Outlined.Person, Icons.Filled.Person),
)

// Вторичные пункты — прячутся под «Ещё» (M3: nav bar держит 3–5 пунктов, остальное
// в overflow-меню), доступны через нижний лист.
private val overflowDestinations = listOf(
    TopLevelDestination("registries", "Реестры", Icons.Outlined.TableChart, Icons.Filled.TableChart),
    TopLevelDestination("employees", "Сотрудники", Icons.Outlined.Groups, Icons.Filled.Groups),
    TopLevelDestination("stats", "Статистика", Icons.Outlined.BarChart, Icons.Filled.BarChart),
    TopLevelDestination("appearance", "Оформление", Icons.Outlined.Palette, Icons.Filled.Palette),
    TopLevelDestination("about", "О приложении", Icons.Outlined.Info, Icons.Filled.Info),
)

// «appearance» — не вкладка, а вложенный экран с кнопкой «назад» и guard'ом на
// выход (нижняя панель на нём скрыта).
private val bottomBarRoutes =
    (primaryDestinations + overflowDestinations).map { it.route }.toSet() - "appearance"

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MainScreen(container: AppContainer) {
    val navController = rememberNavController()
    val backStackEntry by navController.currentBackStackEntryAsState()
    val currentRoute = backStackEntry?.destination?.route
    val showBottomBar = currentRoute in bottomBarRoutes

    val conversations by container.messengerRepo.conversations.collectAsStateWithLifecycle()
    val totalUnread = conversations.sumOf { it.unreadCount }

    var showMoreSheet by remember { mutableStateOf(false) }
    val moreSheetState = rememberModalBottomSheetState()
    val scope = rememberCoroutineScope()

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

    // Deep link приглашения (App Links) — открываем превью; verify/reset уже не
    // актуальны для вошедшего пользователя, просто гасим.
    LaunchedEffect(Unit) {
        container.pendingDeepLink.collect { link ->
            when (link) {
                is DeepLink.Invite -> {
                    container.pendingDeepLink.value = null
                    navController.navigate("invite/${link.token}") { launchSingleTop = true }
                }
                null -> {}
                else -> container.pendingDeepLink.value = null
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
                NavigationBar {
                    primaryDestinations.forEach { destination ->
                        NavigationBarItem(
                            selected = currentRoute == destination.route,
                            onClick = { navController.navigateTop(destination.route) },
                            icon = {
                                BottomBarIcon(
                                    destination = destination,
                                    selected = currentRoute == destination.route,
                                    badge = if (destination.route == "chats") totalUnread else 0,
                                )
                            },
                            label = { Text(destination.label) },
                        )
                    }
                    val moreSelected = currentRoute in overflowDestinations.map { it.route }
                    NavigationBarItem(
                        selected = moreSelected,
                        onClick = { showMoreSheet = true },
                        icon = {
                            Icon(
                                if (moreSelected) Icons.Filled.MoreHoriz else Icons.Outlined.MoreHoriz,
                                contentDescription = "Ещё",
                            )
                        },
                        label = { Text("Ещё") },
                    )
                }
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
            composable("registries") {
                RegistriesScreen(
                    container = container,
                    onOpenRegistry = { registryId -> navController.navigate("registry/$registryId") },
                )
            }
            composable(
                route = "registry/{registryId}",
                arguments = listOf(navArgument("registryId") { type = NavType.LongType }),
            ) { entry ->
                RegistryRecordsScreen(
                    container = container,
                    registryId = entry.arguments?.getLong("registryId") ?: 0L,
                    onBack = { navController.popBackStack() },
                    onOpenRecord = { registryId, recordId ->
                        navController.navigate("registry_record/$registryId/$recordId")
                    },
                )
            }
            composable(
                route = "registry_record/{registryId}/{recordId}",
                arguments = listOf(
                    navArgument("registryId") { type = NavType.LongType },
                    navArgument("recordId") { type = NavType.LongType },
                ),
            ) { entry ->
                RegistryRecordScreen(
                    container = container,
                    registryId = entry.arguments?.getLong("registryId") ?: 0L,
                    recordId = entry.arguments?.getLong("recordId") ?: 0L,
                    onBack = { navController.popBackStack() },
                )
            }
            composable("appearance") {
                AppearanceScreen(container = container, onBack = { navController.popBackStack() })
            }
            composable("profile") {
                ProfileScreen(container = container)
            }
            composable("about") {
                AboutScreen(
                    container = container,
                    onOpenChat = { id -> navController.navigate("chat/$id") },
                )
            }
            composable(
                route = "invite/{token}",
                arguments = listOf(navArgument("token") { type = NavType.StringType }),
            ) { entry ->
                AcceptInviteScreen(
                    container = container,
                    token = entry.arguments?.getString("token").orEmpty(),
                    onAccepted = {
                        navController.navigate("tasks") {
                            popUpTo(navController.graph.findStartDestination().id) { inclusive = false }
                            launchSingleTop = true
                        }
                    },
                    onBack = { navController.popBackStack() },
                )
            }
        }
    }

    if (showMoreSheet) {
        ModalBottomSheet(
            onDismissRequest = { showMoreSheet = false },
            sheetState = moreSheetState,
        ) {
            Column(modifier = Modifier.fillMaxWidth().navigationBarsPadding()) {
                overflowDestinations.forEach { destination ->
                    val selected = currentRoute == destination.route
                    ListItem(
                        headlineContent = { Text(destination.label) },
                        leadingContent = {
                            Icon(
                                if (selected) destination.selectedIcon else destination.icon,
                                contentDescription = null,
                            )
                        },
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable {
                                // appearance — вложенный экран (push с back), остальные — вкладки.
                                if (destination.route == "appearance") {
                                    navController.navigate("appearance") { launchSingleTop = true }
                                } else {
                                    navController.navigateTop(destination.route)
                                }
                                scope.launch { moreSheetState.hide() }.invokeOnCompletion {
                                    if (!moreSheetState.isVisible) showMoreSheet = false
                                }
                            },
                    )
                }
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

// Переход к верхнеуровневому разделу с сохранением/восстановлением состояния вкладок.
private fun NavController.navigateTop(route: String) {
    navigate(route) {
        popUpTo(graph.findStartDestination().id) { saveState = true }
        launchSingleTop = true
        restoreState = true
    }
}

@Composable
private fun BottomBarIcon(destination: TopLevelDestination, selected: Boolean, badge: Int) {
    val icon = if (selected) destination.selectedIcon else destination.icon
    if (badge > 0) {
        BadgedBox(badge = { Badge { Text(badge.toString()) } }) {
            Icon(icon, contentDescription = destination.label)
        }
    } else {
        Icon(icon, contentDescription = destination.label)
    }
}
