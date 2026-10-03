<template>
  <div class="sm-backdrop" :data-taskbar="prefs.taskbarSide" @pointerdown.self="desktop.startOpen = false">
    <!-- Во весь экран — тот же «Пуск» в планшетном виде: плитки и категории
         стола крупно, аккаунт и лента колонкой слева. Занимает весь стол:
         панель задач на это время прячется (DesktopTaskbar), выход — кнопкой
         рядом с Hola, запуском раздела или Esc. -->
    <div v-if="desktop.startFull" class="sm-tablet">
      <MobileStart platform="desktop" tablet-layout>
        <template #brand-actions>
          <button class="sm-full" type="button" title="Обычное меню" @click="desktop.startFull = false">
            <span class="material-symbols-outlined">close_fullscreen</span>
          </button>
        </template>
      </MobileStart>
    </div>

    <!-- Две стеклянные панели без общей подложки: слева запуск разделов,
         справа лента «Моя активность». Разделяет их только пустота. -->
    <section v-else class="start-menu" :class="{ 'no-activity': !prefs.startActivity }" role="menu">
      <div class="sm-panel sm-main">
        <header class="sm-head">
          <button class="sm-brand" type="button" title="О приложении" @click="openAbout">
            <BrandWordmark />
          </button>
          <span class="sm-screen-title">{{ showFavorites ? 'Избранное' : 'Все разделы' }}</span>
          <button class="sm-full" type="button" title="Во весь экран" @click="desktop.startFull = true">
            <span class="material-symbols-outlined">open_in_full</span>
          </button>
        </header>

        <!-- Два экрана, как в «Пуске» Windows 11: избранное — то, чем человек
             пользуется каждый день, все разделы — полный каталог. Меню всегда
             открывается на избранном. -->
        <div class="sm-screens">
          <Transition :name="showFavorites ? 'sm-back' : 'sm-fwd'" mode="out-in">
            <div v-if="showFavorites" key="favorites" class="sm-body">
              <div v-if="favoriteApps.length" class="sm-tiles">
                <button
                  v-for="(app, i) in favoriteApps"
                  :key="app.id"
                  class="sm-tile"
                  :class="[`is-${sizeOf(app)}`, { dragging: drag.appId === app.id }]"
                  type="button"
                  draggable="true"
                  :title="app.title"
                  @dragstart="onFavDragStart(app, $event)"
                  @dragover.prevent.stop="onFavDragOver(app)"
                  @drop.prevent="onDragEnd"
                  @dragend="onDragEnd"
                  @click="launch(app, $event)"
                  @auxclick.middle.prevent="desktop.open(app.path, { newWindow: true })"
                  @contextmenu.prevent.stop="openAppMenu(app, 'tile', $event)"
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
              </div>

              <p v-else class="sm-empty">
                Избранное пусто. Откройте «Все разделы» и отметьте звёздочкой то, чем пользуетесь каждый день.
              </p>
            </div>

            <div v-else key="all" class="sm-body">
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
                    <div class="sm-rows" @dragover.prevent="onDragOverGroup(group)" @drop.prevent="onDragEnd">
                      <div
                        v-for="app in group.items"
                        :key="app.id"
                        class="sm-row"
                        :class="{ dragging: drag.appId === app.id }"
                        draggable="true"
                        @dragstart="onDragStart(group, app, $event)"
                        @dragover.prevent.stop="onDragOver(group, app)"
                        @drop.prevent="onDragEnd"
                        @dragend="onDragEnd"
                        @contextmenu.prevent.stop="openAppMenu(app, 'row', $event)"
                      >
                        <button
                          class="sm-row-open"
                          type="button"
                          :title="app.title"
                          @click="launch(app, $event)"
                          @auxclick.middle.prevent="desktop.open(app.path, { newWindow: true })"
                        >
                          <span class="material-symbols-outlined sm-row-icon">{{ app.icon }}</span>
                          <span class="sm-row-title">{{ app.title }}</span>
                          <span v-if="badgeOf(app)" class="sm-row-badge" :class="{ alert: badgeOf(app) === '!' }">
                            {{ badgeOf(app) }}
                          </span>
                        </button>
                        <button
                          class="sm-row-star"
                          :class="{ on: isFavorite(app.id) }"
                          type="button"
                          :title="isFavorite(app.id) ? 'Убрать из избранного' : 'В избранное'"
                          @click="toggleFavorite(app.id)"
                        >
                          <span class="material-symbols-outlined">star</span>
                        </button>
                      </div>

                      <p v-if="!group.items.length" class="sm-group-empty">Перетащите сюда раздел</p>
                    </div>
                  </div>
                </div>
              </section>

              <button class="sm-add-group" type="button" @click="createGroup">
                <span class="material-symbols-outlined">add</span>
                Новая категория
              </button>
            </div>
          </Transition>
        </div>

        <!-- Без избранного переключать нечего: меню — один каталог. -->
        <div v-if="prefs.startFavorites" class="sm-switch">
          <AppButton
            v-if="showFavorites"
            variant="text"
            size="sm"
            label="Все разделы"
            trailing-icon="chevron_right"
            @click="screen = 'all'"
          />
          <AppButton
            v-else
            variant="text"
            size="sm"
            icon="chevron_left"
            label="Избранное"
            @click="screen = 'favorites'"
          />
        </div>

        <!-- Подвал: кто я, активная компания, настройки и выход — одной строкой. -->
        <footer class="sm-foot">
          <button
            class="sm-user"
            type="button"
            :title="shortFio(auth.user?.fio) || 'Аккаунт'"
            @click="launchPath('/settings?section=account')"
          >
            <img loading="lazy" decoding="async" class="sm-avatar" :src="avatarSrc" :alt="auth.user?.fio || 'Аккаунт'" />
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

      <ActivityPanel v-if="prefs.startActivity" class="sm-panel sm-activity" @open="launchPath" />
    </section>

    <ContextMenu
      :visible="appMenu.open"
      :x="appMenu.x"
      :y="appMenu.y"
      :items="appMenuItems"
      @select="onAppMenuSelect"
      @close="appMenu.open = false"
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
import { avatarUrl } from '@/utils/avatar.js'
import { shortFio } from '@/utils/people.js'
import { useScreenLock } from '@/composables/useScreenLock.js'
import { computed, defineAsyncComponent, nextTick, onMounted, reactive, ref } from 'vue'
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
import { DEFAULT_FAVORITES, menuGroups } from '@/desktop/apps.js'
import { tileFaces } from '@/desktop/liveTiles.js'
import { useLiveTilesStore } from '@/stores/liveTiles.js'
import { useUnitsStore } from '@/stores/units.js'
import BrandWordmark from '@/components/common/BrandWordmark.vue'
import CompanySelect from '@/components/common/CompanySelect.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import LiveTile from './LiveTile.vue'
import ActivityPanel from './ActivityPanel.vue'

