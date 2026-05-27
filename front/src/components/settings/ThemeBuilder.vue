<template>
  <div class="tb">
    <!-- Hero-превью текущей темы -->
    <div class="tb-hero">
      <div class="tb-hero-preview" aria-hidden="true">
        <!-- Мок интерфейса: сайдбар, шапка, кнопки, badge — наглядно показывает,
             как будут выглядеть основные элементы. Все цвета — токены. -->
        <div class="hp-frame">
          <div class="hp-sidebar">
            <div class="hp-dot" />
            <div class="hp-stripe" />
            <div class="hp-stripe short" />
            <div class="hp-stripe short" />
          </div>
          <div class="hp-content">
            <div class="hp-row">
              <span class="hp-pill primary">Активные</span>
              <span class="hp-pill ghost">Архив</span>
            </div>
            <div class="hp-card">
              <div class="hp-card-title" />
              <div class="hp-card-line" />
              <div class="hp-card-line short" />
              <div class="hp-card-foot">
                <span class="hp-btn">Начать</span>
                <span class="hp-tag tag-a">design</span>
                <span class="hp-tag tag-b">copy</span>
              </div>
            </div>
            <div class="hp-mini-card">
              <div class="hp-card-line" />
              <div class="hp-card-line short" />
            </div>
          </div>
        </div>
      </div>

      <div class="tb-hero-info">
        <div class="tb-hero-eyebrow">
          <span class="material-symbols-outlined">palette</span>
          Внешний вид
        </div>
        <h3 class="tb-hero-title">Сделайте Groove своим</h3>
        <p class="tb-hero-sub">
          Выберите готовую тему, поиграйте с цветами или импортируйте чужую —
          палитра пересчитается мгновенно во всём интерфейсе.
        </p>
        <div class="tb-hero-actions">
          <button class="btn-lucky" @click="surpriseMe" title="Случайная гармоничная тема">
            <span class="material-symbols-outlined">auto_awesome</span>
            Мне повезёт
          </button>
          <button class="btn-ghost" @click="resetToCurrent" title="Сбросить правки">
            <span class="material-symbols-outlined">refresh</span>
            Сбросить
          </button>
        </div>
      </div>
    </div>

    <!-- Светлая / Тёмная — сегментированный переключатель -->
    <div class="tb-card">
      <div class="tb-card-head">
        <div class="tb-card-head-icon" data-tone="secondary">
          <span class="material-symbols-outlined">contrast</span>
        </div>
        <div>
          <h4 class="tb-card-title">Режим оформления</h4>
          <p class="tb-card-sub">Подстройте платформу под освещение в комнате.</p>
        </div>
      </div>
      <div class="seg-group" role="tablist">
        <button
          class="seg-btn"
          :class="{ active: !themeStore.dark }"
          role="tab"
          :aria-selected="!themeStore.dark"
          @click="themeStore.dark && themeStore.toggleDark()"
        >
          <span class="material-symbols-outlined">light_mode</span>
          Светлая
        </button>
        <button
          class="seg-btn"
          :class="{ active: themeStore.dark }"
          role="tab"
          :aria-selected="themeStore.dark"
          @click="!themeStore.dark && themeStore.toggleDark()"
        >
          <span class="material-symbols-outlined">dark_mode</span>
          Тёмная
        </button>
        <span class="seg-indicator" :data-pos="themeStore.dark ? 'right' : 'left'" />
      </div>
    </div>

    <!-- Галерея готовых тем -->
    <div class="tb-card">
      <div class="tb-card-head">
        <div class="tb-card-head-icon" data-tone="primary">
          <span class="material-symbols-outlined">view_carousel</span>
        </div>
        <div>
          <h4 class="tb-card-title">Готовые темы</h4>
          <p class="tb-card-sub">Полноценные палитры, которые уже подобраны и проверены.</p>
        </div>
      </div>
      <div class="preset-gallery">
        <button
          v-for="preset in themeStore.presetNames"
          :key="preset"
          class="preset-tile"
          :class="{ active: themeStore.currentPreset === preset }"
          @click="applyPreset(preset)"
          :title="themeStore.presetLabels[preset]"
        >
          <div class="pt-preview" :style="previewStyle(preset)">
            <span class="pt-c c1" :style="{ background: themeStore.getVars(preset).primary }"></span>
            <span class="pt-c c2" :style="{ background: themeStore.getVars(preset).secondary }"></span>
            <span class="pt-c c3" :style="{ background: themeStore.getVars(preset).tertiary }"></span>
            <span
              v-if="themeStore.currentPreset === preset"
              class="pt-check"
              :style="{ background: themeStore.getVars(preset).primary }"
            >
              <span class="material-symbols-outlined">check</span>
            </span>
          </div>
          <span class="pt-name">{{ themeStore.presetLabels[preset] }}</span>
        </button>
      </div>
    </div>

    <!-- Конструктор -->
    <div class="tb-card">
      <div class="tb-card-head">
        <div class="tb-card-head-icon" data-tone="tertiary">
          <span class="material-symbols-outlined">tune</span>
        </div>
        <div>
          <h4 class="tb-card-title">Свои цвета</h4>
          <p class="tb-card-sub">Покрутите ручки — палитра обновится мгновенно. Сохраните, если понравится.</p>
        </div>
      </div>

      <div class="color-grid">
        <label
          v-for="(label, key) in colorLabels"
          :key="key"
          class="color-swatch"
          :style="{ '--swatch-color': customVars[key] }"
        >
          <div class="cs-circle">
            <div class="cs-fill" :style="{ background: customVars[key] }" />
            <span class="cs-edit-icon">
              <span class="material-symbols-outlined">edit</span>
            </span>
            <input
              type="color"
              class="cs-input"
              v-model="customVars[key]"
              @input="onLivePreview"
            />
          </div>
          <div class="cs-text">
            <span class="cs-label">{{ label.title }}</span>
            <span class="cs-hint">{{ label.hint }}</span>
            <span class="cs-hex">{{ customVars[key].toUpperCase() }}</span>
          </div>
        </label>
      </div>

      <div class="save-row">
        <InputText
          v-model="customThemeName"
          placeholder="Дайте теме название — например, «Утро в офисе»"
          class="save-input"
        />
        <button class="btn-filled" @click="saveCustom" :disabled="!customThemeName.trim()">
          <span class="material-symbols-outlined">bookmark_add</span>
          Сохранить
        </button>
      </div>
    </div>

    <!-- Мои темы -->
    <div v-if="themeStore.customThemes.length" class="tb-card">
      <div class="tb-card-head">
        <div class="tb-card-head-icon" data-tone="primary">
          <span class="material-symbols-outlined">bookmarks</span>
        </div>
        <div>
          <h4 class="tb-card-title">Мои темы</h4>
          <p class="tb-card-sub">Сохранённые вами палитры — можно применить в один клик.</p>
        </div>
      </div>
      <div class="custom-list">
        <div
          v-for="t in themeStore.customThemes"
          :key="t.name"
          class="custom-tile"
          :class="{ active: themeStore.currentPreset === t.name }"
        >
          <div class="ct-preview" @click="themeStore.applyTheme(t.name)">
            <span class="pt-c" :style="{ background: t.vars.primary }"></span>
            <span class="pt-c" :style="{ background: t.vars.secondary }"></span>
            <span class="pt-c" :style="{ background: t.vars.tertiary }"></span>
          </div>
          <div class="ct-info">
            <span class="ct-name">{{ t.name }}</span>
            <div class="ct-actions">
              <button class="ct-btn" @click="themeStore.applyTheme(t.name)" title="Применить">
                <span class="material-symbols-outlined">check_circle</span>
                Применить
              </button>
              <button class="ct-btn danger" @click="themeStore.deleteCustomTheme(t.name)" title="Удалить">
                <span class="material-symbols-outlined">delete</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Импорт / Экспорт -->
    <div class="tb-card">
      <div class="tb-card-head">
        <div class="tb-card-head-icon" data-tone="secondary">
          <span class="material-symbols-outlined">swap_vert</span>
        </div>
        <div>
          <h4 class="tb-card-title">Импорт и экспорт</h4>
          <p class="tb-card-sub">Поделиться темой с коллегой или сохранить настройки на будущее.</p>
        </div>
      </div>
      <div class="io-row">
        <button class="btn-tonal" @click="themeStore.exportTheme(themeStore.currentPreset)">
          <span class="material-symbols-outlined">download</span>
          Скачать JSON
        </button>
        <label class="btn-tonal file-btn">
          <span class="material-symbols-outlined">upload</span>
          Загрузить JSON
          <input type="file" accept=".json" @change="importTheme" style="display:none" />
        </label>
      </div>
    </div>
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
  primary:   { title: 'Основной',   hint: 'Главный акцент: кнопки и активные элементы' },
  secondary: { title: 'Вторичный',  hint: 'Поддерживающий: ссылки и второстепенные акценты' },
  tertiary:  { title: 'Третичный',  hint: 'Третий тон для выделений и плашек' },
  neutral:   { title: 'Нейтральный', hint: 'Гамма фонов и поверхностей' },
}

