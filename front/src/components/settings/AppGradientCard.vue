<template>
  <!-- Градиентное сияние — оформление фона приложения, поэтому живёт рядом с
       обоями рабочего стола, а не в палитрах: цвета оно берёт из активной темы
       и следует за ней само. -->
  <div class="ag-card">
    <header class="ag-head">
      <span class="ag-icon material-symbols-outlined">blur_on</span>
      <div class="ag-head-text">
        <h3>Фон приложения</h3>
        <p>
          Мягкие цветные пятна под разделами — как на экране входа. Цвета берутся
          из активной темы и меняются вместе с ней.
        </p>
      </div>
    </header>

    <SwitchRow
      :model-value="themeStore.bgGradient.enabled"
      title="Градиентное сияние"
      hint="Выключено — под окнами останется ровный фон темы."
      icon="gradient"
      @update:model-value="themeStore.setBgGradientEnabled"
    />

    <Transition name="ag-reveal">
      <div v-if="themeStore.bgGradient.enabled" class="ag-actions">
        <button class="btn-glass" type="button" @click="themeStore.regenerateBgGradient()">
          <span class="material-symbols-outlined">shuffle</span>
          Другая композиция
        </button>
        <button class="btn-glass" type="button" @click="themeStore.resetBgGradient()">
          <span class="material-symbols-outlined">restart_alt</span>
          Стандартная
        </button>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import SwitchRow from '@/components/common/SwitchRow.vue'
import { useThemeStore } from '@/stores/theme.js'

const themeStore = useThemeStore()
</script>

<style scoped>
.ag-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 18px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
}

.ag-head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.ag-icon {
  display: grid;
  place-items: center;
  width: 40px;
  min-width: 40px;
  max-width: 40px;
  height: 40px;
  min-height: 40px;
  max-height: 40px;
  border-radius: var(--radius-md);
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  font-size: 22px;
}

.ag-head-text h3 {
  margin: 0 0 2px;
  font-size: 1rem;
  font-weight: 600;
}

.ag-head-text p {
  margin: 0;
  font-size: 0.83rem;
  line-height: 1.4;
  color: var(--color-text-dim);
}

.ag-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.ag-reveal-enter-active, .ag-reveal-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.ag-reveal-enter-from, .ag-reveal-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