// Планшетный вид нужен только по кнопке — его чанк не грузим вместе с меню.
const MobileStart = defineAsyncComponent(() => import('@/components/mobile/MobileStart.vue'))

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

const screen = ref('favorites')
// Избранное можно выключить в настройках — тогда меню целиком каталог.
const showFavorites = computed(() => prefs.startFavorites && screen.value === 'favorites')

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

const appById = computed(() => {
  const map = new Map()
  for (const g of groups.value) for (const a of g.items) map.set(a.id, a)
  return map
})

/* ── Избранное ── */
const favoriteIds = computed(() => prefs.favoritesList(PLATFORM, DEFAULT_FAVORITES))
const isFavorite = (id) => favoriteIds.value.includes(id)
const toggleFavorite = (id) => prefs.toggleFavorite(PLATFORM, id, DEFAULT_FAVORITES)

// Недоступные сейчас разделы (сменилась компания, выключена фича) не
// показываются, но из списка не выпадают — вернутся, когда снова доступны.
const favoriteApps = computed(() => favoriteIds.value.map((id) => appById.value.get(id)).filter(Boolean))

onMounted(() => {
  /* Сводки живых плиток рабочий стол тянет заранее и обновляет по таймеру —
     здесь лишь подстраховка: свежие данные запрос не повторяют (TTL стора).
     Живые плитки — только в избранном, у строк каталога сводок нет. */
  if (prefs.liveTiles && showFavorites.value) {
    const ids = favoriteApps.value.map((a) => a.id).filter((id) => prefs.isTileLive(id))
    live.refresh(ids).catch(() => {})
  }
})

