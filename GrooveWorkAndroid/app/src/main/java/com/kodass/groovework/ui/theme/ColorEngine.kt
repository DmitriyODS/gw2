package com.kodass.groovework.ui.theme

import androidx.compose.material3.ColorScheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.ui.graphics.Color
import kotlin.math.atan2
import kotlin.math.cbrt
import kotlin.math.cos
import kotlin.math.pow
import kotlin.math.roundToInt
import kotlin.math.sin

// Генерация полной M3-схемы из seed-цветов в пространстве OKLCH — тот же подход,
// что у веб-версии (CSS-токены oklch(L C H)). Тон роли задаётся светлотой L,
// оттенок/насыщенность берутся из seed'а. Светлота на роль фиксирована, поэтому
// контраст текста сохраняется при ЛЮБОМ оттенке (в т.ч. жёлтом/салатовом).
//
// Схему собираем через lightColorScheme/darkColorScheme: у их параметров есть
// дефолты на ВСЕ поля, поэтому код переживёт добавление новых ролей в будущих
// версиях Material3 (важно на compose-bom-alpha) — заполняем лишь известные.

// Оттенок (радианы) и хрома seed-цвета; светлоту подставляем потоново.
private data class HueChroma(val h: Double, val c: Double)

// Фиксированная палитра ошибки (M3-красный) — не зависит от темы.
private val ERROR = HueChroma(h = 0.51, c = 0.18) // ≈29° в OKLCH

private fun parseHex(hex: String): Triple<Double, Double, Double> {
    val clean = hex.trim().removePrefix("#")
    val v = when (clean.length) {
        3 -> clean.map { "$it$it" }.joinToString("")
        6 -> clean
        8 -> clean.substring(2) // отбрасываем альфу
        else -> "1195ed"
    }
    val r = v.substring(0, 2).toInt(16) / 255.0
    val g = v.substring(2, 4).toInt(16) / 255.0
    val b = v.substring(4, 6).toInt(16) / 255.0
    return Triple(r, g, b)
}

private fun srgbToLinear(c: Double): Double =
    if (c <= 0.04045) c / 12.92 else ((c + 0.055) / 1.055).pow(2.4)

private fun linearToSrgb(c: Double): Double {
    val v = if (c <= 0.0031308) c * 12.92 else 1.055 * c.pow(1.0 / 2.4) - 0.055
    return v.coerceIn(0.0, 1.0)
}

// sRGB hex → оттенок и хрома OKLCH (светлота отбрасывается — задаётся потоново).
private fun hueChroma(hex: String): HueChroma {
    val (r0, g0, b0) = parseHex(hex)
    val r = srgbToLinear(r0); val g = srgbToLinear(g0); val b = srgbToLinear(b0)
    val l = 0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b
    val m = 0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b
    val s = 0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b
    val l_ = cbrt(l); val m_ = cbrt(m); val s_ = cbrt(s)
    val okA = 1.9779984951 * l_ - 2.4285922050 * m_ + 0.4505937099 * s_
    val okB = 0.0259040371 * l_ + 0.7827717662 * m_ - 0.8086757660 * s_
    return HueChroma(h = atan2(okB, okA), c = kotlin.math.hypot(okA, okB))
}

// OKLCH (L 0..1, C, H рад) → sRGB Color (с клипом в гамут).
private fun oklch(lightness: Double, chroma: Double, hue: Double): Color {
    val a = chroma * cos(hue)
    val b = chroma * sin(hue)
    val l_ = lightness + 0.3963377774 * a + 0.2158037573 * b
    val m_ = lightness - 0.1055613458 * a - 0.0638541728 * b
    val s_ = lightness - 0.0894841775 * a - 1.2914855480 * b
    val l = l_ * l_ * l_; val m = m_ * m_ * m_; val s = s_ * s_ * s_
    val r = linearToSrgb(4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s)
    val g = linearToSrgb(-1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s)
    val bl = linearToSrgb(-0.0041960863 * l - 0.7034186147 * m + 1.7076147010 * s)
    return Color(r.toFloat(), g.toFloat(), bl.toFloat())
}

// Акцентный тон: хрому приглушаем к краям светлоты — иначе светлые контейнеры
// уходят в пересвет, а тёмные «грязнят». В центре насыщенность максимальная.
private fun accent(hc: HueChroma, l: Double): Color {
    val edge = (kotlin.math.abs(l - 0.5) * 2.0)
    val c = (hc.c.coerceIn(0.05, 0.20)) * (1.0 - 0.5 * edge)
    return oklch(l, c, hc.h)
}

// Нейтральный/нейтрально-вариантный тон: лёгкий тинт оттенком нейтрали.
private fun neutral(h: Double, l: Double, chroma: Double = 0.006): Color = oklch(l, chroma, h)

