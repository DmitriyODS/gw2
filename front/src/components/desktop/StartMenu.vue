<template>
  <div class="sm-backdrop" @pointerdown.self="desktop.startOpen = false">
    <section class="start-menu" role="menu">
      <div class="sm-columns">
        <!-- Левая колонка: марка, компания, плитки и карточка пользователя —
             всё, что относится к запуску разделов, живёт только здесь. -->
        <div class="sm-main">
        <header class="sm-head">
          <button class="sm-brand" type="button" title="О приложении" @click="openAbout">
            <span class="sm-word-groove">Groove</span>
            <span class="sm-word-work">Work</span>
            <span v-if="majorVersion" class="sm-word-work">{{ majorVersion }}</span>
          </button>
          <CompanySelect variant="row" class="sm-company" />
        </header>

        <div class="sm-body">
        <section v-for="group in visibleGroups" :key="group.key" class="sm-group">
          <div class="sm-group-head" @contextmenu.prevent="openGroupMenu(group, $event)">
            <button
              class="sm-group-toggle"
              type="button"
              :title="prefs.isCollapsed(group.key) ? 'Развернуть раздел' : 'Свернуть раздел'"
              @click="prefs.toggleCollapsed(group.key)"
            >
              <span
                class="material-symbols-outlined sm-group-chev"
                :class="{ collapsed: prefs.isCollapsed(group.key) }"
              >expand_more</span>
              <InputText
                v-if="editing === group.key"
                ref="editorRef"
                v-model="editLabel"
                class="sm-group-input"
                @click.stop
                @keyup.enter="commitRename(group)"
                @keyup.esc="editing = null"
                @blur="commitRename(group)"
              />
              <span v-else class="sm-group-label">{{ group.label }}</span>
              <span class="sm-group-count">{{ group.items.length }}</span>
            </button>
            <button
              class="sm-group-more"
              type="button"
              title="Настроить раздел"
              @click.stop="openGroupMenu(group, $event)"
            >
              <span class="material-symbols-outlined">more_horiz</span>
            </button>
          </div>

          <!-- Свёрнутый раздел схлопывается по высоте (grid-template-rows). -->
          <div class="sm-group-body" :class="{ collapsed: prefs.isCollapsed(group.key) }">
            <div class="sm-group-inner">
              <div class="sm-tiles" @dragover.prevent="onDragOverGroup(group)" @drop.prevent="onDragEnd">
                <button
                  v-for="(app, i) in group.items"
                  :key="app.id"
                  class="sm-tile"
                  :class="[`is-${sizeOf(app)}`, { pinned: prefs.isPinned(app.id), dragging: drag.appId === app.id }]"
                  type="button"
                  draggable="true"
                  :title="app.title"
                  @dragstart="onDragStart(group, app, $event)"
                  @dragover.prevent.stop="onDragOver(group, app)"
                  @drop.prevent="onDragEnd"
                  @dragend="onDragEnd"
                  @click="launch(app, $event)"
                  @auxclick.middle.prevent="desktop.open(app.path, { newWindow: true })"
                  @contextmenu.prevent.stop="openTileMenu(app, $event)"
                  @pointerenter="hovered = app.id"
                  @pointerleave="hovered = hovered === app.id ? null : hovered"
                >
                  <LiveTile
                    :title="app.title"
                    :icon="app.icon"
                    :faces="facesOf(app)"
                    :wide="sizeOf(app) === 'wide'"
                    :order="i"
                    :paused="hovered === app.id || !!drag.appId"
                  />
                  <span v-if="badgeOf(app)" class="sm-tile-badge" :class="{ alert: badgeOf(app) === '!' }">
                    {{ badgeOf(app) }}
                  </span>
                  <span v-if="prefs.isPinned(app.id)" class="sm-tile-pin material-symbols-outlined">keep</span>
                </button>

                <p v-if="!group.items.length" class="sm-group-empty">Перетащите сюда плитку</p>
              </div>
            </div>
          </div>
        </section>

        <button class="sm-add-group" type="button" @click="createGroup">
          <span class="material-symbols-outlined">add</span>
          Новый раздел
        </button>
        </div>

        <footer class="sm-foot">
          <button class="sm-user" type="button" title="Профиль" @click="launchPath('/profile')">
            <img class="sm-avatar" :src="avatarSrc" :alt="auth.user?.fio" />
            <span class="sm-user-name">{{ auth.user?.fio || 'Профиль' }}</span>
          </button>
          <button class="sm-icon-btn" type="button" title="Настройки" @click="launchPath('/settings')">
            <span class="material-symbols-outlined">settings</span>
          </button>
          <button class="sm-icon-btn danger" type="button" title="Выйти" @click="auth.logout()">
            <span class="material-symbols-outlined">logout</span>
          </button>
        </footer>
        </div>

        <!-- Правая колонка целиком отдана ленте: ни выбора компании, ни
             карточки пользователя над ней и под ней. -->
        <ActivityPanel class="sm-activity" @open="launchPath" />
      </div>
    </section>

    <ContextMenu
      :visible="tileMenu.open"
      :x="tileMenu.x"
      :y="tileMenu.y"
      :items="tileMenuItems"
      @select="onTileMenuSelect"
      @close="tileMenu.open = false"
    />

    <ContextMenu
      :visible="groupMenu.open"
      :x="groupMenu.x"
      :y="groupMenu.y"
      :items="groupMenuItems"
      @select="onGroupMenuSelect"
      @close="groupMenu.open = false"
    />
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import InputText from 'primevue/inputtext'
import { useAuthStore } from '@/stores/auth.js'
import { useDesktopStore } from '@/stores/desktop.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { usePortalStore } from '@/stores/portal.js'
import { useTasksStore } from '@/stores/tasks.js'
import { usePetsStore } from '@/stores/pets.js'
import { usePermission } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { useAppVersion } from '@/composables/useAppVersion.js'
import { menuGroups } from '@/desktop/apps.js'
import { tileFaces } from '@/desktop/liveTiles.js'
import { useLiveTilesStore } from '@/stores/liveTiles.js'
import { useUnitsStore } from '@/stores/units.js'
import CompanySelect from '@/components/common/CompanySelect.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import LiveTile from './LiveTile.vue'
import ActivityPanel from './ActivityPanel.vue'