const avatarSrc = computed(() => (auth.user ? avatarUrl(auth.user) : ''))

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

/* ── Перетаскивание ─────────────────────────────────────────────
   В избранном меняется порядок плиток; в каталоге — порядок внутри категории
   и принадлежность ей. Всё пишется в личные настройки, поэтому раскладка
   едет между устройствами. */
const drag = reactive({ appId: null, groupKey: null })

function startDrag(appId, e) {
  drag.appId = appId
  e.dataTransfer.effectAllowed = 'move'
  // Firefox не начинает перетаскивание без данных в буфере.
  e.dataTransfer.setData('text/plain', appId)
}

function onFavDragStart(app, e) {
  startDrag(app.id, e)
}

function onFavDragOver(app) {
  if (!drag.appId || app.id === drag.appId) return
  const ids = favoriteApps.value.map((a) => a.id)
  const from = ids.indexOf(drag.appId)
  const to = ids.indexOf(app.id)
  if (from < 0 || to < 0) return
  ids.splice(to, 0, ...ids.splice(from, 1))
  // Скрытые сейчас разделы остаются в списке — в его конце.
  prefs.setFavorites(PLATFORM, [...ids, ...favoriteIds.value.filter((id) => !ids.includes(id))])
}

function onDragStart(group, app, e) {
  startDrag(app.id, e)
  drag.groupKey = group.key
}

function onDragOver(group, app) {
  if (!drag.appId || !drag.groupKey || app.id === drag.appId) return
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

/** Перетаскивание на свободное место категории — раздел встаёт в конец. */
function onDragOverGroup(group) {
  if (!drag.appId || !drag.groupKey || group.key === drag.groupKey) return
  const ids = [...group.items.map((a) => a.id), drag.appId]
  prefs.moveTileToGroup(PLATFORM, drag.appId, group.key, ids)
  drag.groupKey = group.key
}

function onDragEnd() {
  drag.appId = null
  drag.groupKey = null
}

/* ── Категории каталога: создание, переименование, удаление ─── */
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
  const key = prefs.addGroup(PLATFORM, 'Новая категория')
  editing.value = key
  editLabel.value = 'Новая категория'
  focusEditor()
}

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

/* ── Контекстное меню раздела: плитка избранного или строка каталога ── */
const appMenu = reactive({ open: false, x: 0, y: 0, appId: null, kind: 'tile' })

const appMenuItems = computed(() => {
  const id = appMenu.appId
  if (!id) return []
  const items = []
  if (appMenu.kind === 'tile') {
    const size = prefs.tileSize(PLATFORM, id, appById.value.get(id)?.tile || 'square')
    items.push(
      { label: 'Широкая плитка', icon: size === 'wide' ? 'check' : 'width_wide', action: 'wide' },
      { label: 'Квадратная плитка', icon: size === 'square' ? 'check' : 'crop_square', action: 'square' },
      { divider: true },
      /* Сводки поимённо: общий тумблер живых плиток главнее — при выключенном
         пункт объясняет, почему плитка «мёртвая», а не молчит. */
      prefs.liveTiles
        ? { label: 'Живая плитка', icon: prefs.isTileLive(id) ? 'check' : 'dashboard', action: 'live' }
        : { label: 'Живые плитки выключены', icon: 'toggle_off', disabled: true },
      { divider: true },
    )
  }
  items.push(
    isFavorite(id)
      ? { label: 'Убрать из избранного', icon: 'star_border', action: 'unfavorite' }
      : { label: 'В избранное', icon: 'star', action: 'favorite' },
    prefs.isPinned(PLATFORM, id)
      ? { label: 'Открепить от панели задач', icon: 'keep_off', action: 'unpin' }
      : { label: 'Закрепить на панели задач', icon: 'keep', action: 'pin' },
    { label: 'Открыть ещё одно окно', icon: 'add', action: 'new' },
  )
  return items
})

