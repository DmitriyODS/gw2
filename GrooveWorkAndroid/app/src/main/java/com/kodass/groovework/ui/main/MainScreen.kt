package com.kodass.groovework.ui.main

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.Chat
import androidx.compose.material.icons.automirrored.filled.Chat
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.TaskAlt
import androidx.compose.material.icons.outlined.Info
import androidx.compose.material.icons.outlined.TaskAlt
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
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
import com.kodass.groovework.ui.tasks.TaskDetailScreen
import com.kodass.groovework.ui.tasks.TasksScreen

private data class TopLevelDestination(
    val route: String,
    val label: String,
    val icon: ImageVector,
    val selectedIcon: ImageVector,
)

private val topLevelDestinations = listOf(
    TopLevelDestination("chats", "Чаты", Icons.AutoMirrored.Outlined.Chat, Icons.AutoMirrored.Filled.Chat),
    TopLevelDestination("tasks", "Задачи", Icons.Outlined.TaskAlt, Icons.Filled.TaskAlt),
    TopLevelDestination("about", "О приложении", Icons.Outlined.Info, Icons.Filled.Info),
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
        val granted = ContextCompat.checkSelfPermission(
            context, Manifest.permission.POST_NOTIFICATIONS
        ) == PackageManager.PERMISSION_GRANTED
        if (!granted) notifPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
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

    Scaffold(
        // Инсеты обрабатывают вложенные Scaffold экранов — иначе статус-бар учитывается дважды.
        contentWindowInsets = WindowInsets(0, 0, 0, 0),
        bottomBar = {
            if (showBottomBar) {
                NavigationBar {
                    topLevelDestinations.forEach { destination ->
                        val selected = currentRoute == destination.route
                        NavigationBarItem(
                            selected = selected,
                            onClick = {
                                navController.navigate(destination.route) {
                                    popUpTo(navController.graph.findStartDestination().id) {
                                        saveState = true
                                    }
                                    launchSingleTop = true
                                    restoreState = true
                                }
                            },
                            icon = {
                                val icon = if (selected) destination.selectedIcon else destination.icon
                                if (destination.route == "chats" && totalUnread > 0) {
                                    BadgedBox(badge = { Badge { Text(totalUnread.toString()) } }) {
                                        Icon(icon, contentDescription = destination.label)
                                    }
                                } else {
                                    Icon(icon, contentDescription = destination.label)
                                }
                            },
                            label = { Text(destination.label) },
                        )
                    }
                }
            }
        },
    ) { innerPadding ->
        NavHost(
            navController = navController,
            startDestination = "chats",
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
            composable("about") {
                AboutScreen(container = container)
            }
        }
    }
}
