<template>
  <!--
    Логотип Groove Work, реагирующий на текущую тему через CSS-переменные.
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
    <mask id="gw-logo-mask" style="mask-type:alpha" maskUnits="userSpaceOnUse" x="0" y="0" width="71" height="71">
      <circle cx="35.197" cy="35.197" r="35.197" fill="black" />
    </mask>
    <g mask="url(#gw-logo-mask)">
      <!-- Фоновый круг — мягкий тон primary -->
      <circle cx="35.197" cy="35.197" r="35.197" class="bg" />
      <!-- Дальняя волна (самая тёмная) — основной акцент -->
      <path
        class="wave-deep"
        d="M-17.0076 34.0765C-17.0076 34.0765 -9.94135 5.77754 7.42573 3.57601C24.7928 1.37451 33.6048 31.4667 50.3846 29.3396C67.1646 27.2125 79.0774 4.89627 79.0774 4.89627L88.45 78.8342L-9.20534 91.2133L-17.0076 34.0765Z"
      />
      <!-- Средняя волна — тёплый промежуточный тон -->
      <path
        class="wave-mid"
        d="M-20.3477 51.1056C-20.3477 51.1056 -10.8051 24.0658 10.3386 21.3856C31.4824 18.7053 41.1404 46.8986 61.5692 44.3089C81.9981 41.7193 97.2149 20.1787 97.2149 20.1787L106.05 89.872L-12.842 104.943L-20.3477 51.1056Z"
      />
      <!-- Ближняя волна — самый светлый, повторяет тон фона -->
      <path
        class="wave-soft"
        d="M-32.5352 57.1393C-32.5352 57.1393 -23.4531 30.8684 -3.21909 28.3035C17.0149 25.7385 26.3144 53.1681 45.8642 50.6899C65.4141 48.2117 79.9383 27.2983 79.9383 27.2983L88.5299 95.0747L-25.2463 109.497L-32.5352 57.1393Z"
      />
    </g>
  </svg>
</template>

<script setup>
defineProps({
  size: { type: [Number, String], default: 56 },
  alt: { type: String, default: 'Groove Work' },
})
</script>

<style scoped>
.gw-logo { display: block; }

.gw-logo .bg        { fill: var(--color-primary-container); }
.gw-logo .wave-deep { fill: var(--color-primary); }
.gw-logo .wave-mid  { fill: color-mix(in oklch, var(--color-primary) 55%, var(--color-tertiary-container)); }
.gw-logo .wave-soft { fill: var(--color-primary-container); }
.gw-logo .mark      { fill: var(--color-on-primary-container); opacity: 0.55; }
</style>
