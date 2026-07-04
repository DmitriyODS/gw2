<template>
  <span class="tv-count">{{ text }}</span>
</template>

<script setup>
// Анимированный счётчик: плавно «доезжает» до нового значения
// вместо мгновенного скачка (count-up при монтировании и смене value).
import { ref, computed, watch, inject, onMounted, onBeforeUnmount } from 'vue'
import { num, formatHoursShort } from './tvFormat.js'

const props = defineProps({
  value:  { type: [Number, String], default: 0 },
  format: { type: String, default: 'int' }, // 'int' | 'hours'
  prefix: { type: String, default: '' },
})

// Часы в рабочем дне приходят из настроек табло (provide в TvView).
const hoursPerDay = inject('tvHoursPerDay', null)

const display = ref(0)
let raf = null

function animateTo(target) {
  cancelAnimationFrame(raf)
  const start = performance.now()
  const startVal = display.value
  const duration = 900
  const step = (now) => {
    const t = Math.min(1, (now - start) / duration)
    const eased = 1 - Math.pow(1 - t, 3)
    display.value = startVal + (Number(target) - startVal) * eased
    if (t < 1) raf = requestAnimationFrame(step)
  }
  raf = requestAnimationFrame(step)
}

onMounted(() => animateTo(num(props.value)))
watch(() => num(props.value), v => animateTo(v))
onBeforeUnmount(() => cancelAnimationFrame(raf))

const text = computed(() => {
  const v = display.value
  const body = props.format === 'hours'
    ? formatHoursShort(v, num(hoursPerDay?.value) || 8)
    : String(Math.round(v))
  return props.prefix + body
})
</script>
