package com.kodass.groovework.ui.theme

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json

private val Context.themeDataStore by preferencesDataStore(name = "theme")

// Состояние персонализации оформления: режим (свет/тьма/система) и палитра из
// четырёх seed-цветов. Хранится локально на устройстве (DataStore "theme"),
// переживает перезапуск. Источник истины для GrooveWorkTheme.
class ThemeController(context: Context, private val scope: CoroutineScope) {
    private val ds = context.applicationContext.themeDataStore

    val mode = MutableStateFlow(ThemeMode.SYSTEM)
    val palette = MutableStateFlow(DEFAULT_PRESET.palette)
    // Сохранённые пользователем темы (свои и сгенерированные).
    val savedThemes = MutableStateFlow<List<SavedTheme>>(emptyList())

    private val json = Json { ignoreUnknownKeys = true }
    private val savedSerializer = ListSerializer(SavedTheme.serializer())

    init {
        // Читаем стартовые значения синхронно (локальный файл, быстро) — чтобы
        // приложение не моргнуло дефолтной темой на первом кадре.
        runCatching {
            val prefs = runBlocking { ds.data.first() }
            prefs[KEY_MODE]?.let { raw -> runCatching { ThemeMode.valueOf(raw) }.getOrNull()?.let { mode.value = it } }
            val p = prefs[KEY_PRIMARY]; val s = prefs[KEY_SECONDARY]
            val t = prefs[KEY_TERTIARY]; val n = prefs[KEY_NEUTRAL]
            if (p != null && s != null && t != null && n != null) {
                palette.value = ThemePalette(p, s, t, n)
            }
            prefs[KEY_SAVED]?.let { raw ->
                runCatching { json.decodeFromString(savedSerializer, raw) }.getOrNull()?.let { savedThemes.value = it }
            }
        }
        // Подписку на дальнейшие изменения файла не ведём: правки идут только
        // через этот же контроллер (один на процесс), StateFlow уже актуален.
    }

    fun setMode(value: ThemeMode) {
        mode.value = value
        persist { it[KEY_MODE] = value.name }
    }

    fun setPalette(value: ThemePalette) {
        palette.value = value
        persist {
            it[KEY_PRIMARY] = value.primary
            it[KEY_SECONDARY] = value.secondary
            it[KEY_TERTIARY] = value.tertiary
            it[KEY_NEUTRAL] = value.neutral
        }
    }

    fun applyPreset(preset: ThemePreset) = setPalette(preset.palette)

    fun surprise() = setPalette(randomPalette())

    // Сохранить текущую палитру под именем (своя/сгенерированная тема).
    fun saveTheme(name: String) {
        val trimmed = name.trim().ifEmpty { "Моя тема ${savedThemes.value.size + 1}" }
        val next = savedThemes.value.filter { it.name != trimmed } + SavedTheme(trimmed, palette.value)
        savedThemes.value = next
        persistSaved(next)
    }

    fun deleteSavedTheme(name: String) {
        val next = savedThemes.value.filter { it.name != name }
        savedThemes.value = next
        persistSaved(next)
    }

    fun reset() {
        setMode(ThemeMode.SYSTEM)
        setPalette(DEFAULT_PRESET.palette)
    }

    private fun persist(block: (androidx.datastore.preferences.core.MutablePreferences) -> Unit) {
        scope.launch { ds.edit(block) }
    }

    private fun persistSaved(list: List<SavedTheme>) {
        val raw = json.encodeToString(savedSerializer, list)
        persist { it[KEY_SAVED] = raw }
    }

    private companion object {
        val KEY_MODE = stringPreferencesKey("mode")
        val KEY_PRIMARY = stringPreferencesKey("primary")
        val KEY_SECONDARY = stringPreferencesKey("secondary")
        val KEY_TERTIARY = stringPreferencesKey("tertiary")
        val KEY_NEUTRAL = stringPreferencesKey("neutral")
        val KEY_SAVED = stringPreferencesKey("saved_themes")
    }
}
