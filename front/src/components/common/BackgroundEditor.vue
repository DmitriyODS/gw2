<script setup>
/* Общий редактор оформления фона (чаты мессенджера и лента портала): живой
   предпросмотр + своя картинка с размытием + градиент-пресеты + узор (SVG-фигуры
   или эмодзи). Правит переданный реактивный `recipe` НА МЕСТЕ (владелец —
   родитель, он же применяет/сбрасывает). Загрузку картинки делегирует `uploadFn`,
   чтобы каждый раздел грузил в своё хранилище. */
import { ref, computed } from 'vue'
import Slider from 'primevue/slider'
import ChatBackgroundLayer from '@/components/messenger/ChatBackgroundLayer.vue'
import EmojiPicker from '@/components/common/EmojiPicker.vue'
import { useNotificationsStore } from '@/stores/notifications.js'
import {
  GRADIENT_PRESETS, PATTERNS, PATTERN_ROLE, IMAGE_BLUR_MAX,
  gradientCss, patternDataUri, randomGradientBlobs, normalizeRecipe,
} from '@/utils/chatBackgrounds.js'

const props = defineProps({
  // Реактивный рабочий рецепт (мутируется на месте).
  recipe: { type: Object, required: true },
  // async (File) => { url }: загрузка картинки в хранилище раздела.
  uploadFn: { type: Function, required: true },
})

const notif = useNotificationsStore()
const uploading = ref(false)
const fileInput = ref(null)

const gradientPreset = computed(() => props.recipe.gradient.preset)
const previewRecipe = computed(() => normalizeRecipe(props.recipe))

function pickPreset(key) {
  props.recipe.gradient.preset = key
  props.recipe.gradient.blobs = null
}

function generate() {
  props.recipe.gradient.preset = 'custom'
  props.recipe.gradient.blobs = randomGradientBlobs()
}

function pickPattern(key) {
  const p = props.recipe.pattern
  p.key = key
  p.emoji = null // фигура и эмодзи взаимоисключимы
  if (key && (!p.alpha || p.alpha < 1)) p.alpha = 6
  if (p.alpha > 15) p.alpha = 15 // потолок фигуры-узора
  if (key && !p.size) p.size = 128
}

function pickEmoji(e) {
  const p = props.recipe.pattern
  p.emoji = e
  p.key = null
  if (!p.alpha || p.alpha < 8) p.alpha = 14 // цветной эмодзи заметнее
  if (!p.size) p.size = 128
}

function pickImageFile() { fileInput.value?.click() }

async function onImagePicked(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    notif.error('Нужен файл-картинка')
    return
  }
  uploading.value = true
  try {
    const res = await props.uploadFn(file)
    props.recipe.image = { url: res.url, blur: props.recipe.image?.blur ?? 12 }
  } catch (err) {
    notif.error(err?.message || 'Не удалось загрузить картинку')
  } finally {
    uploading.value = false
  }
}

function removeImage() { props.recipe.image = null }

// IMAGE_BLUR_MAX используется в шаблоне (импорт доступен из script setup).

function patternSwatchStyle(key) {
  const uri = patternDataUri(key)
  return {
    backgroundColor: `var(--color-${PATTERN_ROLE})`,
    maskImage: uri, WebkitMaskImage: uri,
    maskSize: '26px 26px', WebkitMaskSize: '26px 26px',
    maskRepeat: 'repeat', WebkitMaskRepeat: 'repeat',
    opacity: 0.55,
  }
}
</script>

