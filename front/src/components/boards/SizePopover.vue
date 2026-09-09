<script setup>
/* Попап величины у кнопки панели инструментов: тот же ползунок с полем, что и
   в свойствах (SizeField), плюс необязательные режимы — ластику нужно ещё
   выбрать, стирает он пиксели или объекты.

   Живёт в body и позиционируется экранными координатами кнопки — как
   контекстные меню: у окна рабочего стола свой transform, и телепорт в тело
   окна сместил бы попап относительно клика. */
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import SizeField from '@/components/boards/SizeField.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  anchor: { type: Object, default: () => ({ x: 0, y: 0 }) },
  value: { type: Number, default: 0 },
  min: { type: Number, default: 0 },
  max: { type: Number, default: 100 },
  unit: { type: String, default: 'px' },
  title: { type: String, default: 'Размер' },
  presets: { type: Array, default: () => [] },
  zeroLabel: { type: String, default: '' },
  // [{ key, label }] — необязательный переключатель режима над величиной.
  modes: { type: Array, default: () => [] },
  mode: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'update:value', 'update:mode'])

const root = ref(null)
const style = ref({ left: '0px', top: '0px' })

function place() {
  const width = 236
  const height = props.modes.length ? 190 : 150
  const x = Math.min(Math.max(8, props.anchor.x - width / 2), window.innerWidth - width - 8)
  // Панель инструментов стоит внизу, поэтому попап обычно раскрывается вверх.
  const above = props.anchor.y - height - 10
  const y = above > 8 ? above : Math.min(props.anchor.y + 10, window.innerHeight - height - 8)
  style.value = { left: `${Math.round(x)}px`, top: `${Math.round(y)}px`, width: `${width}px` }
}

function onOutside(e) {
  if (root.value && !root.value.contains(e.target)) emit('update:modelValue', false)
}

watch(() => props.modelValue, (open) => { if (open) place() })
watch(() => props.anchor, place, { deep: true })

onMounted(() => {
  document.addEventListener('pointerdown', onOutside, true)
  window.addEventListener('resize', place)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onOutside, true)
  window.removeEventListener('resize', place)
})
</script>

<template>
  <Teleport to="body">
    <div v-if="modelValue" ref="root" class="sp" :style="style" @pointerdown.stop>
      <header class="sp-head">
        <span class="sp-title">{{ title }}</span>
        <button
          type="button"
          class="sp-close"
          title="Закрыть"
          aria-label="Закрыть"
          @click="emit('update:modelValue', false)"
        >
          <span class="material-symbols-outlined">close</span>
        </button>
      </header>

      <div v-if="modes.length" class="sp-modes">
        <button
          v-for="m in modes"
          :key="m.key"
          type="button"
          class="sp-mode"
          :class="{ 'is-on': mode === m.key }"
          @click="emit('update:mode', m.key)"
        >{{ m.label }}</button>
      </div>

      <SizeField
        :model-value="value"
        :min="min"
        :max="max"
        :unit="unit"
        :presets="presets"
        :zero-label="zeroLabel"
        @update:model-value="(v) => emit('update:value', v)"
      />
    </div>
  </Teleport>
</template>

<style scoped>
.sp {
  position: fixed;
  z-index: 1200;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  /* Плотная подложка, как у панелей редактора: под попапом рисунок. */
  background: var(--color-surface);
  box-shadow: var(--shadow-3);
}

.sp-head { display: flex; align-items: center; gap: 6px; }
.sp-title { flex: 1; font-size: 13px; font-weight: 600; }

.sp-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  max-width: 24px;
  min-height: 24px;
  max-height: 24px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
}

.sp-close:hover { background: var(--color-surface-variant); }
.sp-close .material-symbols-outlined { font-size: 18px; }

.sp-modes { display: flex; gap: 4px; }

.sp-mode {
  flex: 1;
  height: 26px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text);
  font-size: 12px;
  cursor: pointer;
}

.sp-mode.is-on { border-color: var(--color-primary); background: var(--color-primary); color: var(--color-on-primary); }
</style>
