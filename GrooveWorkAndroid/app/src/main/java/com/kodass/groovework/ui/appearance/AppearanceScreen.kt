package com.kodass.groovework.ui.appearance

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Casino
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.DarkMode
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.LightMode
import androidx.compose.material.icons.filled.Restore
import androidx.compose.material.icons.filled.Save
import androidx.compose.material.icons.outlined.BrightnessAuto
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.kodass.groovework.AppContainer
import com.kodass.groovework.ui.theme.SavedTheme
import com.kodass.groovework.ui.theme.THEME_PRESETS
import com.kodass.groovework.ui.theme.ThemeMode
import com.kodass.groovework.ui.theme.ThemePalette
import com.kodass.groovework.ui.theme.fromHex
import com.kodass.groovework.ui.theme.presetKeyFor

private data class ColorSlot(val title: String, val get: (ThemePalette) -> String, val set: (ThemePalette, String) -> ThemePalette)

private val COLOR_SLOTS = listOf(
    ColorSlot("Основной", { it.primary }, { p, v -> p.copy(primary = v) }),
    ColorSlot("Вторичный", { it.secondary }, { p, v -> p.copy(secondary = v) }),
    ColorSlot("Третичный", { it.tertiary }, { p, v -> p.copy(tertiary = v) }),
    ColorSlot("Нейтральный", { it.neutral }, { p, v -> p.copy(neutral = v) }),
)

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun AppearanceScreen(container: AppContainer, onBack: () -> Unit) {
    val theme = container.theme
    val mode by theme.mode.collectAsStateWithLifecycle()
    val palette by theme.palette.collectAsStateWithLifecycle()
    val saved by theme.savedThemes.collectAsStateWithLifecycle()

    val activePreset = presetKeyFor(palette)
    val savedMatch = saved.firstOrNull { it.palette == palette }
    // «Грязно» — текущая палитра не совпадает ни с пресетом, ни с сохранённой темой.
    val isDirty = activePreset == null && savedMatch == null

    var editingSlot by remember { mutableStateOf<ColorSlot?>(null) }
    var showSaveDialog by remember { mutableStateOf(false) }
    var exitAfterSave by remember { mutableStateOf(false) }
    var showExitGuard by remember { mutableStateOf(false) }

    fun attemptExit() { if (isDirty) showExitGuard = true else onBack() }

    BackHandler(enabled = true) { attemptExit() }

    editingSlot?.let { slot ->
        ColorPickerDialog(
            title = slot.title,
            initial = slot.get(palette),
            onDismiss = { editingSlot = null },
            onConfirm = { hex -> theme.setPalette(slot.set(palette, hex)); editingSlot = null },
        )
    }

    if (showSaveDialog) {
        var name by remember { mutableStateOf(savedMatch?.name ?: "") }
        AlertDialog(
            onDismissRequest = { showSaveDialog = false; exitAfterSave = false },
            title = { Text("Сохранить тему") },
            text = {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    singleLine = true,
                    label = { Text("Название темы") },
                    placeholder = { Text("Моя тема") },
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    theme.saveTheme(name)
                    showSaveDialog = false
                    if (exitAfterSave) { exitAfterSave = false; onBack() }
                }) { Text("Сохранить") }
            },
            dismissButton = { TextButton(onClick = { showSaveDialog = false; exitAfterSave = false }) { Text("Отмена") } },
        )
    }

    if (showExitGuard) {
        AlertDialog(
            onDismissRequest = { showExitGuard = false },
            title = { Text("Тема не сохранена") },
            text = { Text("Сохраните свою тему, чтобы вернуться к ней позже, или сбросьте изменения.") },
            confirmButton = {
                TextButton(onClick = { showExitGuard = false; exitAfterSave = true; showSaveDialog = true }) {
                    Text("Сохранить")
                }
            },
            dismissButton = {
                Row {
                    TextButton(onClick = { showExitGuard = false; theme.reset(); onBack() }) { Text("Сбросить") }
                    TextButton(onClick = { showExitGuard = false }) { Text("Остаться") }
                }
            },
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Оформление") },
                navigationIcon = {
                    IconButton(onClick = { attemptExit() }) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Назад")
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(22.dp),
        ) {
            // ── Режим ──
            SectionTitle("Режим")
            SingleChoiceSegmentedButtonRow(modifier = Modifier.fillMaxWidth()) {
                val options = listOf(
                    Triple(ThemeMode.LIGHT, "Светлая", Icons.Filled.LightMode),
                    Triple(ThemeMode.SYSTEM, "Системная", Icons.Outlined.BrightnessAuto),
                    Triple(ThemeMode.DARK, "Тёмная", Icons.Filled.DarkMode),
                )
                options.forEachIndexed { index, (value, label, icon) ->
                    SegmentedButton(
                        selected = mode == value,
                        onClick = { theme.setMode(value) },
                        shape = SegmentedButtonDefaults.itemShape(index, options.size),
                        icon = { Icon(icon, contentDescription = null, modifier = Modifier.size(18.dp)) },
                    ) { Text(label) }
                }
            }

            // ── Цветовые схемы ──
            SectionTitle("Цветовая схема")
            FlowRow(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                THEME_PRESETS.forEach { preset ->
                    SchemeCard(
                        label = preset.label,
                        palette = preset.palette,
                        selected = activePreset == preset.key,
                        onClick = { theme.applyPreset(preset) },
                    )
                }
            }

            // ── Мои темы ──
            if (saved.isNotEmpty()) {
                SectionTitle("Мои темы")
                FlowRow(
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    saved.forEach { st: SavedTheme ->
                        SchemeCard(
                            label = st.name,
                            palette = st.palette,
                            selected = savedMatch?.name == st.name,
                            onClick = { theme.setPalette(st.palette) },
                            onDelete = { theme.deleteSavedTheme(st.name) },
                        )
                    }
                }
            }

            // ── Свой набор цветов ──
            SectionTitle("Свой цвет")
            Text(
                text = "Подберите оттенки под себя — остальная палитра соберётся автоматически.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            COLOR_SLOTS.forEach { slot ->
                ColorRow(title = slot.title, hex = slot.get(palette), onClick = { editingSlot = slot })
            }

            // ── Действия ──
            FilledTonalButton(
                onClick = { exitAfterSave = false; showSaveDialog = true },
                enabled = isDirty,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Icon(Icons.Filled.Save, contentDescription = null, modifier = Modifier.size(18.dp))
                Text(if (isDirty) "Сохранить тему" else "Тема сохранена", modifier = Modifier.padding(start = 8.dp))
            }
            Row(
                modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                OutlinedButton(onClick = { theme.surprise() }, modifier = Modifier.weight(1f)) {
                    Icon(Icons.Filled.Casino, contentDescription = null, modifier = Modifier.size(18.dp))
                    Text("Мне повезёт", modifier = Modifier.padding(start = 8.dp))
                }
                OutlinedButton(onClick = { theme.reset() }, modifier = Modifier.weight(1f)) {
                    Icon(Icons.Filled.Restore, contentDescription = null, modifier = Modifier.size(18.dp))
                    Text("Сбросить", modifier = Modifier.padding(start = 8.dp))
                }
            }
        }
    }
}

