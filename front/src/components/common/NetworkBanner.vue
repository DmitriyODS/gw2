<template>
  <Teleport to="body">
    <!-- appear: компонент монтируется вместе с первым показом (ленивый чанк). -->
    <Transition name="net-banner" mode="out-in" appear>
      <div
        v-if="view"
        :key="view.kind"
        class="net-banner"
        :class="view.kind"
        role="status"
        aria-live="polite"
      >
        <span class="material-symbols-outlined" :class="{ spin: view.kind === 'connecting' }">{{ view.icon }}</span>
        <span class="net-text">{{ view.text }}</span>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed } from 'vue'
import { useNetworkStore } from '@/stores/network.js'

const network = useNetworkStore()

const VIEWS = {
  offline: { kind: 'offline', icon: 'wifi_off', text: 'Нет подключения к интернету' },
  connecting: { kind: 'connecting', icon: 'progress_activity', text: 'Подключение…' },
  restored: { kind: 'restored', icon: 'wifi', text: 'Подключение восстановлено' },
}

const view = computed(() => {
  if (network.status !== 'online') return VIEWS[network.status]
  return network.restored ? VIEWS.restored : null
})
</script>

<style scoped>
/* Под панелью статусов телефона (её толщину каркас публикует на корне),
   на рабочем столе — у верхней кромки. Поверх окон, но ниже тостов. */
.net-banner {
  position: fixed;
  z-index: 11450;
  top: calc(var(--statusbar-height, 0px) + env(safe-area-inset-top, 0px) + 8px);
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: calc(100vw - 32px);
  padding: 8px 16px;
  border-radius: var(--radius-full);
  background: var(--color-surface-highest);
  color: var(--color-text);
  box-shadow: var(--shadow-lg);
  font-size: 13px;
  font-weight: 600;
  pointer-events: none;
}

.net-banner.offline {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

.net-banner.restored {
  background: var(--color-success-container);
  color: var(--color-on-success-container);
}

.net-banner .material-symbols-outlined { font-size: 18px; }

.net-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.spin { animation: netSpin 1.2s linear infinite; }

@keyframes netSpin {
  to { transform: rotate(360deg); }
}

.net-banner-enter-active,
.net-banner-leave-active { transition: opacity 0.2s, transform 0.2s; }
.net-banner-enter-from,
.net-banner-leave-to { opacity: 0; transform: translate(-50%, -8px); }

@media (prefers-reduced-motion: reduce) {
  .spin { animation: none; }
}
</style>
