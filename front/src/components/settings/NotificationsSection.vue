<template>
  <div class="ns">
    <AppCard title="Звук и тишина">
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

    <AppCard title="Как показывать">
      <!-- Угол — только про рабочий стол: на телефоне карточка всегда выходит
           сверху, снизу её закрыла бы панель задач. -->
      <AppRow
        v-if="!isMobile"
        title="Откуда выезжают"
        hint="Угол экрана, в котором собирается стопка уведомлений."
      >
        <Select
          :model-value="notifyPrefs.corner"
          :options="TOAST_CORNERS"
          option-label="label"
          option-value="key"
          @update:model-value="notifyPrefs.corner = $event"
        />
      </AppRow>

      <AppRow
        title="Сколько висит"
        hint="Пока указатель на карточке, отсчёт замирает — прочитать успеете."
      >
        <AppTabs
          variant="tint"
          :model-value="notifyPrefs.life"
          :tabs="lifeTabs"
          dense
          @update:model-value="notifyPrefs.life = $event"
        />
      </AppRow>

      <AppSwitchRow
        :model-value="notifyPrefs.onLockScreen"
        title="Показывать на заблокированном экране"
        hint="Выключено — на запертом экране уведомления не видны; они дождутся разблокировки."
        @update:model-value="notifyPrefs.onLockScreen = $event"
      />
    </AppCard>

    <AppCard
      title="От каких разделов"
      hint="Выключенный раздел перестаёт присылать уведомления — его события останутся в самом разделе. Звонки приходят всегда."
    >
      <AppSwitchRow
        v-for="s in visibleSources"
        :key="s.key"
        :model-value="notifyPrefs.sources[s.key] !== false"
        :title="s.label"
        :hint="s.hint"
        @update:model-value="setSourceEnabled(s.key, $event)"
      />
    </AppCard>
  </div>
</template>

<script setup>
/* Всё про уведомления в одном месте: звук и тишина (раньше жили в «Общих»),
   поведение всплывающих карточек и разделы-источники. */
import { computed, ref } from 'vue'
import Select from 'primevue/select'
import AppCard from '@/components/ui/AppCard.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import { isNotifySoundOn, setNotifySound } from '@/utils/systemNotify.js'
import {
  NOTIFY_SOURCES, TOAST_CORNERS, TOAST_LIVES, notifyPrefs, setSourceEnabled,
} from '@/utils/notifySettings.js'
import { SUBSCRIPTIONS_VISIBLE } from '@/utils/release.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { useNotifyMute } from '@/composables/useNotifyMute.js'

const { isMobile } = useBreakpoint()
const { usesGroove } = useCompanySettings()
const { muted, untilLabel, mute, unmute } = useNotifyMute()
const soundOn = ref(isNotifySoundOn())

const lifeTabs = TOAST_LIVES.map((l) => ({ key: l.key, label: l.label }))

// Разделы, которых у человека нет, в списке не нужны: тумблер «Питомцы» без
// грувиков и «Счета» без магазина обещали бы несуществующие уведомления.
const visibleSources = computed(() => NOTIFY_SOURCES.filter((s) => {
  if (s.key === 'pets') return usesGroove.value
  if (s.key === 'billing') return SUBSCRIPTIONS_VISIBLE
  return true
}))

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
</script>

<style scoped>
.ns {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
