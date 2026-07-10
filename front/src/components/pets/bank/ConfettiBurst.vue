<template>
  <!-- Праздничный залп конфетти: полноэкранный, ни на что не кликается.
       Дофаминовый банкинг правильно празднует ЗДОРОВЫЕ события (цель копилки,
       погашение кредита, сбор собран) — не траты. -->
  <Teleport to="body">
    <div v-if="pieces.length" class="cb-layer" aria-hidden="true">
      <span
        v-for="p in pieces"
        :key="p.id"
        class="cb-piece"
        :style="p.style"
      >{{ p.emoji }}</span>
    </div>
  </Teleport>
</template>

<script setup>
import { onBeforeUnmount, ref } from 'vue'

const EMOJIS = ['🎉', '✨', '💛', '⭐', '🎊', '💫']
const LIFETIME_MS = 1800

const pieces = ref([])
let seq = 0
let clearTimer = null

// Императивный запуск: родитель зовёт через ref — залп самоочищается.
function burst(count = 26) {
  const fresh = Array.from({ length: count }, () => {
    seq += 1
    return {
      id: seq,
      emoji: EMOJIS[Math.floor(Math.random() * EMOJIS.length)],
      style: {
        left: `${8 + Math.random() * 84}%`,
        fontSize: `${13 + Math.random() * 14}px`,
        animationDelay: `${Math.random() * 0.35}s`,
        animationDuration: `${1 + Math.random() * 0.7}s`,
        '--cb-drift': `${(Math.random() - 0.5) * 160}px`,
        '--cb-spin': `${(Math.random() - 0.5) * 540}deg`,
      },
    }
  })
  pieces.value = [...pieces.value, ...fresh]
  clearTimeout(clearTimer)
  clearTimer = setTimeout(() => { pieces.value = [] }, LIFETIME_MS)
}

defineExpose({ burst })

onBeforeUnmount(() => clearTimeout(clearTimer))
</script>

<style scoped>
.cb-layer {
  position: fixed;
  inset: 0;
  z-index: 11000;
  pointer-events: none;
  overflow: hidden;
}
.cb-piece {
  position: absolute;
  top: -28px;
  line-height: 1;
  animation: cb-fall 1.3s ease-in forwards;
}
@keyframes cb-fall {
  0% { transform: translate(0, 0) rotate(0deg); opacity: 1; }
  100% { transform: translate(var(--cb-drift, 0), 105vh) rotate(var(--cb-spin, 360deg)); opacity: 0.9; }
}
@media (prefers-reduced-motion: reduce) {
  .cb-piece { animation-duration: 0.6s; }
}
</style>