const DEFAULT_NEUTRAL = '#e8e6ea'

const customVars = reactive({
  primary:   '#e040fb',
  secondary: '#00bfa5',
  tertiary:  '#3d6ce7',
  neutral:   DEFAULT_NEUTRAL,
})

const customThemeName = ref('')

watch(
  () => themeStore.currentPreset,
  (preset) => {
    const vars = themeStore.getVars(preset)
    Object.assign(customVars, vars)
    if (!vars.neutral) customVars.neutral = DEFAULT_NEUTRAL
  },
  { immediate: true },
)

function previewStyle(preset) {
  const v = themeStore.getVars(preset)
  return {
    '--prv-primary': v.primary,
    '--prv-secondary': v.secondary,
    '--prv-tertiary': v.tertiary,
    '--prv-neutral': v.neutral || '#f1eff3',
  }
}

function applyPreset(name) {
  themeStore.applyTheme(name)
}

function onLivePreview() {
  themeStore.applyVars({ ...customVars })
}

function resetToCurrent() {
  const vars = themeStore.getVars(themeStore.currentPreset)
  Object.assign(customVars, vars)
  if (!vars.neutral) customVars.neutral = DEFAULT_NEUTRAL
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
  notif.success(`Тема «${name}» сохранена`)
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
      notif.success(`Тема «${json.name}» импортирована`)
    } catch {
      notif.error('Неверный формат файла темы')
    }
  }
  reader.readAsText(file)
  event.target.value = ''
}
</script>

