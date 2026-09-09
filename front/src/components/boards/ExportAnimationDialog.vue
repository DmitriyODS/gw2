<script setup>
/* Выгрузка анимации: формат, кадры цикла, частота, размер и прозрачный фон.

   Диалог нужен потому, что у ролика и у гифки разные ответы на одни и те же
   вопросы: GIF умеет прозрачность и зацикливание, но тяжелеет от каждого
   кадра, а MP4 наоборот — компактный, но всегда с фоном. Показывать оба набора
   настроек сразу значило бы врать про половину из них. */
import { computed, ref, watch } from 'vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppSwitch from '@/components/ui/AppSwitch.vue'
import Slider from 'primevue/slider'
import { FPS_RANGE } from '@/utils/boardScene.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  // Кадры доски: [{ id, name }] — по ним считается диапазон цикла.
  frames: { type: Array, default: () => [] },
  fps: { type: Number, default: FPS_RANGE.def },
  busy: { type: Boolean, default: false },
  progress: { type: Number, default: 0 },
})

const emit = defineEmits(['update:modelValue', 'export'])

const FORMATS = [
  { key: 'mp4', label: 'Видео MP4', hint: 'Компактный ролик для мессенджеров и презентаций' },
  { key: 'gif', label: 'Анимация GIF', hint: 'Зацикливается сама, умеет прозрачный фон' },
]

const format = ref('gif')
const from = ref(1)
const to = ref(1)
const rate = ref(props.fps)
const loop = ref(0)
const transparent = ref(false)
const size = ref(900)

const count = computed(() => props.frames.length)
const picked = computed(() => Math.max(0, to.value - from.value + 1))
const seconds = computed(() => (picked.value / Math.max(1, rate.value)).toFixed(1))
const isGif = computed(() => format.value === 'gif')
// Оценка веса гифки: кадр целиком, палитра 8 бит, LZW жмёт примерно вдвое.
const weight = computed(() => {
  if (!isGif.value) return ''
  const side = size.value
  const mb = (side * side * 0.5 * picked.value) / 1024 / 1024
  return mb > 0.1 ? `≈ ${mb.toFixed(1)} МБ` : ''
})

watch(() => props.modelValue, (open) => {
  if (!open) return
  from.value = 1
  to.value = Math.max(1, count.value)
  rate.value = props.fps
})

// Диапазон не должен «схлопываться»: правый край не уходит левее левого.
watch(from, (v) => { if (v > to.value) to.value = v })
watch(to, (v) => { if (v < from.value) from.value = v })

function start() {
  emit('export', {
    format: format.value,
    from: from.value - 1,
    to: to.value - 1,
    fps: rate.value,
    loop: isGif.value ? loop.value : 0,
    transparent: isGif.value && transparent.value,
    maxSide: size.value,
  })
}

function close() {
  emit('update:modelValue', false)
}
</script>

<template>
  <AppDialog
    :model-value="modelValue"
    title="Сохранить анимацию"
    :subtitle="`Кадров на доске: ${count}`"
    size="sm"
    :busy="busy"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: busy ? `Собираем… ${Math.round(progress * 100)}%` : 'Сохранить' },
    ]"
    @confirm="start"
    @cancel="close"
    @update:model-value="(v) => !v && close()"
  >
    <div class="ea">
      <div class="ea-formats">
        <button
          v-for="f in FORMATS"
          :key="f.key"
          type="button"
          class="ea-format"
          :class="{ 'is-on': format === f.key }"
          @click="format = f.key"
        >
          <span class="ea-format-name">{{ f.label }}</span>
          <span class="ea-format-hint">{{ f.hint }}</span>
        </button>
      </div>

      <div class="ea-field">
        <span class="ea-label">Зациклить кадры: с {{ from }} по {{ to }} — {{ picked }} шт., {{ seconds }} с</span>
        <Slider :model-value="from" :min="1" :max="Math.max(1, count)" @update:model-value="(v) => (from = v)" />
        <Slider :model-value="to" :min="1" :max="Math.max(1, count)" @update:model-value="(v) => (to = v)" />
      </div>

      <div class="ea-field">
        <span class="ea-label">Частота · {{ rate }} кадров/с</span>
        <Slider :model-value="rate" :min="FPS_RANGE.min" :max="30" @update:model-value="(v) => (rate = v)" />
      </div>

      <div class="ea-field">
        <span class="ea-label">Размер по большей стороне · {{ size }} px {{ weight }}</span>
        <Slider :model-value="size" :min="240" :max="isGif ? 1200 : 1920" :step="60" @update:model-value="(v) => (size = v)" />
      </div>

      <template v-if="isGif">
        <div class="ea-field">
          <span class="ea-label">Повторы · {{ loop === 0 ? 'бесконечно' : loop }}</span>
          <Slider :model-value="loop" :min="0" :max="20" @update:model-value="(v) => (loop = v)" />
        </div>
        <AppSwitch v-model="transparent" label="Прозрачный фон (без белого листа)" />
      </template>
      <p v-else class="ea-hint">
        Видео всегда идёт с фоном: прозрачность в MP4 не хранится. Нужен фон без листа —
        сохраняйте GIF.
      </p>
    </div>
  </AppDialog>
</template>

<style scoped>
.ea { display: flex; flex-direction: column; gap: 14px; }
.ea-formats { display: flex; gap: 8px; }

.ea-format {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 10px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
}

.ea-format.is-on { border-color: var(--color-primary); background: var(--color-primary-container); color: var(--color-on-primary-container); }
.ea-format-name { font-size: 13px; font-weight: 600; }
.ea-format-hint { font-size: 11px; opacity: 0.75; line-height: 1.3; }

.ea-field { display: flex; flex-direction: column; gap: 6px; }
.ea-label { font-size: 12px; color: var(--color-text-muted); }
.ea-hint { margin: 0; font-size: 11px; color: var(--color-text-muted); line-height: 1.4; }
</style>
