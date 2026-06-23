package com.kodass.groovework.ui.login

import android.net.Uri
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import com.kodass.groovework.AppContainer
import com.kodass.groovework.DeepLink

// Навигация неавторизованных экранов: вход, регистрация, подтверждение email,
// сброс пароля. Deep link verify/reset из писем (App Links) ведёт сюда; invite
// остаётся pending до входа (его подхватит MainScreen).
@Composable
fun AuthFlow(container: AppContainer) {
    val navController = rememberNavController()

    LaunchedEffect(Unit) {
        container.pendingDeepLink.collect { link ->
            when (link) {
                is DeepLink.VerifyEmail -> {
                    container.pendingDeepLink.value = null
                    navController.navigate("verify?email=${Uri.encode(link.email)}&token=${link.token}")
                }
                is DeepLink.ResetPassword -> {
                    container.pendingDeepLink.value = null
                    navController.navigate("reset?token=${link.token}")
                }
                // Invite требует авторизации — оставляем pending для MainScreen.
                else -> {}
            }
        }
    }

    NavHost(navController = navController, startDestination = "login") {
        composable("login") {
            LoginScreen(
                container = container,
                onRegister = { navController.navigate("register") },
                onForgot = { navController.navigate("forgot") },
                onNeedVerify = { email -> navController.navigate("verify?email=${Uri.encode(email)}") },
            )
        }
        composable("register") {
            RegisterScreen(
                container = container,
                onBack = { navController.popBackStack() },
                onRegistered = { email ->
                    navController.navigate("verify?email=${Uri.encode(email)}") {
                        popUpTo("login")
                    }
                },
            )
        }
        composable(
            route = "verify?email={email}&token={token}",
            arguments = listOf(
                navArgument("email") { type = NavType.StringType; defaultValue = "" },
                navArgument("token") { type = NavType.StringType; defaultValue = "" },
            ),
        ) { entry ->
            VerifyEmailScreen(
                container = container,
                email = entry.arguments?.getString("email").orEmpty(),
                token = entry.arguments?.getString("token").orEmpty(),
                onBack = { navController.popBackStack() },
            )
        }
        composable("forgot") {
            ForgotPasswordScreen(
                container = container,
                onBack = { navController.popBackStack() },
            )
        }
        composable(
            route = "reset?token={token}",
            arguments = listOf(navArgument("token") { type = NavType.StringType; defaultValue = "" }),
        ) { entry ->
            ResetPasswordScreen(
                container = container,
                token = entry.arguments?.getString("token").orEmpty(),
                onDone = { navController.popBackStack("login", inclusive = false) },
            )
        }
    }
}
