<template>
  <Dialog
    :visible="modelValue"
    @update:visible="onVisibleChange"
    modal
    :draggable="false"
    :show-header="false"
    :closable="closable"
    :close-on-escape="closable"
    :style="rootStyle"
    :pt="rootPt"
  >
    <div class="app-dialog" :class="[`tone-${tone}`, `size-${size}`, { 'has-icon': showIcon }]">
      <!-- Шапка: иконка-тон + заголовок/подзаголовок + крестик. -->
      <header v-if="hasHeader" class="dlg-header">
        <div v-if="showIcon" class="dlg-icon" :class="`tone-${tone}`">
          <span class="material-symbols-outlined">{{ resolvedIcon }}</span>
        </div>
        <div class="dlg-title-wrap">
          <slot name="title">
            <h3 v-if="title" class="dlg-title">{{ title }}</h3>
          </slot>
          <slot name="subtitle">
            <p v-if="subtitle" class="dlg-subtitle">{{ subtitle }}</p>
          </slot>
        </div>
        <button
          v-if="closable && showClose"
          class="dlg-close"
          type="button"
          aria-label="Закрыть"
          @click="cancel"
        >
          <span class="material-symbols-outlined">close</span>
        </button>
      </header>

      <!-- Тело: дефолтный слот. Скроллится при переполнении. -->
      <div class="dlg-body" :class="{ 'no-padding': bodyNoPadding }">
        <slot />
      </div>

      <!-- Подвал: либо кастомный (slot=footer), либо встроенный набор кнопок. -->
      <footer v-if="$slots.footer || actions.length" class="dlg-footer">
        <slot name="footer">
          <!-- Слева — «Отмена» и кастомные кнопки слота (например, «Удалить»);
               справа — главные действия: футер разносит их space-between. -->
          <div class="dlg-footer-start">
            <slot name="footer-start" />
            <template v-for="(a, i) in cancelActions" :key="`c${i}`">
              <button :class="actionClass(a)" :disabled="a.disabled" type="button" @click="onAction(a)">
                <span v-if="a.icon" class="material-symbols-outlined">{{ a.icon }}</span>
                {{ a.label }}
              </button>
            </template>
          </div>
          <div class="dlg-footer-end">
            <template v-for="(a, i) in mainActions" :key="i">
              <button
                :class="actionClass(a)"
                :disabled="a.disabled || (a.kind === 'confirm' && busy)"
                type="button"
                @click="onAction(a)"
              >
                <span v-if="a.kind === 'confirm' && busy" class="dlg-spinner" aria-hidden="true" />
                <span v-else-if="a.icon" class="material-symbols-outlined">{{ a.icon }}</span>
                {{ a.label }}
              </button>
            </template>
          </div>
        </slot>
      </footer>
    </div>
  </Dialog>
</template>

<script setup>
import { computed, onBeforeUnmount, watch } from 'vue'
import Dialog from 'primevue/dialog'
import { registerOpenModal, unregisterOpenModal } from '@/composables/useOpenModals.js'

/* Унифицированный диалог в стиле Material You Expressive.
   Использование:
     <AppDialog v-model="open" tone="danger" icon="delete"
       title="Удалить файл?" subtitle="Это действие нельзя отменить."
       :actions="[
         { kind: 'cancel', label: 'Отмена' },
         { kind: 'confirm', label: 'Удалить', icon: 'delete' },
       ]"
       @confirm="doDelete"
     >
       <!-- произвольное тело -->
     </AppDialog>

   Тоны: primary | tertiary | success | warning | danger | neutral.
   Размеры: sm (380px) | md (520px) | lg (720px) | xl (920px).
   `mobile` ("sheet" — нижний sheet, "full" — полный экран, "auto" — авто).
   `actions[]` — кнопки футера. kind: 'confirm' | 'cancel' | 'neutral'. */

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: '' },
  subtitle: { type: String, default: '' },
  icon: { type: String, default: '' },         // material-symbols-outlined name
  tone: {
    type: String,
    default: 'primary',
    validator: v => ['primary', 'tertiary', 'success', 'warning', 'danger', 'neutral'].includes(v),
  },
  size: {
    type: String,
    default: 'md',
    validator: v => ['sm', 'md', 'lg', 'xl'].includes(v),
  },
  mobile: {
    type: String,
    default: 'auto',
    validator: v => ['auto', 'sheet', 'full'].includes(v),
  },
  closable: { type: Boolean, default: true },
  showClose: { type: Boolean, default: true },
  showIcon: { type: Boolean, default: true },
  busy: { type: Boolean, default: false },
  bodyNoPadding: { type: Boolean, default: false },
  actions: {
    type: Array,
    default: () => [],
    // [{ kind: 'cancel'|'confirm'|'neutral', label, icon?, tone?, disabled? }]
  },
  // Доп. CSS-классы для root и mask (например, поднять z-index над CallView).
  // Это безопаснее, чем хардкодить число — родитель управляет своими слоями.
  dialogClass: { type: [String, Array, Object], default: '' },
  maskClass: { type: [String, Array, Object], default: '' },
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel'])

