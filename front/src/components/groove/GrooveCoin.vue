<template>
  <svg class="groove-coin" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
    <circle cx="12" cy="12" r="10.5" fill="var(--groove-coin-rim)" />
    <circle cx="12" cy="12" r="8.6" fill="var(--groove-coin-face)" />
    <!-- Галочка закрытой задачи: грувы зарабатываются работой. -->
    <path
      d="M7.6 12.4l3.1 3.1 5.9-6.4"
      fill="none"
      stroke="var(--groove-coin-mark)"
      stroke-width="2.4"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
    <path
      d="M6.4 9.3a7.2 7.2 0 0 1 3.2-3.1"
      fill="none"
      stroke="var(--groove-coin-shine)"
      stroke-width="1.5"
      stroke-linecap="round"
    />
  </svg>
</template>

<style scoped>
/* Размер — как у эмодзи: масштабируется вместе с font-size родителя. */
.groove-coin {
  width: 1em;
  height: 1em;
  display: inline-block;
  vertical-align: -0.12em;
  flex-shrink: 0;
}
</style>
