<script setup>
/* Общий редактор фона: обои рабочего стола и экрана блокировки, фон чатов и
   ленты портала. Миниатюра 16:9 всегда на виду (`PreviewLayout`).

   Рецептом владеет родитель и применяет его СРАЗУ: каждая правка приходит к
   нему новым объектом через `update:recipe`. Основа фона одна — готовые обои,
   своя картинка, цвет или градиент, — узор ложится поверх любой. Сохранённые
   раньше сочетания слоёв рисуются как прежде и сводятся к одной основе при
   первой правке. Загрузку картинки делегирует `uploadFn`. */
import { ref, computed, watch } from 'vue'
import Slider from 'primevue/slider'
import AppTabs from '@/components/ui/AppTabs.vue'
import BackgroundPreview from '@/components/common/BackgroundPreview.vue'
import PreviewLayout from '@/components/common/PreviewLayout.vue'
import EmojiPicker from '@/components/common/EmojiPicker.vue'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useThemeStore } from '@/stores/theme.js'
import {
  GRADIENT_PRESETS, PATTERNS, PATTERN_ROLE, IMAGE_BLUR_MAX, SOLID_ROLES,
  gradientCss, patternDataUri, randomGradientBlobs, normalizeRecipe, solidCss,
} from '@/utils/chatBackgrounds.js'

const props = defineProps({
  recipe: { type: Object, required: true },
  // async (File) => { url }: загрузка картинки в хранилище раздела.
  uploadFn: { type: Function, required: true },
  // Сцена миниатюры: рабочий стол или переписка.
  preview: { type: String, default: 'chat' }, // chat | desktop
  // Готовые картинки: [{ key, label, light, dark }]. Пусто — вкладки нет.
  presets: { type: Array, default: () => [] },
  // Недавно загруженные картинки (адреса). Пусто — ряда нет.
  recent: { type: Array, default: () => [] },
})

const emit = defineEmits(['update:recipe', 'uploaded', 'forget-recent'])

const notif = useNotificationsStore()
const theme = useThemeStore()
const uploading = ref(false)
const fileInput = ref(null)

const previewRecipe = computed(() => normalizeRecipe(props.recipe))

function update(patch) {
  emit('update:recipe', { ...props.recipe, ...patch })
}

const PLAIN = { preset: 'plain', blobs: null }

/* ── Основа ── */
const BASES = computed(() => [
  ...(props.presets.length ? [{ value: 'preset', label: 'Готовые' }] : []),
  { value: 'image', label: 'Своя картинка' },
  { value: 'solid', label: 'Цвет' },
  { value: 'gradient', label: 'Градиент' },
])

function baseOf(r) {
  if (r.image?.key && props.presets.length) return 'preset'
  if (r.image?.url) return 'image'
  if (r.solid?.role) return 'solid'
  return 'gradient'
}

/* Вкладка основы следует за рецептом, но сама его не меняет: открыть «Цвет»
   и передумать — фон остаётся прежним, пока не выбран вариант. */
const base = ref(baseOf(props.recipe))
watch(() => baseOf(props.recipe), (b) => { base.value = b })

function setImage(image) {
  update({ image, solid: null, gradient: { ...PLAIN } })
}

function pickWallpaper(w) {
  setImage({ key: w.key, url: w.light, dark: w.dark, blur: props.recipe.image?.blur ?? 0 })
}

function pickRecent(url) {
  setImage({ url, blur: props.recipe.image?.blur ?? 0 })
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
    setImage({ url: res.url, blur: 0 })
    emit('uploaded', res.url)
  } catch (err) {
    notif.error(err?.message || 'Не удалось загрузить картинку')
  } finally {
    uploading.value = false
  }
}

function pickSolid(role) {
  update({
    solid: { role, amount: props.recipe.solid?.amount ?? 40 },
    image: null,
    gradient: { ...PLAIN },
  })
}

