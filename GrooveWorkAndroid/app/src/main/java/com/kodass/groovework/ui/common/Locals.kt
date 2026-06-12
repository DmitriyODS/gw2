package com.kodass.groovework.ui.common

import androidx.compose.runtime.staticCompositionLocalOf

// Базовый адрес сервера для абсолютных URL картинок (/uploads, identicon).
val LocalServerUrl = staticCompositionLocalOf { "" }

fun avatarUrl(serverUrl: String, userId: Long, avatarPath: String?): String =
    if (!avatarPath.isNullOrBlank()) {
        "$serverUrl/uploads/${avatarPath.trimStart('/')}"
    } else {
        "$serverUrl/api/users/$userId/identicon"
    }