<style scoped>
.tb {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 880px;
}

/* ── Hero ─────────────────────────────────────────────────────── */
.tb-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(0, 1fr);
  gap: 20px;
  padding: 24px;
  background: linear-gradient(
    135deg,
    color-mix(in oklch, var(--color-primary-container) 80%, transparent),
    color-mix(in oklch, var(--color-tertiary-container) 80%, transparent)
  );
  border: 1px solid var(--color-outline-dim);
  border-radius: 28px;
  overflow: hidden;
  position: relative;
}

.tb-hero::before {
  content: '';
  position: absolute;
  inset: -40px;
  background:
    radial-gradient(circle at 20% 0%, color-mix(in oklch, var(--color-primary) 20%, transparent), transparent 50%),
    radial-gradient(circle at 80% 100%, color-mix(in oklch, var(--color-secondary) 18%, transparent), transparent 50%);
  pointer-events: none;
  z-index: 0;
}

.tb-hero > * { position: relative; z-index: 1; }

.tb-hero-info {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 8px;
}

.tb-hero-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  background: color-mix(in oklch, var(--color-on-primary-container) 8%, transparent);
  color: var(--color-on-primary-container);
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  width: max-content;
  letter-spacing: 0.02em;
}

.tb-hero-eyebrow .material-symbols-outlined { font-size: 16px; }

