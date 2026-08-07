<template>
  <!-- v-show, а не v-if: свёрнутое окно остаётся смонтированным и сохраняет
       своё состояние (фильтры, черновики, прокрутку) — как в настольной ОС.
       Transition с appear даёт единую анимацию: окно улетает в панель задач и
       возвращается оттуда же. -->
  <Transition name="winfly" appear>
  <section
    v-show="!win.minimized"
    class="win"
    :class="{
      focused: isFocused,
      busy: dragging || resizing,
      snapped: win.mode !== 'normal',
      maximized: win.mode === 'max',
    }"
    :style="style"
    @pointerdown="desktop.focus(win.id)"
  >
    <header
      class="win-bar"
      @pointerdown="onBarDown"
      @dblclick="desktop.toggleMaximize(win.id)"
    >
      <button
        v-if="canBack"
        class="win-nav"
        type="button"
        title="Назад"
        @click="desktop.back(win.id)"
      >
        <span class="material-symbols-outlined">arrow_back</span>
      </button>
      <span class="win-icon material-symbols-outlined">{{ app?.icon || 'web_asset' }}</span>
      <h2 class="win-title">{{ title }}</h2>
      <div class="win-btns">
        <button class="win-btn" type="button" title="Свернуть" @click="desktop.minimize(win.id)">
          <span class="material-symbols-outlined">remove</span>
        </button>
        <button
          class="win-btn"
          type="button"
          :title="win.mode === 'normal' ? 'Развернуть' : 'Восстановить'"
          @click="desktop.toggleMaximize(win.id)"
        >
          <span class="material-symbols-outlined">
            {{ win.mode === 'normal' ? 'crop_square' : 'filter_none' }}
          </span>
        </button>
        <button class="win-btn danger" type="button" title="Закрыть" @click="desktop.close(win.id)">
          <span class="material-symbols-outlined">close</span>
        </button>
      </div>
    </header>

    <!-- Тело окна — оно же хост модалок раздела: диалоги телепортируются сюда,
         а не в body, поэтому накрывают своё окно, а не весь рабочий стол. -->
    <div ref="bodyEl" class="win-body main-content">
      <WindowContent :win="win" />
    </div>

    <div
      v-for="dir in RESIZE_DIRS"
      :key="dir"
      class="win-rz"
      :class="`rz-${dir}`"
      @pointerdown.stop="onResizeDown(dir, $event)"
    />
  </section>
  </Transition>
</template>

<script setup>
import { computed, ref } from 'vue'
import router from '@/router/index.js'
import { useDesktopStore } from '@/stores/desktop.js'
import { appById, windowTitle } from '@/desktop/apps.js'
import { snapZoneAt } from '@/desktop/geometry.js'
import { provideWindowHost } from '@/desktop/windowHost.js'
import WindowContent from './WindowContent.vue'

const props = defineProps({
  win: { type: Object, required: true },
})

const RESIZE_DIRS = ['n', 's', 'e', 'w', 'ne', 'nw', 'se', 'sw']
// Порог, после которого развёрнутое окно «отрывается» от края при перетаскивании.
const TEAR_OFF = 8

const desktop = useDesktopStore()

const app = computed(() => appById(props.win.appId))
const title = computed(() => windowTitle(app.value, router.resolve(props.win.path)))
const isFocused = computed(() => desktop.focusedId === props.win.id)
const canBack = computed(() => props.win.hi > 0)

const dragging = ref(false)
const resizing = ref(false)

const bodyEl = ref(null)
provideWindowHost(bodyEl)

/* --fly-* — вектор от центра окна к центру панели задач: по нему окно
   «улетает» при сворачивании и оттуда же появляется. */
const style = computed(() => {
  const bar = desktop.taskbarRect
  const cx = bar.w ? bar.x + bar.w / 2 : window.innerWidth / 2
  const cy = bar.h ? bar.y + bar.h / 2 : window.innerHeight
  return {
    transform: `translate3d(${props.win.x}px, ${props.win.y}px, 0)`,
    width: `${props.win.w}px`,
    height: `${props.win.h}px`,
    zIndex: props.win.z,
    '--fly-x': `${Math.round(cx - (props.win.x + props.win.w / 2))}px`,
    '--fly-y': `${Math.round(cy - (props.win.y + props.win.h / 2))}px`,
  }
})