@Composable
private fun SectionTitle(text: String) {
    Text(text = text, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
}

// Крупная карточка цветовой схемы в духе Material You: полоса-превью из seed-цветов,
// название и отметка выбора. Для сохранённых тем — кнопка удаления в углу.
@Composable
private fun SchemeCard(
    label: String,
    palette: ThemePalette,
    selected: Boolean,
    onClick: () -> Unit,
    onDelete: (() -> Unit)? = null,
) {
    Surface(
        onClick = onClick,
        shape = RoundedCornerShape(22.dp),
        color = MaterialTheme.colorScheme.surfaceContainerLow,
        border = if (selected) BorderStroke(2.dp, MaterialTheme.colorScheme.primary)
        else BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant),
        modifier = Modifier.width(158.dp),
    ) {
        Column(modifier = Modifier.padding(10.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
            Box {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(64.dp)
                        .clip(RoundedCornerShape(14.dp)),
                ) {
                    Box(modifier = Modifier.weight(2f).fillMaxHeight().background(Color.fromHex(palette.primary)))
                    Box(modifier = Modifier.weight(1f).fillMaxHeight().background(Color.fromHex(palette.secondary)))
                    Box(modifier = Modifier.weight(1f).fillMaxHeight().background(Color.fromHex(palette.tertiary)))
                }
                if (onDelete != null) {
                    Surface(
                        onClick = onDelete,
                        shape = CircleShape,
                        color = MaterialTheme.colorScheme.surface.copy(alpha = 0.85f),
                        modifier = Modifier.align(Alignment.TopEnd).padding(4.dp).size(26.dp),
                    ) {
                        Icon(
                            Icons.Filled.Delete,
                            contentDescription = "Удалить тему",
                            tint = MaterialTheme.colorScheme.error,
                            modifier = Modifier.padding(5.dp),
                        )
                    }
                }
            }
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = label,
                    style = MaterialTheme.typography.labelLarge,
                    maxLines = 1,
                    modifier = Modifier.weight(1f),
                )
                if (selected) {
                    Icon(
                        Icons.Filled.Check,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(18.dp),
                    )
                }
            }
        }
    }
}

@Composable
private fun ColorRow(title: String, hex: String, onClick: () -> Unit) {
    Surface(
        onClick = onClick,
        shape = RoundedCornerShape(14.dp),
        color = MaterialTheme.colorScheme.surfaceContainerLow,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 12.dp),
        ) {
            Box(
                modifier = Modifier
                    .size(28.dp)
                    .clip(CircleShape)
                    .background(Color.fromHex(hex)),
            )
            Column(modifier = Modifier.padding(start = 14.dp).weight(1f)) {
                Text(title, style = MaterialTheme.typography.bodyLarge)
                Text(
                    hex.uppercase(),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}