.tb-hero-title {
  margin: 0;
  font-size: 26px;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--color-on-primary-container);
  line-height: 1.15;
}

.tb-hero-sub {
  margin: 0 0 6px;
  font-size: 14px;
  line-height: 1.5;
  color: color-mix(in oklch, var(--color-on-primary-container) 80%, transparent);
  max-width: 380px;
}

.tb-hero-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 6px;
}

.btn-lucky {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 11px 22px;
  border-radius: 999px;
  border: 0;
  cursor: pointer;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-on-primary);
  background: linear-gradient(120deg, var(--color-primary), var(--color-tertiary));
  box-shadow: 0 8px 20px color-mix(in oklch, var(--color-primary) 35%, transparent);
  transition: transform 0.15s, box-shadow 0.15s;
}

.btn-lucky:hover { transform: translateY(-2px); box-shadow: 0 12px 28px color-mix(in oklch, var(--color-primary) 45%, transparent); }
.btn-lucky:active { transform: translateY(0); }
.btn-lucky .material-symbols-outlined { font-size: 18px; }

.btn-ghost {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 10px 18px;
  background: color-mix(in oklch, var(--color-on-primary-container) 8%, transparent);
  border: 0;
  border-radius: 999px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-on-primary-container);
  transition: background 0.15s;
}

.btn-ghost:hover { background: color-mix(in oklch, var(--color-on-primary-container) 14%, transparent); }
.btn-ghost .material-symbols-outlined { font-size: 18px; }

/* ── Mock-превью интерфейса ──────────────────────────────────── */
.tb-hero-preview {
  border-radius: 20px;
  overflow: hidden;
  box-shadow: 0 16px 36px color-mix(in oklch, var(--color-scrim) 25%, transparent);
  background: var(--color-surface);
  border: 1px solid color-mix(in oklch, var(--color-outline-dim) 50%, transparent);
}

.hp-frame {
  display: grid;
  grid-template-columns: 56px 1fr;
  height: 100%;
  min-height: 180px;
}

.hp-sidebar {
  background: var(--color-surface-low);
  padding: 12px 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.hp-dot {
  width: 24px;
  height: 24px;
  border-radius: 8px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-tertiary));
}

.hp-stripe {
  height: 8px;
  background: var(--color-surface-highest);
  border-radius: 4px;
}

.hp-stripe.short { width: 60%; }