/* ── Перетаскивание за заголовок ───────────────────────────────
   Развёрнутое/прижатое окно «отрывается» от края после порога и продолжает
   ехать за указателем, сохраняя относительную точку захвата.
   У края экрана подсвечивается зона прилипания — половина, четверть или
   полный экран; отпускание применяет её. */
function onBarDown(e) {
  if (e.button !== 0 || e.target.closest('button')) return
  desktop.focus(props.win.id)

  // Точка отсчёта переезжает в момент «отрыва» окна от края — иначе смещение
  // считалось бы дважды (от исходного нажатия и от новой позиции окна).
  let startX = e.clientX
  let startY = e.clientY
  const start = { x: props.win.x, y: props.win.y, w: props.win.w, h: props.win.h }
  const grabRatio = (e.clientX - start.x) / start.w
  let torn = props.win.mode === 'normal'
  let zone = null

  e.currentTarget.setPointerCapture(e.pointerId)
  dragging.value = true

  const onMove = (ev) => {
    const dx = ev.clientX - startX
    const dy = ev.clientY - startY

    if (!torn) {
      if (Math.abs(dx) < TEAR_OFF && Math.abs(dy) < TEAR_OFF) return
      torn = true
      // Восстанавливаем «нормальный» размер и подставляем окно под курсор,
      // сохраняя точку захвата заголовка.
      desktop.unmaximize(props.win.id)
      start.w = props.win.w
      start.h = props.win.h
      start.x = ev.clientX - props.win.w * grabRatio
      start.y = ev.clientY - 18
      startX = ev.clientX
      startY = ev.clientY
      desktop.setPosition(props.win.id, start.x, start.y)
      return
    }

    desktop.setPosition(props.win.id, start.x + dx, start.y + dy)
    zone = snapZoneAt(ev.clientX, ev.clientY, desktop.area)
    desktop.snapPreview = zone ? { zone, rect: desktop.zoneRect(zone) } : null
  }

  const onUp = () => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
    window.removeEventListener('pointercancel', onUp)
    dragging.value = false
    desktop.snapPreview = null
    if (zone) desktop.snapTo(props.win.id, zone)
    else desktop.setPosition(props.win.id, props.win.x, props.win.y, { commit: true })
  }

  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
  window.addEventListener('pointercancel', onUp)
}

/* ── Изменение размера за грани и углы ─────────────────────── */
function onResizeDown(dir, e) {
  if (e.button !== 0) return
  desktop.focus(props.win.id)

  const startX = e.clientX
  const startY = e.clientY
  const start = { x: props.win.x, y: props.win.y, w: props.win.w, h: props.win.h }
  const min = { w: app.value?.min?.[0] ?? 380, h: app.value?.min?.[1] ?? 280 }

  e.currentTarget.setPointerCapture(e.pointerId)
  resizing.value = true

  const onMove = (ev) => {
    const dx = ev.clientX - startX
    const dy = ev.clientY - startY
    const rect = { ...start }

    if (dir.includes('e')) rect.w = Math.max(min.w, start.w + dx)
    if (dir.includes('s')) rect.h = Math.max(min.h, start.h + dy)
    if (dir.includes('w')) {
      rect.w = Math.max(min.w, start.w - dx)
      rect.x = start.x + (start.w - rect.w)
    }
    if (dir.includes('n')) {
      rect.h = Math.max(min.h, start.h - dy)
      rect.y = start.y + (start.h - rect.h)
    }
    desktop.setRect(props.win.id, rect)
  }

  const onUp = () => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
    window.removeEventListener('pointercancel', onUp)
    resizing.value = false
    desktop.setRect(props.win.id, { x: props.win.x, y: props.win.y, w: props.win.w, h: props.win.h }, { commit: true })
  }

  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
  window.addEventListener('pointercancel', onUp)
}
</script>

<style scoped>
/* Окно — акриловая панель с собственной тенью. Позиция через transform
   (композит-слой: перетаскивание не вызывает layout), размер — width/height. */
.win {
  position: absolute;
  top: 0;
  left: 0;
  pointer-events: auto; /* слой окон сквозной, само окно — нет */
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: color-mix(in oklch, var(--color-surface) 88%, transparent);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  /* Тень окна нейтральная и мягкая: общие --shadow-* тонированы primary и на
     цветных обоях смотрятся подкрашенными. */
  box-shadow: 0 6px 24px color-mix(in oklch, var(--color-text) 7%, transparent);
  will-change: transform;
  transition: box-shadow 0.2s ease, border-color 0.2s ease;
}