function pickGradient(key) {
  update({ gradient: { preset: key, blobs: null }, image: null, solid: null })
}

function generateGradient() {
  update({ gradient: { preset: 'custom', blobs: randomGradientBlobs() }, image: null, solid: null })
}

const setBlur = (blur) => update({ image: { ...props.recipe.image, blur } })
const setSolidAmount = (amount) => update({ solid: { ...props.recipe.solid, amount } })

/* ── Узор ── */
const setPattern = (patch) => update({ pattern: { ...props.recipe.pattern, ...patch } })

function pickPattern(key) {
  const p = { ...props.recipe.pattern, key, emoji: null } // фигура и эмодзи взаимоисключимы
  if (key && (!p.alpha || p.alpha < 1)) p.alpha = 6
  if (p.alpha > 15) p.alpha = 15 // потолок фигуры-узора
  if (key && !p.size) p.size = 128
  update({ pattern: p })
}

function pickEmoji(e) {
  const p = { ...props.recipe.pattern, emoji: e, key: null }
  if (!p.alpha || p.alpha < 8) p.alpha = 14 // цветной эмодзи заметнее
  if (!p.size) p.size = 128
  update({ pattern: p })
}

const hasPattern = computed(() => !!(props.recipe.pattern?.key || props.recipe.pattern?.emoji))

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
  <PreviewLayout>
    <template #preview>
      <BackgroundPreview :recipe="previewRecipe" :scene="preview" />
    </template>
    <template v-if="$slots.actions" #actions><slot name="actions" /></template>

    <!-- Основа -->
    <section class="bge-section">
      <h4 class="bge-title">Основа</h4>
      <AppTabs v-model="base" variant="tint" dense :tabs="BASES" class="bge-bases" />

      <div v-if="base === 'preset'" class="bge-papers">
        <button
          v-for="w in presets"
          :key="w.key"
          type="button"
          class="bge-paper"
          :class="{ active: recipe.image?.key === w.key }"
          :title="w.label"
          @click="pickWallpaper(w)"
        >
          <img loading="lazy" decoding="async" :src="theme.dark ? w.dark : w.light" alt="" />
          <span class="bge-paper-label">{{ w.label }}</span>
        </button>
      </div>

      <template v-else-if="base === 'image'">
        <div class="bge-papers">
          <button type="button" class="bge-paper bge-upload" :disabled="uploading" @click="pickImageFile">
            <span class="material-symbols-outlined">{{ uploading ? 'hourglass_top' : 'add_photo_alternate' }}</span>
            <span>{{ uploading ? 'Загружаем…' : 'Загрузить' }}</span>
          </button>
          <div
            v-for="url in recent"
            :key="url"
            class="bge-paper bge-recent"
            :class="{ active: !recipe.image?.key && recipe.image?.url === url }"
          >
            <button type="button" class="bge-recent-pick" title="Поставить эту картинку" @click="pickRecent(url)">
              <img loading="lazy" decoding="async" :src="url" alt="" />
            </button>
            <button type="button" class="bge-recent-remove" title="Убрать из недавних" @click="emit('forget-recent', url)">
              <span class="material-symbols-outlined">close</span>
            </button>
          </div>
        </div>
        <input ref="fileInput" type="file" accept="image/*" hidden @change="onImagePicked" />
      </template>

      <template v-else-if="base === 'solid'">
        <div class="bge-swatches">
          <button
            v-for="r in SOLID_ROLES"
            :key="r.key"
            type="button"
            class="bge-swatch"
            :class="{ active: recipe.solid?.role === r.key }"
            :title="r.label"
            :style="{ backgroundColor: solidCss({ role: r.key, amount: recipe.solid?.amount ?? 40 }) }"
            @click="pickSolid(r.key)"
          />
        </div>
        <div v-if="recipe.solid" class="bge-slider">
          <label>Насыщенность</label>
          <Slider :model-value="recipe.solid.amount" :min="0" :max="100" class="bge-slider-ctl" @update:model-value="setSolidAmount" />
          <span class="bge-slider-val">{{ recipe.solid.amount }}</span>
        </div>
      </template>

      <div v-else class="bge-swatches">
        <button
          v-for="p in GRADIENT_PRESETS"
          :key="p.key"
          type="button"
          class="bge-swatch"
          :class="{ active: !recipe.image && !recipe.solid && recipe.gradient.preset === p.key }"
          :title="p.label"
          :style="{ backgroundImage: gradientCss(p.blobs) }"
          @click="pickGradient(p.key)"
        >
          <span v-if="!p.blobs.length" class="material-symbols-outlined bge-swatch-ico">block</span>
        </button>
        <button
          type="button"
          class="bge-swatch bge-swatch-alt"
          :class="{ active: !recipe.image && !recipe.solid && recipe.gradient.preset === 'custom' }"
          title="Случайная композиция"
          :style="recipe.gradient.preset === 'custom' ? { backgroundImage: gradientCss(recipe.gradient.blobs) } : null"
          @click="generateGradient"
        >
          <span class="material-symbols-outlined bge-swatch-ico">casino</span>
        </button>
      </div>

      <!-- Размытие — свойство картинки, какой бы она ни была. -->
      <div v-if="(base === 'preset' || base === 'image') && recipe.image" class="bge-slider">
        <label>Размытие</label>
        <Slider :model-value="recipe.image.blur" :min="0" :max="IMAGE_BLUR_MAX" class="bge-slider-ctl" @update:model-value="setBlur" />
        <span class="bge-slider-val">{{ recipe.image.blur }}</span>
      </div>
    </section>

    <!-- Узор поверх основы -->
    <section class="bge-section">
      <h4 class="bge-title">Узор поверх</h4>
      <div class="bge-swatches">
        <button
          v-for="p in PATTERNS"
          :key="p.key || 'none'"
          type="button"
          class="bge-swatch bge-swatch-alt"
          :class="{ active: recipe.pattern.key === p.key && !recipe.pattern.emoji }"
          :title="p.label"
          @click="pickPattern(p.key)"
        >
          <span v-if="p.key" class="bge-pat-fill" :style="patternSwatchStyle(p.key)" />
          <span v-else class="material-symbols-outlined bge-swatch-ico">block</span>
        </button>
        <div
          v-if="recipe.pattern.emoji"
          class="bge-swatch bge-swatch-alt active bge-swatch-emoji"
          title="Эмодзи-узор"
        >{{ recipe.pattern.emoji }}</div>
        <div class="bge-swatch bge-swatch-alt bge-swatch-pick" title="Эмодзи как узор">
          <EmojiPicker @pick="pickEmoji" />
        </div>
      </div>

      <template v-if="hasPattern">
        <div class="bge-slider">
          <label>Насыщенность</label>
          <Slider :model-value="recipe.pattern.alpha" :min="1" :max="recipe.pattern.emoji ? 30 : 15" class="bge-slider-ctl" @update:model-value="(v) => setPattern({ alpha: v })" />
          <span class="bge-slider-val">{{ recipe.pattern.alpha }}</span>
        </div>
        <div class="bge-slider">
          <label>Размер</label>
          <Slider :model-value="recipe.pattern.size" :min="64" :max="240" :step="8" class="bge-slider-ctl" @update:model-value="(v) => setPattern({ size: v })" />
          <span class="bge-slider-val">{{ recipe.pattern.size }}</span>
        </div>
      </template>
    </section>
  </PreviewLayout>
