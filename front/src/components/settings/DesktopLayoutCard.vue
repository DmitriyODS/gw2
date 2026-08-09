<template>
  <!-- Каркас — свойство УСТРОЙСТВА, а не человека: настройка живёт локально и
       не уезжает на сервер вместе с обоями и плитками (см. useShellMode). -->
  <AppCard
    title="Раскладка"
    hint="Окна удобны мышью, плитки и зоны — пальцем. По умолчанию выбираем сами: сенсорный экран получает планшетную раскладку, остальные — окна."
  >
    <AppRow
      title="Каркас приложения"
      hint="Настройка только для этого устройства — на других она останется прежней."
    >
      <!-- full-width: в узкой строке управление переезжает под текст и
           растягивается (см. AppRow) — без него вкладки жались влево, а справа
           оставалась пустая полоса. -->
      <AppTabs
        variant="tint"
        :model-value="mode"
        :tabs="MODE_TABS"
        full-width
        dense
        @update:model-value="setShellMode"
      />
    </AppRow>

    <AppInfoBar :message="activeText" />
  </AppCard>
</template>

<script setup>
import { computed } from 'vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import { setShellMode, shellModeSetting, useShellMode } from '@/composables/useShellMode.js'

const MODE_TABS = [
  { key: 'auto', label: 'Авто' },
  { key: 'windows', label: 'Окна' },
  { key: 'tablet', label: 'Планшет' },
]

const mode = computed(() => shellModeSetting.value)
const { shell } = useShellMode()

const activeText = computed(() => {
  if (shell.value === 'phone') {
    return 'Экран узкий — сейчас работает раскладка телефона: разделы во весь экран и панель задач снизу. Выбор выше начнёт действовать на большом экране.'
  }
  if (shell.value === 'tablet') {
    return 'Сейчас планшетная раскладка: раздел открывается во весь экран, а второй можно поставить рядом — долгим нажатием на плитке или кнопке панели задач («Открыть рядом»). Границу зон двигают перетаскиванием.'
  }
  return 'Сейчас рабочий стол с окнами: разделы открываются окнами, их можно двигать, раскладывать рядом и сворачивать.'
})
</script>
