<template>
  <!-- В body: каркасы имеют свои стекинг-контексты и transform, а уведомление
       обязано лежать поверх окон, панелей и модалок. -->
  <Teleport to="body">
    <TransitionGroup
      v-if="visible"
      name="tst"
      tag="div"
      class="tst"
      :class="[isMobile ? 'mobile' : `at-${notifyPrefs.corner}`]"
    >
      <article
        v-for="t in notif.toasts"
        :key="t.id"
        class="tst-item"
        :class="[`is-${t.severity}`, { paused: t.id === hoverId }]"
        role="status"
        @pointerenter="hoverId = t.id"
        @pointerleave="hoverId = null"
      >
        <span class="tst-ic material-symbols-outlined">{{ ICONS[t.severity] || ICONS.info }}</span>

        <div class="tst-body">
          <p v-if="t.summary" class="tst-title">{{ t.summary }}</p>
          <p v-if="t.detail" class="tst-text">{{ t.detail }}</p>
        </div>

        <button type="button" class="tst-close" aria-label="Закрыть" @click="notif.dismiss(t.id)">
          <span class="material-symbols-outlined">close</span>
        </button>

        <!-- Полоска жизни: она же показывает, что под курсором отсчёт замер. -->
        <span
          v-if="t.life"
          class="tst-life"
          :style="{ animationDuration: `${t.life}ms` }"
          @animationend="notif.dismiss(t.id)"
        />
      </article>
    </TransitionGroup>
  </Teleport>
</template>

<script setup>
/* Всплывающие уведомления приложения: стопка карточек в верхнем углу (на
   телефоне — по центру под панелью статусов), новое встаёт первым, соседние
   съезжают вниз плавно.
 *
 * Свой компонент вместо PrimeVue Toast: нужен наш стеклянный стиль и, главное,
 * ПАУЗА ПОД КУРСОРОМ — прочитать длинный текст ошибки за четыре секунды
 * успевает не каждый. Отсчёт ведёт САМА полоска жизни (CSS-анимация,
 * `animation-play-state: paused` под курсором) — таймеров в JS нет, поэтому
 * пауза и возобновление ничего не рассинхронизируют. */
import { computed, ref } from 'vue'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useScreenLock } from '@/composables/useScreenLock.js'
import { notifyPrefs } from '@/utils/notifySettings.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const ICONS = {
  success: 'check_circle',
  error: 'error',
  warn: 'warning',
  info: 'info',
}

const notif = useNotificationsStore()
const { isMobile } = useBreakpoint()
const screenLock = useScreenLock()

/* На запертом экране содержимое уведомлений видно любому, кто подойдёт к
   столу, — поэтому по умолчанию стопка там не показывается. Карточки при этом
   копятся: сняв блокировку, человек увидит их обычным порядком. */
const visible = computed(() => notifyPrefs.onLockScreen || !screenLock.locked.value)

// Под курсором держим карточку открытой, пока указатель с неё не уйдёт.
const hoverId = ref(null)
</script>

<style scoped>
/* Поверх всего: выше плашки активного юнита, модалки питомца и диалогов над
   ней — уведомление не должно прятаться ни за чем. Угол выбирает человек
   («Настройки → Уведомления»); у нижних углов стопка растёт вверх, поэтому
   свежая карточка всегда ближе к краю экрана. */
.tst {
  position: fixed;
  z-index: 11500;
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: min(360px, calc(100vw - 32px));
  pointer-events: none;
}

/* Сверху по центру: стопка выравнивается по середине экрана, поэтому ей нужны
   оба края и авто-поля — фиксированной ширины с одним `right` тут мало. */
.tst.at-top-center {
  top: 16px;
  right: 0;
  left: 0;
  margin: 0 auto;
  align-items: center;
}

.tst.at-top-right { top: 16px; right: 16px; }
.tst.at-top-left { top: 16px; left: 16px; }

.tst.at-bottom-right {
  bottom: calc(var(--taskbar-height, 0px) + 16px);
  right: 16px;
  flex-direction: column-reverse;
}

.tst.at-bottom-left {
  bottom: calc(var(--taskbar-height, 0px) + 16px);
  left: 16px;
  flex-direction: column-reverse;
}

