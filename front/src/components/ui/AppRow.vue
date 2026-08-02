<template>
  <!-- Корень всегда div: строка часто несёт справа собственные кнопки действий,
       а вложенная в <button> кнопка — невалидная разметка, которую браузер
       разбирает по-своему. Роль и клавиатура добавляются вручную. -->
  <div
    class="row"
    :class="[`tone-${tone}`, { clickable, disabled, selected, dense, plain, stacks, 'is-stacked': stack }]"
    :role="clickable ? 'button' : undefined"
    :tabindex="clickable && !disabled ? 0 : undefined"
    :aria-disabled="clickable && disabled ? 'true' : undefined"
    :aria-current="selected ? 'true' : undefined"
    @click="onClick"
    @keydown.enter.prevent="onClick"
    @keydown.space.prevent="onClick"
  >
    <span v-if="icon || slots.lead" class="row-lead">
      <slot name="lead"><span class="material-symbols-outlined row-icon">{{ icon }}</span></slot>
    </span>

    <span class="row-text">
      <strong class="row-title">{{ title }}</strong>
      <small v-if="hint || slots.hint" class="row-hint"><slot name="hint">{{ hint }}</slot></small>
    </span>

    <span v-if="slots.default || clickable" class="row-control">
      <slot />
      <span v-if="chevron" class="material-symbols-outlined row-chev">{{ chevronIcon }}</span>
    </span>
  </div>
</template>

<script setup>
/* Строка — самый частый кирпич платформы: пункт настройки, пункт навигации,
   элемент списка. Слева необязательный значок или превью, в середине название с
   пояснением, справа управление либо шеврон.

   Раскладку строка считает от СВОЕЙ ширины (`@container`), а не от экрана:
   раздел живёт окном рабочего стола, которое пользователь волен ужать. В узкой
   строке ШИРОКОЕ управление переезжает под текст и растягивается, компактное
   (переключатель, шеврон) остаётся справа — как в настройках мобильных ОС.
   Дубль `@media` — для заводского WebView старых Android без `@container`. */
import { computed, useSlots } from 'vue'

const props = defineProps({
  title: { type: String, required: true },
  hint: { type: String, default: '' },
  icon: { type: String, default: '' },
  /** Строка сама по себе действие: подсвечивается, ведёт шевроном, шлёт click. */
  clickable: { type: Boolean, default: false },
  /** Выбранный пункт навигации — тинт primary-container. */
  selected: { type: Boolean, default: false },
  /** Смысловая рамка: просроченное, ошибочное, успешное состояние строки. */
  tone: {
    type: String,
    default: 'neutral',
    validator: (v) => ['neutral', 'primary', 'danger', 'success', 'warning'].includes(v),
  },
  disabled: { type: Boolean, default: false },
  dense: { type: Boolean, default: false },
  /** Без собственного тела: строка внутри карточки-списка, разделители рисует список. */
  plain: { type: Boolean, default: false },
  /** Всегда в столбик — управлению мало места при любой ширине. */
  stack: { type: Boolean, default: false },
  /** Управление компактное (переключатель, значок) — не переносить его вниз. */
  inline: { type: Boolean, default: false },
  /** Шеврон справа: авто у clickable-строк без своего управления. */
  chevronIcon: { type: String, default: 'chevron_right' },
  showChevron: { type: Boolean, default: null },
})

const emit = defineEmits(['click'])
const slots = useSlots()

const stacks = computed(() => !props.inline && !!slots.default)
const chevron = computed(() =>
  props.showChevron === null ? props.clickable && !slots.default : props.showChevron,
)

function onClick(e) {
  if (props.clickable && !props.disabled) emit('click', e)
}
</script>

<style scoped>
.row {
  container-type: inline-size;
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 14px 16px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit;
  text-align: left;
  transition: border-color 0.18s ease, background 0.18s ease, color 0.18s ease;
}

.row.dense { padding: 10px 12px; gap: 10px; }

.row.plain {
  border-color: transparent;
  border-radius: var(--radius-md);
  background: none;
  box-shadow: none;
}

.row.clickable { cursor: pointer; }

.row.clickable:hover:not(.disabled):not(.selected) {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
  background: var(--glass-hover-bg, var(--glass-bg)), var(--acrylic-card-bg);
}

.row.selected {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
  background: var(--glass-bg), var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.row.selected .row-hint,
.row.selected .row-icon,
.row.selected .row-chev { color: inherit; opacity: 0.85; }

.row.disabled { opacity: 0.6; }
.row.disabled.clickable { cursor: default; }

.row.tone-danger:not(.selected) { border-color: color-mix(in oklch, var(--color-error) 45%, var(--acrylic-border)); }
.row.tone-primary:not(.selected) { border-color: color-mix(in oklch, var(--color-primary) 40%, var(--acrylic-border)); }
.row.tone-success:not(.selected) { border-color: color-mix(in oklch, var(--color-success) 40%, var(--acrylic-border)); }
.row.tone-warning:not(.selected) { border-color: color-mix(in oklch, var(--color-warning) 40%, var(--acrylic-border)); }

/* `lead` — не декоративная плашка со значком (их в интерфейсе нет), а значок
   пункта либо содержательное превью: миниатюра фона, аватар, схема раскладки. */
.row-lead {
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.row-icon { font-size: 21px; color: var(--color-text-dim); }

.row-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.row-title {
  font-size: 0.95rem;
  font-weight: 600;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row.dense .row-title { font-size: 0.9rem; }

.row-hint {
  font-size: 0.82rem;
  line-height: 1.4;
  color: var(--color-text-dim);
}

.row-control {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  min-width: 0;
}

.row-chev { font-size: 19px; color: var(--color-text-dim); }

.row.is-stacked {
  flex-direction: column;
  align-items: stretch;
  gap: 12px;
}

.row.is-stacked .row-control { justify-content: stretch; }
.row.is-stacked .row-control > :deep(*) { flex: 1; min-width: 0; }

@container (max-width: 460px) {
  .row.stacks { flex-direction: column; align-items: stretch; gap: 12px; }
  .row.stacks .row-control { justify-content: stretch; }
  .row.stacks .row-control > :deep(*) { flex: 1; min-width: 0; }
}

@media (max-width: 560px) {
  .row.stacks { flex-direction: column; align-items: stretch; gap: 12px; }
  .row.stacks .row-control { justify-content: stretch; }
  .row.stacks .row-control > :deep(*) { flex: 1; min-width: 0; }
}
</style>