private fun white() = Color.White

fun grooveColorScheme(palette: ThemePalette, dark: Boolean): ColorScheme {
    val p = hueChroma(palette.primary)
    val s = hueChroma(palette.secondary)
    val t = hueChroma(palette.tertiary)
    val nH = hueChroma(palette.neutral).h

    return if (dark) {
        darkColorScheme(
            primary = accent(p, 0.82), onPrimary = accent(p, 0.25),
            primaryContainer = accent(p, 0.36), onPrimaryContainer = accent(p, 0.90),
            secondary = accent(s, 0.82), onSecondary = accent(s, 0.25),
            secondaryContainer = accent(s, 0.36), onSecondaryContainer = accent(s, 0.90),
            tertiary = accent(t, 0.82), onTertiary = accent(t, 0.25),
            tertiaryContainer = accent(t, 0.36), onTertiaryContainer = accent(t, 0.90),
            error = accent(ERROR, 0.80), onError = accent(ERROR, 0.25),
            errorContainer = accent(ERROR, 0.40), onErrorContainer = accent(ERROR, 0.90),
            background = neutral(nH, 0.16), onBackground = neutral(nH, 0.90),
            surface = neutral(nH, 0.16), onSurface = neutral(nH, 0.90),
            surfaceVariant = neutral(nH, 0.40, 0.012), onSurfaceVariant = neutral(nH, 0.80, 0.012),
            outline = neutral(nH, 0.60, 0.012), outlineVariant = neutral(nH, 0.40, 0.012),
            scrim = Color.Black,
            inverseSurface = neutral(nH, 0.90), inverseOnSurface = neutral(nH, 0.25),
            inversePrimary = accent(p, 0.50),
            surfaceDim = neutral(nH, 0.14), surfaceBright = neutral(nH, 0.34),
            surfaceContainerLowest = neutral(nH, 0.11), surfaceContainerLow = neutral(nH, 0.18),
            surfaceContainer = neutral(nH, 0.20), surfaceContainerHigh = neutral(nH, 0.25),
            surfaceContainerHighest = neutral(nH, 0.30),
        )
    } else {
        lightColorScheme(
            primary = accent(p, 0.52), onPrimary = white(),
            primaryContainer = accent(p, 0.90), onPrimaryContainer = accent(p, 0.24),
            secondary = accent(s, 0.54), onSecondary = white(),
            secondaryContainer = accent(s, 0.90), onSecondaryContainer = accent(s, 0.24),
            tertiary = accent(t, 0.54), onTertiary = white(),
            tertiaryContainer = accent(t, 0.90), onTertiaryContainer = accent(t, 0.24),
            error = accent(ERROR, 0.50), onError = white(),
            errorContainer = accent(ERROR, 0.91), onErrorContainer = accent(ERROR, 0.24),
            background = neutral(nH, 0.985), onBackground = neutral(nH, 0.18),
            surface = neutral(nH, 0.985), onSurface = neutral(nH, 0.18),
            surfaceVariant = neutral(nH, 0.90, 0.012), onSurfaceVariant = neutral(nH, 0.40, 0.012),
            outline = neutral(nH, 0.58, 0.012), outlineVariant = neutral(nH, 0.80, 0.012),
            scrim = Color.Black,
            inverseSurface = neutral(nH, 0.25), inverseOnSurface = neutral(nH, 0.96),
            inversePrimary = accent(p, 0.82),
            surfaceDim = neutral(nH, 0.88), surfaceBright = neutral(nH, 0.985),
            surfaceContainerLowest = white(), surfaceContainerLow = neutral(nH, 0.97),
            surfaceContainer = neutral(nH, 0.955), surfaceContainerHigh = neutral(nH, 0.935),
            surfaceContainerHighest = neutral(nH, 0.915),
        )
    }
}

// Цвет из OKLCH-параметров (оттенок в градусах) сразу в hex — для генератора
// случайных гармоничных тем («Мне повезёт»).
fun oklchHex(lightness: Double, chroma: Double, hueDeg: Double): String =
    oklch(lightness, chroma, Math.toRadians(hueDeg)).toHex()

// Color → "#RRGGBB" (для хранения и превью в конструкторе).
fun Color.toHex(): String {
    val r = (red * 255).roundToInt().coerceIn(0, 255)
    val g = (green * 255).roundToInt().coerceIn(0, 255)
    val b = (blue * 255).roundToInt().coerceIn(0, 255)
    return "#%02X%02X%02X".format(r, g, b)
}

fun Color.Companion.fromHex(hex: String): Color {
    val (r, g, b) = parseHex(hex)
    return Color(r.toFloat(), g.toFloat(), b.toFloat())
}