.tst.at-top-center .tst-enter-from,
.tst.at-top-center .tst-leave-to { transform: translateY(-16px) scale(0.98); }

.tst.at-top-center .tst-leave-active { position: absolute; right: auto; left: auto; }

.tst.at-top-left .tst-enter-from,
.tst.at-top-left .tst-leave-to,
.tst.at-bottom-left .tst-enter-from,
.tst.at-bottom-left .tst-leave-to { transform: translateX(-24px) scale(0.98); }

.tst.at-bottom-left .tst-leave-active,
.tst.at-top-left .tst-leave-active { right: auto; left: 0; }

/* На телефоне — по центру под панелью статусов каркаса (её толщина живёт на
   корне документа): у правого угла карточка налезала бы на вырез. */
.tst.mobile {
  top: calc(var(--statusbar-height, 0px) + env(safe-area-inset-top, 0px) + 10px);
  right: 12px;
  left: 12px;
  width: auto;
  align-items: center;
}

.tst.mobile .tst-item { width: min(420px, 100%); }

/* Лёгкое стекло: сильное размытие фона, чуть заметная подложка и светлый блик
   по верхней кромке — карточка «лежит на» интерфейсе, а не закрашивает его.
   Цветных полос и заливок нет: тон несёт только значок. */
.tst-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  overflow: hidden;
  padding: 12px 14px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--glass-bg), var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur-strong);
  backdrop-filter: var(--acrylic-blur-strong);
  box-shadow: var(--glass-edge), var(--shadow-lg);
  color: var(--color-text);
  pointer-events: auto;
}

.tst-item.is-success { --tone: var(--color-success); }
.tst-item.is-error { --tone: var(--color-error); }
.tst-item.is-warn { --tone: var(--color-warning); }
.tst-item.is-info { --tone: var(--color-primary); }

/* Значок — единственное цветное пятно карточки: мягкий кружок тона вместо
   полосы у кромки. */
.tst-ic {
  flex-shrink: 0;
  display: grid;
  place-items: center;
  width: 32px;
  min-width: 32px;
  height: 32px;
  min-height: 32px;
  border-radius: 50%;
  background: color-mix(in oklch, var(--tone, var(--color-primary)) 14%, transparent);
  color: var(--tone, var(--color-primary));
  font-size: 19px;
}

.tst-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.tst-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

/* Текст ошибки бывает длинным и может нести адрес или имя файла без пробелов. */
.tst-text {
  margin: 0;
  font-size: 13px;
  line-height: 1.45;
  color: var(--color-text-dim);
  overflow-wrap: anywhere;
}

.tst-close {
  flex-shrink: 0;
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
  border-radius: 50%;
  background: none;
  color: var(--color-text-muted);
  cursor: pointer;
}

.tst-close:hover { background: var(--color-surface-variant); color: var(--color-text); }
.tst-close .material-symbols-outlined { font-size: 16px; }

/* Полоска жизни — деликатная линия по нижней кромке: она же показывает, что
   под курсором отсчёт замер. Тон приглушён — карточка остаётся спокойной. */
.tst-life {
  position: absolute;
  left: 0;
  bottom: 0;
  height: 2px;
  background: color-mix(in oklch, var(--tone, var(--color-primary)) 55%, transparent);
  animation: tst-life linear forwards;
}

.tst-item.paused .tst-life { animation-play-state: paused; }

@keyframes tst-life {
  from { width: 100%; }
  to { width: 0; }
}

/* Появление — выезд с той стороны, где стоит стопка; уход — сжатие, чтобы
   соседи не прыгали. */
.tst-enter-from,
.tst-leave-to {
  opacity: 0;
  transform: translateX(24px) scale(0.98);
}

.tst-enter-active,
.tst-leave-active,
.tst-move { transition: opacity 0.2s ease, transform 0.24s cubic-bezier(0.2, 0, 0, 1); }

/* Уходящая карточка выпадает из потока — иначе стопка схлопывается рывком. */
.tst-leave-active { position: absolute; right: 0; width: 100%; }

.tst.mobile .tst-enter-from,
.tst.mobile .tst-leave-to { transform: translateY(-16px) scale(0.98); }

@media (prefers-reduced-motion: reduce) {
  .tst-enter-active,
  .tst-leave-active,
  .tst-move { transition: none; }
}
</style>
