<template>
  <aside
    class="wp"
    :class="{ rail: collapsed }"
    :style="{ width: `${width}px`, minWidth: `${width}px` }"
    aria-label="Разделы"
  >
    <!-- ── Шапка: марка, ассистент, уведомления и сворачивание ── -->
    <header class="wp-head">
      <button class="wp-brand" type="button" title="О приложении" @click="launchPath('/settings?section=about')">
        <Logo :size="30" />
      </button>

      <div class="wp-head-tools">
        <button
          class="wp-icon-btn"
          type="button"
          :class="{ active: desktop.holaOpen }"
          title="Hola ассистент — поиск, команды и чат (Ctrl+K)"
          @click="desktop.holaOpen = !desktop.holaOpen"
        >
          <HolaIcon :size="20" />
        </button>

        <button
          class="wp-icon-btn"
          type="button"
          :class="{ active: desktop.notifOpen, muted: notifyMuted }"
          :title="notifyMuted ? `Уведомления отключены ${muteUntilLabel}` : 'Уведомления'"
          @click="desktop.notifOpen = !desktop.notifOpen"
          @contextmenu.prevent="openBellMenu($event)"
        >
          <span class="material-symbols-outlined">{{ notifyMuted ? 'notifications_off' : 'notifications' }}</span>
          <span v-if="alerts" class="wp-dot">{{ alerts > 99 ? '99+' : alerts }}</span>
        </button>

        <button
          class="wp-icon-btn"
          type="button"
          :title="collapsed ? 'Развернуть панель' : 'Свернуть панель'"
          @click="emit('toggle')"
        >
          <span class="material-symbols-outlined">
            {{ collapsed ? 'left_panel_open' : 'left_panel_close' }}
          </span>
        </button>
      </div>
    </header>

    <!-- Идёт работа: чип активного юнита живёт здесь — панели задач с ним больше нет. -->
    <button v-if="unit" class="wp-unit" type="button" title="Идёт работа — открыть юнит" @click="expand">
      <span class="material-symbols-outlined">timer</span>
      <span v-if="!collapsed" class="wp-unit-clock">{{ clock }}</span>
    </button>

    <!-- Два вида: избранное (закреплённые) и все разделы по категориям.
         Переключатель появляется, только когда есть что закреплять — иначе он
         вёл бы на заведомо пустой экран. -->
    <AppTabs
      v-if="!collapsed && hasPinned"
      class="wp-view"
      :model-value="view"
      :tabs="VIEW_TABS"
      variant="tint"
      dense
      full-width
      @update:model-value="prefs.setWidgetsView"
    />

    <!-- ── Виджеты разделов ── -->
    <div class="wp-body">
      <!-- ИЗБРАННОЕ: плоский список — категории тут ни к чему, порядок задаёт
           сам человек перетаскиванием. -->
      <div v-if="!collapsed && view === 'pinned'" class="wp-tiles">
        <WidgetTile
          v-for="(app, i) in pinnedApps"
          :key="app.id"
          :app="app"
          :faces="facesOf(app)"
          :badge="badgeOf(app)"
          :active="isActive(app)"
          :opened="isOpen(app)"
          :dragging="drag.appId === app.id"
          :order="i"
          :paused="hovered === app.id || !!drag.appId"
          draggable="true"
          @dragstart="onPinDragStart(app, $event)"
          @dragover.prevent.stop="onPinDragOver(app)"
          @drop.prevent="onDragEnd"
          @dragend="onDragEnd"
          @click="launch(app)"
          @contextmenu.prevent.stop="openTileMenu(app, $event)"
          @pointerenter="hovered = app.id"
          @pointerleave="hovered = hovered === app.id ? null : hovered"
        />
      </div>

      <template v-else-if="!collapsed">
        <section v-for="group in visibleGroups" :key="group.key" class="wp-group">
          <div class="wp-group-head" @contextmenu.prevent="openGroupMenu(group, $event)">
            <button
              class="wp-group-toggle"
              type="button"
              :title="prefs.isCollapsed(PLATFORM, group.key) ? 'Развернуть раздел' : 'Свернуть раздел'"
              @click="prefs.toggleCollapsed(PLATFORM, group.key)"
            >
              <span
                class="material-symbols-outlined wp-group-chev"
                :class="{ collapsed: prefs.isCollapsed(PLATFORM, group.key) }"
              >expand_more</span>
              <InputText
                v-if="editing === group.key"
                ref="editorRef"
                v-model="editLabel"
                class="wp-group-input"
                @click.stop
                @keyup.enter="commitRename(group)"
                @keyup.esc="editing = null"
                @blur="commitRename(group)"
              />
              <span v-else class="wp-group-label">{{ group.label }}</span>
              <span class="wp-group-count">{{ group.items.length }}</span>
            </button>
            <button
              class="wp-group-more"
              type="button"
              title="Настроить раздел"
              @click.stop="openGroupMenu(group, $event)"
            >
              <span class="material-symbols-outlined">more_horiz</span>
            </button>
          </div>

          <!-- Свёрнутый раздел схлопывается по высоте (grid-template-rows). -->
          <div class="wp-group-body" :class="{ collapsed: prefs.isCollapsed(PLATFORM, group.key) }">
            <div class="wp-group-inner">
              <div class="wp-tiles" @dragover.prevent="onDragOverGroup(group)" @drop.prevent="onDragEnd">
                <WidgetTile
                  v-for="(app, i) in group.items"
                  :key="app.id"
                  :app="app"
                  :faces="facesOf(app)"
                  :badge="badgeOf(app)"
                  :active="isActive(app)"
                  :opened="isOpen(app)"
                  :dragging="drag.appId === app.id"
                  :order="i"
                  :paused="hovered === app.id || !!drag.appId"
                  draggable="true"
                  @dragstart="onDragStart(group, app, $event)"
                  @dragover.prevent.stop="onDragOver(group, app)"
                  @drop.prevent="onDragEnd"
                  @dragend="onDragEnd"
                  @click="launch(app)"
                  @contextmenu.prevent.stop="openTileMenu(app, $event)"
                  @pointerenter="hovered = app.id"
                  @pointerleave="hovered = hovered === app.id ? null : hovered"
                />

                <p v-if="!group.items.length" class="wp-group-empty">Перетащите сюда виджет</p>
              </div>
            </div>
          </div>
        </section>

        <button class="wp-add-group" type="button" @click="createGroup">
          <span class="material-symbols-outlined">add</span>
          Новый раздел
        </button>
      </template>

      <!-- Свёрнутая панель: одни значки подряд — без групп и сводок. -->
      <template v-else>
        <button
          v-for="app in railApps"
          :key="app.id"
          class="wp-rail-btn"
          :class="{ active: isActive(app), opened: isOpen(app) }"
          type="button"
          :title="app.title"
          @click="launch(app)"
          @contextmenu.prevent="openTileMenu(app, $event)"
        >
          <span class="material-symbols-outlined">{{ app.icon }}</span>
          <span v-if="badgeOf(app)" class="wp-dot">{{ badgeOf(app) === '!' ? '!' : badgeOf(app) }}</span>
        </button>
      </template>
    </div>

    <!-- ── Подвал: кто я, компания, настройки и выход ── -->
    <footer class="wp-foot">
      <div class="wp-foot-row">
        <button
          class="wp-user"
          type="button"
          :title="shortFio(auth.user?.fio) || 'Аккаунт'"
          @click="launchPath('/settings?section=account')"
        >
          <img class="wp-avatar" :src="avatarSrc" :alt="auth.user?.fio || 'Аккаунт'" />
        </button>
        <CompanySelect compact />
        <button class="wp-icon-btn" type="button" title="Настройки" @click="launchPath('/settings')">
          <span class="material-symbols-outlined">settings</span>
        </button>
        <!-- Запереть экран: сессия жива, приложение закрывается пин-кодом. -->
        <button
          v-if="screenLock.enabled.value"
          class="wp-icon-btn"
          type="button"
          title="Заблокировать (Ctrl+L)"
          @click="screenLock.lock()"
        >
          <span class="material-symbols-outlined">lock</span>
        </button>
        <button class="wp-icon-btn danger" type="button" title="Выйти" @click="logoutAsk = true">
          <span class="material-symbols-outlined">logout</span>
        </button>
      </div>
      <div v-if="!collapsed" class="wp-clock" :title="fullDate">{{ time }} · {{ date }}</div>
    </footer>

    <ContextMenu
      :visible="menu.open"
      :x="menu.x"
      :y="menu.y"
      :items="menuItems"
      @select="onMenuSelect"
      @close="menu.open = false"
    />

    <!-- Выход — из тех действий, что делают одним промахом мыши: спрашиваем. -->
    <AppDialog
      v-model="logoutAsk"
      tone="danger"
      size="sm"
      title="Выйти из системы?"
      subtitle="Открытые разделы закроются, для возврата понадобится войти заново."
      :actions="LOGOUT_ACTIONS"
      @confirm="auth.logout()"
    />
  </aside>
