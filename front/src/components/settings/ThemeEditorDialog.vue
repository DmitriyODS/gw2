<template>
  <!-- Конструктор палитры. Цвета применяются к интерфейсу сразу как живое
       превью, поэтому закрытие без сохранения откатывает их обратно. -->
  <AppDialog
    :model-value="modelValue"
    size="md"
    :title="source ? `Тема «${source.name}»` : 'Своя тема'"
    subtitle="Крутите ручки — палитра пересчитывается во всём интерфейсе мгновенно."
    :actions="actions"
    @update:model-value="onVisible"
    @confirm="save"
  >
    <div class="te-preview" aria-hidden="true">
      <div class="tp-side">
        <span class="tp-dot" />
        <span class="tp-line" />
        <span class="tp-line short" />
      </div>
      <div class="tp-main">
        <div class="tp-row">
          <span class="tp-pill primary">Активные</span>
          <span class="tp-pill ghost">Архив</span>
        </div>
        <div class="tp-card">
          <span class="tp-line wide" />
          <span class="tp-line" />
          <div class="tp-card-foot">
            <span class="tp-btn">Начать</span>
            <span class="tp-tag" />
          </div>
        </div>
      </div>
    </div>

    <div class="te-colors">
      <label
        v-for="(label, key) in COLOR_LABELS"
        :key="key"
        class="te-color"
      >
        <span class="tec-circle" :style="{ background: vars[key] }">
          <span class="material-symbols-outlined">edit</span>
          <input type="color" v-model="vars[key]" @input="preview" />
        </span>
        <span class="tec-text">
          <span class="tec-name">{{ label.title }}</span>
          <span class="tec-hint">{{ label.hint }}</span>
          <span class="tec-hex">{{ (vars[key] || '').toUpperCase() }}</span>
        </span>
      </label>
    </div>

    <div class="te-name">
      <label class="te-name-label" for="te-name-input">Название темы</label>
      <InputText id="te-name-input" v-model="name" placeholder="Например, «Утренний кофе»" />
      <p v-if="nameTaken" class="te-name-hint">Тема с таким именем уже есть — она будет перезаписана.</p>
    </div>

    <template #footer-start>
      <AppButton icon="auto_awesome" label="Мне повезёт" @click="lucky" />
    </template>
  </AppDialog>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import InputText from 'primevue/inputtext'
import AppDialog from '@/components/ui/AppDialog.vue'
import { useThemeStore } from '@/stores/theme.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  /** Своя тема для правки; null — создаём новую от текущей палитры. */
  source: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue'])

const themeStore = useThemeStore()
const notif = useNotificationsStore()

const COLOR_LABELS = {
  primary: { title: 'Основной', hint: 'Кнопки и активные элементы' },
  secondary: { title: 'Вторичный', hint: 'Ссылки и второстепенные акценты' },
  tertiary: { title: 'Третичный', hint: 'Выделения и плашки' },
  neutral: { title: 'Нейтральный', hint: 'Фоны и поверхности' },
}

const DEFAULT_NEUTRAL = '#e8e6ea'

const vars = reactive({ primary: '', secondary: '', tertiary: '', neutral: DEFAULT_NEUTRAL })
const name = ref('')
/** Палитра, к которой возвращаемся, если закрыть окно без сохранения. */
let restore = null

const nameTaken = computed(() => {
  const n = name.value.trim()
  return !!n && n !== props.source?.name && themeStore.customThemes.some((t) => t.name === n)
})

const actions = computed(() => [
  { kind: 'cancel', label: 'Отмена' },
  { kind: 'confirm', label: 'Сохранить', icon: 'bookmark_add', disabled: !name.value.trim() },
])

watch(() => props.modelValue, (open) => {
  if (!open) return
  const base = props.source?.vars || themeStore.getVars(themeStore.currentPreset)
  restore = { ...themeStore.getVars(themeStore.currentPreset) }
  Object.assign(vars, { ...base, neutral: base.neutral || DEFAULT_NEUTRAL })
  name.value = props.source?.name || nextName()
  preview()
})