function openAppMenu(app, kind, e) {
  appMenu.appId = app.id
  appMenu.kind = kind
  appMenu.x = e.clientX
  appMenu.y = e.clientY
  appMenu.open = true
}

function onAppMenuSelect(action) {
  const id = appMenu.appId
  if (!id) return
  if (action === 'wide' || action === 'square') prefs.setTileSize(PLATFORM, id, action)
  else if (action === 'live') {
    prefs.toggleTileLive(id)
    // Включили обратно — сводку этой плитки надо подтянуть: её не опрашивали.
    if (prefs.isTileLive(id)) live.refresh([id]).catch(() => {})
  }
  else if (action === 'favorite' || action === 'unfavorite') {
    toggleFavorite(id)
    if (action === 'favorite' && prefs.isTileLive(id)) live.refresh([id]).catch(() => {})
  }
  else if (action === 'pin') prefs.pin(PLATFORM, id)
  else if (action === 'unpin') prefs.unpin(PLATFORM, id)
  else if (action === 'new') {
    const app = appById.value.get(id)
    if (app) { desktop.open(app.path, { newWindow: true }); desktop.startOpen = false }
  }
}

/* ── Контекстное меню категории ─────────────────────────────── */
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
    { label: 'Новая категория', icon: 'add', action: 'create' },
  ]
  if (group.custom) {
    items.push({ divider: true })
    items.push({ label: 'Удалить категорию', icon: 'delete', action: 'remove', danger: true })
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
  // Разделы удалённой категории возвращаются в свои родные.
  else if (action === 'remove') prefs.removeGroup(PLATFORM, group.key)
}
</script>

<style scoped>
/* Панель задач сверху — меню выезжает вниз; по бокам — от своего края, а по
   вертикали центрируется: тянуться от кнопки «Пуск» через весь экран незачем. */
.sm-backdrop[data-taskbar='top'] .start-menu {
  top: calc(var(--taskbar-height) + 20px);
  bottom: auto;
  transform-origin: top center;
}

.sm-backdrop[data-taskbar='left'] .start-menu,
.sm-backdrop[data-taskbar='right'] .start-menu {
  top: 50%;
  bottom: auto;
  left: auto;
  transform: translateY(-50%);
  height: min(640px, calc(100dvh - 48px));
}

.sm-backdrop[data-taskbar='left'] .start-menu {
  left: calc(var(--taskbar-height) + 20px);
  transform-origin: center left;
}

.sm-backdrop[data-taskbar='right'] .start-menu {
  right: calc(var(--taskbar-height) + 20px);
  transform-origin: center right;
}

/* Прозрачная подложка на весь экран: клик мимо меню закрывает его, как в ОС. */
.sm-backdrop {
  position: fixed;
  inset: 0;
  z-index: 950;
}

/* Сам «Пуск» — только раскладка двух панелей: своей подложки у него нет.
   Высота постоянная, поэтому переключение экранов меню не дёргает. */
