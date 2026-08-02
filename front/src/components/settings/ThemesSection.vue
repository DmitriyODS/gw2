<template>
  <div class="ts">
    <!-- ── Режим оформления ──────────────────────────────────── -->
    <AppCard>
      <AppRow
        title="Режим оформления"
        hint="Светлая или тёмная тема, как в системе либо по расписанию."
        stack
      >
        <div class="mode-seg" role="tablist">
          <button
            v-for="m in THEME_MODES"
            :key="m.value"
            class="mode-btn"
            :class="{ active: themeStore.mode === m.value }"
            role="tab"
            type="button"
            :aria-selected="themeStore.mode === m.value"
            @click="themeStore.setMode(m.value)"
          >
            <span class="material-symbols-outlined">{{ m.icon }}</span>
            <span class="mode-label">{{ m.label }}</span>
          </button>
        </div>
      </AppRow>

      <Transition name="ts-reveal">
        <AppRow
          v-if="themeStore.mode === 'schedule'"
          title="Расписание"
          hint="Когда включать и выключать тёмную тему."
          stack
        >
          <div class="mode-times">
            <label class="mode-time">
              <span class="mode-time-label">Включить тёмную тему:</span>
              <TimePicker
                :model-value="themeStore.schedule.from"
                icon="dark_mode"
                @update:model-value="(v) => onSchedule('from', v)"
              />
            </label>
            <label class="mode-time">
              <span class="mode-time-label">Выключить тёмную тему:</span>
              <TimePicker
                :model-value="themeStore.schedule.to"
                icon="light_mode"
                @update:model-value="(v) => onSchedule('to', v)"
              />
            </label>
          </div>
        </AppRow>
      </Transition>
    </AppCard>

    <!-- ── Встроенные темы ───────────────────────────────────── -->
    <SettingsAccordion title="Встроенные темы">
      <div class="theme-grid">
        <ThemeCard
          v-for="preset in themeStore.presetNames"
          :key="preset"
          :name="themeStore.presetLabels[preset]"
          :vars="themeStore.getVars(preset)"
          :active="themeStore.currentPreset === preset"
          @apply="themeStore.applyTheme(preset)"
        />
      </div>
    </SettingsAccordion>

    <!-- ── Магазин: витрина ещё не открыта, поэтому пока только вход ── -->
    <SettingsAccordion title="Загруженные из магазина">
      <AppRow
        title="Темы из магазина"
        hint="Здесь появятся темы, которые вы возьмёте в магазине оформления."
      >
        <button class="ts-store-btn" type="button" @click="router.push('/store')">
          <span class="material-symbols-outlined">shopping_bag</span>
          В магазин
        </button>
      </AppRow>
    </SettingsAccordion>

    <!-- ── Свои темы ─────────────────────────────────────────── -->
    <SettingsAccordion title="Созданные мной" :badge="themeStore.customThemes.length || ''">
      <div class="theme-grid">
        <ThemeCard
          v-for="t in themeStore.customThemes"
          :key="t.name"
          :name="t.name"
          :vars="t.vars"
          :active="themeStore.currentPreset === t.name"
          editable
          @apply="themeStore.applyTheme(t.name)"
          @edit="openEditor(t)"
          @remove="askRemove(t)"
        />

        <button class="theme-new" type="button" @click="openEditor(null)">
          <span class="material-symbols-outlined">add</span>
          <span>Создать свою</span>
        </button>
      </div>

      <div class="ts-io">
        <label class="ts-io-btn">
          <span class="material-symbols-outlined">upload</span>
          Импортировать из файла
          <input type="file" accept=".json" @change="onImport" />
        </label>
        <button class="ts-io-btn" type="button" @click="themeStore.exportTheme(themeStore.currentPreset)">
          <span class="material-symbols-outlined">download</span>
          Сохранить текущую в файл
        </button>
      </div>
    </SettingsAccordion>

    <ThemeEditorDialog v-model="editorOpen" :source="editorSource" />

    <ConfirmDialog
      :visible="!!removeTarget"
      header="Удалить тему?"
      :message="`Тема «${removeTarget?.name}» будет удалена без возможности вернуть.`"
      confirm-label="Удалить"
      :danger-confirm="true"
      @confirm="confirmRemove"
      @cancel="removeTarget = null"
    />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import TimePicker from '@/components/common/TimePicker.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppRow from '@/components/ui/AppRow.vue'
import SettingsAccordion from '@/components/settings/SettingsAccordion.vue'
import ThemeCard from '@/components/settings/ThemeCard.vue'
import ThemeEditorDialog from '@/components/settings/ThemeEditorDialog.vue'
import { useThemeStore } from '@/stores/theme.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const themeStore = useThemeStore()
const notif = useNotificationsStore()
const router = useRouter()

