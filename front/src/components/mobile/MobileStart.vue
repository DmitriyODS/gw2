<template>
  <!-- Стартовый экран сенсорного каркаса — то же меню «Пуск», что и на рабочем
       столе (те же разделы, марка над плитками, порядок и размеры плиток из
       личных настроек), но во весь экран.

       Телефон: одна колонка плиток, а компания, юнит и уведомления живут в
       общей панели статусов. Планшет: места хватает на всё меню целиком —
       слева колонка аккаунта (то, что на столе ютилось в подвале одной
       строкой) вместе с лентой активности, справа плитки. -->
  <div class="mstart" :class="{ tablet }" @contextmenu.self.prevent="openDeskMenu">
    <!-- ── Колонка аккаунта (планшет) ──
         На столе это подвал меню: аватар, компания и четыре значка в ряд. На
         большом сенсорном экране ряд мелких значков — плохая цель для пальца,
         поэтому здесь то же самое стоит колонкой с подписями. -->
    <aside v-if="tablet" class="mst-rail">
      <header class="mst-rail-head">
        <h2 class="mst-rail-title">Аккаунт</h2>
      </header>

      <button class="mst-rail-user" type="button" @click="open('/settings?section=account')">
        <img class="mst-rail-avatar" :src="avatarSrc" :alt="auth.user?.fio || 'Аккаунт'" />
        <span class="mst-rail-who">
          <span class="mst-rail-name">{{ shortFio(auth.user?.fio) || 'Аккаунт' }}</span>
          <span v-if="auth.user?.post" class="mst-rail-post">{{ auth.user.post }}</span>
        </span>
      </button>

      <div class="mst-rail-company"><CompanySelect /></div>

      <nav class="mst-rail-actions">
        <button class="mst-rail-btn" type="button" @click="open('/settings')">
          <span class="material-symbols-outlined">settings</span>
          <span>Настройки</span>
        </button>
        <button
          v-if="screenLock.enabled.value"
          class="mst-rail-btn"
          type="button"
          @click="lock"
        >
          <span class="material-symbols-outlined">lock</span>
          <span>Заблокировать</span>
        </button>
        <button class="mst-rail-btn danger" type="button" @click="logoutAsk = true">
          <span class="material-symbols-outlined">logout</span>
          <span>Выйти</span>
        </button>
      </nav>

      <!-- Лента последних действий — здесь же, под кнопками: своей колонки она
           не заслуживает, а колонка аккаунта без неё полупустая. -->
      <ActivityPanel class="mst-activity" @open="open" />
    </aside>

    <div class="mstart-body" @contextmenu.self.prevent="openDeskMenu">
      <!-- Марка — там же, где в меню «Пуск» рабочего стола: над плитками. -->
      <button class="mst-brand" type="button" title="О приложении" @click="open('/settings?section=about')">
        <BrandWordmark />
      </button>

      <section v-for="group in visibleGroups" :key="group.key" class="mst-group">
        <button class="mst-group-head" type="button" @click="prefs.toggleCollapsed(platform, group.key)">
          <span class="mst-group-label">{{ group.label }}</span>
          <span
            class="material-symbols-outlined mst-group-chev"
            :class="{ collapsed: prefs.isCollapsed(platform, group.key) }"
          >expand_more</span>
        </button>

        <div class="mst-group-body" :class="{ collapsed: prefs.isCollapsed(platform, group.key) }">
          <div class="mst-group-inner">
            <div class="mst-tiles">
              <button
                v-for="(app, i) in group.items"
                :key="app.id"
                class="mst-tile"
                :class="[`is-${sizeOf(app)}`, { pinned: prefs.isPinned(platform, app.id) }]"
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
                  dense
                />
                <span v-if="badgeOf(app)" class="mst-badge" :class="{ alert: badgeOf(app) === '!' }">
                  {{ badgeOf(app) }}
                </span>
                <span v-if="prefs.isPinned(platform, app.id)" class="mst-pin material-symbols-outlined">keep</span>
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
import { avatarUrl } from '@/utils/pets.js'
import { shortFio } from '@/utils/people.js'
import BrandWordmark from '@/components/common/BrandWordmark.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import CompanySelect from '@/components/common/CompanySelect.vue'
import LiveTile from '@/components/desktop/LiveTile.vue'
import ActivityPanel from '@/components/desktop/ActivityPanel.vue'

// Спрашиваем про выход редко — диалог (а с ним PrimeVue Dialog) грузим лениво.
const AppDialog = defineAsyncComponent(() => import('@/components/ui/AppDialog.vue'))