</template>

<script setup>
/**
 * Боковая панель каркаса «Виджеты»: марка и общие кнопки сверху, живые плитки
 * разделов по категориям в середине, аккаунт и выход снизу. Заменяет собой и
 * панель задач, и меню «Пуск» — поэтому всё, что жило в них (ассистент,
 * уведомления, чип идущего юнита, компания, часы), собрано здесь.
 *
 * Плитки всегда широкие: панель — одна колонка, выбирать размер не из чего.
 * Раскладка (свои разделы, порядок, переносы, свёрнутость) — своя, отдельная
 * от «Пуска» рабочего стола: панель узкая, и порядок в ней другой.
 */
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import InputText from 'primevue/inputtext'
import { useAuthStore } from '@/stores/auth.js'
import { useDesktopStore } from '@/stores/desktop.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { useLiveTilesStore } from '@/stores/liveTiles.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { usePortalStore } from '@/stores/portal.js'
import { useTasksStore } from '@/stores/tasks.js'
import { usePetsStore } from '@/stores/pets.js'
import { useUnitsStore } from '@/stores/units.js'
import { useActiveUnit } from '@/composables/useActiveUnit.js'
import { useElapsed } from '@/composables/useElapsed.js'
import { useDesktopNotifications } from '@/composables/useDesktopNotifications.js'
import { useNotifyMute } from '@/composables/useNotifyMute.js'
import { useScreenLock } from '@/composables/useScreenLock.js'
import { usePermission } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { menuGroups } from '@/desktop/apps.js'
import { tileFaces } from '@/desktop/liveTiles.js'
import { avatarUrl } from '@/utils/pets.js'
import { shortFio } from '@/utils/people.js'
import Logo from '@/components/common/Logo.vue'
import HolaIcon from '@/components/common/HolaIcon.vue'
import CompanySelect from '@/components/common/CompanySelect.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import WidgetTile from './WidgetTile.vue'

