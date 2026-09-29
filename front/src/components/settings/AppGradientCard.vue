<template>
  <!-- Градиентное сияние — оформление фона приложения, поэтому живёт рядом с
       обоями рабочего стола, а не в палитрах: цвета оно берёт из активной темы
       и следует за ней само. -->
  <AppCard
    title="Фон приложения"
    hint="Мягкие цветные пятна под разделами — как на экране входа. Цвета берутся из активной темы и меняются вместе с ней."
  >
    <AppSwitchRow
      :model-value="themeStore.bgGradient.enabled"
      title="Градиентное сияние"
      hint="Выключено — под окнами останется ровный фон темы."
      @update:model-value="themeStore.setBgGradientEnabled"
    />

    <Transition name="ag-reveal">
      <div v-if="themeStore.bgGradient.enabled" class="ag-actions">
        <AppButton
          icon="shuffle"
          label="Другая композиция"
          @click="themeStore.regenerateBgGradient()"
        />
        <AppButton icon="restart_alt" label="Стандартная" @click="themeStore.resetBgGradient()" />
      </div>
    </Transition>

    <!-- Про железо, а не про вкус: плотные панели разгружают видеокарту. -->
    <AppSwitchRow
      :model-value="reduceTransparency"
      title="Меньше прозрачности"
      :hint="transparencyChoice
        ? 'Плотные панели вместо размытого стекла — легче для видеокарты и батареи.'
        : 'Плотные панели вместо размытого стекла — легче для видеокарты и батареи. Сейчас — как в системе.'"
      @update:model-value="setReduceTransparency"
    />
    <div v-if="transparencyChoice" class="ag-actions">
      <AppButton variant="text" icon="settings_suggest" label="Как в системе" @click="setReduceTransparency(null)" />
    </div>
  </AppCard>
</template>

<script setup>
import AppCard from '@/components/ui/AppCard.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'
import { useThemeStore } from '@/stores/theme.js'
import { reduceTransparency, setReduceTransparency, transparencyChoice } from '@/utils/transparency.js'

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
