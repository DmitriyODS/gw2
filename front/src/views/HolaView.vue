<template>
  <!-- Экран-адрес /hola: в обоих каркасах ссылку перехватывает сам каркас и
       показывает панель Hola поверх всего (HolaPopup), поэтому сюда попадают
       только контексты без каркаса. -->
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
