<template>
  <!-- Фирменный знак Яндекса (исходник — public/icons/yandex_logo.svg).
       Цвета бренда заданы значениями намеренно: чужая айдентика не следует
       нашей теме — то же исключение, что и у шаблонов писем. -->
  <svg
    class="brand-logo"
    :width="size"
    :height="size"
    viewBox="0 0 36 36"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    role="img"
    :aria-label="alt"
  >
    <path
      d="M3.06006 18C3.06006 9.7155 9.77406 3 18.0601 3C26.3431 3 33.0601 9.7155 33.0601 18C33.0601 26.2845 26.3431 33 18.0601 33C9.77406 33 3.06006 26.2845 3.06006 18Z"
      fill="#FC3F1D"
    />
    <path
      d="M19.9801 11.499H18.5941C16.0531 11.499 14.7166 12.786 14.7166 14.6835C14.7166 16.8285 15.6406 17.8335 17.5381 19.122L19.1056 20.178L14.6011 26.9085H11.2351L15.2776 20.8875C12.9526 19.221 11.6476 17.6025 11.6476 14.865C11.6476 11.433 14.0401 9.09 18.5776 9.09H23.0821V26.892H19.9801V11.499Z"
      fill="white"
    />
  </svg>
</template>

<script setup>
defineProps({
  size: { type: [Number, String], default: 20 },
  alt: { type: String, default: 'Яндекс' },
})
</script>

<style scoped>
.brand-logo { display: block; flex-shrink: 0; }
</style>
