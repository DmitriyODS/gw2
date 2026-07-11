<template>
  <!-- Настройки десктоп-обёртки (Electron): виден только внутри неё.
       Тумблеры применяются мгновенно (IPC-мост GrooveDesktop). -->
  <div v-if="desktop" class="dac">
    <header class="dac-head">
      <span class="dac-icon material-symbols-outlined">desktop_windows</span>
      <div class="dac-head-text">
        <h3>Приложение для компьютера</h3>
        <p>Поведение окна, трея и уведомлений этой установки Groove Work.</p>
      </div>
    </header>

    <label class="dac-row">
      <div class="dac-row-text">
        <span class="dac-row-title">Автозапуск при входе в систему</span>
        <span class="dac-row-desc">Приложение стартует свёрнутым в трей — уведомления приходят сразу.</span>
      </div>
      <ToggleSwitch :model-value="s.autostart" @update:model-value="set('autostart', $event)" />
    </label>

<!-- Свернуть в трей при скрытом значке — ловушка (окно не вернуть),
         поэтому без значка тумблер сворачивания недоступен. -->
    <label v-if="s.trayIcon" class="dac-row">
      <div class="dac-row-text">
        <span class="dac-row-title">Сворачивать в трей при закрытии</span>
        <span class="dac-row-desc">Крестик прячет окно, приложение живёт в трее; выключено — закрывает совсем.</span>
      </div>
      <ToggleSwitch :model-value="s.closeToTray" @update:model-value="set('closeToTray', $event)" />
    </label>

    <label class="dac-row">
      <div class="dac-row-text">
        <span class="dac-row-title">Значок в трее</span>
        <span class="dac-row-desc">Быстрый доступ к окну и выходу из меню значка.</span>
      </div>
      <ToggleSwitch :model-value="s.trayIcon" @update:model-value="set('trayIcon', $event)" />
    </label>

    <label class="dac-row">
      <div class="dac-row-text">
        <span class="dac-row-title">Не беспокоить</span>
        <span class="dac-row-desc">Без звука и уведомлений о сообщениях; входящие звонки показываются всегда.</span>
      </div>
      <ToggleSwitch :model-value="muted" @update:model-value="setMuted" />
    </label>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import ToggleSwitch from 'primevue/toggleswitch'
import { isNotifyMuted, setNotifyMuted } from '@/utils/systemNotify.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const desktop = window.GrooveDesktop
const notify = useNotificationsStore()

const s = reactive({ autostart: false, closeToTray: true, trayIcon: true })
const muted = ref(isNotifyMuted())

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
  muted.value = v
  setNotifyMuted(v)
}
</script>

<style scoped>
.dac {
  display: flex;
  flex-direction: column;
  gap: 4px;
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  padding: 18px 20px;
  margin-top: 16px;
}
.dac-head { display: flex; gap: 14px; align-items: center; margin-bottom: 8px; }
.dac-icon {
  width: 44px; height: 44px;
  border-radius: var(--radius-md);
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  display: grid; place-items: center;
  font-size: 24px;
  flex-shrink: 0;
}
.dac-head-text h3 { margin: 0 0 2px; font-size: 15px; font-weight: 700; }
.dac-head-text p { margin: 0; font-size: 13px; color: var(--color-text-dim); }

.dac-row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 2px;
  border-top: 1px solid var(--color-outline-dim);
  cursor: pointer;
}
.dac-row-text { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.dac-row-title { font-size: 14px; font-weight: 600; }
.dac-row-desc { font-size: 12.5px; color: var(--color-text-dim); line-height: 1.35; }
</style>