// Своя раскладка плиток: панель узкая, «Пуск» стола к ней отношения не имеет.
const PLATFORM = 'widgets'

const VIEW_TABS = [
  { key: 'pinned', label: 'Избранное', icon: 'push_pin' },
  { key: 'all', label: 'Все', icon: 'apps' },
]

defineProps({
  /** Ширина панели в пикселях — считает каркас (границы зависят от экрана). */
  width: { type: Number, required: true },
  /** Панель свёрнута до полоски значков. */
  collapsed: { type: Boolean, default: false },
})

const emit = defineEmits(['toggle'])

const auth = useAuthStore()
const desktop = useDesktopStore()
const prefs = useDesktopPrefsStore()
const live = useLiveTilesStore()
const messenger = useMessengerStore()
const portal = usePortalStore()
const tasks = useTasksStore()
const pets = usePetsStore()
const units = useUnitsStore()
const screenLock = useScreenLock()
const { expand } = useActiveUnit()
const { count: alerts } = useDesktopNotifications()
const { muted: notifyMuted, untilLabel: muteUntilLabel, mute, unmute } = useNotifyMute()
const { isSuperAdmin, hasActiveCompany } = usePermission()
const { settings } = useCompanySettings()

const unit = computed(() => units.activeUnit)
const { clock } = useElapsed(() => unit.value?.datetime_start)

const groups = computed(() => menuGroups({
  hasCompany: hasActiveCompany(),
  isSuperAdmin: isSuperAdmin(),
  settings: settings.value,
}, prefs.layout(PLATFORM)))

/* Пустой раздел показываем, только когда пользователь перекладывал плитки или
   завёл свои разделы: иначе в панели висели бы «мёртвые» заголовки. */
const visibleGroups = computed(() =>
  groups.value.filter((g) => g.items.length || g.custom || prefs.customized(PLATFORM)))

