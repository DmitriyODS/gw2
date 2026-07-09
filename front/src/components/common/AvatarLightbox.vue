<template>
  <Teleport to="body">
    <transition name="lb">
      <div v-if="modelValue" class="avatar-lb" @click.self="close">
        <button class="lb-close" @click="close" aria-label="Закрыть">
          <span class="material-symbols-outlined">close</span>
        </button>
        <div class="lb-stage">
          <img :src="src" :alt="alt || 'Фото'" class="lb-img" @click.stop />
        </div>
        <div v-if="caption" class="lb-caption">{{ caption }}</div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup>
import { watch, onBeforeUnmount } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, required: true },
  src: { type: String, required: true },
  alt: { type: String, default: '' },
  caption: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

function close() {
  emit('update:modelValue', false)
}

function onKey(e) {
  if (e.key === 'Escape' && props.modelValue) close()
}

watch(() => props.modelValue, (v) => {
  if (v) {
    document.addEventListener('keydown', onKey)
    document.body.style.overflow = 'hidden'
  } else {
    document.removeEventListener('keydown', onKey)
    document.body.style.overflow = ''
  }
}, { immediate: true })

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKey)
  document.body.style.overflow = ''
})
</script>

<style scoped>
.avatar-lb {
  position: fixed;
  inset: 0;
  z-index: 10100;
  background: var(--color-scrim, rgba(0, 0, 0, 0.85));
  display: grid;
  grid-template-rows: 1fr auto;
  place-items: center;
  padding: 24px;
  gap: 12px;
}

.lb-close {
  position: absolute;
  top: max(12px, env(safe-area-inset-top, 0px));
  right: max(12px, env(safe-area-inset-right, 0px));
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: none;
  display: grid;
  place-items: center;
  background: color-mix(in oklab, var(--color-surface) 60%, transparent);
  color: var(--color-on-surface);
  cursor: pointer;
  -webkit-backdrop-filter: blur(8px);
  backdrop-filter: blur(8px);
}
.lb-close .material-symbols-outlined { font-size: 24px; }
.lb-close:hover { background: var(--color-surface); }

.lb-stage {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  min-height: 0;
}
.lb-img {
  max-width: min(92vw, 1200px);
  max-height: 80dvh;
  object-fit: contain;
  border-radius: var(--radius-lg, 16px);
  box-shadow: var(--shadow-lg);
  background: var(--color-surface-container);
}
.lb-caption {
  color: var(--color-on-surface);
  font-size: 14px;
  text-align: center;
  background: color-mix(in oklab, var(--color-surface) 70%, transparent);
  padding: 6px 12px;
  border-radius: var(--radius-full, 999px);
  -webkit-backdrop-filter: blur(8px);
  backdrop-filter: blur(8px);
}

.lb-enter-active, .lb-leave-active { transition: opacity .18s; }
.lb-enter-from, .lb-leave-to { opacity: 0; }
.lb-enter-active .lb-img, .lb-leave-active .lb-img { transition: transform .22s cubic-bezier(0.34, 1.56, 0.64, 1); }
.lb-enter-from .lb-img, .lb-leave-to .lb-img { transform: scale(.92); }
</style>
