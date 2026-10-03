<script setup>
// Брендовый лоадер: круг логотипа, внутри которого дрейфуют три волны
// (как на самом логотипе). Замена стандартному спиннеру на экранах загрузки.
import { computed, useId } from 'vue'

const props = defineProps({
  size: { type: Number, default: 88 },
  /* block — лоадер занимает всю ширину и встаёт по центру области загрузки.
     Сам по себе он inline-block и прижимается к левому верхнему углу, а ждать
     раздел приходится глядя именно на него: в разделах, панелях и диалогах
     нужен этот режим, а не обёртка-«центровка» в каждом файле. */
  block: { type: Boolean, default: false },
  /** Высота области ожидания в режиме block (0 — по содержимому). */
  minHeight: { type: Number, default: 240 },
})

const uid = useId()
const clipId = `bl-clip-${uid}`
const discId = `bl-disc-${uid}`

// В блочном режиме размер держит сам svg, а контейнер растягивается и центрирует.
const rootStyle = computed(() => (props.block
  ? { minHeight: props.minHeight ? `${props.minHeight}px` : undefined }
  : { width: `${props.size}px`, height: `${props.size}px` }))
</script>

<template>
  <div
    class="brand-loader"
    :class="{ 'is-block': block }"
    :style="rootStyle"
    role="status"
    aria-label="Загрузка"
  >
    <svg viewBox="0 0 71 71" :width="size" :height="size">
      <defs>
        <clipPath :id="clipId">
          <circle cx="35.5" cy="35.5" r="35.5" />
        </clipPath>
        <!-- Шайба логотипа: свет сверху слева, фаска по кромке. -->
        <radialGradient :id="discId" cx="0.32" cy="0.22" r="0.95">
          <stop offset="0" class="bl-st-hi" />
          <stop offset="0.55" class="bl-st" />
          <stop offset="1" class="bl-st-lo" />
        </radialGradient>
      </defs>
      <circle cx="35.5" cy="35.5" r="35.5" :fill="`url(#${discId})`" />
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
      <circle cx="35.5" cy="35.5" r="34.9" class="bl-rim" />
    </svg>
  </div>
</template>

<style scoped>
.brand-loader {
  display: inline-block;
  flex: none;
}

/* Блочный режим: ждать раздел приходится глядя на лоадер — он должен быть в
   центре области, а не в её углу. */
.brand-loader.is-block {
  display: grid;
  place-items: center;
  width: 100%;
  flex: 1;
}

/* Та же шайба, что у логотипа (BrandLogo): второй цвет темы, утемнённый. */
.bl-st-hi { stop-color: color-mix(in oklch, var(--color-secondary) 70%, white); }
.bl-st    { stop-color: color-mix(in oklch, var(--color-secondary) 80%, black); }
.bl-st-lo { stop-color: color-mix(in oklch, var(--color-secondary) 45%, black); }

.bl-rim {
  fill: none;
  stroke: color-mix(in oklch, white 30%, transparent);
  stroke-width: 1.2;
}

.bl-wave {
  will-change: transform;
  animation: bl-drift linear infinite;
}

.bl-wave-back {
  fill: color-mix(in oklch, var(--color-primary-container) 70%, white);
  animation-duration: 5.2s;
}

.bl-wave-mid {
  fill: color-mix(in oklch, var(--color-primary) 45%, var(--color-tertiary));
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
