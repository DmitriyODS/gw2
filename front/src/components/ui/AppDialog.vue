<template>
  <Dialog
    :visible="modelValue"
    @update:visible="onVisibleChange"
    modal
    :append-to="host"
    :draggable="false"
    :show-header="false"
    :closable="closable"
    :close-on-escape="closable"
    :style="rootStyle"
    :pt="rootPt"
  >
    <div class="app-dialog" :class="[`tone-${tone}`, `size-${size}`]">
      <!-- Шапка: заголовок с подзаголовком и крестик. Декоративной иконки-плашки
           у диалогов нет — как и во всём остальном интерфейсе. -->
      <header v-if="hasHeader" class="dlg-header">
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
      <!-- Тела может не быть вовсе: у диалога-подтверждения весь текст живёт в
           подзаголовке, и пустая вставленная карточка выглядела бы белой
           полосой ни о чём. Прокручивается ОБОЛОЧКА, а не сама карточка: тогда
           ползунок идёт по краю стекла, снаружи от неё, и не режет скруглённый
           угол. -->
      <div v-if="$slots.default" class="dlg-scroll">
        <div class="dlg-body" :class="{ 'no-padding': bodyNoPadding }">
          <slot />
        </div>
      </div>

      <!-- Подвал: кастомный (slot=footer), слоты footer-start/footer-end или
           встроенный набор кнопок (actions). -->
      <footer v-if="$slots.footer || $slots['footer-start'] || $slots['footer-end'] || actions.length" class="dlg-footer">
        <slot name="footer">
          <!-- Слева — «Отмена» и кастомные кнопки слота (например, «Удалить»);
               справа — главные действия: футер разносит их space-between. -->
          <div class="dlg-footer-start">
            <slot name="footer-start" />
            <AppButton
              v-for="(a, i) in cancelActions"
              :key="`c${i}`"
              variant="glass"
              :icon="a.icon"
              :label="a.label"
              :disabled="a.disabled"
              @click="onAction(a)"
            />
          </div>
          <div class="dlg-footer-end">
            <slot name="footer-end" />
            <AppButton
              v-for="(a, i) in mainActions"
              :key="i"
              :variant="a.kind === 'confirm' ? 'filled' : 'glass'"
              :tone="actionTone(a)"
              :icon="a.icon"
              :label="a.label"
              :loading="a.kind === 'confirm' && busy"
              :disabled="a.disabled"
              @click="onAction(a)"
            />
          </div>
        </slot>
      </footer>
    </div>
  </Dialog>
</template>

<script setup>
import { computed, onBeforeUnmount, watch } from 'vue'
import Dialog from 'primevue/dialog'
import AppButton from './AppButton.vue'
import { registerOpenModal, unregisterOpenModal } from '@/composables/useOpenModals.js'
import { useModalHost } from '@/desktop/windowHost.js'

/* Унифицированный диалог в стиле Material You Expressive.
   Использование:
     <AppDialog v-model="open" tone="danger"
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

/* В режиме рабочего стола диалог раздела живёт в СВОЁМ окне (см.
   desktop/windowHost.js): маска накрывает окно, а не экран. Вне окна — прежний
   body. */
const { host, inWindow } = useModalHost()

// Регистрируем открытие в глобальном счётчике — плавающие виджеты прячутся,
// пока открыт хоть один диалог (см. composables/useOpenModals.js).
watch(() => props.modelValue, (open, prev) => {
  if (open && !prev) registerOpenModal()
  else if (!open && prev) unregisterOpenModal()
}, { immediate: true })
onBeforeUnmount(() => {
  if (props.modelValue) unregisterOpenModal()
})

const hasHeader = computed(() =>
  !!(props.title || props.subtitle || props.closable)
)

// Ширины размеров — в глобальном style ниже (.dlg-size-*).
// dvh, не vh: на мобильных vh = высота при СКРЫТОЙ панели браузера, поэтому
// модалка получалась выше видимой области и обрезалась сверху/снизу (а
// нижний sheet «уезжал» под адресную строку — выглядело узкой полоской).
// В окне рабочего стола эти габариты клампит .gw-in-window-mask (main.css):
// экран там больше окна, и без клампа модалку срезала бы граница окна.
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
  mask: { class: ['app-dialog-mask', { 'gw-in-window-mask': inWindow.value }, props.maskClass] },
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

/* Тон главной кнопки наследуется от диалога, если не задан явно: у диалога
   удаления кнопка «Удалить» красная без лишних указаний. AppButton знает
   primary | danger | success | neutral — остальные тона к ним и сводим. */
function actionTone(a) {
  if (a.kind !== 'confirm') return 'primary'
  const t = a.tone || props.tone
  if (t === 'danger' || t === 'warning') return 'danger'
  if (t === 'success') return 'success'
  return 'primary'
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
  padding: 20px 24px 14px;
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
/* Область прокрутки: ползунок остаётся на стекле, у самой кромки диалога.
   Правое поле у карточки меньше — его добирает сам ползунок. */
.dlg-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  scrollbar-gutter: stable;
}

/* Тело — ПЛОТНАЯ карточка, ВСТАВЛЕННАЯ в стеклянную оболочку: у неё свои
   скруглённые края и поля от кромок диалога, поэтому стекло остаётся видно
   рамкой вокруг — в шапке, подвале и по бокам. Просвечивающее насквозь тело
   читалось плохо: сквозь него проступал интерфейс под диалогом. */
.dlg-body {
  margin: 0 7px;
  padding: 16px 18px 14px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg, 20px);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: 14px;
  line-height: 1.5;
}

/* Содержимое во всю карточку (таблицы, холсты) — поля снимаются, но вставка
   и скругление остаются: иначе край содержимого торчал бы из-под стекла. */
.dlg-body.no-padding { padding: 0; overflow: hidden; }

/* Без шапки и без подвала вставка всё равно нужна: карточка не должна
   срастаться с кромкой стекла. */
.app-dialog:not(:has(.dlg-header)) .dlg-scroll { padding-top: 14px; }

/* Вложенные AppDialog в :has() не попадают (PrimeVue телепортирует каждый
   диалог в body). */
.app-dialog:not(:has(.dlg-footer)) .dlg-scroll { padding-bottom: 14px; }

/* Без тела шапка сливалась бы с кнопками — держим их на расстоянии. */
.app-dialog:not(:has(.dlg-scroll)) .dlg-header { padding-bottom: 4px; }
.app-dialog:not(:has(.dlg-scroll)) .dlg-footer { padding-top: 16px; }

/* Подвал — на стекле, как и шапка: действия обрамляют вставленную карточку. */
.dlg-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 24px 14px;
}

.dlg-footer-start { display: flex; gap: 8px; }
.dlg-footer-end {
  display: flex;
  gap: 8px;
  margin-left: auto;
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
  .dlg-body { margin-left: 5px; margin-right: 5px; padding-left: 14px; padding-right: 14px; }
  .dlg-footer { padding: 10px 20px calc(14px + env(safe-area-inset-bottom, 0px)); }
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
  /* Sheet прижат к нижней кромке экрана: телу без футера нужен ещё
     и запас под home-индикатор. */
  .app-dialog-root:is(.mobile-auto, .mobile-sheet) .app-dialog:not(:has(.dlg-footer)) .dlg-scroll {
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
