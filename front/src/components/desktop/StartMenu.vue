<template>
  <div class="sm-backdrop" :data-taskbar="prefs.taskbarSide" @pointerdown.self="desktop.startOpen = false">
    <section class="start-menu" :class="{ full }" role="menu">
      <div class="sm-columns">
        <!-- Левая колонка: марка, компания, плитки и карточка пользователя —
             всё, что относится к запуску разделов, живёт только здесь. -->
        <div class="sm-main">
        <div class="sm-panel">
        <header class="sm-head">
          <button class="sm-brand" type="button" title="О приложении" @click="openAbout">
            <BrandWordmark />
          </button>
          <button
            class="sm-full"
            type="button"
            :title="full ? 'Свернуть меню' : 'Развернуть на весь экран'"
            @click="full = !full"
          >
            <span class="material-symbols-outlined">{{ full ? 'close_fullscreen' : 'open_in_full' }}</span>
          </button>
        </header>

        <div class="sm-body">
        <section v-for="group in visibleGroups" :key="group.key" class="sm-group">
          <div class="sm-group-head" @contextmenu.prevent="openGroupMenu(group, $event)">
            <button
              class="sm-group-toggle"
              type="button"
              :title="prefs.isCollapsed(PLATFORM, group.key) ? 'Развернуть раздел' : 'Свернуть раздел'"
              @click="prefs.toggleCollapsed(PLATFORM, group.key)"
            >
              <span
                class="material-symbols-outlined sm-group-chev"
                :class="{ collapsed: prefs.isCollapsed(PLATFORM, group.key) }"
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
          <div class="sm-group-body" :class="{ collapsed: prefs.isCollapsed(PLATFORM, group.key) }">
            <div class="sm-group-inner">
              <div class="sm-tiles" @dragover.prevent="onDragOverGroup(group)" @drop.prevent="onDragEnd">
                <button
                  v-for="(app, i) in group.items"
                  :key="app.id"
                  class="sm-tile"
                  :class="[`is-${sizeOf(app)}`, { pinned: prefs.isPinned(PLATFORM, app.id), dragging: drag.appId === app.id }]"
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
                  <span v-if="prefs.isPinned(PLATFORM, app.id)" class="sm-tile-pin material-symbols-outlined">keep</span>
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
        </div>

        <!-- Подвал: кто я, активная компания, настройки и выход — одной строкой. -->
        <footer class="sm-foot">
          <button
            class="sm-user"
            type="button"
            :title="auth.user?.fio || 'Аккаунт'"
            @click="launchPath('/settings?section=account')"
          >
            <img class="sm-avatar" :src="avatarSrc" :alt="auth.user?.fio || 'Аккаунт'" />
          </button>
          <div class="sm-company">
            <CompanySelect />
          </div>
          <button class="sm-icon-btn" type="button" title="Настройки" @click="launchPath('/settings')">
            <span class="material-symbols-outlined">settings</span>
          </button>
          <!-- Запереть экран: сессия остаётся живой, приложение закрывается
               пин-кодом. Кнопка видна, только когда блокировка включена. -->
          <button
            v-if="screenLock.enabled.value"
            class="sm-icon-btn"
            type="button"
            title="Заблокировать (Ctrl+L)"
            @click="lockScreen"
          >
            <span class="material-symbols-outlined">lock</span>
          </button>
          <button class="sm-icon-btn danger" type="button" title="Выйти" @click="logoutAsk = true">
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

    <!-- Выход — из тех действий, что делают одним промахом мыши: спрашиваем. -->
    <AppDialog
      v-model="logoutAsk"
      tone="danger"
      size="sm"
      title="Выйти из системы?"
      subtitle="Открытые окна закроются, для возврата понадобится войти заново."
      :actions="LOGOUT_ACTIONS"
      @confirm="auth.logout()"
    />
  </div>
</template>

<script setup>
import { avatarUrl } from '@/utils/pets.js'
import { useScreenLock } from '@/composables/useScreenLock.js'
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
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
import { menuGroups } from '@/desktop/apps.js'
import { tileFaces } from '@/desktop/liveTiles.js'
import { useLiveTilesStore } from '@/stores/liveTiles.js'
import { useUnitsStore } from '@/stores/units.js'
import BrandWordmark from '@/components/common/BrandWordmark.vue'
import CompanySelect from '@/components/common/CompanySelect.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import LiveTile from './LiveTile.vue'
import ActivityPanel from './ActivityPanel.vue'

// Раскладка «Пуска» стола хранится отдельно от мобилы — desktopPrefs держит
// обе, здесь работаем только со своей.
const PLATFORM = 'desktop'

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

