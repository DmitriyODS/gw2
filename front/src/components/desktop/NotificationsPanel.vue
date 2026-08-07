<template>
  <div class="np-backdrop" :class="{ full }" @pointerdown.self="emit('close')">
    <section class="np-panel" :class="{ full }" :style="anchorStyle" role="dialog" aria-label="Уведомления">
      <header class="np-head">
        <h3 class="np-title">Уведомления</h3>
        <button
          class="np-clear"
          type="button"
          title="Очистить все"
          :disabled="!items.length"
          @click="clearAll"
        >
          <span class="material-symbols-outlined">delete_sweep</span>
        </button>
      </header>

      <div class="np-body">
        <article
          v-for="item in items"
          :key="item.key"
          class="np-item"
          @click="go(item)"
        >
          <header class="np-item-head">
            <span class="np-item-icon material-symbols-outlined" :class="item.tone">{{ item.icon }}</span>
            <span class="np-item-title">{{ item.title }}</span>
            <button class="np-item-close" type="button" title="Убрать" @click.stop="dismiss(item)">
              <span class="material-symbols-outlined">close</span>
            </button>
          </header>
          <p class="np-item-text">{{ item.text }}</p>
        </article>

        <p v-if="!items.length" class="np-empty">Новых уведомлений нет</p>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useDesktopStore } from '@/stores/desktop.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { useDesktopNotifications } from '@/composables/useDesktopNotifications.js'

/* full — мобильный каркас: центр уведомлений разворачивается во весь экран
   между панелями статусов и задач (якориться к кнопке на телефоне некуда). */
const props = defineProps({
  full: { type: Boolean, default: false },
})

const emit = defineEmits(['close'])

const desktop = useDesktopStore()
const prefs = useDesktopPrefsStore()
// Список общий с бейджем на кнопке уведомлений — см. composable.
const { items, dismiss, clearAll } = useDesktopNotifications()

/* Панель раскрывается ровно из кнопки уведомлений: её центр совпадает с
   центром кнопки, у краёв экрана — прижимается с отступом. */
const PANEL_W = 420

const anchorStyle = computed(() => {
  // Во весь экран якорь не нужен — раскладку целиком держит CSS.
  if (props.full) return {}
  const width = Math.min(PANEL_W, window.innerWidth - 24)
  const bar = desktop.taskbarRect
  const gap = 24

  // Вертикальная панель задач: центр уведомлений выезжает вбок от неё и
  // держится по вертикали у кнопки, а не по центру экрана.
  if (prefs.taskbarVertical) {
    const style = { width: `${width}px`, bottom: 'auto', top: `${Math.max(12, Math.round(bar.y))}px` }
    if (prefs.taskbarSide === 'left') style.left = `${Math.round(bar.x + bar.w + gap)}px`
    else style.right = `${Math.round(window.innerWidth - bar.x + gap)}px`
    return style
  }

  const center = desktop.bellCenter || window.innerWidth - 60
  const left = Math.min(Math.max(12, center - width / 2), window.innerWidth - width - 12)
  const style = { left: `${Math.round(left)}px`, width: `${width}px` }
  // Панель сверху — центр уведомлений раскрывается вниз.
  if (prefs.taskbarSide === 'top') {
    style.top = `calc(var(--taskbar-height) + ${gap}px)`
    style.bottom = 'auto'
  }
  return style
})

function go(item) {
  desktop.open(item.path)
  emit('close')
}
</script>

<style scoped>
.np-backdrop {
  position: fixed;
  inset: 0;
  z-index: 950;
}

.np-panel {
  position: absolute;
  bottom: calc(var(--taskbar-height) + 24px);
  max-height: min(820px, calc(100dvh - var(--taskbar-height) - 48px));
  display: flex;
  flex-direction: column;
  padding: 18px;
  gap: 14px;
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
  /* Панель выезжает из кнопки уведомлений и въезжает в неё обратно. */
  transform-origin: bottom center;
  transition: opacity 0.18s ease, translate 0.22s cubic-bezier(0.2, 0, 0, 1),
    scale 0.22s cubic-bezier(0.2, 0, 0, 1);
}

/* Мобильный каркас: центр уведомлений во весь экран между панелями. Слой ниже
   панелей каркаса (900) — иначе повторное нажатие кнопки-колокольчика попадало
   бы в подложку, а не в саму кнопку, и панель открывалась заново. */
.np-backdrop.full { z-index: 890; }

.np-panel.full {
  top: calc(var(--statusbar-height, 0px) + env(safe-area-inset-top, 0px));
  bottom: calc(var(--taskbar-height) + env(safe-area-inset-bottom, 0px));
  left: 0;
  right: 0;
  max-height: none;
  border: none;
  border-radius: 0;
  box-shadow: none;
  /* Та же матовая подложка, что у панели ассистента во весь экран: под ней
     обои и плитки, поэтому стекло плотнее и размытие сильнее. */
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur-strong);
  backdrop-filter: var(--acrylic-blur-strong);
  /* Кнопка-колокольчик на телефоне сверху — панель и выезжает сверху. */
  transform-origin: top center;
}

.np-enter-from .np-panel,
.np-leave-to .np-panel {
  opacity: 0;
  translate: 0 22px;
  scale: 0.92;
}

.np-enter-from .np-panel.full,
.np-leave-to .np-panel.full { translate: 0 -22px; }

.np-head {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  padding: 0 2px;
}

.np-title { margin: 0; font-size: 20px; font-weight: 700; flex: 1; color: var(--color-text); }

.np-clear {
  width: 36px;
  min-width: 36px;
  max-width: 36px;
  height: 36px;
  min-height: 36px;
  max-height: 36px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  transition: background 0.15s, color 0.15s, opacity 0.15s;
}

.np-clear:disabled { opacity: 0.4; cursor: default; }
.np-clear:not(:disabled):hover { background: color-mix(in oklch, var(--color-primary) 12%, transparent); color: var(--color-primary); }
.np-clear .material-symbols-outlined { font-size: 22px; }

.np-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-right: 2px;
}

/* Карточка уведомления — стеклянная плашка, как плитки меню «Пуск». */
.np-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}

.np-item:hover {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
  background: color-mix(in oklch, var(--color-primary) 6%, var(--glass-bg));
}

.np-item-head { display: flex; align-items: center; gap: 10px; }

.np-item-icon { font-size: 22px; flex-shrink: 0; color: var(--color-text); }
.np-item-icon.alert { color: var(--color-error); }

.np-item-title {
  flex: 1;
  min-width: 0;
  font-size: 15px;
  font-weight: 500;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.np-item-close {
  width: 26px;
  min-width: 26px;
  max-width: 26px;
  height: 26px;
  min-height: 26px;
  max-height: 26px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.np-item-close:hover { background: color-mix(in oklch, var(--color-error) 14%, transparent); color: var(--color-error); }
.np-item-close .material-symbols-outlined { font-size: 18px; }

.np-item-text {
  margin: 0;
  font-size: 13.5px;
  color: var(--color-text-dim);
}

.np-empty {
  margin: 0;
  padding: 32px 16px;
  text-align: center;
  color: var(--color-text-dim);
  font-size: 13.5px;
}
</style>
