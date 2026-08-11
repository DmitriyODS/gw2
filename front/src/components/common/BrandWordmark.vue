<template>
  <!-- Марка «Groove Work N» — одна и та же надпись в меню «Пуск» рабочего
       стола и в шапке стартового экрана телефона. Мажорная версия приезжает
       с сервера (см. useAppVersion), в бандл не зашивается. -->
  <span class="gw-mark" :style="{ fontSize: `${size}px` }">
    <span class="gw-mark-groove">Groove</span>
    <span class="gw-mark-work">Work</span>
    <span v-if="majorVersion" class="gw-mark-work">{{ majorVersion }}</span>
  </span>
</template>

<script setup>
import { onMounted } from 'vue'
import { useAppVersion } from '@/composables/useAppVersion.js'

defineProps({
  /** Кегль надписи: марка стоит и заголовком «Пуска», и подписью в подвале. */
  size: { type: Number, default: 26 },
})

const { majorVersion, load } = useAppVersion()

onMounted(load)
</script>

<style scoped>
.gw-mark {
  display: flex;
  align-items: baseline;
  /* Промежуток в em — марка ставится и крупно (меню «Пуск»), и мелко (подвал). */
  gap: 0.27em;
  font-size: 26px;
  /* ExtraBlack вариативного Roboto Flex — фирменное начертание wordmark.
     Кнопка не наследует шрифт документа сама, поэтому задаём явно, а вес
     дублируем осью вариативного шрифта. */
  font-family: 'Roboto Flex', 'Roboto', sans-serif;
  font-weight: 1000;
  font-variation-settings: 'wght' 1000;
  letter-spacing: 0.2px;
  white-space: nowrap;
}

.gw-mark-groove { color: var(--color-primary); }
.gw-mark-work { color: var(--color-text); }
</style>
