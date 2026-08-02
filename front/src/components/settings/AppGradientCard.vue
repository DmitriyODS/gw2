<template>
  <!-- Градиентное сияние — оформление фона приложения, поэтому живёт рядом с
       обоями рабочего стола, а не в палитрах: цвета оно берёт из активной темы
       и следует за ней само. -->
  <SettingCard
    title="Фон приложения"
    hint="Мягкие цветные пятна под разделами — как на экране входа. Цвета берутся из активной темы и меняются вместе с ней."
  >
    <SwitchRow
      :model-value="themeStore.bgGradient.enabled"
      title="Градиентное сияние"
      hint="Выключено — под окнами останется ровный фон темы."
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
  </SettingCard>
</template>

<script setup>
import SettingCard from '@/components/common/SettingCard.vue'
import SwitchRow from '@/components/common/SwitchRow.vue'
import { useThemeStore } from '@/stores/theme.js'

const themeStore = useThemeStore()
</script>

<style scoped>
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