.start-menu {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  bottom: calc(var(--taskbar-height) + 20px);
  width: min(860px, calc(100vw - 24px));
  height: min(640px, calc(100dvh - var(--taskbar-height) - 40px));
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  gap: 12px;
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

/* Каждая панель — самостоятельное плавающее стекло: общей подложки, которая
   размывала бы фон за ними, больше нет. */
.start-menu .sm-panel {
  min-height: 0;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  box-shadow: var(--shadow-lg), var(--glass-edge);
}

.sm-main {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 12px 8px;
}

/* Без ленты меню — одна колонка и уже: каталогу и плиткам хватает. */
.start-menu.no-activity {
  grid-template-columns: minmax(0, 1fr);
  width: min(600px, calc(100vw - 24px));
}

/* Узкое окно — лента уступает место разделам. */
@media (max-width: 860px) {
  .start-menu { grid-template-columns: minmax(0, 1fr); }
  .sm-activity { display: none; }
}

/* ── Шапка: марка и название экрана ── */
.sm-head {
  display: flex;
  align-items: center;
  gap: 12px;
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
  font-size: 22px;
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

.sm-screen-title {
  margin-left: auto;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-dim);
}

.sm-full {
  width: 34px;
  min-width: 34px;
  max-width: 34px;
  height: 34px;
  min-height: 34px;
  max-height: 34px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
  padding: 0;
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.sm-full:hover { background: color-mix(in oklch, var(--color-primary) 10%, transparent); color: var(--color-text); }
.sm-full .material-symbols-outlined { font-size: 20px; }

/* Планшетный вид — на весь стол, без полей и скруглений. Панель задач спрятана,
   поэтому резерв под неё у полотна обнуляем. */
.sm-tablet {
  position: absolute;
  inset: 0;
  overflow: hidden;
  --taskbar-height: 0px;
  transform-origin: bottom center;
  transition: opacity 0.2s ease, scale 0.24s cubic-bezier(0.2, 0, 0, 1);
}

.sm-enter-from .sm-tablet,
.sm-leave-to .sm-tablet {
  opacity: 0;
  scale: 0.96;
}

/* ── Экраны ── */
.sm-screens {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.sm-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  /* Жёлоб под полосу прокрутки резервируется всегда — иначе она ложится
     поверх плиток, а при появлении дёргает всю сетку. Сама полоса — еле
     заметная: меню и так плотное, яркая линия сбоку только мешает. */
  scrollbar-gutter: stable;
  padding-right: 6px;
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

/* Каталог — «вглубь» (въезжает справа), избранное — «назад» (слева). */
.sm-fwd-enter-active, .sm-fwd-leave-active,
.sm-back-enter-active, .sm-back-leave-active {
  transition: opacity 0.14s ease, transform 0.18s cubic-bezier(0.2, 0, 0, 1);
}

.sm-fwd-enter-from, .sm-back-leave-to { opacity: 0; transform: translate3d(24px, 0, 0); }
.sm-fwd-leave-to, .sm-back-enter-from { opacity: 0; transform: translate3d(-24px, 0, 0); }

.sm-switch {
  display: flex;
  justify-content: flex-end;
  flex-shrink: 0;
}

.sm-empty {
  margin: auto;
  max-width: 340px;
  padding: 20px;
  text-align: center;
  font-size: 13.5px;
  line-height: 1.45;
  color: var(--color-text-dim);
}

/* ── Избранное: живые плитки ──
   Сетка 4 колонки: широкая плитка занимает две, квадратная — одну. */
.sm-tiles {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.sm-tile {
  position: relative;
  grid-column: span 1;
  /* Содержимое плитки рисует LiveTile — он растягивается на всю площадь. */
  display: flex;
  align-items: stretch;
  height: 104px;
  padding: 12px;
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
.sm-tile.dragging,
.sm-row.dragging { opacity: 0.45; }

.sm-tile:hover {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
  background: linear-gradient(color-mix(in oklch, var(--color-primary) 6%, transparent), color-mix(in oklch, var(--color-primary) 6%, transparent)), var(--glass-bg);
}

.sm-tile-badge,
.sm-row-badge {
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

.sm-tile-badge {
  position: absolute;
  top: 10px;
  right: 10px;
}

.sm-tile-badge.alert,
.sm-row-badge.alert {
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

/* ── Все разделы: категории компактных строк ── */
.sm-group-head {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 4px;
}

.sm-group-toggle {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  height: 28px;
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
  font-size: 13px;
  font-weight: 600;
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
  font-size: 13px;
}

.sm-group-count {
  margin-left: 4px;
  font-size: 12px;
  color: color-mix(in oklch, var(--color-text-dim) 70%, transparent);
}

.sm-group-more {
  width: 28px;
  min-width: 28px;
  max-width: 28px;
  height: 28px;
  min-height: 28px;
  max-height: 28px;
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

/* Сворачивание категории: 1fr → 0fr даёт плавную высоту без замеров JS. */
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

.sm-rows {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(180px, 100%), 1fr));
  gap: 2px 6px;
  min-height: 36px;
}

.sm-row {
  display: flex;
  align-items: center;
  min-width: 0;
  border-radius: var(--radius-md);
  transition: background 0.15s;
}

.sm-row:hover { background: color-mix(in oklch, var(--color-primary) 8%, transparent); }

.sm-row-open {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  height: 36px;
  padding: 0 4px 0 8px;
  border: none;
  background: transparent;
  color: var(--color-text);
  font-size: 13.5px;
  text-align: left;
  cursor: pointer;
}

.sm-row-icon {
  font-size: 20px;
  color: var(--color-primary);
}

.sm-row-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Звёздочка избранного: у отмеченных видна всегда, у остальных — под
   курсором, чтобы каталог не рябил пустыми контурами. */
.sm-row-star {
  width: 30px;
  min-width: 30px;
  max-width: 30px;
  height: 30px;
  min-height: 30px;
  max-height: 30px;
  display: grid;
  place-items: center;
  margin-right: 3px;
  padding: 0;
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.15s, color 0.15s, background 0.15s;
}

.sm-row-star .material-symbols-outlined { font-size: 18px; }
.sm-row:hover .sm-row-star,
.sm-row-star:focus-visible { opacity: 1; }
.sm-row-star:hover { background: color-mix(in oklch, var(--color-primary) 12%, transparent); }

.sm-row-star.on {
  opacity: 1;
  color: var(--color-primary);
}

.sm-row-star.on .material-symbols-outlined { font-variation-settings: 'FILL' 1; }

.sm-group-empty {
  grid-column: 1 / -1;
  margin: 0;
  padding: 10px;
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
  flex-shrink: 0;
  height: 36px;
  border: 1px dashed var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: transparent;
  color: var(--color-text-dim);
  font-size: 13px;
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

/* ── Подвал: низкая строка внутри панели, отбитая линией ── */
.sm-foot {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  margin: 0 -4px;
  padding: 8px 4px 0;
  border-top: 1px solid var(--acrylic-border);
}

/* Аватар — круглая кнопка входа в аккаунт (имя не дублируем: оно в подсказке). */
.sm-user {
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
  border: 1px solid var(--acrylic-border);
  border-radius: 50%;
  background: var(--glass-bg);
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

.sm-company {
  flex: 1;
  min-width: 0;
  display: flex;
}

/* Выбор компании — той же высоты, что и кнопки подвала. */
.sm-company :deep(.company-button) {
  height: 36px;
  min-height: 36px;
}

.sm-icon-btn {
  width: 36px;
  min-width: 36px;
  max-width: 36px;
  height: 36px;
  min-height: 36px;
  max-height: 36px;
  display: grid;
  place-items: center;
  padding: 0;
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.sm-icon-btn .material-symbols-outlined { font-size: 20px; }

.sm-icon-btn:hover {
  background: color-mix(in oklch, var(--color-primary) 10%, transparent);
  color: var(--color-primary);
}

.sm-icon-btn.danger { color: var(--color-error); }
.sm-icon-btn.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
</style>