const THEME_MODES = [
  { value: 'system', label: 'Системная', icon: 'brightness_auto' },
  { value: 'light', label: 'Светлая', icon: 'light_mode' },
  { value: 'dark', label: 'Тёмная', icon: 'dark_mode' },
  { value: 'schedule', label: 'По времени', icon: 'schedule' },
]

function onSchedule(field, v) {
  if (!v) return
  themeStore.setSchedule(
    field === 'from' ? v : themeStore.schedule.from,
    field === 'to' ? v : themeStore.schedule.to,
  )
}

/* ── Конструктор ── */
const editorOpen = ref(false)
const editorSource = ref(null)

function openEditor(theme) {
  editorSource.value = theme
  editorOpen.value = true
}

/* ── Удаление своей темы ── */
const removeTarget = ref(null)

function askRemove(theme) {
  removeTarget.value = theme
}

function confirmRemove() {
  themeStore.deleteCustomTheme(removeTarget.value.name)
  notif.success(`Тема «${removeTarget.value.name}» удалена`)
  removeTarget.value = null
}

async function onImport(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return
  try {
    const json = JSON.parse(await file.text())
    themeStore.importTheme(json)
    notif.success(`Тема «${json.name}» импортирована`)
  } catch {
    notif.error('Неверный формат файла темы')
  }
}
</script>

<style scoped>
.ts {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

/* ── Сегментированный переключатель режима ── */
.mode-seg {
  display: flex;
  width: 100%;
  gap: 4px;
  padding: 5px;
  border-radius: 999px;
  background: var(--color-surface-low);
  border: 1px solid var(--acrylic-border);
}

.mode-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  min-width: 0;
  padding: 9px 12px;
  border: none;
  border-radius: 999px;
  background: none;
  color: var(--color-text-dim);
  font-size: 0.88rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s ease, color 0.2s ease;
}

.mode-btn .material-symbols-outlined { font-size: 20px; }

.mode-btn:hover:not(.active) { background: var(--color-surface-high); }

.mode-btn.active {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.mode-label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.mode-times {
  display: grid;
  width: 100%;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.mode-time {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.mode-time-label {
  padding-left: 4px;
  font-size: 0.82rem;
  color: var(--color-text-dim);
}

/* ── Сетка тем ── */
.theme-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
  gap: 10px;
}

.theme-new {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px;
  border: 1px dashed var(--color-outline-dim);
  border-radius: var(--radius-lg);
  background: none;
  color: var(--color-text-dim);
  font-size: 0.88rem;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.2s ease, color 0.2s ease, background 0.2s ease;
}

.theme-new:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
  background: var(--color-surface-low);
}

.theme-new .material-symbols-outlined { font-size: 20px; }

/* ── Вход в магазин тем ── */
.ts-store-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 11px 18px;
  border: 1px solid var(--acrylic-border);
  border-radius: 999px;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 0.92rem;
  font-weight: 600;
  cursor: pointer;
  transition: filter 0.2s ease;
}

.ts-store-btn:hover { filter: brightness(1.04); }

/* ── Импорт/экспорт и кнопки-действия секций ── */
.ts-io {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 14px;
}

.ts-io-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border: 1px solid var(--acrylic-border);
  border-radius: 999px;
  background: var(--color-surface-low);
  color: var(--color-text);
  font-size: 0.86rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s ease;
}

.ts-io-btn:hover { background: var(--color-surface-high); }

.ts-io-btn .material-symbols-outlined { font-size: 20px; }

.ts-io-btn input[type="file"] { display: none; }

/* ── Раскрытие ── */
.ts-reveal-enter-active, .ts-reveal-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.ts-reveal-enter-from, .ts-reveal-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

/* Узкая ПАНЕЛЬ раздела (её контейнер объявляет SettingsView), а не экран:
   в половинном окне рабочего стола четыре сегмента в строку не помещаются —
   становимся сеткой 2×2. Дорожка при этом теряет форму пилюли: скругление
   999px на двух рядах выглядит сломанным. */
@container (max-width: 640px) {
  .mode-seg {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 6px;
    border-radius: var(--radius-lg);
  }
  .mode-btn { justify-content: flex-start; padding: 10px 14px; }
  .mode-times { grid-template-columns: 1fr; }
}

/* Дубль для заводского WebView старых Android (chrome87 не знает @container);
   там окно всё равно во весь экран. */
@media (max-width: 640px) {
  .mode-seg {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 6px;
    border-radius: var(--radius-lg);
  }
  .mode-btn { justify-content: flex-start; padding: 10px 14px; }
  .mode-times { grid-template-columns: 1fr; }
}

/* Совсем узко (одна колонка) — четыре строки списком. */
@container (max-width: 380px) {
  .mode-seg { grid-template-columns: 1fr; }
}

@media (max-width: 380px) {
  .mode-seg { grid-template-columns: 1fr; }
}
</style>
