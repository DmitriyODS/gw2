<template>
  <AppDialog
    :model-value="modelValue"
    title="Настройки табло"
    subtitle="Применяются сразу и запоминаются на этом устройстве"
    icon="tune"
    tone="primary"
    size="md"
    :actions="[{ kind: 'cancel', label: 'Готово' }]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
  >
    <div class="tvs">
      <div class="tvs-group-title">Слайды</div>
      <label v-for="s in slides" :key="s.id" class="tvs-row">
        <Checkbox :model-value="!isDisabled(s.id)" binary @update:model-value="toggle(s.id, $event)" />
        <span class="tvs-main">
          <span class="tvs-title">{{ s.title }}</span>
          <span v-if="s.settingsNote" class="tvs-desc">{{ s.settingsNote }}</span>
        </span>
      </label>

      <div class="tvs-group-title">Ротация</div>
      <div class="tvs-slider-row">
        <span class="tvs-slider-label">Длительность слайда</span>
        <Slider
          :model-value="settings.slideSec"
          :min="5" :max="30" :step="1"
          class="tvs-slider"
          @update:model-value="patch('slideSec', $event)"
        />
        <span class="tvs-slider-value">{{ settings.slideSec }} с</span>
      </div>
      <div class="tvs-slider-row">
        <span class="tvs-slider-label">Брендовый слайд</span>
        <Slider
          :model-value="settings.brandSec"
          :min="5" :max="60" :step="1"
          class="tvs-slider"
          @update:model-value="patch('brandSec', $event)"
        />
        <span class="tvs-slider-value">{{ settings.brandSec }} с</span>
      </div>

      <div class="tvs-group-title">Форматирование</div>
      <div class="tvs-slider-row">
        <span class="tvs-slider-label">Часов в рабочем дне</span>
        <Slider
          :model-value="settings.hoursPerDay"
          :min="4" :max="24" :step="1"
          class="tvs-slider"
          @update:model-value="patch('hoursPerDay', $event)"
        />
        <span class="tvs-slider-value">{{ settings.hoursPerDay }} ч</span>
      </div>
      <p class="tvs-hint">Большие объёмы часов показываются как «N дн M ч» — рабочий день считается по этой длине.</p>
    </div>
  </AppDialog>
</template>

<script setup>
// Настройки табло: включение слайдов, длительность ротации, длина рабочего
// дня. Хранятся в localStorage (пишет TvView), применяются без перезагрузки.
import Checkbox from 'primevue/checkbox'
import Slider from 'primevue/slider'
import AppDialog from '@/components/common/AppDialog.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  // Каталог слайдов для чекбоксов: [{id, title, settingsNote?}]
  slides: { type: Array, default: () => [] },
  // Текущие настройки: {disabled: [], slideSec, brandSec, hoursPerDay}
  settings: { type: Object, required: true },
})

const emit = defineEmits(['update:modelValue', 'update:settings'])

function isDisabled(id) {
  return props.settings.disabled.includes(id)
}

function toggle(id, enabled) {
  const disabled = enabled
    ? props.settings.disabled.filter(x => x !== id)
    : [...props.settings.disabled, id]
  emit('update:settings', { ...props.settings, disabled })
}

function patch(key, val) {
  emit('update:settings', { ...props.settings, [key]: val })
}
</script>

<style scoped>
.tvs { display: flex; flex-direction: column; gap: 8px; }

.tvs-group-title {
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--color-text-dim);
  margin-top: 10px;
}

.tvs-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 8px 12px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  cursor: pointer;
}

.tvs-row:hover { background: var(--color-surface-high); }

.tvs-main { display: flex; flex-direction: column; min-width: 0; }
.tvs-title { font-size: 14px; font-weight: 600; color: var(--color-text); }
.tvs-desc { font-size: 12px; color: var(--color-text-dim); }

.tvs-slider-row {
  display: grid;
  grid-template-columns: minmax(140px, 1fr) 2fr 56px;
  align-items: center;
  gap: 14px;
  padding: 8px 12px;
}

.tvs-slider-label { font-size: 14px; font-weight: 600; color: var(--color-text); }
.tvs-slider { width: 100%; }
.tvs-slider-value {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-primary);
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.tvs-hint {
  margin: 0;
  padding: 0 12px;
  font-size: 12px;
  color: var(--color-text-dim);
}
</style>
