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

    <!-- Личный ключ YouGile: только для рядовых участников компании —
         администратор подключает компанию целиком в карточке компании. -->
    <AppCard v-if="showYougile" title="Интеграция с YouGile">
      <YougileUserSettings />
    </AppCard>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppRow from '@/components/ui/AppRow.vue'
import YougileUserSettings from '@/components/settings/YougileUserSettings.vue'
import { SEARCH_ENGINES, getSearchEngine, setSearchEngine } from '@/utils/webSearch.js'

defineProps({
  showYougile: { type: Boolean, default: false },
})

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