// Регистрируем открытие в глобальном счётчике — плавающие виджеты прячутся,
// пока открыт хоть один диалог (см. composables/useOpenModals.js).
watch(() => props.modelValue, (open, prev) => {
  if (open && !prev) registerOpenModal()
  else if (!open && prev) unregisterOpenModal()
}, { immediate: true })
onBeforeUnmount(() => {
  if (props.modelValue) unregisterOpenModal()
})

// Иконка по умолчанию для каждого тона — если её не передали явно.
const TONE_ICON_DEFAULT = {
  primary: 'info',
  tertiary: 'lightbulb',
  success: 'check_circle',
  warning: 'warning',
  danger: 'error',
  neutral: 'help',
}

const resolvedIcon = computed(() => props.icon || TONE_ICON_DEFAULT[props.tone])

const hasHeader = computed(() =>
  !!(props.title || props.subtitle || props.showIcon || props.closable)
)

// Ширины размеров — в глобальном style ниже (.dlg-size-*).
// dvh, не vh: на мобильных vh = высота при СКРЫТОЙ панели браузера, поэтому
// модалка получалась выше видимой области и обрезалась сверху/снизу (а
// нижний sheet «уезжал» под адресную строку — выглядело узкой полоской).
const SIZE_MAX_H = {
  sm: 'calc(100dvh - 48px)',
  md: 'calc(100dvh - 48px)',
  lg: 'calc(100dvh - 48px)',
  xl: 'calc(100dvh - 32px)',
}

// Ширина — классом dlg-size-* (не инлайном): на широких экранах md
// расширяется до lg медиазапросом (инлайн-width это перебил бы).
const rootStyle = computed(() => ({
  maxWidth: 'calc(100vw - 24px)',
  maxHeight: SIZE_MAX_H[props.size],
}))

const rootPt = computed(() => ({
  root: { class: ['app-dialog-root', `dlg-size-${props.size}`, `mobile-${props.mobile}`, props.dialogClass] },
  mask: { class: ['app-dialog-mask', props.maskClass] },
  content: { class: 'app-dialog-content' },
}))

function onVisibleChange(v) {
  if (!v) cancel()
}

function cancel() {
  emit('update:modelValue', false)
  emit('cancel')
}

function onAction(a) {
  if (a.kind === 'cancel') {
    cancel()
    return
  }
  if (a.kind === 'confirm') {
    emit('confirm')
    // Закрытие — на совести родителя (через v-model). Если он хочет автоматом —
    // передаст closeOnConfirm. Чаще нужно дождаться async-операции.
    return
  }
  // 'neutral' — кастомное действие, родитель ловит через onClick свойство.
  if (typeof a.onClick === 'function') a.onClick()
}

// «Отмена» — к левому краю футера, главные действия — к правому.
const cancelActions = computed(() => props.actions.filter((a) => a.kind === 'cancel'))
const mainActions = computed(() => props.actions.filter((a) => a.kind !== 'cancel'))

function actionClass(a) {
  if (a.kind === 'cancel') return 'dlg-btn dlg-btn-text'
  if (a.kind === 'confirm') {
    // Тон confirm-кнопки наследуется от диалога, если не задан явно.
    const t = a.tone || props.tone
    if (t === 'danger') return 'dlg-btn dlg-btn-filled tone-danger'
    if (t === 'warning') return 'dlg-btn dlg-btn-filled tone-warning'
    if (t === 'success') return 'dlg-btn dlg-btn-filled tone-success'
    if (t === 'tertiary') return 'dlg-btn dlg-btn-filled tone-tertiary'
    return 'dlg-btn dlg-btn-filled tone-primary'
  }
  return 'dlg-btn dlg-btn-tonal'
}
</script>

<style scoped>
.app-dialog {
  display: flex;
  flex-direction: column;
  min-height: 0;
  flex: 1 1 auto;
  border-radius: var(--radius-xl, 28px);
  background: transparent; /* фон даёт акриловый .app-dialog-root */
  overflow: hidden;
}

/* Шапка — иконка-тон + текст + крестик. Текстовый блок (заголовок+подпись)
   центрируется по вертикали относительно иконки раздела. */
.dlg-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 24px 24px 8px;
}

/* Иконка-бейдж диалога: матовое стекло единого стиля, тон задаёт только
   цвет символа (сплошные тональные круги выбивались из акриловой системы). */
