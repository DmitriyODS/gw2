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
      <header v-if="hasHeader" class="ad-header">
        <div v-if="showIcon" class="ad-icon" :class="`tone-${tone}`">
          <span class="material-symbols-outlined">{{ resolvedIcon }}</span>
        </div>
        <div class="ad-title-wrap">
          <slot name="title">
            <h3 v-if="title" class="ad-title">{{ title }}</h3>
          </slot>
          <slot name="subtitle">
            <p v-if="subtitle" class="ad-subtitle">{{ subtitle }}</p>
          </slot>
        </div>
        <button
          v-if="closable && showClose"
          class="ad-close"
          type="button"
          aria-label="Закрыть"
          @click="cancel"
        >
          <span class="material-symbols-outlined">close</span>
        </button>
      </header>

      <!-- Тело: дефолтный слот. Скроллится при переполнении. -->
      <div class="ad-body" :class="{ 'no-padding': bodyNoPadding }">
        <slot />
      </div>

      <!-- Подвал: либо кастомный (slot=footer), либо встроенный набор кнопок. -->
      <footer v-if="$slots.footer || actions.length" class="ad-footer">
        <slot name="footer">
          <!-- Левые кнопки (например, «Удалить» на форме редактирования). -->
          <div v-if="$slots['footer-start']" class="ad-footer-start">
            <slot name="footer-start" />
          </div>
          <div class="ad-footer-end">
            <template v-for="(a, i) in actions" :key="i">
              <button
                :class="actionClass(a)"
                :disabled="a.disabled || (a.kind === 'confirm' && busy)"
                type="button"
                @click="onAction(a)"
              >
                <span v-if="a.kind === 'confirm' && busy" class="ad-spinner" aria-hidden="true" />
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
import { computed } from 'vue'
import Dialog from 'primevue/dialog'

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

const SIZE_WIDTH = { sm: '380px', md: '520px', lg: '720px', xl: '920px' }
// dvh, не vh: на мобильных vh = высота при СКРЫТОЙ панели браузера, поэтому
// модалка получалась выше видимой области и обрезалась сверху/снизу (а
// нижний sheet «уезжал» под адресную строку — выглядело узкой полоской).
const SIZE_MAX_H = {
  sm: 'calc(100dvh - 48px)',
  md: 'calc(100dvh - 48px)',
  lg: 'calc(100dvh - 48px)',
  xl: 'calc(100dvh - 32px)',
}

const rootStyle = computed(() => ({
  width: SIZE_WIDTH[props.size],
  maxWidth: 'calc(100vw - 24px)',
  maxHeight: SIZE_MAX_H[props.size],
}))

