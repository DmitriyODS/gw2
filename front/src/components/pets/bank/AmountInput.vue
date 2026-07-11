<template>
  <!-- Кастомное поле суммы кудосов: НИКАКИХ нативных браузерных спиннеров
       (type=text + inputmode=numeric, как FieldInput для чисел) — только
       цифры, опциональный кламп к max, монетка справа. -->
  <div class="kai" :class="`kai--${size}`">
    <input
      ref="inputEl"
      type="text"
      inputmode="numeric"
      autocomplete="off"
      :value="modelValue ?? ''"
      :placeholder="placeholder"
      :disabled="disabled"
      @input="onInput"
    />
    <KudosCoin v-if="coin" class="kai-coin" />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'

const props = defineProps({
  modelValue: { type: Number, default: null },
  placeholder: { type: String, default: 'Сумма' },
  max: { type: Number, default: null },
  coin: { type: Boolean, default: true },
  disabled: { type: Boolean, default: false },
  /* md — крупное поле карточек Вклад/Кредит; sm — инлайн в копилках/сборах. */
  size: {
    type: String,
    default: 'md',
    validator: (v) => ['md', 'sm'].includes(v),
  },
})
const emit = defineEmits(['update:modelValue'])

const inputEl = ref(null)
defineExpose({ focus: () => inputEl.value?.focus() })

function onInput(e) {
  const digits = e.target.value.replace(/\D/g, '')
  if (!digits) {
    e.target.value = ''
    emit('update:modelValue', null)
    return
  }
  let n = parseInt(digits, 10)
  if (props.max != null && n > props.max) n = props.max
  // Санитизация видна сразу (вставка «12abc» → «12», сверх max — кламп).
  e.target.value = String(n)
  emit('update:modelValue', n)
}
</script>

<style scoped>
.kai {
  position: relative;
  display: flex;
  align-items: center;
  width: 100%;
}
.kai input {
  width: 100%;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit;
  font-weight: 700;
  appearance: none;
  -webkit-appearance: none;
}
.kai input:focus { outline: none; border-color: var(--color-primary); }
.kai input:disabled { opacity: 0.55; }
.kai input::placeholder { font-weight: 500; color: var(--color-text-dim); }
.kai-coin { position: absolute; right: 12px; pointer-events: none; }

.kai--md input { font-size: 17px; padding: 11px 40px 11px 14px; }
.kai--md .kai-coin { font-size: 18px; }
.kai--sm input { font-size: 13px; padding: 8px 30px 8px 10px; border-radius: var(--radius-sm); }
.kai--sm .kai-coin { font-size: 13px; right: 9px; }
</style>
