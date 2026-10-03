<template>
  <!--
    Логотип Groove Work, реагирующий на текущую тему через CSS-переменные.
    Смысл прежний — три волны «грува» в круге; материал — скеоморфный: круг —
    выпуклая матовая шайба второго цвета темы (во флагманской — графит) с
    бликом сверху и фаской по кромке, волны — акцент темы со светом сверху.
    Используется в шапке сайдбара, экранах входа, «О приложении», TV-режиме.
    Для favicon/PWA/системных уведомлений работает статический public/logo.svg.
  -->
  <svg
    class="gw-logo"
    :width="size"
    :height="size"
    viewBox="0 0 71 71"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    role="img"
    :aria-label="alt"
  >
    <defs>
      <mask :id="ids.mask" style="mask-type:alpha" maskUnits="userSpaceOnUse" x="0" y="0" width="71" height="71">
        <circle cx="35.197" cy="35.197" r="35.197" fill="black" />
      </mask>
      <!-- Шайба: свет падает сверху слева. -->
      <radialGradient :id="ids.disc" cx="0.32" cy="0.22" r="0.95">
        <stop offset="0" class="st-disc-hi" />
        <stop offset="0.55" class="st-disc" />
        <stop offset="1" class="st-disc-lo" />
      </radialGradient>
      <linearGradient :id="ids.deep" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" class="st-deep-hi" />
        <stop offset="0.5" class="st-deep" />
      </linearGradient>
      <linearGradient :id="ids.mid" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" class="st-mid-hi" />
        <stop offset="0.5" class="st-mid" />
      </linearGradient>
      <linearGradient :id="ids.soft" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" class="st-soft-hi" />
        <stop offset="0.5" class="st-soft" />
      </linearGradient>
      <!-- Фаска: светлая кромка сверху, тёмная снизу. -->
      <linearGradient :id="ids.rim" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" class="st-rim-hi" />
        <stop offset="1" class="st-rim-lo" />
      </linearGradient>
    </defs>

    <g :mask="`url(#${ids.mask})`">
      <circle cx="35.197" cy="35.197" r="35.197" :fill="`url(#${ids.disc})`" />
      <path
        :fill="`url(#${ids.deep})`"
        d="M-17.0076 34.0765C-17.0076 34.0765 -9.94135 5.77754 7.42573 3.57601C24.7928 1.37451 33.6048 31.4667 50.3846 29.3396C67.1646 27.2125 79.0774 4.89627 79.0774 4.89627L88.45 78.8342L-9.20534 91.2133L-17.0076 34.0765Z"
      />
      <path
        :fill="`url(#${ids.mid})`"
        d="M-20.3477 51.1056C-20.3477 51.1056 -10.8051 24.0658 10.3386 21.3856C31.4824 18.7053 41.1404 46.8986 61.5692 44.3089C81.9981 41.7193 97.2149 20.1787 97.2149 20.1787L106.05 89.872L-12.842 104.943L-20.3477 51.1056Z"
      />
      <path
        :fill="`url(#${ids.soft})`"
        d="M-32.5352 57.1393C-32.5352 57.1393 -23.4531 30.8684 -3.21909 28.3035C17.0149 25.7385 26.3144 53.1681 45.8642 50.6899C65.4141 48.2117 79.9383 27.2983 79.9383 27.2983L88.5299 95.0747L-25.2463 109.497L-32.5352 57.1393Z"
      />
      <!-- Тонкая тень гребня ближней волны — волны лежат слоями, а не в одной плоскости. -->
      <path
        class="crest"
        d="M-32.5352 57.1393C-32.5352 57.1393 -23.4531 30.8684 -3.21909 28.3035C17.0149 25.7385 26.3144 53.1681 45.8642 50.6899C65.4141 48.2117 79.9383 27.2983 79.9383 27.2983"
      />
    </g>
    <circle cx="35.197" cy="35.197" r="34.6" class="rim" :stroke="`url(#${ids.rim})`" />
  </svg>
</template>

<script setup>
import { useId } from 'vue'

defineProps({
  size: { type: [Number, String], default: 56 },
  alt: { type: String, default: 'Groove Work' },
})

// id маски и градиентов уникальны на экземпляр: при нескольких логотипах
// дубль ссылался бы на СКРЫТЫЙ первый экземпляр, и заливка пропадала.
const uid = useId()
const ids = {
  mask: `gw-logo-mask-${uid}`,
  disc: `gw-logo-disc-${uid}`,
  deep: `gw-logo-deep-${uid}`,
  mid: `gw-logo-mid-${uid}`,
  soft: `gw-logo-soft-${uid}`,
  rim: `gw-logo-rim-${uid}`,
}
</script>

<style scoped>
.gw-logo { display: block; }

/* Шайба — второй цвет темы, утемнённый: во флагманской это графит. */
.st-disc-hi { stop-color: color-mix(in oklch, var(--color-secondary) 70%, white); }
.st-disc    { stop-color: color-mix(in oklch, var(--color-secondary) 80%, black); }
.st-disc-lo { stop-color: color-mix(in oklch, var(--color-secondary) 45%, black); }

.st-deep-hi { stop-color: color-mix(in oklch, var(--color-primary) 80%, white); }
.st-deep    { stop-color: var(--color-primary); }
.st-mid-hi  { stop-color: color-mix(in oklch, var(--color-tertiary) 75%, white); }
.st-mid     { stop-color: color-mix(in oklch, var(--color-primary) 45%, var(--color-tertiary)); }
.st-soft-hi { stop-color: white; }
.st-soft    { stop-color: color-mix(in oklch, var(--color-primary-container) 70%, white); }

.st-rim-hi { stop-color: color-mix(in oklch, white 45%, transparent); }
.st-rim-lo { stop-color: color-mix(in oklch, black 30%, transparent); }

.rim { fill: none; stroke-width: 1.2; }
.crest { fill: none; stroke: color-mix(in oklch, black 18%, transparent); stroke-width: 0.8; }
</style>
