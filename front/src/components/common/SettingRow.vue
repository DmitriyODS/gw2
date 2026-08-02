<template>
  <!-- Строка настройки — базовый кирпич раздела «Настройки»: слева название и
       пояснение, справа управление (вкладки, переключатель, кнопка) либо
       шеврон, если строка сама куда-то ведёт.

       Раскладка считается от ширины СТРОКИ (`@container`), а не экрана: панель
       настроек живёт окном рабочего стола, а на телефоне занимает весь экран.
       В узкой строке ШИРОКОЕ управление (вкладки, поля, кнопки) переезжает под
       текст и растягивается — иначе название сжимается в колонку в два слова;
       компактное (переключатель, шеврон) остаётся справа, как в настройках
       мобильных ОС. -->
  <component
    :is="clickable ? 'button' : 'div'"
    class="srow"
    :class="{ clickable, disabled, stacks, 'is-stacked': stack }"
    :type="clickable ? 'button' : undefined"
    :disabled="clickable && disabled ? true : undefined"
    @click="onClick"
  >
    <span v-if="slots.lead" class="srow-lead"><slot name="lead" /></span>

    <span class="srow-text">
      <strong class="srow-title">{{ title }}</strong>
      <small v-if="hint || slots.hint" class="srow-hint"><slot name="hint">{{ hint }}</slot></small>
    </span>

    <span v-if="slots.default || clickable" class="srow-control">
      <slot />
      <span v-if="clickable && !slots.default" class="material-symbols-outlined srow-chev">chevron_right</span>
    </span>
  </component>
</template>

<script setup>
import { computed, useSlots } from 'vue'

const props = defineProps({
  title: { type: String, required: true },
  hint: { type: String, default: '' },
  /** Строка сама по себе действие: подсвечивается, ведёт шевроном, шлёт click. */
  clickable: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  /** Всегда в столбик — управлению мало места при любой ширине. */
  stack: { type: Boolean, default: false },
  /** Управление компактное (переключатель, значок) — не переносить его вниз. */
  inline: { type: Boolean, default: false },
})

const emit = defineEmits(['click'])

const slots = useSlots()

// Переносим вниз только широкое управление: у переключателя и шеврона внизу
// строки был бы растянутый на всю ширину «хвост».
const stacks = computed(() => !props.inline && !!slots.default)

function onClick() {
  if (props.clickable && !props.disabled) emit('click')
}
</script>

<style scoped>
.srow {
  container-type: inline-size;
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 14px 16px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  text-align: left;
  transition: border-color 0.2s ease, background 0.2s ease;
}

.srow.clickable { cursor: pointer; }
.srow.clickable:hover:not(.disabled) { border-color: var(--color-primary); }
.srow.disabled { opacity: 0.6; }
.srow.disabled.clickable { cursor: default; }

/* Слот `lead` — не декоративная плашка со значком (их в интерфейсе нет), а
   содержательное превью: миниатюра фона, аватар, схема раскладки. */
.srow-lead {
  flex-shrink: 0;
  display: flex;
}

.srow-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.srow-title {
  font-size: 0.95rem;
  font-weight: 600;
  line-height: 1.3;
}

.srow-hint {
  font-size: 0.82rem;
  line-height: 1.4;
  color: var(--color-text-dim);
}

.srow-control {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  min-width: 0;
}

.srow-chev { color: var(--color-text-dim); }

/* Заданный руками столбик и узкая строка ведут себя одинаково. Дубль @media —
   для заводского WebView старых Android, который @container не знает (там
   ширина строки практически совпадает с шириной экрана). */
.srow.is-stacked {
  flex-direction: column;
  align-items: stretch;
  gap: 12px;
}

.srow.is-stacked .srow-control { justify-content: stretch; }
.srow.is-stacked .srow-control > :deep(*) { flex: 1; min-width: 0; }

@container (max-width: 460px) {
  .srow.stacks { flex-direction: column; align-items: stretch; gap: 12px; }
  .srow.stacks .srow-control { justify-content: stretch; }
  .srow.stacks .srow-control > :deep(*) { flex: 1; min-width: 0; }
}

@media (max-width: 560px) {
  .srow.stacks { flex-direction: column; align-items: stretch; gap: 12px; }
  .srow.stacks .srow-control { justify-content: stretch; }
  .srow.stacks .srow-control > :deep(*) { flex: 1; min-width: 0; }
}
</style>
