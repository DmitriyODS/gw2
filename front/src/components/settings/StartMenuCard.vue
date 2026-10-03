<template>
  <!-- Что показывает меню «Пуск» — на столе, планшете и телефоне одинаково.
       Настройки личные и синхронизируются между устройствами. -->
  <AppCard title="Меню «Пуск»">
    <AppSwitchRow
      :model-value="prefs.startFavorites"
      title="Показывать избранное"
      hint="Меню открывается на избранных разделах. Выключено — сразу на всех разделах."
      @update:model-value="prefs.setStartFavorites"
    />
    <AppSwitchRow
      :model-value="prefs.startActivity"
      title="Показывать последнюю активность"
      hint="Недавно открытые разделы и созданные вами записи — рядом с разделами на рабочем столе и планшете."
      @update:model-value="prefs.setStartActivity"
    />
  </AppCard>
</template>

<script setup>
import AppCard from '@/components/ui/AppCard.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'

const prefs = useDesktopPrefsStore()
</script>