const rootPt = computed(() => ({
  root: { class: ['app-dialog-root', `mobile-${props.mobile}`, props.dialogClass] },
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

function actionClass(a) {
  if (a.kind === 'cancel') return 'ad-btn ad-btn-text'
  if (a.kind === 'confirm') {
    // Тон confirm-кнопки наследуется от диалога, если не задан явно.
    const t = a.tone || props.tone
    if (t === 'danger') return 'ad-btn ad-btn-filled tone-danger'
    if (t === 'warning') return 'ad-btn ad-btn-filled tone-warning'
    if (t === 'success') return 'ad-btn ad-btn-filled tone-success'
    if (t === 'tertiary') return 'ad-btn ad-btn-filled tone-tertiary'
    return 'ad-btn ad-btn-filled tone-primary'
  }
  return 'ad-btn ad-btn-tonal'
}
</script>

<style scoped>
.app-dialog {
  display: flex;
  flex-direction: column;
  min-height: 0;
  flex: 1 1 auto;
  border-radius: var(--radius-xl, 28px);
  background: var(--color-surface);
  overflow: hidden;
}

/* Шапка — иконка-тон + текст + крестик. */
.ad-header {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 24px 24px 8px;
}

.ad-icon {
  width: 56px;
  height: 56px;
  flex-shrink: 0;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.ad-icon.tone-tertiary {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.ad-icon.tone-success {
  background: var(--color-success-container, var(--color-tertiary-container));
  color: var(--color-on-success-container, var(--color-on-tertiary-container));
}
.ad-icon.tone-warning {
  background: var(--color-warning-container, var(--color-tertiary-container));
  color: var(--color-on-warning-container, var(--color-on-tertiary-container));
}
.ad-icon.tone-danger {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}
.ad-icon.tone-neutral {
  background: var(--color-surface-high);
  color: var(--color-text);
}

.ad-icon .material-symbols-outlined {
  font-size: 28px;
  font-variation-settings: 'FILL' 1, 'wght' 500, 'GRAD' 0, 'opsz' 32;
}

.ad-title-wrap {
  flex: 1;
  min-width: 0;
  padding-top: 8px;
}

.ad-title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  letter-spacing: -0.1px;
  color: var(--color-text);
  line-height: 1.25;
}

.ad-subtitle {
  margin: 4px 0 0;
  font-size: 14px;
  color: var(--color-text-dim);
  line-height: 1.45;
}

.ad-close {
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

.ad-close:hover {
  background: var(--color-surface-low);
  color: var(--color-text);
}

.ad-close .material-symbols-outlined { font-size: 20px; }

/* Тело. */
.ad-body {
  padding: 12px 24px 4px;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
  color: var(--color-text);
  font-size: 14px;
  line-height: 1.5;
}

.ad-body.no-padding { padding: 0; }

.app-dialog.has-icon .ad-body {
  padding-left: 24px;
  padding-right: 24px;
}

/* Когда шапки нет — тело не должно «лепиться» к верху. */
.app-dialog:not(:has(.ad-header)) .ad-body { padding-top: 24px; }

/* Подвал. */
.ad-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 16px 24px 20px;
}

.ad-footer-start { display: flex; gap: 8px; }
.ad-footer-end {
  display: flex;
  gap: 8px;
  margin-left: auto;
}

/* Кнопки. */
.ad-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 40px;
  padding: 0 18px;
  border: none;
  border-radius: var(--radius-full);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  position: relative;
  isolation: isolate;
  overflow: hidden;
  transition: box-shadow 0.18s ease, transform 0.12s ease;
}

.ad-btn::before {
  content: '';
  position: absolute;
  inset: 0;
  background: currentColor;
  opacity: 0;
  transition: opacity 0.18s ease;
  z-index: -1;
}

.ad-btn:hover::before { opacity: 0.08; }
.ad-btn:focus-visible::before { opacity: 0.12; }
.ad-btn:active::before { opacity: 0.16; }
.ad-btn:active { transform: scale(0.98); }
.ad-btn:disabled { opacity: 0.55; cursor: not-allowed; transform: none; }
.ad-btn:disabled::before { opacity: 0; }

.ad-btn .material-symbols-outlined { font-size: 18px; }

.ad-btn-text {
  background: transparent;
  color: var(--color-primary);
  padding: 0 14px;
}

.ad-btn-tonal {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}

.ad-btn-filled.tone-primary {
  background: var(--color-primary);
  color: var(--color-on-primary);
}
.ad-btn-filled.tone-danger {
  background: var(--color-error);
  color: var(--color-on-error);
}
.ad-btn-filled.tone-warning {
  background: var(--color-warning, var(--color-tertiary));
  color: var(--color-on-warning, var(--color-on-tertiary));
}
.ad-btn-filled.tone-success {
  background: var(--color-success);
  color: var(--color-on-success);
}
.ad-btn-filled.tone-tertiary {
  background: var(--color-tertiary);
  color: var(--color-on-tertiary);
}

.ad-btn-filled:hover { box-shadow: var(--shadow-sm); }

/* Спиннер для busy-кнопки. */
.ad-spinner {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 2px solid currentColor;
  border-right-color: transparent;
  animation: ad-spin 0.6s linear infinite;
}

@keyframes ad-spin {
  to { transform: rotate(360deg); }
}

/* Мобильная адаптация: на ≤600 — bottom sheet (по умолчанию),
   полный экран (mobile="full") или авто (sheet). */
@media (max-width: 600px) {
  .app-dialog {
    border-radius: var(--radius-xl, 28px) var(--radius-xl, 28px) 0 0;
    /* dvh — учитывает динамическую панель браузера; иначе sheet выше экрана. */
    max-height: 90dvh;
  }
  .ad-header { padding: 20px 20px 4px; }
  .ad-body { padding-left: 20px; padding-right: 20px; }
  .ad-footer { padding: 16px 20px calc(20px + env(safe-area-inset-bottom, 0px)); }
  .ad-footer-end { flex-wrap: wrap; justify-content: flex-end; }
}
</style>

<!-- Глобальные стили для маски и контейнера диалога (не scoped — PrimeVue
     рендерит их вне дерева компонента через Teleport). -->
<style>
.app-dialog-root {
  border-radius: var(--radius-xl, 28px) !important;
  background: var(--color-surface) !important;
  border: 1px solid var(--color-outline-dim) !important;
  box-shadow: var(--shadow-xl, 0 24px 60px rgba(0, 0, 0, 0.25)) !important;
  overflow: hidden !important;
  display: flex !important;
  flex-direction: column !important;
}

.app-dialog-mask {
  background: var(--color-scrim, color-mix(in oklch, var(--color-text) 60%, transparent)) !important;
  backdrop-filter: blur(2px);
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
@media (max-width: 600px) {
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
  }
}
</style>
