<template>
  <div class="gs">
    <AppCard title="Hola ассистент">
      <!-- Выдачу поисковика Hola открывает новой вкладкой браузера; здесь
           выбирается тот, что идёт в результатах первым. -->
      <AppRow
        title="Поиск в интернете"
        hint="С какого поисковика начинать, когда ответа нет внутри приложения."
      >
        <AppTabs variant="tint" :model-value="engine" :tabs="engineTabs" dense @update:model-value="pickEngine" />
      </AppRow>
    </AppCard>

    <!-- Настройки самой десктоп-обёртки: автозапуск, трей, поведение крестика.
         Карточка сама себя прячет вне Electron. -->
    <DesktopAppCard />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppRow from '@/components/ui/AppRow.vue'
import { SEARCH_ENGINES, getSearchEngine, setSearchEngine } from '@/utils/webSearch.js'
import DesktopAppCard from './DesktopAppCard.vue'

const engineTabs = SEARCH_ENGINES.map((e) => ({ key: e.key, label: e.label }))
const engine = ref(getSearchEngine())

function pickEngine(key) {
  engine.value = key
  setSearchEngine(key)
}
</script>

<style scoped>
.gs {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
