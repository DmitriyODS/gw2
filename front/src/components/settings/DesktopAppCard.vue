<template>
  <!-- Настройки десктоп-обёртки (Electron): видна только внутри неё.
       Тумблеры применяются мгновенно (IPC-мост GrooveDesktop). -->
  <SettingCard
    v-if="desktop"
    class="dac"
    title="Приложение для компьютера"
    hint="Поведение окна, трея и уведомлений этой установки Groove Work."
  >
    <SwitchRow
      :model-value="s.autostart"
      title="Автозапуск при входе в систему"
      hint="Приложение стартует свёрнутым в трей — уведомления приходят сразу."
      @update:model-value="set('autostart', $event)"
    />

    <!-- Свернуть в трей при скрытом значке — ловушка (окно не вернуть),
         поэтому без значка тумблер сворачивания недоступен. -->
    <SwitchRow
      v-if="s.trayIcon"
      :model-value="s.closeToTray"
      title="Сворачивать в трей при закрытии"
      hint="Крестик прячет окно, приложение живёт в трее; выключено — закрывает совсем."
      @update:model-value="set('closeToTray', $event)"
    />

    <SwitchRow
      :model-value="s.trayIcon"
      title="Значок в трее"
      hint="Быстрый доступ к окну и выходу из меню значка."
      @update:model-value="set('trayIcon', $event)"
    />

    <SwitchRow
      :model-value="muted"
      title="Не беспокоить"
      :hint="muteHint"
      @update:model-value="setMuted"
    />
  </SettingCard>
</template>

<script setup>
import { computed, onMounted, reactive } from 'vue'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useNotifyMute } from '@/composables/useNotifyMute.js'
import SettingCard from '@/components/common/SettingCard.vue'
import SwitchRow from '@/components/common/SwitchRow.vue'

const desktop = window.GrooveDesktop
const notify = useNotificationsStore()

const s = reactive({ autostart: false, closeToTray: true, trayIcon: true })
// Состояние общее с ПКМ-меню колокольчика на панели задач — тумблер не должен
// расходиться с ним (там тишину можно включить и на срок).
const { muted, untilLabel: muteUntilLabel, mute, unmute } = useNotifyMute()

const muteHint = computed(() => {
  const base = 'Без звука и всплывающих уведомлений ОС — сами уведомления копятся в центре; входящие звонки показываются всегда.'
  return muted.value && muteUntilLabel.value !== 'навсегда'
    ? `${base} Сейчас тишина ${muteUntilLabel.value}.`
    : base
})

onMounted(async () => {
  if (!desktop?.getSettings) return
  try {
    Object.assign(s, await desktop.getSettings())
  } catch { /* старая обёртка без настроек — оставим дефолты */ }
})

async function set(key, value) {
  s[key] = value
  try {
    const res = await desktop.setSetting(key, value)
    if (res && !res.error) Object.assign(s, res)
  } catch {
    notify.warn('Обёртка не поддерживает эту настройку — обновите приложение')
    return
  }
  // Выключенный значок трея гасит и «сворачивать в трей»: иначе окно,
  // спрятанное крестиком, нечем вызвать обратно. Новая обёртка делает это
  // сама (и уже вернула settings), для старой — добиваем отдельным вызовом.
  if (key === 'trayIcon' && !value && s.closeToTray) await set('closeToTray', false)
}

function setMuted(v) {
  if (v) mute(null)
  else unmute()
}
</script>

<style scoped>
.dac { margin-top: 16px; }
</style>
