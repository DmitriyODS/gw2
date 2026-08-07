<template>
  <!-- Панель статусов мобильного каркаса: активная компания слева, справа —
       идущий юнит, уведомления и аккаунт. Видна всегда, в том числе поверх
       открытого раздела. -->
  <header class="msbar">
    <div class="msb-company">
      <CompanySelect />
    </div>

    <button v-if="unit" class="msb-unit" type="button" title="Идёт работа — открыть юнит" @click="expand">
      <span class="material-symbols-outlined">timer</span>
      <span class="msb-clock">{{ clock }}</span>
    </button>

    <button
      class="msb-bell"
      type="button"
      :class="{ active: desktop.notifOpen, muted: notifyMuted }"
      :title="notifyMuted ? `Уведомления отключены ${muteUntilLabel}` : 'Уведомления'"
      aria-label="Уведомления"
      @click="toggleNotifications"
      @pointerdown="bellPress.start(null, $event)"
      @pointermove="bellPress.move($event)"
      @pointerup="bellPress.cancel()"
      @pointercancel="bellPress.cancel()"
      @contextmenu.prevent="openBellMenu($event)"
    >
      <span class="material-symbols-outlined">{{ notifyMuted ? 'notifications_off' : 'notifications' }}</span>
      <span v-if="alerts" class="msb-dot">{{ alerts > 99 ? '99+' : alerts }}</span>
    </button>

    <button class="msb-user" type="button" :title="auth.user?.fio || 'Аккаунт'" @click="desktop.open('/settings?section=account')">
      <img class="msb-avatar" :src="avatarSrc" :alt="auth.user?.fio || 'Аккаунт'" />
    </button>

    <ContextMenu
      :visible="menu.open"
      :x="menu.x"
      :y="menu.y"
      :items="menuItems"
      @select="onMenuSelect"
      @close="menu.open = false"
    />
  </header>
</template>

<script setup>
import { avatarUrl } from '@/utils/pets.js'
import { computed, reactive } from 'vue'
import { useAuthStore } from '@/stores/auth.js'
import { useDesktopStore } from '@/stores/desktop.js'
import { useUnitsStore } from '@/stores/units.js'
import { useActiveUnit } from '@/composables/useActiveUnit.js'
import { useDesktopNotifications } from '@/composables/useDesktopNotifications.js'
import { useNotifyMute } from '@/composables/useNotifyMute.js'
import { useElapsed } from '@/composables/useElapsed.js'
import { useLongPress } from '@/composables/useLongPress.js'
import CompanySelect from '@/components/common/CompanySelect.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'

const auth = useAuthStore()
const desktop = useDesktopStore()
const units = useUnitsStore()
const { expand } = useActiveUnit()
const { count: alerts } = useDesktopNotifications()
const { muted: notifyMuted, untilLabel: muteUntilLabel, mute, unmute } = useNotifyMute()

const unit = computed(() => units.activeUnit)
const { clock } = useElapsed(() => unit.value?.datetime_start)

const avatarSrc = computed(() => {
  const user = auth.user
  if (!user) return ''
  return avatarUrl(user)
})

/* ── «Не беспокоить» ──────────────────────────────────────────
   Долгое нажатие на колокольчик глушит звук и всплывашки на срок или насовсем;
   карточки в центре уведомлений при этом продолжают копиться. */
const menu = reactive({ open: false, x: 0, y: 0 })
const bellPress = useLongPress((_, e) => openBellMenu(e))

const MUTE_OPTIONS = [
  { label: '10 минут', minutes: 10 },
  { label: '30 минут', minutes: 30 },
  { label: '1 час', minutes: 60 },
  { label: '4 часа', minutes: 240 },
  { label: '8 часов', minutes: 480 },
  { label: 'Навсегда', minutes: null },
]

const menuItems = computed(() => {
  if (notifyMuted.value) {
    return [{ label: `Включить уведомления (${muteUntilLabel.value})`, icon: 'notifications_active', action: 'unmute' }]
  }
  return [{
    label: 'Отключить уведомления',
    icon: 'notifications_off',
    children: MUTE_OPTIONS.map((o) => ({
      label: o.label,
      icon: o.minutes ? 'schedule' : 'do_not_disturb_on',
      action: `mute:${o.minutes ?? 'forever'}`,
      danger: !o.minutes,
    })),
  }]
})

function openBellMenu(e) {
  menu.x = e.clientX
  menu.y = e.clientY
  menu.open = true
}

function onMenuSelect(action) {
  if (action === 'unmute') return unmute()
  if (action?.startsWith('mute:')) {
    const arg = action.slice(5)
    mute(arg === 'forever' ? null : Number(arg))
  }
}

function toggleNotifications() {
  if (bellPress.consumed()) return
  desktop.notifOpen = !desktop.notifOpen
}
</script>

<style scoped>
/* Панель прижата к верхней кромке: без полей и скруглений, высота — только
   содержимое плюс системный вырез. */
.msbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 900;
  display: flex;
  align-items: center;
  gap: 6px;
  height: calc(var(--statusbar-height) + env(safe-area-inset-top, 0px));
  padding: 0 10px;
  padding-top: env(safe-area-inset-top, 0px);
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border-bottom: 1px solid var(--acrylic-border);
}

.msbar button {
  -webkit-tap-highlight-color: transparent;
  border: none;
  background: none;
  cursor: pointer;
}

/* Общий выбор компании, ужатый до строки статусов: значок и название, стрелка
   лишняя — список всё равно раскрывается по нажатию. */
.msb-company {
  flex: 1;
  min-width: 0;
  display: flex;
}

.msb-company :deep(.company-button) {
  height: 32px;
  min-height: 32px;
  padding: 0 8px;
  gap: 8px;
  border: none;
  border-radius: var(--radius-md);
  background: none;
  box-shadow: none;
  font-size: 13px;
}

.msb-company :deep(.company-button-ico) { font-size: 19px; }
.msb-company :deep(.company-button-chev) { display: none; }

.msb-unit {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  min-height: 28px;
  padding: 0 8px;
  border-radius: var(--radius-md);
  background: color-mix(in oklch, var(--color-success) 18%, transparent);
  color: var(--color-text);
  font-size: 12px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.msb-unit .material-symbols-outlined { font-size: 16px; color: var(--color-success); }

.msb-bell {
  position: relative;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  width: 34px;
  min-width: 34px;
  height: 34px;
  min-height: 34px;
  padding: 0;
  border-radius: 50%;
  color: var(--color-text);
  transition: background 0.15s ease, color 0.15s ease, scale 0.12s ease;
}

.msb-bell .material-symbols-outlined { font-size: 20px; }
.msb-bell:active { scale: 0.9; }
.msb-bell.muted { color: var(--color-text-dim); }

.msb-bell.active {
  background: color-mix(in oklch, var(--color-primary) 18%, transparent);
  color: var(--color-primary);
}

.msb-dot {
  position: absolute;
  top: 0;
  right: -2px;
  min-width: 15px;
  height: 15px;
  padding: 0 4px;
  border-radius: 8px;
  background: var(--color-error);
  color: var(--color-on-primary);
  font-size: 10px;
  font-weight: 700;
  line-height: 15px;
  text-align: center;
}

.msb-user {
  flex-shrink: 0;
  width: 32px;
  min-width: 32px;
  height: 32px;
  min-height: 32px;
  padding: 0;
  border-radius: 50%;
  overflow: hidden;
}

.msb-avatar {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

@media (prefers-reduced-motion: reduce) {
  .msbar button { transition: none; }
}
</style>