</template>

<style scoped>
.bge-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.bge-title {
  margin: 0;
  font-size: 12.5px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--color-text-dim);
}

.bge-bases { max-width: 100%; flex-wrap: wrap; }

/* ── Картинки: готовые, загрузка, недавние ── */
.bge-papers {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(120px, 100%), 1fr));
  gap: 10px;
}

.bge-paper {
  position: relative;
  display: flex;
  flex-direction: column;
  padding: 0;
  overflow: hidden;
  border: 2px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  color: var(--color-text-dim);
  cursor: pointer;
  transition: border-color 0.15s;
}

.bge-paper:hover { border-color: color-mix(in oklch, var(--color-primary) 45%, var(--color-outline-dim)); }
.bge-paper.active { border-color: var(--color-primary); }

.bge-paper img {
  width: 100%;
  aspect-ratio: 16 / 9;
  object-fit: cover;
  display: block;
}

.bge-paper-label {
  padding: 5px 8px 6px;
  font-size: 11.5px;
  font-weight: 650;
  text-align: left;
  overflow-wrap: anywhere;
}

.bge-paper.active .bge-paper-label { color: var(--color-primary); }

.bge-upload {
  aspect-ratio: 16 / 9;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border-style: dashed;
  font-size: 12px;
  font-weight: 650;
}

