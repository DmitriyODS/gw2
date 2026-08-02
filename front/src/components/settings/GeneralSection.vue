<template>
  <div class="gs">
    <SettingCard title="Hola ассистент">
      <!-- Выдачу поисковика Hola открывает новой вкладкой браузера; здесь
           выбирается тот, что идёт в результатах первым. -->
      <SettingRow
        title="Поиск в интернете"
        hint="С какого поисковика начинать, когда ответа нет внутри приложения."
      >
        <PillTabs :model-value="engine" :tabs="engineTabs" compact @update:model-value="pickEngine" />
      </SettingRow>
    </SettingCard>

    <!-- Личный ключ YouGile: только для рядовых участников компании —
         администратор подключает компанию целиком в карточке компании. -->
    <SettingCard v-if="showYougile" title="Интеграция с YouGile">
      <YougileUserSettings />
    </SettingCard>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import PillTabs from '@/components/common/PillTabs.vue'
import SettingCard from '@/components/common/SettingCard.vue'
import SettingRow from '@/components/common/SettingRow.vue'
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