.hp-content {
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.hp-row { display: flex; gap: 6px; }

.hp-pill {
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 600;
}

.hp-pill.primary {
  background: var(--color-primary);
  color: var(--color-on-primary);
}

.hp-pill.ghost {
  background: var(--color-surface-high);
  color: var(--color-text-dim);
}

.hp-card {
  padding: 10px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.hp-mini-card {
  padding: 10px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  gap: 5px;
  opacity: 0.7;
}

.hp-card-title {
  height: 9px;
  width: 70%;
  border-radius: 4px;
  background: var(--color-text);
  opacity: 0.85;
}

.hp-card-line {
  height: 6px;
  background: var(--color-surface-highest);
  border-radius: 3px;
}

.hp-card-line.short { width: 50%; }

.hp-card-foot {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
}

.hp-btn {
  padding: 4px 10px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  border-radius: 999px;
  font-size: 10px;
  font-weight: 600;
}

.hp-tag {
  padding: 3px 8px;
  border-radius: 999px;
  font-size: 9px;
  font-weight: 600;
}

.hp-tag.tag-a {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}

.hp-tag.tag-b {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}

/* ── Базовая карточка ────────────────────────────────────────── */
.tb-card {
  padding: 20px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: 22px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.tb-card-head {
  display: flex;
  align-items: center;
  gap: 12px;
}

.tb-card-head-icon {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}

.tb-card-head-icon[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.tb-card-head-icon[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.tb-card-head-icon[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }

.tb-card-head-icon .material-symbols-outlined { font-size: 22px; }

.tb-card-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text);
  line-height: 1.2;
}

.tb-card-sub {
  margin: 2px 0 0;
  font-size: 13px;
  color: var(--color-text-dim);
  line-height: 1.4;
}

/* ── Segmented светлая/тёмная ───────────────────────────────── */
.seg-group {
  position: relative;
  display: inline-grid;
  grid-template-columns: 1fr 1fr;
  padding: 4px;
  background: var(--color-surface-high);
  border-radius: 999px;
  width: 100%;
  max-width: 320px;
}

.seg-btn {
  position: relative;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 18px;
  background: transparent;
  border: 0;
  border-radius: 999px;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: color 0.2s;
}

.seg-btn.active { color: var(--color-on-primary); }
.seg-btn .material-symbols-outlined { font-size: 18px; }

.seg-indicator {
  position: absolute;
  z-index: 0;
  top: 4px;
  bottom: 4px;
  left: 4px;
  width: calc(50% - 4px);
  background: var(--color-primary);
  border-radius: 999px;
  transition: transform 0.3s cubic-bezier(0.22, 1, 0.36, 1);
  box-shadow: 0 4px 14px color-mix(in oklch, var(--color-primary) 35%, transparent);
}

.seg-indicator[data-pos="right"] { transform: translateX(100%); }

/* ── Галерея пресетов ───────────────────────────────────────── */
.preset-gallery {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
}

.preset-tile {
  background: transparent;
  border: 0;
  padding: 0;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: stretch;
  text-align: center;
}

.pt-preview {
  position: relative;
  aspect-ratio: 16 / 10;
  border-radius: 18px;
  background: var(--prv-neutral, var(--color-surface-low));
  border: 2px solid var(--color-outline-dim);
  overflow: hidden;
  display: flex;
  transition: transform 0.18s, border-color 0.18s, box-shadow 0.18s;
}

.preset-tile:hover .pt-preview {
  transform: translateY(-3px);
  box-shadow: 0 10px 24px color-mix(in oklch, var(--color-scrim) 18%, transparent);
}

.preset-tile.active .pt-preview {
  border-color: var(--prv-primary);
  box-shadow: 0 0 0 4px color-mix(in oklch, var(--prv-primary) 18%, transparent);
}

.pt-c {
  flex: 1;
  height: 100%;
}

.pt-c.c1 { flex: 1.6; }
.pt-c.c2 { flex: 1; }
.pt-c.c3 { flex: 0.8; }

.pt-check {
  position: absolute;
  right: 8px;
  bottom: 8px;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  color: var(--color-on-primary);
  box-shadow: 0 4px 12px color-mix(in oklch, var(--color-scrim) 25%, transparent);
}

.pt-check .material-symbols-outlined { font-size: 18px; }

.pt-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text);
  line-height: 1.2;
}

.preset-tile.active .pt-name { color: var(--color-primary); }

/* ── Color swatches ──────────────────────────────────────────── */
.color-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
}

.color-swatch {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  background: var(--color-surface-low);
  border: 1px solid var(--color-outline-dim);
  border-radius: 18px;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
  position: relative;
}

.color-swatch:hover {
  border-color: color-mix(in oklch, var(--swatch-color) 70%, var(--color-outline-dim));
  background: var(--color-surface);
}

.cs-circle {
  position: relative;
  width: 52px;
  height: 52px;
  border-radius: 50%;
  flex-shrink: 0;
}

.cs-fill {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  box-shadow:
    inset 0 0 0 1px color-mix(in oklch, var(--color-on-surface) 8%, transparent),
    0 6px 14px color-mix(in oklch, var(--swatch-color) 35%, transparent);
}

.cs-edit-icon {
  position: absolute;
  right: -2px;
  bottom: -2px;
  width: 22px;
  height: 22px;
  background: var(--color-surface);
  color: var(--color-text);
  border-radius: 50%;
  display: grid;
  place-items: center;
  box-shadow: 0 2px 6px color-mix(in oklch, var(--color-scrim) 25%, transparent);
}

