package com.kodass.groovework.ui.tasks

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

// 8 пастельных цветов-тегов задач — набор продублирован с domain.TaskColors
// tasksvc и front/src/utils/taskColors.js.
val TaskColorNames = listOf("red", "orange", "amber", "green", "teal", "blue", "violet", "pink")

data class TaskTagColors(val container: Color, val accent: Color)

private val lightTagColors = mapOf(
    "red" to TaskTagColors(Color(0xFFFFDAD6), Color(0xFFBA1A1A)),
    "orange" to TaskTagColors(Color(0xFFFFDCC5), Color(0xFF9A4B00)),
    "amber" to TaskTagColors(Color(0xFFFFE9B3), Color(0xFF8A6400)),
    "green" to TaskTagColors(Color(0xFFD7F5D0), Color(0xFF2E7D32)),
    "teal" to TaskTagColors(Color(0xFFCCF1EC), Color(0xFF00796B)),
    "blue" to TaskTagColors(Color(0xFFD6E8FF), Color(0xFF1565C0)),
    "violet" to TaskTagColors(Color(0xFFE9DDFF), Color(0xFF6A4FA3)),
    "pink" to TaskTagColors(Color(0xFFFFD8EC), Color(0xFFC2185B)),
)

private val darkTagColors = mapOf(
    "red" to TaskTagColors(Color(0xFF5C1F1B), Color(0xFFFFB4AB)),
    "orange" to TaskTagColors(Color(0xFF5C3A1A), Color(0xFFFFB77C)),
    "amber" to TaskTagColors(Color(0xFF584400), Color(0xFFF2C94C)),
    "green" to TaskTagColors(Color(0xFF1F4D24), Color(0xFF98D783)),
    "teal" to TaskTagColors(Color(0xFF0E4A43), Color(0xFF7FD4C7)),
    "blue" to TaskTagColors(Color(0xFF1C3F61), Color(0xFF9FCAFF)),
    "violet" to TaskTagColors(Color(0xFF453266), Color(0xFFCFBCFF)),
    "pink" to TaskTagColors(Color(0xFF5C2342), Color(0xFFFFAFD3)),
)

@Composable
fun taskTagColors(name: String?): TaskTagColors? {
    if (name.isNullOrBlank()) return null
    return if (isSystemInDarkTheme()) darkTagColors[name] else lightTagColors[name]
}

@Composable
fun taskAccentColor(name: String?): Color? = taskTagColors(name)?.accent
