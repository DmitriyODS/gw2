<template>
  <!-- Раздел про то, КАК приложение раскладывает разделы и чем ими управлять.
       Оформление (обои, градиент фона) живёт в «Темах и оформлении»: оно про
       внешний вид и одинаково работает во всех каркасах. Настройки самой
       десктоп-обёртки — в «Общих»: они про установленное приложение. -->
  <AppStack>
    <DesktopLayoutCard />
    <DesktopShellCard v-if="windowsShell" />
    <StartMenuCard />
    <DesktopTilesCard />
  </AppStack>
</template>

<script setup>
import { computed } from 'vue'
import AppStack from '@/components/ui/AppStack.vue'
import DesktopLayoutCard from './DesktopLayoutCard.vue'
import DesktopShellCard from './DesktopShellCard.vue'
import DesktopTilesCard from './DesktopTilesCard.vue'
import StartMenuCard from './StartMenuCard.vue'
import { useShellMode } from '@/composables/useShellMode.js'

const { shell } = useShellMode()

/* Сторона панели задач — про ОКОННЫЙ каркас: у телефона, планшета и
   «Виджетов» панель на своём месте. */
const windowsShell = computed(() => shell.value === 'windows')
</script>