/* Избранное — закреплённые разделы в своём порядке; недоступные (сменилась
   компания, отключилась фича) просто выпадают из списка. */
const pinnedApps = computed(() => {
  const byId = new Map(groups.value.flatMap((g) => g.items).map((a) => [a.id, a]))
  return prefs.pinnedList(PLATFORM).map((id) => byId.get(id)).filter(Boolean)
})

const hasPinned = computed(() => pinnedApps.value.length > 0)

/* Что показываем на самом деле: пока ничего не закреплено, «Избранное» вело бы
   на пустой экран — панель остаётся на «Всех». */
const view = computed(() => (hasPinned.value && prefs.widgetsView === 'pinned' ? 'pinned' : 'all'))

// Свёрнутая панель показывает тот же набор, что и развёрнутая, одним списком.
const railApps = computed(() => (view.value === 'pinned'
  ? pinnedApps.value
  : groups.value.flatMap((g) => g.items)))

const appById = computed(() => {
  const map = new Map()
  for (const g of groups.value) for (const a of g.items) map.set(a.id, a)
  return map
})

const avatarSrc = computed(() => (auth.user ? avatarUrl(auth.user) : ''))

/* ── Плитки ────────────────────────────────────────────────── */
const hovered = ref(null)

const liveCtx = computed(() => ({ data: live.data, messenger, portal, pets, units, auth }))

function facesOf(app) {
  // Живые плитки выключены — общим тумблером или у этой плитки — обычный значок.
  return prefs.isTileLive(app.id) ? tileFaces(app.id, liveCtx.value) : []
}

function badgeOf(app) {
  if (app.id === 'messenger') return messenger.totalUnread || 0
  if (app.id === 'portal') return portal.unread || 0
  if (app.id === 'tasks') return tasks.myActiveCount || 0
  if (app.id === 'pets') return pets.pet?.sick ? '!' : 0
  return 0
}

function windowOf(app) {
  return desktop.windows.find((w) => w.appId === app.id) || null
}

const isOpen = (app) => !!windowOf(app)
const isActive = (app) => !desktop.startOpen && desktop.focused?.appId === app.id

/* Плитка — переключатель приложений: открытый раздел возвращается ТАМ, ГДЕ его
   оставили (окна нет, а перезапуск по корневому пути стёр бы место в разделе).
   Пустой холст уходит сам: и focus, и open гасят startOpen в сторе.
   Повторный клик по разделу, который уже на экране, наоборот возвращает холст —
   иначе к ленте активности было бы не вернуться, не закрыв все разделы. */
function launch(app) {
  const win = windowOf(app)
  if (!win) return desktop.open(app.path)
  if (isActive(app)) desktop.startOpen = true
  else desktop.focus(win.id)
}

function launchPath(path) {
  desktop.open(path)
}

/* ── Перетаскивание плиток ─────────────────────────────────────
   Внутри раздела меняется порядок, между разделами — принадлежность плитки. */
const drag = reactive({ appId: null, groupKey: null })

function onDragStart(group, app, e) {
  drag.appId = app.id
  drag.groupKey = group.key
  e.dataTransfer.effectAllowed = 'move'
  // Firefox не начинает перетаскивание без данных в буфере.
  e.dataTransfer.setData('text/plain', app.id)
}

function onDragOver(group, app) {
  if (!drag.appId || app.id === drag.appId) return
  const ids = group.items.map((a) => a.id)

  if (group.key === drag.groupKey) {
    const from = ids.indexOf(drag.appId)
    const to = ids.indexOf(app.id)
    if (from < 0 || to < 0) return
    ids.splice(to, 0, ...ids.splice(from, 1))
    prefs.setGroupOrder(PLATFORM, group.key, ids)
    return
  }

  const to = ids.indexOf(app.id)
  ids.splice(to < 0 ? ids.length : to, 0, drag.appId)
  prefs.moveTileToGroup(PLATFORM, drag.appId, group.key, ids)
  drag.groupKey = group.key
}

/** Перетаскивание на свободное место раздела — плитка встаёт в конец. */
function onDragOverGroup(group) {
  if (!drag.appId || group.key === drag.groupKey) return
  const ids = [...group.items.map((a) => a.id), drag.appId]
  prefs.moveTileToGroup(PLATFORM, drag.appId, group.key, ids)
  drag.groupKey = group.key
}