onMounted(() => {
  /* Сводки живых плиток рабочий стол тянет заранее и обновляет по таймеру —
     здесь лишь подстраховка: свежие данные запрос не повторяют (TTL стора). */
  if (prefs.liveTiles) {
    const ids = groups.value
      .flatMap((g) => g.items.map((a) => a.id))
      .filter((id) => prefs.isTileLive(id))
    live.refresh(ids).catch(() => {})
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
}, prefs.layout(PLATFORM)))

/* Пустой раздел показываем, только когда пользователь перекладывал плитки или
   завёл свои разделы: иначе в меню висели бы «мёртвые» заголовки. */
const visibleGroups = computed(() =>
  groups.value.filter((g) => g.items.length || g.custom || prefs.customized(PLATFORM)))

const avatarSrc = computed(() => {
  const user = auth.user
  if (!user) return ''
  return avatarUrl(user)
})

const appById = computed(() => {
  const map = new Map()
  for (const g of groups.value) for (const a of g.items) map.set(a.id, a)
  return map
})

const sizeOf = (app) => prefs.tileSize(PLATFORM, app.id, app.tile || 'square')

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
  if (label && label !== group.label) prefs.renameGroup(PLATFORM, group.key, label)
  editing.value = null
}

function createGroup() {
  const key = prefs.addGroup(PLATFORM, 'Новый раздел')
  editing.value = key
  editLabel.value = 'Новый раздел'
  focusEditor()
}

/* ── Контекстное меню плитки: размер и закрепление ───────────── */
/* Полноэкранное меню: разворачивается кнопкой в шапке, а в настройках можно
   выбрать, чтобы оно ВСЕГДА открывалось во весь экран (тогда кнопка
   сворачивает его до обычной панели на время сеанса). */
const full = ref(prefs.startFullscreen)
watch(() => prefs.startFullscreen, (on) => { full.value = on })

// Пока меню развёрнуто, панель задач прячется: экран занимает только меню.
watch(full, (on) => { desktop.startFull = on }, { immediate: true })
onBeforeUnmount(() => { desktop.startFull = false })

// Подтверждение выхода: кнопка стоит рядом с настройками, промахнуться легко.
const logoutAsk = ref(false)

const screenLock = useScreenLock()

function lockScreen() {
  screenLock.lock()
  desktop.startOpen = false
}
const LOGOUT_ACTIONS = [
  { kind: 'cancel', label: 'Остаться' },
  { kind: 'confirm', label: 'Выйти', icon: 'logout' },
]

const tileMenu = reactive({ open: false, x: 0, y: 0, appId: null })