<template>
  <div class="bg-editor">
    <!-- Живой предпросмотр -->
    <div class="cbg-preview">
      <ChatBackgroundLayer :recipe="previewRecipe" />
      <div class="cbg-bubbles">
        <div class="cbg-bubble in">Пример карточки</div>
        <div class="cbg-bubble out">А вот и фон 🎨</div>
        <div class="cbg-bubble in">Красота ✨</div>
      </div>
    </div>

    <!-- Своя картинка -->
    <div class="cbg-section">
      <div class="cbg-section-title">Своя картинка</div>
      <div class="cbg-image-row">
        <button type="button" class="btn-glass cbg-image-btn" :disabled="uploading" @click="pickImageFile">
          <span class="material-symbols-outlined">{{ uploading ? 'hourglass_top' : 'add_photo_alternate' }}</span>
          {{ uploading ? 'Загружаем…' : (recipe.image ? 'Заменить картинку' : 'Загрузить картинку') }}
        </button>
        <button v-if="recipe.image" type="button" class="btn-glass cbg-image-remove" @click="removeImage">
          <span class="material-symbols-outlined">delete</span>
          Убрать
        </button>
        <input ref="fileInput" type="file" accept="image/*" hidden @change="onImagePicked" />
      </div>
      <div v-if="recipe.image" class="cbg-slider">
        <label>Размытие</label>
        <Slider v-model="recipe.image.blur" :min="0" :max="IMAGE_BLUR_MAX" class="cbg-slider-ctl" />
        <span class="cbg-slider-val">{{ recipe.image.blur }}</span>
      </div>
    </div>

    <!-- Градиент -->
    <div class="cbg-section">
      <div class="cbg-section-title">Градиент</div>
      <div class="cbg-swatches">
        <button
          v-for="p in GRADIENT_PRESETS"
          :key="p.key"
          type="button"
          class="cbg-swatch"
          :class="{ active: gradientPreset === p.key }"
          :title="p.label"
          :style="{ backgroundImage: gradientCss(p.blobs) }"
          @click="pickPreset(p.key)"
        >
          <span v-if="!p.blobs.length" class="material-symbols-outlined cbg-swatch-ico">block</span>
        </button>
        <button
          type="button"
          class="cbg-swatch cbg-swatch-gen"
          :class="{ active: gradientPreset === 'custom' }"
          title="Свой (сгенерировать)"
          :style="gradientPreset === 'custom' ? { backgroundImage: gradientCss(recipe.gradient.blobs) } : null"
          @click="generate"
        >
          <span class="material-symbols-outlined cbg-swatch-ico">casino</span>
        </button>
      </div>
    </div>

    <!-- Узор -->
    <div class="cbg-section">
      <div class="cbg-section-title">Узор</div>
      <div class="cbg-swatches">
        <button
          v-for="p in PATTERNS"
          :key="p.key || 'none'"
          type="button"
          class="cbg-swatch cbg-swatch-pat"
          :class="{ active: recipe.pattern.key === p.key && !recipe.pattern.emoji }"
          :title="p.label"
          @click="pickPattern(p.key)"
        >
          <span v-if="p.key" class="cbg-pat-fill" :style="patternSwatchStyle(p.key)" />
          <span v-else class="material-symbols-outlined cbg-swatch-ico">block</span>
        </button>
        <div
          v-if="recipe.pattern.emoji"
          class="cbg-swatch cbg-swatch-pat active cbg-swatch-emoji"
          title="Эмодзи-узор"
        >{{ recipe.pattern.emoji }}</div>
        <div class="cbg-swatch cbg-swatch-pat cbg-swatch-pick" title="Эмодзи как узор">
          <EmojiPicker @pick="pickEmoji" />
        </div>
      </div>

      <template v-if="recipe.pattern.key || recipe.pattern.emoji">
        <div class="cbg-slider">
          <label>Насыщённость</label>
          <Slider v-model="recipe.pattern.alpha" :min="1" :max="recipe.pattern.emoji ? 30 : 15" class="cbg-slider-ctl" />
          <span class="cbg-slider-val">{{ recipe.pattern.alpha }}</span>
        </div>
        <div class="cbg-slider">
          <label>Размер</label>
          <Slider v-model="recipe.pattern.size" :min="64" :max="240" :step="8" class="cbg-slider-ctl" />
          <span class="cbg-slider-val">{{ recipe.pattern.size }}</span>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.cbg-preview {
  position: relative;
  isolation: isolate;
  height: 150px;
  border-radius: var(--radius-lg);
  overflow: hidden;
  border: 1px solid var(--color-outline-dim);
  margin-bottom: 16px;
}

.cbg-bubbles {
  position: relative;
  z-index: 1;
  height: 100%;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  justify-content: center;
}

.cbg-bubble {
  max-width: 72%;
  padding: 8px 12px;
  border-radius: 16px;
  font-size: 13px;
  line-height: 1.3;
  box-shadow: var(--shadow-sm);
}

.cbg-bubble.in {
  align-self: flex-start;
  background: var(--color-surface-high);
  color: var(--color-text);
  border-bottom-left-radius: 5px;
}

.cbg-bubble.out {
  align-self: flex-end;
  background: var(--color-primary);
  color: var(--color-on-primary);
  border-bottom-right-radius: 5px;
}

.cbg-section { margin-bottom: 16px; }

.cbg-section-title {
  font-size: 12.5px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--color-text-dim);
  margin-bottom: 8px;
}

.cbg-image-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.cbg-image-btn,
.cbg-image-remove {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.cbg-image-btn .material-symbols-outlined,
.cbg-image-remove .material-symbols-outlined { font-size: 20px; }

.cbg-swatches {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.cbg-swatch {
  width: 52px;
  min-width: 52px;
  max-width: 52px;
  height: 52px;
  min-height: 52px;
  max-height: 52px;
  border-radius: var(--radius-md);
  border: 2px solid var(--color-outline-dim);
  background-color: var(--color-bg);
  background-size: cover;
  cursor: pointer;
  display: grid;
  place-items: center;
  overflow: hidden;
  padding: 0;
  transition: border-color 0.15s, transform 0.12s;
}

.cbg-swatch:hover { transform: scale(1.06); }

.cbg-swatch.active {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px var(--color-primary) inset;
}

.cbg-swatch-ico {
  font-size: 22px;
  color: var(--color-text-dim);
}

.cbg-swatch-gen { background-color: var(--color-surface-high); }

.cbg-swatch-pat { position: relative; background-color: var(--color-surface-high); }

.cbg-swatch-emoji { font-size: 26px; line-height: 1; }

.cbg-swatch-pick { padding: 0; }
.cbg-swatch-pick :deep(.emoji-picker-wrap),
.cbg-swatch-pick :deep(.emoji-btn) {
  width: 100%;
  height: 100%;
  min-height: 0;
  border: none;
  background: transparent;
  border-radius: 0;
}
.cbg-swatch-pick :deep(.emoji-btn .material-symbols-outlined) {
  font-size: 24px;
  color: var(--color-text-dim);
}

.cbg-pat-fill {
  position: absolute;
  inset: 0;
}

.cbg-slider {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 12px;
}

.cbg-slider label {
  width: 108px;
  flex-shrink: 0;
  font-size: 13px;
  color: var(--color-text);
}

.cbg-slider-ctl { flex: 1; }

.cbg-slider-val {
  width: 34px;
  text-align: right;
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: var(--color-text-dim);
}

@media (max-width: 560px) {
  .cbg-slider label { width: 84px; }
  .cbg-swatch {
    width: 46px;
    min-width: 46px;
    max-width: 46px;
    height: 46px;
    min-height: 46px;
    max-height: 46px;
  }
}
</style>
