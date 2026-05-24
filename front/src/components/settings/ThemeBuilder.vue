<template>
  <div class="theme-builder">
    <!-- Режим -->
    <section class="theme-section">
      <h4>Режим</h4>
      <div class="toggle-group">
        <button
          :class="{ active: !themeStore.dark }"
          @click="themeStore.dark && themeStore.toggleDark()"
        >
          Светлая
        </button>
        <button
          :class="{ active: themeStore.dark }"
          @click="!themeStore.dark && themeStore.toggleDark()"
        >
          Тёмная
        </button>
      </div>
    </section>

    <!-- Цветовые схемы -->
    <section class="theme-section">
      <h4>Цветовые схемы</h4>
      <div class="preset-grid">
        <button
          v-for="preset in themeStore.presetNames"
          :key="preset"
          class="preset-btn"
          :class="{ active: themeStore.currentPreset === preset }"
          @click="themeStore.applyTheme(preset)"
        >
          <span
            class="color-dot"
            :style="{ background: themeStore.getVars(preset).primary }"
          ></span>
          {{ themeStore.presetLabels[preset] }}
        </button>
      </div>
    </section>

    <!-- Конструктор -->
    <section class="theme-section">
      <h4>Конструктор тем</h4>
      <p class="builder-hint">
        Выберите ключевые цвета — вся палитра пересчитывается автоматически. Основной,
        вторичный и третичный задают кнопки и акценты; «Фон / нейтральный» — общую гамму
        фонов и поверхностей (работает и в светлой, и в тёмной теме).
      </p>
      <div class="builder-actions">
        <button class="btn-lucky" @click="surpriseMe" title="Случайная гармоничная тема">
          <span class="material-symbols-outlined">casino</span>
          Мне повезёт
        </button>
      </div>
      <div class="color-pickers">
        <div
          v-for="(label, key) in colorLabels"
          :key="key"
          class="color-picker-row"
        >
          <label class="color-picker-label">{{ label }}</label>
          <input type="color" v-model="customVars[key]" class="color-input" @input="onLivePreview" />
          <span class="color-hex">{{ customVars[key] }}</span>
        </div>
      </div>
      <div class="custom-theme-form">
        <InputText v-model="customThemeName" placeholder="Название темы" />
        <button class="btn-primary" @click="saveCustom" :disabled="!customThemeName.trim()">
          Сохранить тему
        </button>
      </div>
    </section>

    <!-- Пользовательские темы -->
    <section v-if="themeStore.customThemes.length" class="theme-section">
      <h4>Мои темы</h4>
      <div
        v-for="t in themeStore.customThemes"
        :key="t.name"
        class="custom-theme-row"
      >
        <span class="custom-theme-name">{{ t.name }}</span>
        <button class="btn-sm" @click="themeStore.applyTheme(t.name)">Применить</button>
        <button class="btn-sm danger" @click="themeStore.deleteCustomTheme(t.name)">Удалить</button>
      </div>
    </section>

    <!-- Импорт / Экспорт -->
    <section class="theme-section">
      <h4>Импорт / Экспорт</h4>
      <div class="io-buttons">
        <button class="btn-secondary" @click="themeStore.exportTheme(themeStore.currentPreset)">
          <span class="material-symbols-outlined">download</span>
          Экспортировать
        </button>
        <label class="btn-secondary file-btn">
          <span class="material-symbols-outlined">upload</span>
          Импортировать
          <input type="file" accept=".json" @change="importTheme" style="display:none" />
        </label>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue'
import InputText from 'primevue/inputtext'
import { useThemeStore } from '@/stores/theme.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const themeStore = useThemeStore()
const notif = useNotificationsStore()

const colorLabels = {
  primary:   'Основной цвет',
  secondary: 'Вторичный (акцент)',
  tertiary:  'Третичный',
  neutral:   'Фон / нейтральный',
}

const DEFAULT_NEUTRAL = '#e8e6ea'

const customVars = reactive({
  primary:   '#e040fb',
  secondary: '#00bfa5',
  tertiary:  '#3d6ce7',
  neutral:   DEFAULT_NEUTRAL,
})

const customThemeName = ref('')

// Sync with current theme
watch(
  () => themeStore.currentPreset,
  (preset) => {
    const vars = themeStore.getVars(preset)
    Object.assign(customVars, vars)
    // Пресет без своей нейтрали — показываем дефолтный нейтральный в пикере.
    if (!vars.neutral) customVars.neutral = DEFAULT_NEUTRAL
  },
  { immediate: true }
)

function onLivePreview() {
  themeStore.applyVars({ ...customVars })
}

function surpriseMe() {
  const t = themeStore.randomTheme()
  Object.assign(customVars, t)
  themeStore.applyVars({ ...customVars })
}