const tileMenuItems = computed(() => {
  const id = tileMenu.appId
  if (!id) return []
  const size = prefs.tileSize(PLATFORM, id, appById.value.get(id)?.tile || 'square')
  return [
    { label: 'Широкая плитка', icon: size === 'wide' ? 'check' : 'width_wide', action: 'wide' },
    { label: 'Квадратная плитка', icon: size === 'square' ? 'check' : 'crop_square', action: 'square' },
    { divider: true },
    /* Сводки поимённо: общий тумблер живых плиток главнее — при выключенном
       пункт объясняет, почему плитка «мёртвая», а не молчит. */
    prefs.liveTiles
      ? {
          label: 'Живая плитка',
          icon: prefs.isTileLive(id) ? 'check' : 'dashboard',
          action: 'live',
        }
      : { label: 'Живые плитки выключены', icon: 'toggle_off', disabled: true },
    { divider: true },
    prefs.isPinned(PLATFORM, id)
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
  if (action === 'wide' || action === 'square') prefs.setTileSize(PLATFORM, id, action)
  else if (action === 'live') {
    prefs.toggleTileLive(id)
    // Включили обратно — сводку этой плитки надо подтянуть: её не опрашивали.
    if (prefs.isTileLive(id)) live.refresh([id]).catch(() => {})
  }
  else if (action === 'pin') prefs.pin(PLATFORM, id)
  else if (action === 'unpin') prefs.unpin(PLATFORM, id)
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
  else if (action === 'collapse') prefs.toggleCollapsed(PLATFORM, group.key)
  else if (action === 'create') createGroup()
  // Плитки удалённого раздела возвращаются в свои родные разделы.
  else if (action === 'remove') prefs.removeGroup(PLATFORM, group.key)
}
</script>

<style scoped>
/* Во весь экран — по-настоящему: без полей, скруглений и панели задач (её
   прячет рабочий стол по desktop.startFull). Меню держит экран, пока не
   выберут раздел или не свернут его обратно. Правило идёт ПОСЛЕ раскладок по
   сторонам панели и перекрывает их. */
.sm-backdrop .start-menu.full {
  inset: 0;
  width: auto;
  max-width: none;
  max-height: none;
  transform: none;
  border-radius: 0;
  border: none;
}

/* Кнопка «во весь экран» — в шапке у правого края. */
.sm-full {
  margin-left: auto;
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
}

.sm-full:hover { background: var(--color-surface-variant); color: var(--color-text); }
.sm-full .material-symbols-outlined { font-size: 20px; }

/* Панель задач сверху — меню выезжает вниз; по бокам — от своего края, а по
   вертикали центрируется: тянуться от кнопки «Пуск» через весь экран незачем. */
.sm-backdrop[data-taskbar='top'] .start-menu {
  top: calc(var(--taskbar-height) + 24px);
  bottom: auto;
  transform-origin: top center;
}

.sm-backdrop[data-taskbar='left'] .start-menu,
.sm-backdrop[data-taskbar='right'] .start-menu {
  top: 50%;
  bottom: auto;
  left: auto;
  transform: translateY(-50%);
  max-height: min(820px, calc(100dvh - 48px));
}

.sm-backdrop[data-taskbar='left'] .start-menu {
  left: calc(var(--taskbar-height) + 24px);
  transform-origin: center left;
}

.sm-backdrop[data-taskbar='right'] .start-menu {
  right: calc(var(--taskbar-height) + 24px);
  transform-origin: center right;
}

/* Прозрачная подложка на весь экран: клик мимо меню закрывает его, как в ОС. */
.sm-backdrop {
  position: fixed;
  inset: 0;
  z-index: 950;
}

.start-menu {
  /* Меню по центру экрана со стороны панели задач (по умолчанию — над ней). */
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  bottom: calc(var(--taskbar-height) + 24px);
  /* Плитки + колонка «Моя активность» (на узком экране колонка скрывается). */
  width: min(900px, calc(100vw - 24px));
  max-height: min(820px, calc(100dvh - var(--taskbar-height) - 48px));
  display: flex;
  flex-direction: column;
  padding: 12px;
  gap: 12px;
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

/* ── Шапка: марка (выбор компании живёт в подвале) ── */
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

.sm-company {
  flex: 1;
  min-width: 0;
  display: flex;
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
  grid-template-columns: minmax(0, 1fr) 300px;
  gap: 14px;
}

.sm-main {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  min-height: 0;
}

/* Стеклянная подложка под марку и плитки — парная к панели «Моя активность». */
.sm-panel {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
}

.sm-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 14px;
  /* Жёлоб под полосу прокрутки резервируется всегда — иначе она ложится
     поверх плиток, а при появлении дёргает всю сетку. Сама полоса — еле
     заметная: меню и так плотное, яркая линия сбоку только мешает. */
  scrollbar-gutter: stable;
  padding-right: 10px;
  scrollbar-width: thin;
  scrollbar-color: color-mix(in oklch, var(--color-text) 14%, transparent) transparent;
}

.sm-body::-webkit-scrollbar { width: 6px; }
.sm-body::-webkit-scrollbar-track { background: transparent; }

.sm-body::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: color-mix(in oklch, var(--color-text) 14%, transparent);
}

/* Под курсором чуть заметнее — чтобы можно было прицелиться. */
.sm-body:hover { scrollbar-color: color-mix(in oklch, var(--color-text) 26%, transparent) transparent; }
.sm-body:hover::-webkit-scrollbar-thumb {
  background: color-mix(in oklch, var(--color-text) 26%, transparent);
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

/* Обычное меню: РОВНО четыре колонки, то есть две широкие плитки в ряд.
   Ширину колонки считает сама сетка — подбирать её в пикселях бессмысленно,
   доступное место зависит от полосы прокрутки и колонки «Моя активность». */
.sm-tiles {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  min-height: 44px;
}

/* Во весь экран плитки НЕ растягиваются: ширина фиксируется той же, что в
   обычном меню, и плитки просто перетекают на освободившееся место. */
.start-menu.full .sm-tiles {
  grid-template-columns: repeat(auto-fill, var(--sm-tile-w, 132px));
  justify-content: start;
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

/* ── Подвал: пользователь, компания, настройки, выход — на своей подложке ── */
.sm-foot {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  padding: 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
}

/* Аватар — круглая кнопка входа в аккаунт (имя не дублируем: оно в подсказке). */
.sm-user {
  display: grid;
  place-items: center;
  flex-shrink: 0;
  width: 52px;
  min-width: 52px;
  max-width: 52px;
  height: 52px;
  min-height: 52px;
  max-height: 52px;
  padding: 0;
  border: 1px solid var(--acrylic-border);
  border-radius: 50%;
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  overflow: hidden;
  cursor: pointer;
  transition: border-color 0.15s;
}

.sm-user:hover { border-color: color-mix(in oklch, var(--color-primary) 34%, var(--acrylic-border)); }

.sm-avatar {
  width: 100%;
  height: 100%;
  object-fit: cover;
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