.dlg-icon {
  width: 56px;
  height: 56px;
  flex-shrink: 0;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  color: var(--color-primary);
}

.dlg-icon.tone-tertiary { color: var(--color-tertiary); }
.dlg-icon.tone-success { color: var(--color-success, var(--color-tertiary)); }
.dlg-icon.tone-warning { color: var(--color-warning, var(--color-tertiary)); }
.dlg-icon.tone-danger { color: var(--color-error); }
.dlg-icon.tone-neutral { color: var(--color-text-dim); }

.dlg-icon .material-symbols-outlined {
  font-size: 28px;
  font-variation-settings: 'FILL' 1, 'wght' 500, 'GRAD' 0, 'opsz' 32;
}

.dlg-title-wrap {
  flex: 1;
  min-width: 0;
}

.dlg-title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  letter-spacing: -0.1px;
  color: var(--color-text);
  line-height: 1.25;
}

.dlg-subtitle {
  margin: 4px 0 0;
  font-size: 14px;
  color: var(--color-text-dim);
  line-height: 1.45;
}

.dlg-close {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  margin: -4px -8px 0 0;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: grid;
  place-items: center;
  transition: background 0.15s, color 0.15s;
}

.dlg-close:hover {
  background: var(--color-surface-low);
  color: var(--color-text);
}

.dlg-close .material-symbols-outlined { font-size: 20px; }

/* Тело. */
.dlg-body {
  padding: 12px 24px 4px;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
  color: var(--color-text);
  font-size: 14px;
  line-height: 1.5;
}

.dlg-body.no-padding { padding: 0; }

.app-dialog.has-icon .dlg-body {
  padding-left: 24px;
  padding-right: 24px;
}

/* Когда шапки нет — тело не должно «лепиться» к верху. */
.app-dialog:not(:has(.dlg-header)) .dlg-body { padding-top: 24px; }

/* Подвал. */
.dlg-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 16px 24px 20px;
}

.dlg-footer-start { display: flex; gap: 8px; }
.dlg-footer-end {
  display: flex;
  gap: 8px;
  margin-left: auto;
}

/* Кнопки. */
.dlg-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 40px;
  padding: 0 18px;
  border: none;
  border-radius: var(--radius-full);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  position: relative;
  isolation: isolate;
  overflow: hidden;
  transition: box-shadow 0.18s ease, transform 0.12s ease;
}

.dlg-btn::before {
  content: '';
  position: absolute;
  inset: 0;
  background: currentColor;
  opacity: 0;
  transition: opacity 0.18s ease;
  z-index: -1;
}

.dlg-btn:hover::before { opacity: 0.08; }
.dlg-btn:focus-visible::before { opacity: 0.12; }
.dlg-btn:active::before { opacity: 0.16; }
.dlg-btn:active { transform: scale(0.98); }
.dlg-btn:disabled { opacity: 0.55; cursor: not-allowed; transform: none; }
.dlg-btn:disabled::before { opacity: 0; }

.dlg-btn .material-symbols-outlined { font-size: 18px; }

.dlg-btn-text {
  background: transparent;
  background: var(--glass-bg);
  box-shadow: var(--glass-edge), inset 0 0 0 1px var(--acrylic-border);
  color: var(--color-text);
  padding: 0 14px;
}

/* Второстепенная кнопка — стеклянная (как глобальная .btn-glass). */
.dlg-btn-tonal {
  background: var(--color-secondary-container);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge), inset 0 0 0 1px var(--acrylic-border);
  color: var(--color-text);
}

/* Главные кнопки — пилюли на градиенте тона (как глобальная .btn-grad). */
.dlg-btn-filled.tone-primary {
  background: var(--grad-primary);
  color: var(--color-on-primary);
}
.dlg-btn-filled.tone-danger {
  background: var(--color-error);
  background: linear-gradient(90deg,
    var(--color-error) 0%,
    color-mix(in oklch, var(--color-error) 45%, var(--color-error-container)) 100%);
  color: var(--color-on-error);
}
.dlg-btn-filled.tone-warning {
  background: var(--color-warning, var(--color-tertiary));
  background: linear-gradient(90deg,
    var(--color-warning, var(--color-tertiary)) 0%,
    color-mix(in oklch, var(--color-warning, var(--color-tertiary)) 45%, var(--color-warning-container, var(--color-tertiary-container))) 100%);
  color: var(--color-on-warning, var(--color-on-tertiary));
}
.dlg-btn-filled.tone-success {
  background: var(--color-success);
  background: linear-gradient(90deg,
    var(--color-success) 0%,
    color-mix(in oklch, var(--color-success) 45%, var(--color-success-container)) 100%);
  color: var(--color-on-success);
}
.dlg-btn-filled.tone-tertiary {
  background: var(--color-tertiary);
  background: linear-gradient(90deg,
    var(--color-tertiary) 0%,
    color-mix(in oklch, var(--color-tertiary) 45%, var(--color-tertiary-container)) 100%);
  color: var(--color-on-tertiary);
}

