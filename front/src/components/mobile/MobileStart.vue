<template>
  <!-- Стартовый экран мобильного каркаса — то же меню «Пуск», что и на рабочем
       столе (те же разделы, марка над плитками, порядок и размеры плиток из
       личных настроек), но во весь экран и в две колонки. Компания, аккаунт,
       юнит и уведомления — в общей панели статусов (MobileStatusBar). -->
  <div class="mstart" @contextmenu.self.prevent="openDeskMenu">
    <div class="mstart-body" @contextmenu.self.prevent="openDeskMenu">
      <!-- Марка — там же, где в меню «Пуск» рабочего стола: над плитками. -->
      <button class="mst-brand" type="button" title="О приложении" @click="open('/settings?section=about')">
        <BrandWordmark />
      </button>

      <section v-for="group in visibleGroups" :key="group.key" class="mst-group">
        <button class="mst-group-head" type="button" @click="prefs.toggleCollapsed(group.key)">
          <span class="mst-group-label">{{ group.label }}</span>
          <span
            class="material-symbols-outlined mst-group-chev"
            :class="{ collapsed: prefs.isCollapsed(group.key) }"
          >expand_more</span>
        </button>

        <div class="mst-group-body" :class="{ collapsed: prefs.isCollapsed(group.key) }">
          <div class="mst-group-inner">
            <div class="mst-tiles">
              <button
                v-for="(app, i) in group.items"
                :key="app.id"
                class="mst-tile"
                :class="[`is-${sizeOf(app)}`, { pinned: prefs.isPinned(app.id) }]"
                type="button"
                :title="app.title"
                @click="launch(app)"
                @pointerdown="longPress.start(app, $event)"
                @pointermove="longPress.move($event)"
                @pointerup="longPress.cancel()"
                @pointercancel="longPress.cancel()"
                @contextmenu.prevent="openTileMenu(app, $event)"
              >
                <LiveTile
                  :title="app.title"
                  :icon="app.icon"
                  :faces="facesOf(app)"
                  :wide="sizeOf(app) === 'wide'"
                  :order="i"
                />
                <span v-if="badgeOf(app)" class="mst-badge" :class="{ alert: badgeOf(app) === '!' }">
                  {{ badgeOf(app) }}
                </span>
                <span v-if="prefs.isPinned(app.id)" class="mst-pin material-symbols-outlined">keep</span>
              </button>
            </div>
          </div>
        </div>
      </section>
    </div>

    <ContextMenu
      :visible="menu.open"
      :x="menu.x"
      :y="menu.y"
      :items="menuItems"
      @select="onMenuSelect"
      @close="menu.open = false"
    />

    <AppDialog
      v-if="logoutAsk"
      v-model="logoutAsk"
      tone="danger"
      size="sm"
      title="Выйти из системы?"
      subtitle="Открытые разделы закроются, для возврата понадобится войти заново."
      :actions="LOGOUT_ACTIONS"
      @confirm="auth.logout()"
    />
  </div>
</template>

<script setup>
import { computed, defineAsyncComponent, reactive, ref } from 'vue'
import { useAuthStore } from '@/stores/auth.js'
import { useScreenLock } from '@/composables/useScreenLock.js'
import { useDesktopStore } from '@/stores/desktop.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { usePortalStore } from '@/stores/portal.js'
import { useTasksStore } from '@/stores/tasks.js'
import { usePetsStore } from '@/stores/pets.js'
import { useUnitsStore } from '@/stores/units.js'
import { useLiveTilesStore } from '@/stores/liveTiles.js'
import { usePermission } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { useLongPress } from '@/composables/useLongPress.js'
import { menuGroups } from '@/desktop/apps.js'
import { tileFaces } from '@/desktop/liveTiles.js'
import BrandWordmark from '@/components/common/BrandWordmark.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import LiveTile from '@/components/desktop/LiveTile.vue'

// Спрашиваем про выход редко — диалог (а с ним PrimeVue Dialog) грузим лениво.
const AppDialog = defineAsyncComponent(() => import('@/components/ui/AppDialog.vue'))