function nextName() {
  const taken = new Set(themeStore.customThemes.map((t) => t.name))
  let n = 1
  while (taken.has(`Моя тема ${n}`)) n++
  return `Моя тема ${n}`
}

function preview() {
  themeStore.applyVars({ ...vars })
}

function lucky() {
  Object.assign(vars, { ...themeStore.randomTheme(), neutral: vars.neutral })
  preview()
}

function onVisible(open) {
  if (!open) {
    // Крестик, Esc или «Отмена» — живое превью откатываем.
    if (restore) themeStore.applyVars(restore)
    restore = null
  }
  emit('update:modelValue', open)
}

function save() {
  const n = name.value.trim()
  if (!n) return
  const saved = { ...vars }
  if (props.source && props.source.name !== n) themeStore.deleteCustomTheme(props.source.name)
  themeStore.saveCustomTheme(n, saved)
  themeStore.applyTheme(n)
  restore = null
  notif.success(`Тема «${n}» сохранена`)
  emit('update:modelValue', false)
}
</script>

<style scoped>
/* ── Мини-макет интерфейса: показывает роль каждого цвета ── */
.te-preview {
  display: flex;
  gap: 10px;
  padding: 12px;
  border-radius: var(--radius-lg);
  background: var(--color-surface-low);
  border: 1px solid var(--acrylic-border);
}

.tp-side {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 64px;
  padding: 10px 8px;
  border-radius: var(--radius-md);
  background: var(--color-surface-high);
}

.tp-dot {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--color-primary);
}

.tp-line {
  height: 8px;
  border-radius: 999px;
  background: var(--color-surface-highest);
}

.tp-line.short { width: 60%; }
.tp-line.wide { width: 80%; background: var(--color-secondary-container); }

.tp-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.tp-row { display: flex; gap: 8px; }

.tp-pill {
  padding: 5px 14px;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 600;
}

.tp-pill.primary {
  background: var(--color-primary);
  color: var(--color-on-primary);
}

.tp-pill.ghost {
  background: var(--color-surface-high);
  color: var(--color-text-dim);
}

.tp-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  border-radius: var(--radius-md);
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
}

.tp-card-foot {
  display: flex;
  align-items: center;
  gap: 8px;
}

.tp-btn {
  padding: 5px 12px;
  border-radius: 999px;
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  font-size: 0.72rem;
  font-weight: 600;
}

.tp-tag {
  width: 46px;
  height: 16px;
  border-radius: 999px;
  background: var(--color-tertiary-container);
}

/* ── Цвета ── */
.te-colors {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;
}

.te-color {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px;
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  cursor: pointer;
}

.te-color:hover { background: var(--color-surface-high); }

.tec-circle {
  position: relative;
  display: grid;
  place-items: center;
  width: 44px;
  min-width: 44px;
  max-width: 44px;
  height: 44px;
  min-height: 44px;
  max-height: 44px;
  border-radius: 50%;
  box-shadow: inset 0 0 0 2px var(--color-surface);
  color: #fff;
  overflow: hidden;
}

.tec-circle .material-symbols-outlined {
  font-size: 18px;
  opacity: 0;
  transition: opacity 0.15s ease;
  mix-blend-mode: difference;
}

.te-color:hover .tec-circle .material-symbols-outlined { opacity: 0.9; }

.tec-circle input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.tec-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.tec-name { font-size: 0.9rem; font-weight: 600; }

.tec-hint {
  font-size: 0.75rem;
  color: var(--color-text-dim);
  line-height: 1.25;
}

.tec-hex {
  font-size: 0.7rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  color: var(--color-text-dim);
}

/* ── Имя ── */
.te-name {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 16px;
}

.te-name-label {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--color-text-dim);
}

.te-name :deep(input) { width: 100%; }

.te-name-hint {
  margin: 0;
  font-size: 0.75rem;
  color: var(--color-warning);
}

@media (max-width: 640px) {
  .te-colors { grid-template-columns: 1fr; }
  .te-preview { display: none; }
}
</style>
