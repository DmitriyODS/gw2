<script>
// Брендовый лоадер: круг логотипа, внутри которого дрейфуют три волны
// (как на самом логотипе). Замена стандартному спиннеру на экранах загрузки.
let uid = 0
</script>

<script setup>
defineProps({
  size: { type: Number, default: 88 },
})

const clipId = `bl-clip-${++uid}`
</script>

<template>
  <div
    class="brand-loader"
    :style="{ width: size + 'px', height: size + 'px' }"
    role="status"
    aria-label="Загрузка"
  >
    <svg viewBox="0 0 71 71" width="100%" height="100%">
      <defs>
        <clipPath :id="clipId">
          <circle cx="35.5" cy="35.5" r="35.5" />
        </clipPath>
      </defs>
      <circle cx="35.5" cy="35.5" r="35.5" class="bl-bg" />
      <g :clip-path="`url(#${clipId})`">
        <!-- Каждая волна — синус с периодом 71 и хвостом на второй период:
             сдвиг на -71px возвращает её в исходную фазу, петля бесшовна. -->
        <path
          class="bl-wave bl-wave-back"
          d="M0 30 Q17.75 22 35.5 30 T71 30 T106.5 30 T142 30 V71 H0 Z"
        />
        <path
          class="bl-wave bl-wave-mid"
          d="M0 38 Q17.75 29 35.5 38 T71 38 T106.5 38 T142 38 V71 H0 Z"
        />
        <path
          class="bl-wave bl-wave-front"
          d="M0 46 Q17.75 38 35.5 46 T71 46 T106.5 46 T142 46 V71 H0 Z"
        />
      </g>
    </svg>
  </div>
</template>

<style scoped>
.brand-loader {
  display: inline-block;
  flex: none;
}

.bl-bg {
  fill: var(--color-primary-container);
}

.bl-wave {
  will-change: transform;
  animation: bl-drift linear infinite;
}

.bl-wave-back {
  fill: color-mix(in oklch, var(--color-primary) 45%, var(--color-primary-container));
  animation-duration: 5.2s;
}

.bl-wave-mid {
  fill: color-mix(in oklch, var(--color-primary) 78%, var(--color-on-primary-container));
  animation-duration: 3.6s;
  animation-direction: reverse;
}

.bl-wave-front {
  fill: var(--color-primary);
  animation-duration: 2.6s;
}

@keyframes bl-drift {
  from { transform: translateX(0); }
  to   { transform: translateX(-71px); }
}

@media (prefers-reduced-motion: reduce) {
  .bl-wave { animation: none; }
}
</style>
