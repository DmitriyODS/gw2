<template>
  <AppCard
    title="Живые плитки"
    hint="Плитки меню «Пуск» по очереди показывают сводку своего раздела: дела на сегодня, ближайшие события, непрочитанное. Настройка личная и синхронизируется на всех ваших устройствах."
  >
    <AppSwitchRow
      :model-value="prefs.liveTiles"
      title="Показывать сводки на плитках"
      hint="Выключено — плитки остаются обычными значками, данные не запрашиваются. Отдельную плитку можно погасить в её меню: правая кнопка мыши, на телефоне — долгое нажатие."
      @update:model-value="prefs.setLiveTiles"
    />
  </AppCard>
</template>

<script setup>
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import AppCard from '@/components/ui/AppCard.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'

const prefs = useDesktopPrefsStore()
</script>
