<template>
  <!-- Настройки десктоп-обёртки (Electron): видна только внутри неё.
       Тумблеры применяются мгновенно (IPC-мост GrooveDesktop). -->
  <AppCard
    v-if="desktop"
    class="dac"
    title="Приложение для компьютера"
    hint="Поведение окна, трея и уведомлений этой установки Groove Work."
  >
    <AppSwitchRow
      :model-value="s.autostart"
      title="Автозапуск при входе в систему"
      hint="Приложение стартует свёрнутым в трей — уведомления приходят сразу."
      @update:model-value="set('autostart', $event)"
    />

    <!-- Свернуть в трей при скрытом значке — ловушка (окно не вернуть),
         поэтому без значка тумблер сворачивания недоступен. -->
    <AppSwitchRow
      v-if="s.trayIcon"
      :model-value="s.closeToTray"
      title="Сворачивать в трей при закрытии"
      hint="Крестик прячет окно, приложение живёт в трее; выключено — закрывает совсем."
      @update:model-value="set('closeToTray', $event)"
    />

    <AppSwitchRow
      :model-value="s.trayIcon"
      title="Значок в трее"
      hint="Быстрый доступ к окну и выходу из меню значка."
      @update:model-value="set('trayIcon', $event)"
    />

  </AppCard>
</template>

<script setup>
import { onMounted, reactive } from 'vue'
import { useNotificationsStore } from '@/stores/notifications.js'
import AppCard from '@/components/ui/AppCard.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'

const desktop = window.GrooveDesktop
const notify = useNotificationsStore()

const s = reactive({ autostart: false, closeToTray: true, trayIcon: true })
// Тишина и звук уведомлений живут в «Настройки → Общие»: они одинаковы на
// всех платформах, а здесь — только про саму десктоп-обёртку.

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
</script>

<style scoped>
.dac { margin-top: 16px; }
</style>