const auth = useAuthStore()
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
const { majorVersion, load: loadVersion } = useAppVersion()

onMounted(() => {
  loadVersion()
  /* Сводки живых плиток рабочий стол тянет заранее и обновляет по таймеру —
     здесь лишь подстраховка: свежие данные запрос не повторяют (TTL стора). */
  if (prefs.liveTiles) {
    live.refresh(groups.value.flatMap((g) => g.items.map((a) => a.id))).catch(() => {})
  }
})

function openAbout() {
  desktop.open('/settings?section=about')
  desktop.startOpen = false
}

const groups = computed(() => menuGroups({
  hasCompany: hasActiveCompany(),
  isSuperAdmin: isSuperAdmin(),
  settings: settings.value,
}, prefs.layout))

/* Пустой раздел показываем, только когда пользователь перекладывал плитки или
   завёл свои разделы: иначе в меню висели бы «мёртвые» заголовки. */
const visibleGroups = computed(() =>
  groups.value.filter((g) => g.items.length || g.custom || prefs.customized))

const avatarSrc = computed(() => {
  const user = auth.user
  if (!user) return ''
  return user.avatar_path ? `/uploads/${user.avatar_path}` : `/api/users/${user.id}/identicon`
})

const appById = computed(() => {
  const map = new Map()
  for (const g of groups.value) for (const a of g.items) map.set(a.id, a)
  return map
})

const sizeOf = (app) => prefs.tileSize(app.id, app.tile || 'square')

/* Грани живой плитки: сводки стора + то, что уже есть в памяти приложения.
   Плитка, которую тащат или под курсором, замирает — см. prop paused. */
const hovered = ref(null)

const liveCtx = computed(() => ({
  data: live.data,
  messenger,
  portal,
  pets,
  units,
  auth,
}))

function facesOf(app) {
  // Живые плитки выключены в настройках — оставляем обычные значки.
  return prefs.liveTiles ? tileFaces(app.id, liveCtx.value) : []
}

function badgeOf(app) {
  if (app.id === 'messenger') return messenger.totalUnread || 0
  if (app.id === 'portal') return portal.unread || 0
  if (app.id === 'tasks') return tasks.myActiveCount || 0
  if (app.id === 'pets') return pets.pet?.sick ? '!' : 0
  return 0
}

