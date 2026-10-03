<script setup>
/* Миниатюра оформления строго 16:9: рабочий стол с окнами и панелью задач
   либо переписка поверх фона. Сцена задана долями кадра, поэтому при любой
   ширине панели она масштабируется целиком, как уменьшенный снимок экрана, и
   не искажается.

   Для стола миниатюра честная: размытие и шаг узора уменьшаются в той же
   пропорции, что и сам экран, а пустые обои показывают то, что каркас рисует
   без них, — сияние фона приложения. */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import ChatBackgroundLayer from '@/components/common/ChatBackgroundLayer.vue'
import { useThemeStore } from '@/stores/theme.js'
import { isBlankRecipe } from '@/utils/chatBackgrounds.js'

const props = defineProps({
  // Нормализованный рецепт фона; null или пустой — фон по умолчанию.
  recipe: { type: Object, default: null },
  scene: { type: String, default: 'desktop' }, // desktop | chat
})

const theme = useThemeStore()
const frame = ref(null)
const width = ref(0)

let ro = null
onMounted(() => {
  ro = new ResizeObserver(([e]) => { width.value = e.contentRect.width })
  ro.observe(frame.value)
})
onBeforeUnmount(() => ro?.disconnect())

const blank = computed(() => isBlankRecipe(props.recipe))

const scale = computed(() => {
  if (props.scene !== 'desktop' || !width.value) return 1
  return Math.min(1, width.value / Math.max(window.innerWidth, 1))
})

/* Сияние фона приложения (main.css, [data-bg-gradient]) в долях кадра: там
   размер пятна задан в vmax, а у широкого экрана vmax — это ширина, поэтому
   по высоте 16:9 радиус в 16/9 раза больше. */
const glow = computed(() => {
  if (props.scene !== 'desktop' || !blank.value) return null
  if (!theme.bgGradient.enabled) return null
  return {
    background: [
      ...theme.bgGradient.blobs.map((b) =>
        `radial-gradient(${b.size}% ${(b.size * 16) / 9}% at ${b.x}% ${b.y}%, ` +
        `color-mix(in oklch, var(--color-${b.role}) calc(${b.alpha}% * var(--bg-grad-dim, 1)), transparent), transparent 64%)`),
      'var(--color-bg)',
    ].join(', '),
  }
})
</script>

<template>
  <div ref="frame" class="bp" :class="`is-${scene}`">
    <div v-if="glow" class="bp-glow" :style="glow" />
    <ChatBackgroundLayer v-else-if="!blank || scene === 'chat'" :recipe="recipe" :scale="scale" />

    <div v-if="scene === 'desktop'" class="bp-desk" aria-hidden="true">
      <div class="bp-win back">
        <span class="bp-win-bar"><i /><i /><i /></span>
        <span class="bp-line" />
        <span class="bp-line short" />
      </div>
      <div class="bp-win front">
        <span class="bp-win-bar"><i /><i /><i /></span>
        <span class="bp-line" />
        <span class="bp-line short" />
        <span class="bp-line mid" />
      </div>
      <div class="bp-taskbar">
        <span class="bp-tb start" />
        <span class="bp-tb btn" />
        <span class="bp-tb btn" />
        <span class="bp-tb btn" />
        <span class="bp-tb clock" />
      </div>
    </div>

    <div v-else class="bp-chat" aria-hidden="true">
      <span class="bp-bubble in" />
      <span class="bp-bubble out" />
      <span class="bp-bubble in short" />
      <span class="bp-bubble out short" />
    </div>
  </div>
</template>

<style scoped>
.bp {
  position: relative;
  isolation: isolate;
  width: 100%;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  /* Пустой фон стола без сияния — та же мягкая подсветка, что у каркаса. */
  background:
    radial-gradient(42% 75% at 6% -6%, color-mix(in oklch, var(--color-primary) 9%, transparent), transparent 62%),
    radial-gradient(38% 68% at 104% 28%, color-mix(in oklch, var(--color-tertiary) 7%, transparent), transparent 60%),
    radial-gradient(46% 82% at 38% 112%, color-mix(in oklch, var(--color-secondary) 6%, transparent), transparent 60%),
    var(--color-bg);
}

.bp-glow {
  position: absolute;
  inset: 0;
  z-index: -1;
}

/* ── Стол: все размеры — доли кадра ── */
.bp-desk {
  position: absolute;
  inset: 0;
}

.bp-win {
  position: absolute;
  display: flex;
  flex-direction: column;
  gap: 7%;
  padding: 2.4% 3%;
  border: 1px solid var(--acrylic-border);
  /* Проценты радиуса — от своих сторон, поэтому пара даёт ровный угол. */
  border-radius: 4% / 6.4%;
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--shadow-md), var(--glass-edge);
}

.bp-win.back { left: 7%; top: 8%; width: 44%; height: 50%; }
.bp-win.front { left: 35%; top: 24%; width: 52%; height: 54%; }

.bp-win-bar {
  display: flex;
  gap: 2%;
  height: 6%;
  margin-bottom: 4%;
}

.bp-win-bar i {
  height: 100%;
  aspect-ratio: 1;
  border-radius: 50%;
  background: var(--color-outline-dim);
}

.bp-line {
  height: 6%;
  border-radius: var(--radius-full);
  background: var(--color-surface-highest);
}

.bp-line.short { width: 55%; }
.bp-line.mid { width: 75%; }

.bp-taskbar {
  position: absolute;
  left: 50%;
  bottom: 3.5%;
  height: 8%;
  width: 36%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 3%;
  padding: 0 2.4%;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--shadow-sm), var(--glass-edge);
}

.bp-tb {
  height: 46%;
  border-radius: var(--radius-full);
  background: var(--color-surface-highest);
}

.bp-tb.start { aspect-ratio: 1; background: var(--color-primary); }
.bp-tb.btn { width: 12%; }
.bp-tb.clock { width: 16%; margin-left: auto; }

/* ── Переписка ── */
.bp-chat {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 5%;
  padding: 0 6%;
}

.bp-bubble {
  height: 13%;
  width: 46%;
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}

.bp-bubble.short { width: 30%; }

.bp-bubble.in {
  align-self: flex-start;
  background: var(--color-surface-high);
}

.bp-bubble.out {
  align-self: flex-end;
  background: var(--color-primary);
}
</style>