const props = defineProps({
  /* Раскладка «Пуска» у телефона и планшета своя (desktopPrefs держит обе
     отдельно от стола) — какая именно, знает каркас. В шаблоне пропс доступен
     по имени, в скрипте — через props. */
  platform: { type: String, default: 'mobile' },
  /* Планшет: плитку можно открыть во ВТОРУЮ зону — в меню появляется пункт. */
  split: { type: Boolean, default: false },
})

const tablet = computed(() => props.platform === 'tablet')

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
}, prefs.layout(props.platform)))

/* Пустой раздел показываем, только когда пользователь перекладывал плитки или
   завёл свои разделы: иначе на экране висели бы «мёртвые» заголовки. */
const visibleGroups = computed(() =>
  groups.value.filter((g) => g.items.length || g.custom || prefs.customized(props.platform)))

const appById = computed(() => {
  const map = new Map()
  for (const g of groups.value) for (const a of g.items) map.set(a.id, a)
  return map
})

// Раздел, индекс и общее число плиток в нём — для меню «переместить выше/ниже».
function tilePosition(appId) {
  for (const g of groups.value) {
    const index = g.items.findIndex((a) => a.id === appId)
    if (index !== -1) return { group: g, index, total: g.items.length }
  }
  return null
}

function moveTile(appId, delta) {
  const pos = tilePosition(appId)
  if (!pos) return
  const to = pos.index + delta
  if (to < 0 || to >= pos.total) return
  const ids = pos.group.items.map((a) => a.id)
  ;[ids[pos.index], ids[to]] = [ids[to], ids[pos.index]]
  prefs.moveTileToGroup(props.platform, appId, pos.group.key, ids)
}

const sizeOf = (app) => prefs.tileSize(props.platform, app.id, app.tile || 'square')

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

const avatarSrc = computed(() => (auth.user ? avatarUrl(auth.user) : ''))

