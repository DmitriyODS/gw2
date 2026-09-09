<script setup>
/* Числовая величина холста — ползунком и вводом: толщина обводки, размер
   ластика, кегль надписи. Один компонент на три места, потому что правило у
   них одно: тянуть удобно на глаз, а точное значение проще набрать.

   Пресеты — не замена вводу, а быстрый доступ к ходовым величинам. */
import InputNumber from 'primevue/inputnumber'
import Slider from 'primevue/slider'

const props = defineProps({
  modelValue: { type: Number, default: 0 },
  min: { type: Number, default: 0 },
  max: { type: Number, default: 100 },
  step: { type: Number, default: 1 },
  unit: { type: String, default: 'px' },
  label: { type: String, default: '' },
  // Ходовые значения кнопками; пустой список — только ползунок и поле.
  presets: { type: Array, default: () => [] },
  // Подпись нулю: у обводки это «без линии», у прочего нуля обычно нет.
  zeroLabel: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue'])

function set(value) {
  const n = Number(value)
  if (!Number.isFinite(n)) return
  emit('update:modelValue', Math.min(props.max, Math.max(props.min, Math.round(n / props.step) * props.step)))
}
</script>

<template>
  <div class="sf">
    <span v-if="label" class="sf-label">
      {{ label }}
      <template v-if="zeroLabel && modelValue === 0"> · {{ zeroLabel }}</template>
    </span>
    <div class="sf-row">
      <Slider
        class="sf-slider"
        :model-value="modelValue"
        :min="min"
        :max="max"
        :step="step"
        :disabled="disabled"
        @update:model-value="set"
      />
      <InputNumber
        class="sf-num"
        :model-value="modelValue"
        :min="min"
        :max="max"
        :step="step"
        :suffix="unit ? ` ${unit}` : ''"
        :disabled="disabled"
        @update:model-value="set"
      />
    </div>
    <div v-if="presets.length" class="sf-presets">
      <button
        v-for="p in presets"
        :key="p"
        type="button"
        class="sf-preset"
        :class="{ 'is-on': modelValue === p }"
        :disabled="disabled"
        @click="set(p)"
      >{{ p }}</button>
      <button
        v-if="zeroLabel"
        type="button"
        class="sf-preset"
        :class="{ 'is-on': modelValue === 0 }"
        :disabled="disabled"
        :title="zeroLabel"
        @click="set(0)"
      >0</button>
    </div>
  </div>
</template>

<style scoped>
.sf { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.sf-label { font-size: 11px; color: var(--color-text-muted); }
.sf-row { display: flex; align-items: center; gap: 8px; min-width: 0; }
.sf-slider { flex: 1; min-width: 0; }

/* Ширину держат ОБА элемента поля: PrimeVue рисует обёртку и <input> внутри,
   и без явной ширины у самого input он оставался по своему размеру и вылезал
   за край панели. */
.sf-num {
  flex: 0 0 76px;
  width: 76px;
  min-width: 0;
  max-width: 76px;
}

.sf-num :deep(input) {
  width: 100%;
  min-width: 0;
  padding: 4px 6px;
  font-size: 12px;
  text-align: right;
  box-sizing: border-box;
}

.sf-presets { display: flex; flex-wrap: wrap; gap: 4px; }

.sf-preset {
  min-width: 32px;
  height: 24px;
  padding: 0 6px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text);
  font-size: 11px;
  cursor: pointer;
}

.sf-preset:hover:not(:disabled) { background: var(--color-surface-variant); }
.sf-preset:disabled { opacity: 0.5; cursor: default; }
.sf-preset.is-on { border-color: var(--color-primary); color: var(--color-primary); }
</style>
