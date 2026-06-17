package com.kodass.groovework.ui.theme

import kotlinx.serialization.Serializable
import kotlin.random.Random

// Режим оформления — как на вебе (gw_theme_mode): следовать системе либо
// принудительно светлая/тёмная.
enum class ThemeMode { LIGHT, DARK, SYSTEM }

// Палитра темы: четыре seed-цвета (как в конструкторе на фронте). Из них
// grooveColorScheme собирает полную M3-схему для светлого и тёмного режима.
@Serializable
data class ThemePalette(
    val primary: String,
    val secondary: String,
    val tertiary: String,
    val neutral: String,
)

// Сохранённая пользователем тема (своя или сгенерированная) — имя + палитра.
@Serializable
data class SavedTheme(val name: String, val palette: ThemePalette)

// Готовый пресет: ключ (для подсветки выбранного), название и палитра.
data class ThemePreset(val key: String, val label: String, val palette: ThemePalette)

// Пресеты повторяют набор веб-версии (utils/groove.js / stores/theme.js),
// плюс фирменный «Бренд» по умолчанию — синий волны логотипа.
val THEME_PRESETS: List<ThemePreset> = listOf(
    ThemePreset("brand", "Бренд", ThemePalette("#1195ed", "#51606f", "#66597b", "#eef1f6")),
    ThemePreset("classic", "Классика", ThemePalette("#9b4dff", "#00bfa5", "#3d6ce7", "#ece8f2")),
    ThemePreset("blue", "Синяя", ThemePalette("#1e88e5", "#00acc1", "#7e57c2", "#e6ecf4")),
    ThemePreset("pink", "Розовая", ThemePalette("#ec4899", "#e91e63", "#ce93d8", "#f5e8ee")),
    ThemePreset("red", "Красная", ThemePalette("#e53935", "#ff7043", "#f06292", "#f4e6e3")),
    ThemePreset("green", "Зелёная", ThemePalette("#2e7d32", "#00897b", "#26a69a", "#e6eee7")),
    ThemePreset("orange", "Оранжевая", ThemePalette("#ef6c00", "#ff6d00", "#fdd835", "#f5ebde")),
    ThemePreset("yellow", "Жёлтая", ThemePalette("#c98300", "#fb8c00", "#43a047", "#f4eedb")),
    ThemePreset("violet", "Фиолетовая", ThemePalette("#7c4dff", "#00b0ff", "#e040fb", "#ebe6f5")),
    ThemePreset("lilac", "Сиреневая", ThemePalette("#9b59b6", "#c77daa", "#7da87e", "#f0e8ef")),
    ThemePreset("sunset", "Закат", ThemePalette("#e8806e", "#e8a07a", "#db8398", "#f1e9dc")),
    ThemePreset("ocean", "Океан", ThemePalette("#0277bd", "#26c6da", "#5e92f3", "#e3edf2")),
    ThemePreset("mint", "Мята", ThemePalette("#16a085", "#1abc9c", "#7fb3a4", "#e4efea")),
    ThemePreset("coffee", "Кофе", ThemePalette("#795548", "#a1887f", "#d4a373", "#efe8e0")),
    ThemePreset("midnight", "Полночь", ThemePalette("#5e7fff", "#7c3aed", "#2dd4bf", "#e6e9f2")),
    ThemePreset("forest", "Лес", ThemePalette("#2f7d4f", "#558b2f", "#a5a96d", "#e6ece2")),
)

val DEFAULT_PRESET: ThemePreset = THEME_PRESETS.first()

// Подбирает ключ пресета по палитре (для подсветки активной плитки); null —
// если палитра кастомная (собрана в конструкторе).
fun presetKeyFor(palette: ThemePalette): String? =
    THEME_PRESETS.firstOrNull { it.palette == palette }?.key

// Случайная гармоничная палитра: базовый оттенок + аналоговые/комплементарные
// сдвиги, умеренная насыщенность; нейтраль — лёгкий тинт того же тона.
fun randomPalette(): ThemePalette {
    val base = Random.nextDouble(0.0, 360.0)
    val c = Random.nextDouble(0.11, 0.16)
    val shift = listOf(150.0, 180.0, 210.0, -150.0).random()
    return ThemePalette(
        primary = oklchHex(0.58, c, base),
        secondary = oklchHex(0.6, c * 0.85, (base + shift) % 360.0),
        tertiary = oklchHex(0.6, c * 0.9, (base + shift / 2.0) % 360.0),
        neutral = oklchHex(0.93, 0.01, base),
    )
}