.bge-upload .material-symbols-outlined { font-size: 24px; }
.bge-upload:disabled { cursor: progress; opacity: 0.7; }

.bge-recent { cursor: default; }

.bge-recent-pick {
  display: block;
  padding: 0;
  border: none;
  background: none;
  cursor: pointer;
}

.bge-recent-remove {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 22px;
  min-width: 22px;
  max-width: 22px;
  height: 22px;
  min-height: 22px;
  max-height: 22px;
  display: grid;
  place-items: center;
  padding: 0;
  border: none;
  border-radius: var(--radius-full);
  background: var(--acrylic-bg-strong);
  color: var(--color-text);
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.15s;
}

.bge-recent:hover .bge-recent-remove,
.bge-recent-remove:focus-visible { opacity: 1; }
.bge-recent-remove .material-symbols-outlined { font-size: 14px; }

@media (hover: none) {
  .bge-recent-remove { opacity: 1; }
}

/* ── Образцы цвета, градиента и узора ── */
.bge-swatches {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.bge-swatch {
  position: relative;
  width: 52px;
  min-width: 52px;
  max-width: 52px;
  height: 52px;
  min-height: 52px;
  max-height: 52px;
  display: grid;
  place-items: center;
  padding: 0;
  overflow: hidden;
  border: 2px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background-color: var(--color-bg);
  background-size: cover;
  cursor: pointer;
  transition: border-color 0.15s;
}

.bge-swatch:hover { border-color: var(--color-primary); }

.bge-swatch.active {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px var(--color-primary) inset;
}

.bge-swatch-ico {
  font-size: 22px;
  color: var(--color-text-dim);
}

.bge-swatch-alt { background-color: var(--color-surface-high); }
.bge-swatch-emoji { font-size: 26px; line-height: 1; }

.bge-swatch-pick :deep(.emoji-picker-wrap),
.bge-swatch-pick :deep(.emoji-btn) {
  width: 100%;
  height: 100%;
  min-height: 0;
  border: none;
  background: transparent;
  border-radius: 0;
}

.bge-swatch-pick :deep(.emoji-btn .material-symbols-outlined) {
  font-size: 24px;
  color: var(--color-text-dim);
}

.bge-pat-fill {
  position: absolute;
  inset: 0;
}

/* ── Ползунки ── */
.bge-slider {
  display: flex;
  align-items: center;
  gap: 12px;
}

.bge-slider label {
  width: 108px;
  flex-shrink: 0;
  font-size: 13px;
  color: var(--color-text);
}

.bge-slider-ctl { flex: 1; min-width: 0; }

.bge-slider-val {
  width: 34px;
  text-align: right;
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  color: var(--color-text-dim);
}

@media (max-width: 560px) {
  .bge-slider label { width: 84px; }

  .bge-swatch {
    width: 46px;
    min-width: 46px;
    max-width: 46px;
    height: 46px;
    min-height: 46px;
    max-height: 46px;
  }
}
</style>