function lock() {
  screenLock.lock()
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
  const pos = tilePosition(id)
  return [
    { label: 'Открыть', icon: 'open_in_new', action: 'open' },
    // Планшет: раздел можно сразу поставить рядом с текущим — второй зоной.
    ...(props.split ? [{ label: 'Открыть рядом', icon: 'splitscreen_right', action: 'side' }] : []),
    { divider: true },
    // Перетаскивания на тач нет (нативный HTML5 DnD, как на столе, тач не
    // понимает) — переставляем по шагу, как на клавиатуре: то же
    // moveTileToGroup, что и drag на рабочем столе.
    { label: 'Переместить выше', icon: 'arrow_upward', action: 'moveUp', disabled: !pos || pos.index === 0 },
    { label: 'Переместить ниже', icon: 'arrow_downward', action: 'moveDown', disabled: !pos || pos.index === pos.total - 1 },
    { divider: true },
    { label: 'Широкая плитка', icon: size === 'wide' ? 'check' : 'width_wide', action: 'wide' },
    { label: 'Квадратная плитка', icon: size === 'square' ? 'check' : 'crop_square', action: 'square' },
    { divider: true },
    prefs.liveTiles
      ? { label: 'Живая плитка', icon: prefs.isTileLive(id) ? 'check' : 'dashboard', action: 'live' }
      : { label: 'Живые плитки выключены', icon: 'toggle_off', disabled: true },
    { divider: true },
    prefs.isPinned(props.platform, id)
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
  if (action === 'side') return void desktop.openSide(appById.value.get(id).path)
  if (action === 'moveUp') return moveTile(id, -1)
  if (action === 'moveDown') return moveTile(id, 1)
  if (action === 'wide' || action === 'square') return prefs.setTileSize(props.platform, id, action)
  if (action === 'live') {
    prefs.toggleTileLive(id)
    // Вернули сводки — их надо подтянуть: выключенную плитку не опрашивали.
    if (prefs.isTileLive(id)) live.refresh([id]).catch(() => {})
    return
  }
  if (action === 'pin') return prefs.pin(props.platform, id)
  if (action === 'unpin') return prefs.unpin(props.platform, id)
}
</script>

<style scoped>
.mstart {
  position: absolute;
  inset: 0;
  /* Ширина боковой колонки — вместе с её полями (box-sizing: border-box). */
  --mst-rail: 300px;
  /* Плитки мельче и плотнее, чем на столе (там — фиксированные 4 колонки по
     112px): на телефоне это даёт свои 3 колонки вместо уменьшенной копии
     настольной сетки — раскладка, а не масштаб. Размер задаёт и ширину
     широкой (две колонки с зазором), и высоту ряда. */
  --mst-tile: 100px;
  --mst-tile-h: 90px;
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

/* ── Планшет: меню «Пуск» во весь экран в три колонки ──
   Слева аккаунт и выход (на столе это подвал одной строкой), по центру плитки,
   справа лента активности — та же, что в правой колонке «Пуска» на столе.
   Верхней панели на планшете нет, поэтому резерв сверху обычный. */
.mstart.tablet {
  display: grid;
  grid-template-columns: var(--mst-rail) minmax(0, 1fr);
  /* Ряд ровно в высоту экрана: без него он тянется по самой длинной колонке
     (плитки), обе панели уезжают под панель задач и обрезаются по-разному —
     а прокручиваться должна каждая колонка внутри себя. */
  grid-template-rows: minmax(0, 100%);
  align-items: stretch;
  /* Зазор между колонками дают их собственные поля — по обе стороны от тонкой
     линии-разделителя. Так у всех трёх колонок один и тот же отступ от кромки
     экрана, чего не выходило у отдельных панелей с их рамками и тенями. */
  --mst-pad: 22px;
  gap: 0;
  padding: var(--mst-pad) 0 0;
  padding-bottom: calc(var(--taskbar-height, 52px) + env(safe-area-inset-bottom, 0px) + var(--mst-pad));
  /* Плитки крупнее телефонных: до них дотягиваются пальцем с расстояния вытянутой
     руки, и колонок на большом экране всё равно помещается много. */
  --mst-tile: 124px;
  --mst-tile-h: 112px;
}

.mstart.tablet > .mst-rail,
.mstart.tablet > .mstart-body { min-height: 0; height: 100%; }

.mstart.tablet > .mstart-body {
  padding: 0 var(--mst-pad);
  gap: 20px;
}

/* Колонка — не панель, а часть одного экрана: ни рамки, ни фона, ни тени.
   Границу обозначает волосяная линия, отбитая полями с обеих сторон. */
.mstart.tablet > .mst-rail {
  border: none;
  border-radius: 0;
  background: none;
  box-shadow: none;
  padding: 0 var(--mst-pad);
  border-right: 1px solid var(--acrylic-border);
}

/* Лента внутри колонки — тоже без своей панели, отбита от кнопок такой же
   линией: колонка читается как один список, а не как две вложенные карточки.
   Селектор с родителем — чтобы перебить собственные стили ActivityPanel по
   специфичности, а не порядком подключения. */
.mst-rail > .mst-activity {
  flex: 1;
  min-height: 0;
  border: none;
  border-top: 1px solid var(--acrylic-border);
  border-radius: 0;
  background: none;
  box-shadow: none;
  padding: 12px 0 0;
  margin-top: 4px;
}

/* ── Колонка аккаунта ──
   Строки те же, что в ленте активности справа: две боковые колонки должны
   читаться как пара, а не как два разных решения. */
.mst-rail {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
  /* Прокручивается лента внутри, а не колонка целиком: аккаунт, компания и
     кнопки должны оставаться на месте. */
  overflow: hidden;
}

.mst-rail-head { display: flex; align-items: center; flex-shrink: 0; padding: 0 4px; }

.mst-rail-title {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.2px;
  color: var(--color-text);
}

.mst-rail-user {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
  transition: background 0.12s;
}

.mst-rail-user:hover { background: color-mix(in oklch, var(--color-primary) 8%, transparent); }

.mst-rail-avatar {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
}

.mst-rail-who { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.mst-rail-name { font-size: 14px; font-weight: 700; overflow-wrap: anywhere; }
.mst-rail-post { font-size: 12px; color: var(--color-text-dim); overflow-wrap: anywhere; }

/* Выбор компании занимает всю ширину колонки — это главный переключатель. */
.mst-rail-company :deep(.company-select),
.mst-rail-company :deep(button) { width: 100%; }

.mst-rail-actions { display: flex; flex-direction: column; gap: 4px; }

.mst-rail-btn {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text);
  font: inherit;
  font-size: 13.5px;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}

.mst-rail-btn .material-symbols-outlined { font-size: 20px; color: var(--color-text-dim); }
.mst-rail-btn:hover { background: color-mix(in oklch, var(--color-primary) 8%, transparent); }
.mst-rail-btn:hover .material-symbols-outlined { color: var(--color-primary); }

.mst-rail-btn.danger { color: var(--color-error); }
.mst-rail-btn.danger .material-symbols-outlined { color: var(--color-error); }
.mst-rail-btn.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.mst-rail-btn.danger:hover .material-symbols-outlined { color: inherit; }

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
  gap: 9px;
}

.mst-tile {
  position: relative;
  height: var(--mst-tile-h);
  min-height: var(--mst-tile-h);
  padding: 9px;
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
