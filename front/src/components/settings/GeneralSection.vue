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

    <!-- Уведомления: работает на всех платформах, а не только в
         десктоп-обёртке. Звук и «не беспокоить» — РАЗНЫЕ вещи: первое глушит
         только сигнал, второе ещё и всплывашки системы. -->
    <AppCard title="Уведомления">
      <AppSwitchRow
        :model-value="soundOn"
        title="Звук уведомлений"
        hint="Короткий сигнал на новое сообщение, напоминание и другие события."
        @update:model-value="toggleSound"
      />
      <AppSwitchRow
        :model-value="muted"
        title="Не беспокоить"
        :hint="muteHint"
        @update:model-value="toggleMute"
      />
    </AppCard>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'
import { SEARCH_ENGINES, getSearchEngine, setSearchEngine } from '@/utils/webSearch.js'
import { isNotifySoundOn, setNotifySound } from '@/utils/systemNotify.js'
import { useNotifyMute } from '@/composables/useNotifyMute.js'

const engineTabs = SEARCH_ENGINES.map((e) => ({ key: e.key, label: e.label }))
const engine = ref(getSearchEngine())

const { muted, untilLabel, mute, unmute } = useNotifyMute()
const soundOn = ref(isNotifySoundOn())

const muteHint = computed(() => (muted.value
  ? `Сигналы и всплывающие уведомления выключены ${untilLabel.value}. Звонки продолжают звонить.`
  : 'Выключает и сигналы, и всплывающие уведомления системы. Звонки не глушатся.'))

function toggleSound(on) {
  soundOn.value = on
  setNotifySound(on)
}

function toggleMute(on) {
  if (on) mute()
  else unmute()
}

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