/* Появление и сворачивание — полёт в панель задач и обратно. translate/scale —
   отдельные свойства, они складываются с transform-позицией окна. */
.winfly-enter-active,
.winfly-leave-active {
  transition: opacity 0.2s ease, translate 0.26s cubic-bezier(0.2, 0, 0, 1),
    scale 0.26s cubic-bezier(0.2, 0, 0, 1);
}

.winfly-enter-from,
.winfly-leave-to {
  opacity: 0;
  translate: var(--fly-x) var(--fly-y);
  scale: 0.28;
}

/* Плавное «доезжание» при прилипании и разворачивании — но не во время
   перетаскивания: там каждый кадр задаёт указатель. */
.win.snapped:not(.busy) {
  transition: transform 0.16s cubic-bezier(0.2, 0, 0, 1), width 0.16s cubic-bezier(0.2, 0, 0, 1),
    height 0.16s cubic-bezier(0.2, 0, 0, 1), box-shadow 0.2s ease;
}

.win.focused {
  border-color: color-mix(in oklch, var(--color-primary) 22%, var(--acrylic-border));
  box-shadow: 0 12px 40px color-mix(in oklch, var(--color-text) 11%, transparent);
}

/* Во время перетаскивания/ресайза содержимое не должно перехватывать
   указатель (iframe'ов нет, но скроллеры и селект текста мешают). */
.win.busy .win-body { pointer-events: none; }
.win.busy { user-select: none; }

/* ── Заголовок ── */
.win-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  flex-shrink: 0;
  padding: 0 8px 0 14px;
  cursor: grab;
  touch-action: none;
  border-bottom: 1px solid color-mix(in oklch, var(--acrylic-border) 60%, transparent);
  background: color-mix(in oklch, var(--color-surface-low) 45%, transparent);
}

.win.busy .win-bar { cursor: grabbing; }

.win-icon {
  font-size: 20px;
  color: var(--color-primary);
  flex-shrink: 0;
}

.win-title {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 14px;
  font-weight: 650;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.win-nav,
.win-btn {
  width: 32px;
  min-width: 32px;
  max-width: 32px;
  height: 32px;
  min-height: 32px;
  max-height: 32px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.win-nav .material-symbols-outlined,
.win-btn .material-symbols-outlined { font-size: 19px; }

.win-nav:hover,
.win-btn:hover {
  background: color-mix(in oklch, var(--color-primary) 12%, transparent);
  color: var(--color-primary);
}

.win-btn.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

.win-btns { display: flex; gap: 2px; flex-shrink: 0; }

/* ── Содержимое ──
   Класс main-content наследует поведение прежнего каркаса: разделы верстались
   под скроллящуюся колонку фиксированной высоты — здесь ровно она. */
.win-body {
  flex: 1;
  min-height: 0;
  position: relative;
  isolation: isolate;
}

/* ── Зоны изменения размера ── */
.win-rz { position: absolute; z-index: 2; }
.rz-n { top: -3px; left: 12px; right: 12px; height: 8px; cursor: ns-resize; }
.rz-s { bottom: -3px; left: 12px; right: 12px; height: 8px; cursor: ns-resize; }
.rz-w { left: -3px; top: 12px; bottom: 12px; width: 8px; cursor: ew-resize; }
.rz-e { right: -3px; top: 12px; bottom: 12px; width: 8px; cursor: ew-resize; }
.rz-nw { top: -3px; left: -3px; width: 16px; height: 16px; cursor: nwse-resize; }
.rz-se { bottom: -3px; right: -3px; width: 16px; height: 16px; cursor: nwse-resize; }
.rz-ne { top: -3px; right: -3px; width: 16px; height: 16px; cursor: nesw-resize; }
.rz-sw { bottom: -3px; left: -3px; width: 16px; height: 16px; cursor: nesw-resize; }

/* Развёрнутое во весь экран окно за грани не тянем (прижатую половину — можно:
   она просто перестаёт быть прижатой). */
.win.maximized .win-rz { display: none; }

/* Полный экран — как обычное одноэкранное приложение: без полей, скруглений,
   рамки и тени (панель задач в этот момент тоже спрятана). */
.win.maximized {
  border-radius: 0;
  border-color: transparent;
  box-shadow: none;
}
</style>