function saveCustom() {
  const name = customThemeName.value.trim()
  if (!name) return
  themeStore.saveCustomTheme(name, { ...customVars })
  themeStore.applyTheme(name)
  notif.success(`Тема "${name}" сохранена`)
  customThemeName.value = ''
}

function importTheme(event) {
  const file = event.target.files[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = (e) => {
    try {
      const json = JSON.parse(e.target.result)
      themeStore.importTheme(json)
      notif.success(`Тема "${json.name}" импортирована`)
    } catch {
      notif.error('Неверный формат файла темы')
    }
  }
  reader.readAsText(file)
  event.target.value = ''
}
</script>

<style scoped>
.theme-builder {
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-width: 720px;
}

.theme-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.theme-section h4 {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: var(--gw-text);
  padding-bottom: 8px;
  border-bottom: 1px solid var(--gw-border);
}

/* Toggle Светлая/Тёмная */
.toggle-group {
  display: flex;
  gap: 0;
  border: 1px solid var(--gw-border);
  border-radius: 10px;
  overflow: hidden;
  width: fit-content;
}

.toggle-group button {
  padding: 8px 24px;
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: 14px;
  color: var(--gw-text-secondary);
  transition: background 0.15s, color 0.15s;
}

.toggle-group button.active {
  background: var(--gw-primary);
  color: var(--color-on-primary);
  font-weight: 600;
}

.toggle-group button:hover:not(.active) {
  background: var(--gw-bg);
}

/* Preset grid */
.preset-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.preset-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border: 1px solid var(--gw-border);
  border-radius: 20px;
  background: var(--gw-surface);
  cursor: pointer;
  font-size: 13px;
  color: var(--gw-text);
  transition: border-color 0.15s, background 0.15s;
}

.preset-btn.active {
  border-color: var(--gw-primary);
  background: var(--gw-bg);
  font-weight: 600;
  color: var(--gw-primary);
}

.preset-btn:hover:not(.active) {
  border-color: var(--gw-primary);
  background: var(--gw-bg);
}

.color-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  flex-shrink: 0;
  box-shadow: 0 0 0 1px color-mix(in oklch, var(--color-text) 10%, transparent);
}

.builder-hint {
  margin: 0;
  font-size: 13px;
  color: var(--gw-text-secondary);
  line-height: 1.5;
}

.builder-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.btn-lucky {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 9px 18px;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-on-tertiary);
  background: linear-gradient(120deg, var(--color-primary), var(--color-tertiary));
  transition: opacity 0.15s, transform 0.1s;
}

.btn-lucky:hover {
  opacity: 0.9;
}

.btn-lucky:active {
  transform: scale(0.98);
}

.btn-lucky .material-symbols-outlined {
  font-size: 18px;
}

/* Color pickers */
.color-pickers {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.color-picker-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.color-picker-label {
  min-width: 160px;
  font-size: 14px;
  color: var(--gw-text);
}

.color-input {
  width: 44px;
  height: 32px;
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  padding: 2px 4px;
  cursor: pointer;
  background: transparent;
}

.color-hex {
  font-size: 13px;
  color: var(--gw-text-secondary);
  font-family: monospace;
  min-width: 80px;
}

.custom-theme-form {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
}

.btn-primary {
  background: var(--gw-primary);
  color: var(--color-on-primary);
  border: none;
  border-radius: 8px;
  padding: 8px 18px;
  font-size: 14px;
  cursor: pointer;
  font-weight: 600;
  transition: background 0.15s;
  white-space: nowrap;
}

.btn-primary:hover:not(:disabled) {
  background: var(--gw-primary-hover);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Custom theme list */
.custom-theme-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: var(--gw-bg);
  border-radius: 8px;
  border: 1px solid var(--gw-border);
}

.custom-theme-name {
  flex: 1;
  font-size: 14px;
  color: var(--gw-text);
  font-weight: 500;
}

.btn-sm {
  padding: 5px 14px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid var(--gw-border);
  background: var(--gw-surface);
  color: var(--gw-text);
  transition: background 0.15s, color 0.15s;
}

.btn-sm:hover {
  background: var(--gw-primary);
  color: var(--color-on-primary);
  border-color: var(--gw-primary);
}

.btn-sm.danger {
  color: var(--color-error);
  border-color: color-mix(in oklch, var(--color-error) 30%, var(--color-outline-dim));
}

.btn-sm.danger:hover {
  background: var(--color-error);
  color: var(--color-on-error);
  border-color: var(--color-error);
}

/* IO buttons */
.io-buttons {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.btn-secondary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  border: 1px solid var(--gw-border);
  background: var(--gw-surface);
  color: var(--gw-text);
  transition: background 0.15s, border-color 0.15s;
}

.btn-secondary:hover {
  background: var(--gw-bg);
  border-color: var(--gw-primary);
  color: var(--gw-primary);
}

.btn-secondary .material-symbols-outlined {
  font-size: 18px;
}

.file-btn {
  cursor: pointer;
}
</style>