const auth = useAuthStore()
const screenLock = useScreenLock()
const desktop = useDesktopStore()
const prefs = useDesktopPrefsStore()
const messenger = useMessengerStore()
const portal = usePortalStore()
const tasks = useTasksStore()
const pets = usePetsStore()
const units = useUnitsStore()
const live = useLiveTilesStore()
const { isSuperAdmin, hasActiveCompany } = usePermission()
const { settings } = useCompanySettings()

const groups = computed(() => menuGroups({
  hasCompany: hasActiveCompany(),
  isSuperAdmin: isSuperAdmin(),
  settings: settings.value,
}, prefs.layout))

/* Пустой раздел показываем, только когда пользователь перекладывал плитки или
   завёл свои разделы: иначе на экране висели бы «мёртвые» заголовки. */
const visibleGroups = computed(() =>
  groups.value.filter((g) => g.items.length || g.custom || prefs.customized))

const appById = computed(() => {
  const map = new Map()
  for (const g of groups.value) for (const a of g.items) map.set(a.id, a)
  return map
})

const sizeOf = (app) => prefs.tileSize(app.id, app.tile || 'square')

const liveCtx = computed(() => ({ data: live.data, messenger, portal, pets, units, auth }))

function facesOf(app) {
  // Сводки выключены — общим тумблером или у этой плитки: обычный значок.
  return prefs.isTileLive(app.id) ? tileFaces(app.id, liveCtx.value) : []
}

function badgeOf(app) {
  if (app.id === 'messenger') return messenger.totalUnread || 0
  if (app.id === 'portal') return portal.unread || 0
  if (app.id === 'tasks') return tasks.myActiveCount || 0
  if (app.id === 'pets') return pets.pet?.sick ? '!' : 0
  return 0
}

function open(path) {
  desktop.open(path)
}

function launch(app) {
  // Клик, которым закончилось долгое нажатие, раздел не открывает.
  if (longPress.consumed()) return
  desktop.open(app.path)
}

/* ── Контекстное меню плитки и пустого места ─────────────────
   Пальцем правой кнопки нет: меню открывает долгое нажатие. */
const menu = reactive({ open: false, x: 0, y: 0, appId: null })
const longPress = useLongPress((app, e) => openTileMenu(app, e))

const menuItems = computed(() => {
  const id = menu.appId
  if (!id) {
    return [
      { label: 'Персонализация', icon: 'wallpaper', action: 'personalize' },
      { label: 'Справка и поддержка', icon: 'help', action: 'help' },
      ...(screenLock.enabled.value
        ? [{ divider: true }, { label: 'Заблокировать', icon: 'lock', action: 'lock' }]
        : []),
      { divider: true },
      { label: 'Выйти', icon: 'logout', action: 'logout', danger: true },
    ]
  }
  const size = sizeOf(appById.value.get(id) || { id })
  return [
    { label: 'Открыть', icon: 'open_in_new', action: 'open' },
    { divider: true },
    { label: 'Широкая плитка', icon: size === 'wide' ? 'check' : 'width_wide', action: 'wide' },
    { label: 'Квадратная плитка', icon: size === 'square' ? 'check' : 'crop_square', action: 'square' },
    { divider: true },
    prefs.liveTiles
      ? { label: 'Живая плитка', icon: prefs.isTileLive(id) ? 'check' : 'dashboard', action: 'live' }
      : { label: 'Живые плитки выключены', icon: 'toggle_off', disabled: true },
    { divider: true },
    prefs.isPinned(id)
      ? { label: 'Открепить от панели задач', icon: 'keep_off', action: 'unpin' }
      : { label: 'Закрепить на панели задач', icon: 'keep', action: 'pin' },
  ]
})

function openTileMenu(app, e) {
  menu.appId = app.id
  menu.x = e.clientX
  menu.y = e.clientY
  menu.open = true
}

function openDeskMenu(e) {
  menu.appId = null
  menu.x = e.clientX
  menu.y = e.clientY
  menu.open = true
}

const logoutAsk = ref(false)
const LOGOUT_ACTIONS = [
  { kind: 'cancel', label: 'Остаться' },
  { kind: 'confirm', label: 'Выйти', icon: 'logout' },
]

