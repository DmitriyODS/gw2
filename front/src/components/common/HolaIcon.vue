<template>
  <!--
    Значок Hola-ассистента: кнопка панели задач, заголовок его окна, плитки.
    Рисуется currentColor — цвет задаёт место применения (приглушённый в покое,
    primary у активной кнопки), поэтому значок живёт вместе с темой.
    Исходник дизайнера — public/icons/hola_icon.svg (тот же контур).
  -->
  <svg
    class="hola-icon"
    :width="size"
    :height="size"
    viewBox="0 0 25 25"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    role="img"
    :aria-label="alt"
  >
    <circle cx="12.5" cy="12.5" r="11.25" stroke="currentColor" stroke-width="2.5" />
    <path d="M24.0732 12.5L24.0732 24" stroke="currentColor" stroke-width="2.5" />
    <circle cx="12.5732" cy="12.5" r="6.5" fill="currentColor" />
  </svg>
</template>

<script setup>
defineProps({
  size: { type: [Number, String], default: 24 },
  alt: { type: String, default: 'Hola' },
})
</script>

<style scoped>
.hola-icon { display: block; flex-shrink: 0; }
</style>
