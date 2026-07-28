<template>
  <!-- Мобильный каркас: у Hola нет ни панели задач, ни окон — он открывается
       обычным экраном раздела. На рабочем столе ту же панель показывает
       всплывающий HolaPopup. -->
  <div class="gw-shell hola-screen">
    <HolaPanel :start-tab="startTab" />
  </div>
</template>

<script setup>
import { useRoute } from 'vue-router'
import HolaPanel from '@/components/hola/HolaPanel.vue'

const route = useRoute()
const startTab = String(route.query.tab || 'search')
</script>

<style scoped>
.hola-screen {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
}
</style>