// Ctrl/Shift/средняя кнопка — ещё одно окно раздела, обычный клик — открыть
// или поднять уже открытое.
function launch(app, e) {
  desktop.open(app.path, { newWindow: !!(e?.ctrlKey || e?.metaKey || e?.shiftKey) })
  desktop.startOpen = false
}

function launchPath(path) {
  desktop.open(path)
  desktop.startOpen = false
}

/* ── Перетаскивание плиток ─────────────────────────────────────
   Внутри раздела меняется порядок, между разделами — принадлежность плитки.
   Всё пишется в личные настройки, поэтому раскладка едет между устройствами. */
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
    prefs.setGroupOrder(group.key, ids)
    return
  }

  const to = ids.indexOf(app.id)
  ids.splice(to < 0 ? ids.length : to, 0, drag.appId)
  prefs.moveTileToGroup(drag.appId, group.key, ids)
  drag.groupKey = group.key
}

/** Перетаскивание на свободное место раздела — плитка встаёт в конец. */
function onDragOverGroup(group) {
  if (!drag.appId || group.key === drag.groupKey) return
  const ids = [...group.items.map((a) => a.id), drag.appId]
  prefs.moveTileToGroup(drag.appId, group.key, ids)
  drag.groupKey = group.key
}

function onDragEnd() {
  drag.appId = null
  drag.groupKey = null
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

function startRename(group) {
  editing.value = group.key
  editLabel.value = group.label
  focusEditor()
}

function commitRename(group) {
  if (editing.value !== group.key) return
  const label = editLabel.value.trim()
  if (label && label !== group.label) prefs.renameGroup(group.key, label)
  editing.value = null
}

function createGroup() {
  const key = prefs.addGroup('Новый раздел')
  editing.value = key
  editLabel.value = 'Новый раздел'
  focusEditor()
}

/* ── Контекстное меню плитки: размер и закрепление ───────────── */
const tileMenu = reactive({ open: false, x: 0, y: 0, appId: null })

const tileMenuItems = computed(() => {
  const id = tileMenu.appId
  if (!id) return []
  const size = prefs.tileSize(id, appById.value.get(id)?.tile || 'square')
  return [
    { label: 'Широкая плитка', icon: size === 'wide' ? 'check' : 'width_wide', action: 'wide' },
    { label: 'Квадратная плитка', icon: size === 'square' ? 'check' : 'crop_square', action: 'square' },
    { divider: true },
    prefs.isPinned(id)
      ? { label: 'Открепить от панели задач', icon: 'keep_off', action: 'unpin' }
      : { label: 'Закрепить на панели задач', icon: 'keep', action: 'pin' },
    { label: 'Открыть ещё одно окно', icon: 'add', action: 'new' },
  ]
})

function openTileMenu(app, e) {
  tileMenu.appId = app.id
  tileMenu.x = e.clientX
  tileMenu.y = e.clientY
  tileMenu.open = true
}

function onTileMenuSelect(action) {
  const id = tileMenu.appId
  if (!id) return
  if (action === 'wide' || action === 'square') prefs.setTileSize(id, action)
  else if (action === 'pin') prefs.pin(id)
  else if (action === 'unpin') prefs.unpin(id)
  else if (action === 'new') {
    const app = appById.value.get(id)
    if (app) { desktop.open(app.path, { newWindow: true }); desktop.startOpen = false }
  }
}

/* ── Контекстное меню раздела ───────────────────────────────── */
const groupMenu = reactive({ open: false, x: 0, y: 0, key: null })

const currentGroup = computed(() => visibleGroups.value.find((g) => g.key === groupMenu.key) || null)

const groupMenuItems = computed(() => {
  const group = currentGroup.value
  if (!group) return []
  const items = [
    { label: 'Переименовать', icon: 'edit', action: 'rename' },
    {
      label: prefs.isCollapsed(group.key) ? 'Развернуть' : 'Свернуть',
      icon: prefs.isCollapsed(group.key) ? 'expand_more' : 'expand_less',
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
})

function openGroupMenu(group, e) {
  groupMenu.key = group.key
  groupMenu.x = e.clientX
  groupMenu.y = e.clientY
  groupMenu.open = true
}

function onGroupMenuSelect(action) {
  const group = currentGroup.value
  if (!group) return
  if (action === 'rename') startRename(group)
  else if (action === 'collapse') prefs.toggleCollapsed(group.key)
  else if (action === 'create') createGroup()
  // Плитки удалённого раздела возвращаются в свои родные разделы.
  else if (action === 'remove') prefs.removeGroup(group.key)
}
</script>

<style scoped>
/* Прозрачная подложка на весь экран: клик мимо меню закрывает его, как в ОС. */
.sm-backdrop {
  position: fixed;
  inset: 0;
  z-index: 950;
}

.start-menu {
  /* Меню всегда по центру экрана над панелью задач. */
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  bottom: calc(var(--taskbar-height) + 24px);
  /* Плитки + колонка последних действий (на узком экране колонка скрывается). */
  width: min(812px, calc(100vw - 24px));
  max-height: min(820px, calc(100dvh - var(--taskbar-height) - 48px));
  display: flex;
  flex-direction: column;
  padding: 22px;
  gap: 16px;
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
  /* Меню выезжает из панели задач и въезжает обратно; классы задаёт
     <Transition> рабочего стола на корне компонента. */
  transform-origin: bottom center;
  transition: opacity 0.2s ease, translate 0.24s cubic-bezier(0.2, 0, 0, 1),
    scale 0.24s cubic-bezier(0.2, 0, 0, 1);
}

.sm-enter-from .start-menu,
.sm-leave-to .start-menu {
  opacity: 0;
  translate: 0 26px;
  scale: 0.92;
}

/* ── Шапка: марка + активная компания ── */
.sm-head {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-shrink: 0;
}

.sm-brand {
  display: flex;
  align-items: baseline;
  gap: 7px;
  padding: 2px 6px;
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  font-size: 26px;
  /* ExtraBlack вариативного Roboto Flex — фирменное начертание wordmark.
     Кнопка не наследует шрифт документа сама, поэтому задаём явно, а вес
     дублируем осью вариативного шрифта. */
  font-family: 'Roboto Flex', 'Roboto', sans-serif;
  font-weight: 1000;
  font-variation-settings: 'wght' 1000;
  letter-spacing: 0.2px;
  cursor: pointer;
  transition: background 0.15s;
}

.sm-brand:hover { background: color-mix(in oklch, var(--color-primary) 10%, transparent); }
.sm-word-groove { color: var(--color-primary); }
.sm-word-work { color: var(--color-text); }

.sm-company { margin-left: auto; min-width: 0; max-width: 300px; }

.sm-company :deep(.company-row) {
  background: var(--glass-bg);
  border: 1px solid var(--acrylic-border);
  box-shadow: var(--glass-edge);
  border-radius: var(--radius-lg);
}

.sm-company :deep(.company-row:hover) {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
  background: var(--glass-bg);
}

/* ── Разделы и плитки ──
   Сетка 4 колонки: широкая плитка занимает две, квадратная — одну.
   Размер, порядок и принадлежность разделу — личные настройки. */
/* Две колонки: слева всё про запуск разделов (марка, компания, плитки,
   пользователь), справа — только лента. Прокручиваются независимо. */
.sm-columns {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 244px;
  gap: 16px;
}

.sm-main {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
  min-height: 0;
}

.sm-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding-right: 2px;
  scrollbar-width: thin;
}

/* Узкое окно — лента уступает место плиткам. */
@media (max-width: 860px) {
  .sm-columns { grid-template-columns: minmax(0, 1fr); }
  .sm-activity { display: none; }
}

.sm-group-head {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 8px;
}

.sm-group-toggle {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  height: 30px;
  padding: 0 6px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.sm-group-toggle:hover { background: color-mix(in oklch, var(--color-primary) 8%, transparent); }

.sm-group-chev {
  font-size: 20px;
  transition: rotate 0.22s cubic-bezier(0.2, 0, 0, 1);
}

.sm-group-chev.collapsed { rotate: -90deg; }

.sm-group-label {
  font-size: 14px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sm-group-input {
  flex: 1;
  min-width: 0;
  max-width: 260px;
  height: 26px;
  padding: 0 8px;
  font-size: 14px;
}

.sm-group-count {
  margin-left: 4px;
  font-size: 12px;
  color: color-mix(in oklch, var(--color-text-dim) 70%, transparent);
}

.sm-group-more {
  width: 30px;
  min-width: 30px;
  max-width: 30px;
  height: 30px;
  min-height: 30px;
  max-height: 30px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.15s, background 0.15s, color 0.15s;
}

.sm-group-head:hover .sm-group-more { opacity: 1; }
.sm-group-more:hover { background: color-mix(in oklch, var(--color-primary) 12%, transparent); color: var(--color-primary); }
.sm-group-more .material-symbols-outlined { font-size: 20px; }

/* Сворачивание раздела: 1fr → 0fr даёт плавную высоту без замеров JS. */
.sm-group-body {
  display: grid;
  grid-template-rows: 1fr;
  transition: grid-template-rows 0.24s cubic-bezier(0.2, 0, 0, 1), opacity 0.18s ease;
}

.sm-group-body.collapsed {
  grid-template-rows: 0fr;
  opacity: 0;
}

.sm-group-inner { overflow: hidden; }

.sm-tiles {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  min-height: 44px;
}

.sm-group-empty {
  grid-column: 1 / -1;
  margin: 0;
  padding: 14px;
  border: 1px dashed var(--acrylic-border);
  border-radius: var(--radius-lg);
  text-align: center;
  font-size: 13px;
  color: var(--color-text-dim);
}

.sm-add-group {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 40px;
  border: 1px dashed var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: transparent;
  color: var(--color-text-dim);
  font-size: 13.5px;
  font-weight: 500;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}

.sm-add-group:hover {
  border-color: color-mix(in oklch, var(--color-primary) 34%, var(--acrylic-border));
  color: var(--color-primary);
  background: color-mix(in oklch, var(--color-primary) 6%, transparent);
}

.sm-add-group .material-symbols-outlined { font-size: 20px; }

.sm-tile {
  position: relative;
  grid-column: span 1;
  /* Содержимое плитки рисует LiveTile — он растягивается на всю площадь. */
  display: flex;
  align-items: stretch;
  height: 112px;
  padding: 14px;
  overflow: hidden;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}

.sm-tile.is-wide { grid-column: span 2; }

/* Перетаскиваемая плитка — приглушена: её «место» уже занято подсказкой
   порядка (соседи разъезжаются сразу). */
.sm-tile.dragging { opacity: 0.45; }

.sm-tile:hover {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
  background: color-mix(in oklch, var(--color-primary) 6%, var(--glass-bg));
}

.sm-tile-badge {
  position: absolute;
  top: 10px;
  right: 10px;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-sm);
  background: color-mix(in oklch, var(--color-primary) 16%, var(--color-surface));
  border: 1px solid color-mix(in oklch, var(--color-primary) 24%, transparent);
  color: var(--color-primary);
  font-size: 11px;
  font-weight: 700;
}

.sm-tile-badge.alert {
  background: var(--color-error-container);
  border-color: color-mix(in oklch, var(--color-error) 30%, transparent);
  color: var(--color-on-error-container);
}

/* Закреплённая на панели задач плитка помечается канцелярской кнопкой. */
.sm-tile-pin {
  position: absolute;
  right: 10px;
  bottom: 10px;
  font-size: 16px;
  color: var(--color-text-dim);
  opacity: 0.7;
}

/* ── Подвал: пользователь, настройки, выход ── */
.sm-foot {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.sm-user {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  height: 52px;
  padding: 0 12px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  cursor: pointer;
  transition: border-color 0.15s;
}

.sm-user:hover { border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border)); }

.sm-avatar {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  object-fit: cover;
  flex-shrink: 0;
}

.sm-user-name {
  font-size: 15px;
  font-weight: 500;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sm-icon-btn {
  width: 52px;
  min-width: 52px;
  max-width: 52px;
  height: 52px;
  min-height: 52px;
  max-height: 52px;
  display: grid;
  place-items: center;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.sm-icon-btn .material-symbols-outlined { font-size: 22px; }

.sm-icon-btn:hover {
  border-color: color-mix(in oklch, var(--color-primary) 32%, var(--acrylic-border));
  color: var(--color-primary);
}

.sm-icon-btn.danger {
  background: var(--color-error-container);
  border-color: color-mix(in oklch, var(--color-error) 20%, transparent);
  color: var(--color-on-error-container);
}

.sm-icon-btn.danger:hover { background: color-mix(in oklch, var(--color-error) 22%, var(--color-error-container)); }
</style>
