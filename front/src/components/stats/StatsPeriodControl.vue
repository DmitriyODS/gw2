<template>
  <div class="period-control">
    <div class="period-display" @click="showPicker = !showPicker">
      <span class="material-symbols-outlined">calendar_month</span>
      {{ period.displayLabel.value }}
    </div>

    <div v-if="showPicker" class="period-picker-overlay" @click.self="showPicker = false">
      <div class="period-picker">
        <DatePicker
          v-model="customRange"
          selection-mode="range"
          date-format="dd.mm.yy"
          inline
          @update:model-value="onCustomRange"
        />
      </div>
    </div>

    <div class="period-buttons">
      <div class="period-group">
        <span class="period-label">день</span>
        <button class="period-btn" @click="period.setDay(1)" title="+1 день">+</button>
        <button class="period-btn" @click="period.setDay(-1)" title="-1 день">−</button>
      </div>
      <div class="period-group">
        <span class="period-label">нед.</span>
        <button class="period-btn" @click="period.setWeek(1)" title="+1 неделя">+</button>
        <button class="period-btn" @click="period.setWeek(-1)" title="-1 неделя">−</button>
      </div>
      <div class="period-group">
        <span class="period-label">мес.</span>
        <button class="period-btn" @click="period.setMonth(1)" title="+1 месяц">+</button>
        <button class="period-btn" @click="period.setMonth(-1)" title="-1 месяц">−</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import DatePicker from 'primevue/datepicker'
import { useStatsPeriod } from '@/composables/useStatsPeriod.js'

const emit = defineEmits(['change'])

const period = useStatsPeriod()
const showPicker = ref(false)
const customRange = ref(null)

function onCustomRange(val) {
  if (Array.isArray(val) && val[0] && val[1]) {
    period.setCustom(val[0], val[1])
    showPicker.value = false
  }
}

watch(
  [period.fromStr, period.toStr],
  ([from, to]) => {
    emit('change', { from, to })
  },
  { immediate: true }
)
</script>

<style scoped>
.period-control {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  padding: 12px 0;
  position: relative;
}

.period-display {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  border: 1px solid var(--gw-border);
  border-radius: 20px;
  cursor: pointer;
  background: var(--gw-surface);
  font-size: 14px;
  color: var(--gw-text);
  transition: border-color 0.15s, background 0.15s;
  user-select: none;
  white-space: nowrap;
}

.period-display:hover {
  border-color: var(--gw-primary);
  background: var(--gw-bg);
}

.period-display .material-symbols-outlined {
  font-size: 18px;
  color: var(--gw-primary);
}

.period-picker-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: flex-start;
  justify-content: flex-start;
}

.period-picker {
  position: absolute;
  top: 56px;
  left: 0;
  z-index: 1001;
  background: var(--gw-surface);
  border: 1px solid var(--gw-border);
  border-radius: var(--gw-radius);
  box-shadow: var(--shadow-lg);
  padding: 8px;
}

.period-buttons {
  display: flex;
  align-items: center;
  gap: 12px;
}

.period-group {
  display: flex;
  align-items: center;
  gap: 4px;
}

.period-label {
  font-size: 13px;
  color: var(--gw-text-secondary);
  margin-right: 2px;
}

.period-btn {
  width: 28px;
  height: 28px;
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  background: var(--gw-surface);
  color: var(--gw-text);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 600;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
  line-height: 1;
  padding: 0;
}

.period-btn:hover {
  background: var(--gw-primary);
  border-color: var(--gw-primary);
  color: var(--color-on-primary);
}

@media (max-width: 768px) {
  .period-control {
    gap: 10px;
    padding: 8px 0;
  }

  .period-display {
    padding: 7px 12px;
    font-size: 13px;
  }

  .period-buttons {
    gap: 8px;
  }

  .period-label {
    font-size: 11px;
  }

  .period-btn {
    width: 26px;
    height: 26px;
    font-size: 14px;
  }
}
</style>