.cs-edit-icon .material-symbols-outlined { font-size: 13px; }

.cs-input {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  border: 0;
  border-radius: 50%;
  opacity: 0;
  cursor: pointer;
  background: transparent;
}

.cs-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.cs-label {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text);
}

.cs-hint {
  font-size: 11px;
  color: var(--color-text-dim);
  line-height: 1.3;
}

.cs-hex {
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', monospace;
  color: var(--color-text-dim);
  letter-spacing: 0.04em;
  margin-top: 2px;
}

/* ── Save row ────────────────────────────────────────────────── */
.save-row {
  display: flex;
  gap: 10px;
  align-items: center;
}

.save-input {
  flex: 1;
}

.save-row :deep(.save-input) {
  border-radius: 999px;
  padding-left: 18px;
}

/* ── Кнопки M3 ───────────────────────────────────────────────── */
.btn-filled, .btn-tonal {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 11px 20px;
  border-radius: 999px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  border: 0;
  white-space: nowrap;
  transition: background 0.15s, transform 0.15s;
}

.btn-filled {
  background: var(--color-primary);
  color: var(--color-on-primary);
}

.btn-filled:hover:not(:disabled) {
  background: color-mix(in oklch, var(--color-primary) 88%, var(--color-on-primary) 12%);
}

.btn-filled:disabled { opacity: 0.55; cursor: not-allowed; }
.btn-filled .material-symbols-outlined { font-size: 18px; }

.btn-tonal {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}

.btn-tonal:hover {
  background: color-mix(in oklch, var(--color-secondary-container) 80%, var(--color-on-secondary-container) 20%);
}

.btn-tonal .material-symbols-outlined { font-size: 18px; }

.io-row {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.file-btn { display: inline-flex; }

/* ── Мои темы ────────────────────────────────────────────────── */
.custom-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
}

.custom-tile {
  display: flex;
  flex-direction: column;
  background: var(--color-surface-low);
  border: 1px solid var(--color-outline-dim);
  border-radius: 18px;
  overflow: hidden;
  transition: border-color 0.15s;
}

.custom-tile:hover { border-color: var(--color-primary); }
.custom-tile.active { border-color: var(--color-primary); box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 14%, transparent); }

.ct-preview {
  display: flex;
  height: 56px;
  cursor: pointer;
}

.ct-info {
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ct-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
}

.ct-actions {
  display: flex;
  gap: 6px;
}

.ct-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 7px 10px;
  border: 0;
  border-radius: 10px;
  background: var(--color-surface-high);
  color: var(--color-text);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.ct-btn:hover { background: var(--color-surface-highest); }
.ct-btn .material-symbols-outlined { font-size: 16px; }

.ct-btn.danger {
  flex: 0;
  padding: 7px;
  color: var(--color-error);
}

.ct-btn.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

/* ── Mobile ─────────────────────────────────────────────────── */
@media (max-width: 900px) {
  .tb-hero {
    grid-template-columns: 1fr;
    padding: 20px;
  }
  .tb-hero-title { font-size: 22px; }
  .tb-hero-preview { order: -1; }
  .hp-frame { min-height: 160px; }
}

@media (max-width: 600px) {
  .tb {
    gap: 12px;
  }
  .tb-card { padding: 16px; border-radius: 18px; }
  .tb-hero { padding: 18px; gap: 16px; border-radius: 22px; }
  .tb-hero-title { font-size: 20px; }
  .tb-hero-actions { width: 100%; }
  .tb-hero-actions .btn-lucky,
  .tb-hero-actions .btn-ghost { flex: 1; justify-content: center; }
  .tb-card-head { gap: 10px; }
  .preset-gallery { grid-template-columns: repeat(auto-fill, minmax(130px, 1fr)); gap: 10px; }
  .color-grid { grid-template-columns: 1fr; }
  .save-row { flex-direction: column; align-items: stretch; }
  .save-row .btn-filled { justify-content: center; }
  .custom-list { grid-template-columns: 1fr; }
}
</style>
