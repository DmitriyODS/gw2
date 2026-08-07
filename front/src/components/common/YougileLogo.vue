<template>
  <!-- Фирменный знак YouGile (исходник — public/icons/youjile_logo.svg).
       Цвета бренда заданы значениями намеренно: чужая айдентика не следует
       нашей теме — то же исключение, что и у шаблонов писем.
       Идентификаторы градиентов уникальны на инстанс: одинаковые id в разных
       копиях знака перетирали бы друг друга. -->
  <svg
    class="brand-logo"
    :width="size"
    :height="Math.round((size * 29) / 31)"
    viewBox="0 0 31 29"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    role="img"
    :aria-label="alt"
  >
    <path
      d="M22.6911 0.0240479C17.3202 0.0240479 15.3966 3.283 16.0339 7.63672C16.7591 13.034 14.7591 14.4702 10.6998 15.7277C6.47652 17.0361 3.68208 23.7647 9.31102 27.6027C12.2377 29.6305 16.49 29.259 19.1751 26.8445C22.5059 23.8491 22.0554 19.4132 23.895 15.7277C25.7232 12.0654 29.5211 11.8726 30.0048 7.29335C30.4179 3.38359 26.29 0.0240479 22.6911 0.0240479Z"
      :fill="`url(#${gradA})`"
    />
    <path
      d="M3.78722 0.974108C-3.17689 3.37862 0.469205 13.8694 7.23095 12.3617C14.9379 10.6434 10.8703 -1.47147 3.78722 0.974108Z"
      :fill="`url(#${gradB})`"
    />
    <defs>
      <linearGradient :id="gradA" x1="33.9926" y1="-10.6961" x2="-0.621925" y2="28.711" gradientUnits="userSpaceOnUse">
        <stop stop-color="#E71A6E" />
        <stop offset="1" stop-color="#FFB313" />
      </linearGradient>
      <linearGradient :id="gradB" x1="3.05058" y1="15.6203" x2="9.21377" y2="-0.213177" gradientUnits="userSpaceOnUse">
        <stop stop-color="#5CF4E7" />
        <stop offset="1" stop-color="#2F97E6" />
      </linearGradient>
    </defs>
  </svg>
</template>

<script setup>
defineProps({
  size: { type: Number, default: 20 },
  alt: { type: String, default: 'YouGile' },
})

const uid = Math.random().toString(36).slice(2, 8)
const gradA = `yg-a-${uid}`
const gradB = `yg-b-${uid}`
</script>

<style scoped>
.brand-logo { display: block; flex-shrink: 0; }
</style>
