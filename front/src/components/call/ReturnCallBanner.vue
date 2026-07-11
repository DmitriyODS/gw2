<template>
  <Teleport to="body">
    <Transition name="return-banner">
      <div v-if="call" class="return-banner" role="alertdialog" aria-label="Незавершённый звонок">
        <div class="rb-icon">
          <span class="material-symbols-outlined">{{ isVideo ? 'videocam' : 'call' }}</span>
        </div>
        <div class="rb-body">
          <div class="rb-title">Незавершённый {{ isVideo ? 'видеозвонок' : 'звонок' }}</div>
          <div class="rb-sub">Вы всё ещё в звонке — можно вернуться</div>
        </div>
        <div class="rb-actions">
          <button class="rb-btn rb-leave" title="Завершить" @click="callStore.dismissRejoin()">
            <span class="material-symbols-outlined">call_end</span>
            <span class="rb-btn-label">Завершить</span>
          </button>
          <button class="rb-btn rb-return" title="Вернуться к звонку" @click="callStore.confirmRejoin()">
            <span class="material-symbols-outlined">{{ isVideo ? 'video_call' : 'phone_in_talk' }}</span>
            <span class="rb-btn-label">Вернуться</span>
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed } from 'vue'
import { useCallStore } from '@/stores/call.js'

const callStore = useCallStore()
const call = computed(() => callStore.rejoinCall)
const isVideo = computed(() => (call.value?.media || 'video') === 'video')
</script>

<style scoped>
.return-banner {
  position: fixed;
  z-index: 11600;
  left: 50%;
  top: calc(16px + env(safe-area-inset-top, 0px));
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 12px;
  max-width: min(440px, calc(100vw - 24px));
  width: max-content;
  padding: 12px 14px;
  /* Плавающий слой — акрил с blur, как FAB и мини-панели. */
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl, 20px);
  box-shadow: var(--glass-edge), var(--shadow-lg);
}

.rb-icon {
  width: 42px;
  height: 42px;
  flex-shrink: 0;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  animation: rbPulse 1.8s ease-in-out infinite;
}

.rb-icon .material-symbols-outlined { font-size: 22px; }

@keyframes rbPulse {
  0%, 100% { box-shadow: 0 0 0 0 color-mix(in oklch, var(--color-primary) 45%, transparent); }
  50%      { box-shadow: 0 0 0 8px color-mix(in oklch, var(--color-primary) 0%, transparent); }
}

.rb-body { min-width: 0; }

.rb-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rb-sub {
  font-size: 12px;
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rb-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.rb-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 9px 14px;
  border: none;
  border-radius: var(--radius-full, 999px);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.12s, filter 0.15s;
}

.rb-btn:hover { transform: translateY(-1px); }
.rb-btn:active { transform: translateY(0); }
.rb-btn .material-symbols-outlined { font-size: 18px; }

.rb-return {
  background: var(--color-primary);
  color: var(--color-on-primary);
}

.rb-leave {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

/* На узких экранах прячем подписи кнопок — остаются круглые иконки. */
@media (max-width: 520px) {
  .return-banner { gap: 10px; padding: 10px 12px; }
  .rb-btn-label { display: none; }
  .rb-btn { padding: 10px; }
}

.return-banner-enter-active,
.return-banner-leave-active { transition: opacity 0.25s, transform 0.25s; }
.return-banner-enter-from,
.return-banner-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-12px);
}
</style>
