package com.kodass.groovework.ui.common

import java.security.SecureRandom

// Безопасный пароль без двусмысленных символов (как генератор веб-регистрации).
private const val PASSWORD_ALPHABET = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
private val secureRandom = SecureRandom()

fun generatePassword(length: Int = 12): String =
    buildString(length) {
        repeat(length) { append(PASSWORD_ALPHABET[secureRandom.nextInt(PASSWORD_ALPHABET.length)]) }
    }
