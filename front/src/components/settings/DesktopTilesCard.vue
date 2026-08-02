<template>
  <SettingCard
    title="Живые плитки"
    hint="Плитки меню «Пуск» по очереди показывают сводку своего раздела: дела на сегодня, ближайшие события, непрочитанное. Настройка личная и синхронизируется на всех ваших устройствах."
  >
    <SwitchRow
      :model-value="prefs.liveTiles"
      title="Показывать сводки на плитках"
      hint="Выключено — плитки остаются обычными значками, данные не запрашиваются."
      @update:model-value="prefs.setLiveTiles"
    />
  </SettingCard>
</template>

<script setup>
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import SettingCard from '@/components/common/SettingCard.vue'
import SwitchRow from '@/components/common/SwitchRow.vue'

const prefs = useDesktopPrefsStore()
</script>
