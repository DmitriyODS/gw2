<template>
  <canvas ref="canvasEl" class="qr-image" :style="{ width: size + 'px', height: size + 'px' }"></canvas>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import QRCode from 'qrcode'

const props = defineProps({
  value: { type: String, required: true },
  size: { type: Number, default: 240 },
})

const canvasEl = ref(null)

// QR обязан быть чёрно-белым с высоким контрастом — иначе камеры его не читают.
// Это осознанное исключение из цветовых токенов (как хардкод в почтовых
// шаблонах): цвета темы сломали бы сканируемость.
async function render() {
  if (!canvasEl.value || !props.value) return
  try {
    await QRCode.toCanvas(canvasEl.value, props.value, {
      width: props.size,
      margin: 1,
      errorCorrectionLevel: 'M',
      color: { dark: '#000000', light: '#ffffff' },
    })
  } catch {
    /* невалидное значение — молча ничего не рисуем */
  }
}

onMounted(render)
watch(() => [props.value, props.size], render)
</script>

<style scoped>
.qr-image {
  display: block;
  border-radius: var(--radius-md, 12px);
  background: #fff;
}
</style>