function onDragEnd() {
  drag.appId = null
  drag.groupKey = null
}

/* Избранное переставляется тем же жестом, что и плитки в категориях, но
   меняет НЕ порядок категории, а сам список закреплённых. */
function onPinDragStart(app, e) {
  drag.appId = app.id
  drag.groupKey = null
  e.dataTransfer.effectAllowed = 'move'
  e.dataTransfer.setData('text/plain', app.id)
}

function onPinDragOver(app) {
  if (!drag.appId || app.id === drag.appId) return
  const ids = pinnedApps.value.map((a) => a.id)
  const from = ids.indexOf(drag.appId)
  const to = ids.indexOf(app.id)
  if (from < 0 || to < 0) return
  ids.splice(to, 0, ...ids.splice(from, 1))
  prefs.setPinnedOrder(PLATFORM, ids)
}

/* ── Разделы: создание, переименование, удаление ───────────── */
const editing = ref(null)
const editLabel = ref('')
const editorRef = ref(null)

function focusEditor() {
  // v-for в шаблоне отдаёт ref массивом — берём единственное активное поле.
  nextTick(() => {
    const el = Array.isArray(editorRef.value) ? editorRef.value[0] : editorRef.value
    el?.$el?.focus?.()
    el?.$el?.select?.()
  })
}

function commitRename(group) {
  if (editing.value !== group.key) return
  const label = editLabel.value.trim()
  if (label && label !== group.label) prefs.renameGroup(PLATFORM, group.key, label)
  editing.value = null
}

function createGroup() {
  const key = prefs.addGroup(PLATFORM, 'Новый раздел')
  editing.value = key
  editLabel.value = 'Новый раздел'
  focusEditor()
}

/* ── Контекстные меню ──────────────────────────────────────────
   Одно меню на три случая (плитка, раздел, уведомления) — три отдельных
   компонента ради одного списка пунктов не нужны. */
const menu = reactive({ open: false, x: 0, y: 0, kind: 'tile', appId: null, groupKey: null })

// Сроки тишины: от «на десять минут отойти» до «сегодня меня нет».
const MUTE_OPTIONS = [
  { label: '10 минут', minutes: 10 },
  { label: '30 минут', minutes: 30 },
  { label: '1 час', minutes: 60 },
  { label: '4 часа', minutes: 240 },
  { label: '8 часов', minutes: 480 },
  { label: 'Навсегда', minutes: null },
]

const currentGroup = computed(() => visibleGroups.value.find((g) => g.key === menu.groupKey) || null)