.dlg-btn-filled:hover { box-shadow: var(--shadow-sm); filter: brightness(1.06); }

/* Спиннер для busy-кнопки. */
.dlg-spinner {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 2px solid currentColor;
  border-right-color: transparent;
  animation: dlg-spin 0.6s linear infinite;
}

@keyframes dlg-spin {
  to { transform: rotate(360deg); }
}

/* Мобильная адаптация: на ≤768 (единый мобильный брейкпоинт приложения,
   см. useBreakpoint) — bottom sheet (по умолчанию), полный экран
   (mobile="full") или авто (sheet). */
@media (max-width: 768px) {
  .app-dialog {
    border-radius: var(--radius-xl, 28px) var(--radius-xl, 28px) 0 0;
    /* dvh — учитывает динамическую панель браузера; иначе sheet выше экрана. */
    max-height: 90dvh;
  }
  .dlg-header { padding: 20px 20px 4px; }
  .dlg-body { padding-left: 20px; padding-right: 20px; }
  .dlg-footer { padding: 16px 20px calc(20px + env(safe-area-inset-bottom, 0px)); }
  .dlg-footer-end { flex-wrap: wrap; justify-content: flex-end; }
}
</style>

<!-- Глобальные стили для маски и контейнера диалога (не scoped — PrimeVue
     рендерит их вне дерева компонента через Teleport). -->
<style>
.app-dialog-root {
  border-radius: var(--radius-xl, 28px) !important;
  /* Акрил: полупрозрачная карточка с блюром контента под ней */
  background: var(--acrylic-bg) !important;
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border) !important;
  box-shadow: var(--shadow-xl, 0 24px 60px rgba(0, 0, 0, 0.25)) !important;
  overflow: hidden !important;
  display: flex !important;
  flex-direction: column !important;
}

/* Ширины размеров. На больших экранах md-диалоги растягиваются до lg —
   520px на десктопе читается слишком узко. */
.app-dialog-root.dlg-size-sm { width: 380px; }
.app-dialog-root.dlg-size-md { width: 520px; }
.app-dialog-root.dlg-size-lg { width: 720px; }
.app-dialog-root.dlg-size-xl { width: 920px; }
@media (min-width: 1200px) {
  .app-dialog-root.dlg-size-md { width: 720px; }
}

/* Маска светлее и с сильным блюром: стекло модалки показывает размытый
   контент страницы, а не тёмную пелену (как панель ассистента). */
.app-dialog-mask {
  background: color-mix(in oklch, var(--color-scrim) 45%, transparent) !important;
  -webkit-backdrop-filter: blur(12px) saturate(1.2);
  backdrop-filter: blur(12px) saturate(1.2);
}

.app-dialog-content {
  padding: 0 !important;
  background: transparent !important;
  border-radius: var(--radius-xl, 28px) !important;
  overflow: hidden !important;
  flex: 1 1 auto !important;
  min-height: 0 !important;
  display: flex !important;
  flex-direction: column !important;
}

/* На мобильном — bottom sheet: прижимаем диалог к низу, скругление сверху. */
@media (max-width: 768px) {
  .app-dialog-root.mobile-auto,
  .app-dialog-root.mobile-sheet {
    position: fixed !important;
    bottom: 0 !important;
    left: 0 !important;
    right: 0 !important;
    width: 100vw !important;
    max-width: 100vw !important;
    margin: 0 !important;
    border-radius: var(--radius-xl, 28px) var(--radius-xl, 28px) 0 0 !important;
  }
  .app-dialog-root.mobile-full {
    position: fixed !important;
    inset: 0 !important;
    width: 100vw !important;
    height: 100dvh !important;
    max-width: 100vw !important;
    max-height: 100dvh !important;
    margin: 0 !important;
    border-radius: 0 !important;
    /* Явная flex-колонка: контент тянется на всю высоту, футер прижат к
       низу — без этого под кнопками остаётся пустая полоса. */
    display: flex !important;
    flex-direction: column !important;
  }
  .app-dialog-root.mobile-full .dlg-footer {
    margin-top: auto;
    padding-bottom: calc(14px + env(safe-area-inset-bottom, 0px));
  }
  /* Внутренний контейнер в full-режиме не ограничен sheet-высотой 90dvh —
     иначе под футером остаётся пустая полоса в 10% экрана. */
  .app-dialog-root.mobile-full .app-dialog {
    max-height: none;
    border-radius: 0;
  }
}
</style>