function onMenuSelect(action) {
  if (action === 'personalize') return open('/settings?section=desktop')
  if (action === 'help') return open('/settings?section=help')
  if (action === 'logout') { logoutAsk.value = true; return }
  if (action === 'lock') { screenLock.lock(); return }
  const id = menu.appId
  if (!id) return
  if (action === 'open') return open(appById.value.get(id).path)
  if (action === 'wide' || action === 'square') return prefs.setTileSize(id, action)
  if (action === 'live') {
    prefs.toggleTileLive(id)
    // Вернули сводки — их надо подтянуть: выключенную плитку не опрашивали.
    if (prefs.isTileLive(id)) live.refresh([id]).catch(() => {})
    return
  }
  if (action === 'pin') return prefs.pin(id)
  if (action === 'unpin') return prefs.unpin(id)
}
</script>

<style scoped>
.mstart {
  position: absolute;
  inset: 0;
  /* Размер плитки фиксирован — от него считается и ширина широкой (две
     колонки с зазором), и высота ряда. */
  --mst-tile: 108px;
  --mst-tile-h: 96px;
  overflow: hidden;
}

/* ── Плитки ─────────────────────────────────────────────────── */
.mstart-body {
  height: 100%;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 0 12px;
  /* Резервы под панели, прижатые к кромкам: лента уходит под них и
     просвечивает сквозь стекло, но в исходном положении крайние плитки видны
     целиком. */
  padding-top: calc(var(--statusbar-height, 44px) + env(safe-area-inset-top, 0px) + 14px);
  padding-bottom: calc(var(--taskbar-height, 52px) + env(safe-area-inset-bottom, 0px) + 16px);
  scrollbar-width: none;
}

.mstart-body::-webkit-scrollbar { display: none; }

/* Марка прокручивается вместе с плитками: панель статусов остаётся компактной,
   а надпись стоит там же, где в меню «Пуск» на рабочем столе. */
.mst-brand {
  align-self: flex-start;
  display: flex;
  padding: 2px 4px;
  margin-bottom: -6px;
  border: none;
  border-radius: var(--radius-md);
  background: none;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
}

.mst-group-head {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 2px 4px 8px;
  border: none;
  background: none;
  cursor: pointer;
}

.mst-group-label {
  font-size: 0.95rem;
  font-weight: 500;
  color: var(--color-text-dim);
}

.mst-group-chev {
  font-size: 18px;
  color: var(--color-text-dim);
  transition: rotate 0.18s ease;
}

.mst-group-chev.collapsed { rotate: -90deg; }

/* Свёрнутый раздел схлопывается по высоте — как в меню «Пуск» на столе. */
.mst-group-body {
  display: grid;
  grid-template-rows: 1fr;
  transition: grid-template-rows 0.22s cubic-bezier(0.2, 0, 0, 1);
}

.mst-group-body.collapsed { grid-template-rows: 0fr; }
.mst-group-inner { overflow: hidden; }

/* Колонок — сколько плиток номинального размера влезает по ширине, и ряд
   делится между ними поровну: сколько поместилось, столько и стоит вплотную
   без «хвоста» пустого места. Влезает одна — она занимает всю ширину. */
.mst-tiles {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(var(--mst-tile), 1fr));
  gap: 10px;
}

.mst-tile {
  position: relative;
  height: var(--mst-tile-h);
  min-height: var(--mst-tile-h);
  padding: 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  cursor: pointer;
  overflow: hidden;
  transition: scale 0.14s ease, background 0.15s ease;
  -webkit-tap-highlight-color: transparent;
}

.mst-tile.is-wide { grid-column: span 2; }

/* В ряд помещается одна плитка — широкой не на что растягиваться. */
@media (max-width: 250px) {
  .mst-tile.is-wide { grid-column: span 1; }
}
.mst-tile:active { scale: 0.97; }

.mst-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 10px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 11px;
  font-weight: 700;
  line-height: 20px;
  text-align: center;
}

.mst-badge.alert { background: var(--color-error); }

.mst-pin {
  position: absolute;
  right: 8px;
  bottom: 8px;
  font-size: 15px;
  color: var(--color-text-dim);
  opacity: 0.7;
}

@media (prefers-reduced-motion: reduce) {
  .mst-group-body,
  .mst-tile { transition: none; }
}
</style>