const menuItems = computed(() => {
  if (menu.kind === 'bell') {
    if (notifyMuted.value) {
      return [{
        label: `Включить уведомления (${muteUntilLabel.value})`,
        icon: 'notifications_active',
        action: 'unmute',
      }]
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
  }

  if (menu.kind === 'group') {
    const group = currentGroup.value
    if (!group) return []
    const items = [
      { label: 'Переименовать', icon: 'edit', action: 'rename' },
      {
        label: prefs.isCollapsed(PLATFORM, group.key) ? 'Развернуть' : 'Свернуть',
        icon: prefs.isCollapsed(PLATFORM, group.key) ? 'expand_more' : 'expand_less',
        action: 'collapse',
      },
      { divider: true },
      { label: 'Новый раздел', icon: 'add', action: 'create' },
    ]
    if (group.custom) {
      items.push({ divider: true })
      items.push({ label: 'Удалить раздел', icon: 'delete', action: 'remove', danger: true })
    }
    return items
  }

  const app = appById.value.get(menu.appId)
  if (!app) return []
  const opened = isOpen(app)
  const pinned = prefs.isPinned(PLATFORM, app.id)
  return [
    { label: opened ? 'Перейти' : 'Открыть', icon: 'open_in_new', action: 'open' },
    pinned
      ? { label: 'Убрать из избранного', icon: 'keep_off', action: 'unpin' }
      : { label: 'Закрепить в избранном', icon: 'keep', action: 'pin' },
    /* Сводки поимённо: общий тумблер живых плиток главнее — при выключенном
       пункт объясняет, почему плитка «мёртвая», а не молчит. */
    prefs.liveTiles
      ? { label: 'Живая плитка', icon: prefs.isTileLive(app.id) ? 'check' : 'dashboard', action: 'live' }
      : { label: 'Живые плитки выключены', icon: 'toggle_off', disabled: true },
    ...(opened
      ? [{ divider: true }, { label: 'Закрыть раздел', icon: 'close', action: 'close', danger: true }]
      : []),
  ]
})

function openMenuAt(e) {
  menu.x = e.clientX
  menu.y = e.clientY
  menu.open = true
}

function openTileMenu(app, e) {
  menu.kind = 'tile'
  menu.appId = app.id
  openMenuAt(e)
}

function openGroupMenu(group, e) {
  menu.kind = 'group'
  menu.groupKey = group.key
  openMenuAt(e)
}

function openBellMenu(e) {
  menu.kind = 'bell'
  openMenuAt(e)
}

function onMenuSelect(action) {
  if (action === 'unmute') return unmute()
  if (action.startsWith?.('mute:')) {
    const arg = action.slice(5)
    return mute(arg === 'forever' ? null : Number(arg))
  }

  if (menu.kind === 'group') {
    const group = currentGroup.value
    if (!group) return
    if (action === 'rename') {
      editing.value = group.key
      editLabel.value = group.label
      focusEditor()
    } else if (action === 'collapse') prefs.toggleCollapsed(PLATFORM, group.key)
    else if (action === 'create') createGroup()
    // Плитки удалённого раздела возвращаются в свои родные разделы.
    else if (action === 'remove') prefs.removeGroup(PLATFORM, group.key)
    return
  }

  const app = appById.value.get(menu.appId)
  if (!app) return
  if (action === 'open') launch(app)
  else if (action === 'pin') prefs.pin(PLATFORM, app.id)
  else if (action === 'unpin') prefs.unpin(PLATFORM, app.id)
  else if (action === 'close') {
    const win = windowOf(app)
    if (win) desktop.close(win.id)
  } else if (action === 'live') {
    prefs.toggleTileLive(app.id)
    // Включили обратно — сводку этой плитки надо подтянуть: её не опрашивали.
    if (prefs.isTileLive(app.id)) live.refresh([app.id]).catch(() => {})
  }
}

/* ── Часы ──────────────────────────────────────────────────── */
const now = ref(new Date())
let timer = null

onMounted(() => { timer = setInterval(() => { now.value = new Date() }, 10000) })
onBeforeUnmount(() => clearInterval(timer))

const time = computed(() => now.value.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' }))
const date = computed(() => now.value.toLocaleDateString('ru-RU', { day: '2-digit', month: 'short' }))
const fullDate = computed(() => now.value.toLocaleDateString('ru-RU', {
  weekday: 'long', day: 'numeric', month: 'long', year: 'numeric',
}))

const LOGOUT_ACTIONS = [
  { kind: 'cancel', label: 'Остаться' },
  { kind: 'confirm', label: 'Выйти', icon: 'logout' },
]

const logoutAsk = ref(false)
</script>

<style scoped>
/* Панель прижата к левой кромке во всю высоту: ни полей, ни скруглений — она
   часть каркаса, а не карточка поверх него. */
.wp {
  position: relative;
  z-index: 900;
  display: flex;
  flex-direction: column;
  gap: 10px;
  height: 100%;
  flex-shrink: 0;
  padding: 12px 10px;
  border-right: 1px solid var(--acrylic-border);
  /* Явный ноль: панель — часть каркаса, а не карточка поверх него, и скругления
     ей не положены ни в одном состоянии. */
  border-radius: 0;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  overflow: hidden;
}

/* Свёрнутая полоска — сплошная лента значков: скругления дробили узкую
   колонку на «таблетки», поэтому подсветка идёт прямоугольником от кромки до
   кромки. */
.wp.rail { padding: 12px 0; }

.wp.rail .wp-head-tools,
.wp.rail .wp-foot-row { gap: 0; align-self: stretch; }

.wp.rail .wp-head { gap: 4px; align-self: stretch; }
.wp.rail .wp-foot { padding-top: 6px; }

/* Всё в полоске — одной шириной и без скруглений: и значки разделов, и
   кнопки шапки с подвалом (компания приходит из общего CompanySelect, поэтому
   её кнопку правим через :deep). Круглым остаётся только фирменный знак и
   само фото в аватаре — это не «скругление панели», а форма самого предмета. */
.wp.rail .wp-brand,
.wp.rail .wp-icon-btn,
.wp.rail .wp-unit,
.wp.rail .wp-rail-btn,
.wp.rail .wp-user,
.wp.rail :deep(.company-button) {
  border-radius: 0;
  width: 100%;
  min-width: 0;
  max-width: none;
}

.wp.rail .wp-icon-btn,
.wp.rail .wp-user,
.wp.rail .wp-unit,
.wp.rail :deep(.company-button) {
  height: 40px;
  min-height: 40px;
  max-height: 40px;
}

/* Фото в аватаре остаётся круглым внутри прямоугольной кнопки. */
.wp.rail .wp-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
}

/* ── Шапка ── */
.wp-head {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.wp.rail .wp-head { flex-direction: column; }

.wp-brand {
  display: grid;
  place-items: center;
  flex-shrink: 0;
  margin-right: auto;
  padding: 4px;
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  cursor: pointer;
}

.wp.rail .wp-brand { margin-right: 0; }

.wp-brand:hover { background: var(--glass-bg); }

.wp-head-tools { display: flex; align-items: center; gap: 6px; }

.wp.rail .wp-head-tools { flex-direction: column; }

.wp-icon-btn {
  position: relative;
  width: 36px;
  min-width: 36px;
  max-width: 36px;
  height: 36px;
  min-height: 36px;
  max-height: 36px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.wp-icon-btn:hover { background: var(--glass-bg); border-color: var(--acrylic-border); }
.wp-icon-btn.active { color: var(--color-primary); border-color: color-mix(in oklch, var(--color-primary) 32%, transparent); }
.wp-icon-btn.muted { color: var(--color-text-dim); }
.wp-icon-btn.danger:hover { color: var(--color-error); }
.wp-icon-btn .material-symbols-outlined { font-size: 21px; }

/* Счётчик поверх значка — общий для колокольчика и свёрнутых плиток. */
.wp-dot {
  position: absolute;
  top: 2px;
  right: 2px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 10px;
  font-weight: 700;
}

/* ── Чип идущего юнита ── */
.wp-unit {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  flex-shrink: 0;
  height: 38px;
  padding: 0 12px;
  border: 1px solid color-mix(in oklch, var(--color-success) 34%, transparent);
  border-radius: var(--radius-lg);
  background: color-mix(in oklch, var(--color-success) 12%, var(--glass-bg));
  color: var(--color-text);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.wp.rail .wp-unit { width: 38px; padding: 0; }

.wp-unit .material-symbols-outlined { font-size: 19px; color: var(--color-success); }
.wp-unit-clock { font-variant-numeric: tabular-nums; }

.wp-view { flex-shrink: 0; }

/* ── Тело: группы и плитки ── */
.wp-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow-y: auto;
  overflow-x: hidden;
  /* Полоса прокрутки в своём жёлобе и всегда видимая: иначе непонятно, какая
     часть панели листается — особенно в свёрнутой полоске, где подписей нет. */
  scrollbar-gutter: stable;
  scrollbar-width: thin;
  scrollbar-color: color-mix(in oklch, var(--color-text) 16%, transparent) transparent;
}

.wp-body::-webkit-scrollbar { width: 6px; }
.wp-body::-webkit-scrollbar-track { background: transparent; }

.wp-body::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: color-mix(in oklch, var(--color-text) 16%, transparent);
}

/* Границы ленты отбиты линиями — видно, где кончается прокручиваемая часть. */
/* Лента разделов — единственная прокручиваемая часть панели, и в свёрнутом
   виде это неочевидно: отбиваем её линиями и лёгкой подложкой, полоса
   прокрутки при этом видима постоянно (см. .wp-body). */
.wp.rail .wp-body {
  gap: 2px;
  padding: 6px 0;
  border-top: 1px solid var(--acrylic-border);
  border-bottom: 1px solid var(--acrylic-border);
  background: color-mix(in oklch, var(--color-text) 4%, transparent);
}

.wp-group { display: flex; flex-direction: column; gap: 6px; }

.wp-group-head { display: flex; align-items: center; gap: 2px; }

.wp-group-toggle {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 4px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-dim);
  font-size: 11.5px;
  font-weight: 700;
  letter-spacing: 0.4px;
  text-transform: uppercase;
  cursor: pointer;
}

.wp-group-toggle:hover { color: var(--color-text); }

.wp-group-chev {
  font-size: 18px;
  transition: rotate 0.18s ease;
}

.wp-group-chev.collapsed { rotate: -90deg; }

.wp-group-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.wp-group-input { flex: 1; min-width: 0; height: 26px; font-size: 12px; }

.wp-group-count { color: var(--color-text-dim); font-weight: 600; }

.wp-group-more {
  display: grid;
  place-items: center;
  width: 26px;
  min-width: 26px;
  max-width: 26px;
  height: 26px;
  min-height: 26px;
  max-height: 26px;
  padding: 0;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
}

.wp-group-more:hover { background: var(--glass-bg); color: var(--color-text); }
.wp-group-more .material-symbols-outlined { font-size: 18px; }

/* Свёртывание раздела: высота едет к нулю, содержимое не размонтируется. */
.wp-group-body {
  display: grid;
  grid-template-rows: 1fr;
  transition: grid-template-rows 0.22s cubic-bezier(0.2, 0, 0, 1);
}

.wp-group-body.collapsed { grid-template-rows: 0fr; }

.wp-group-inner { min-height: 0; overflow: hidden; }

.wp-tiles { display: flex; flex-direction: column; gap: 8px; }

/* Открытый раздел помечен полосой у кромки, активный — залит: свёрнутая
   полоска отвечает тем же языком, что и плитки (см. WidgetTile.vue). */
.wp-rail-btn.opened::before {
  content: '';
  position: absolute;
  left: 0;
  top: 22%;
  bottom: 22%;
  width: 3px;
  border-radius: 0 var(--radius-full) var(--radius-full) 0;
  background: color-mix(in oklch, var(--color-primary) 55%, transparent);
}

.wp-rail-btn.active::before { background: var(--color-primary); top: 12%; bottom: 12%; }

.wp-group-empty {
  margin: 0;
  padding: 12px;
  border: 1px dashed var(--acrylic-border);
  border-radius: var(--radius-lg);
  text-align: center;
  font-size: 12.5px;
  color: var(--color-text-dim);
}

.wp-add-group {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  flex-shrink: 0;
  height: 36px;
  border: 1px dashed var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: transparent;
  color: var(--color-text-dim);
  font-size: 12.5px;
  cursor: pointer;
}

.wp-add-group:hover { color: var(--color-primary); border-color: color-mix(in oklch, var(--color-primary) 32%, var(--acrylic-border)); }
.wp-add-group .material-symbols-outlined { font-size: 18px; }

/* ── Свёрнутая полоска ── */
.wp-rail-btn {
  position: relative;
  display: grid;
  place-items: center;
  width: 44px;
  min-width: 44px;
  max-width: 44px;
  height: 44px;
  min-height: 44px;
  max-height: 44px;
  flex-shrink: 0;
  padding: 0;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.wp-rail-btn:hover { background: var(--glass-bg); border-color: var(--acrylic-border); }
.wp-rail-btn.active { background: color-mix(in oklch, var(--color-primary) 14%, var(--glass-bg)); color: var(--color-primary); }
.wp-rail-btn .material-symbols-outlined { font-size: 22px; }

/* ── Подвал ── */
.wp-foot {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-top: 10px;
  border-top: 1px solid var(--acrylic-border);
}

.wp-foot-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.wp.rail .wp-foot-row { flex-direction: column; }

/* Аватар и компания — такие же прозрачные значки, как настройки и выход:
   собственная подложка выделяла их белыми плашками среди ровного ряда. */
.wp-user {
  display: grid;
  place-items: center;
  flex-shrink: 0;
  width: 36px;
  min-width: 36px;
  max-width: 36px;
  height: 36px;
  min-height: 36px;
  max-height: 36px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: 50%;
  background: transparent;
  overflow: hidden;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.wp-user:hover,
.wp-foot :deep(.company-button:hover) {
  background: var(--glass-bg);
  border-color: var(--acrylic-border);
}

.wp-foot :deep(.company-button) {
  border-color: transparent;
  background: transparent;
  box-shadow: none;
}

.wp-avatar { width: 100%; height: 100%; object-fit: cover; }


.wp-clock {
  text-align: center;
  font-size: 11.5px;
  color: var(--color-text-dim);
  font-variant-numeric: tabular-nums;
}
</style>
