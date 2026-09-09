<script setup>
/* Лента кадров покадровой анимации: миниатюры, перестановка, проигрывание,
   частота кадров и «калька» (соседние кадры полупрозрачно на холсте).

   Кадр — не отдельная сцена, а метка у объектов: объект с `frame` виден только
   в своём кадре, объект без метки — во всех (это фон анимации). Поэтому лента
   правит ту же сцену, что и холст, и ничего своего не хранит, кроме
   проигрывания и миниатюр. */
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import { FPS_RANGE, MAX_FRAMES, newId, normalizeScene, orderedObjects, sceneBounds } from '@/utils/boardScene.js'
import { renderScene } from '@/utils/boardRender.js'

const props = defineProps({
  scene: { type: Object, required: true },
  frame: { type: String, default: '' },
  onionDepth: { type: Number, default: 1 },
  readOnly: { type: Boolean, default: false },
  exporting: { type: Boolean, default: false },
})

const emit = defineEmits(['update:scene', 'update:frame', 'update:onionDepth', 'update:playing', 'export', 'close'])

const THUMB_W = 84
const THUMB_H = 56
// Больше миниатюр в ленте всё равно не разглядеть, а перерисовка их всех на
// каждую правку сцены заметно тормозит холст.
const THUMB_LIMIT = 60

const thumbs = ref([])          // canvas-элементы миниатюр по индексу кадра
const playing = ref(false)
const menu = ref({ visible: false, x: 0, y: 0, items: [] })

let timer = null
let thumbTimer = null

const scene = computed(() => normalizeScene(props.scene))
const frames = computed(() => scene.value.animation?.frames || [])
const fps = computed(() => scene.value.animation?.fps || FPS_RANGE.def)
const index = computed(() => Math.max(0, frames.value.findIndex((f) => f.id === props.frame)))
const duration = computed(() => (frames.value.length / Math.max(1, fps.value)).toFixed(1))

function countIn(frameId) {
  return scene.value.objects.filter((o) => o.frame === frameId).length
}

// Объекты без кадра видны всегда — их показываем отдельным счётчиком.
const commonCount = computed(() => scene.value.objects.filter((o) => !o.frame).length)

function apply(next) {
  emit('update:scene', next)
}

function setAnimation(animation, objects) {
  apply({ ...scene.value, animation, objects: objects || scene.value.objects })
}

// ── Правка ленты ─────────────────────────────────────────────────

function addFrame(after = index.value) {
  if (frames.value.length >= MAX_FRAMES) return
  const created = { id: newId(), name: `Кадр ${frames.value.length + 1}` }
  const list = [...frames.value]
  list.splice(after + 1, 0, created)
  setAnimation({ fps: fps.value, frames: list })
  emit('update:frame', created.id)
}

/** Дубликат кадра: копия его объектов с новыми id — основа покадровой работы. */
function duplicateFrame(id = props.frame) {
  if (frames.value.length >= MAX_FRAMES) return
  const source = frames.value.find((f) => f.id === id)
  if (!source) return
  const created = { id: newId(), name: `${source.name} — копия` }
  const list = [...frames.value]
  list.splice(frames.value.indexOf(source) + 1, 0, created)
  const copies = scene.value.objects
    .filter((o) => o.frame === id)
    .map((o) => ({ ...JSON.parse(JSON.stringify(o)), id: newId(), frame: created.id }))
  setAnimation({ fps: fps.value, frames: list }, [...scene.value.objects, ...copies])
  emit('update:frame', created.id)
}

/** Удаление кадра уносит и его объекты: они больше нигде не показываются. */
function removeFrame(id = props.frame) {
  if (frames.value.length < 2) return
  const list = frames.value.filter((f) => f.id !== id)
  const objects = scene.value.objects.filter((o) => o.frame !== id)
  setAnimation({ fps: fps.value, frames: list }, objects)
  if (props.frame === id) emit('update:frame', list[Math.max(0, index.value - 1)].id)
}

function clearFrame(id = props.frame) {
  const count = countIn(id)
  if (count && !window.confirm(`Очистить кадр (${count} объектов)?`)) return
  apply({ ...scene.value, objects: scene.value.objects.filter((o) => o.frame !== id) })
}

function moveFrame(id, delta) {
  const list = [...frames.value]
  const i = list.findIndex((f) => f.id === id)
  const j = i + delta
  if (i < 0 || j < 0 || j >= list.length) return
  ;[list[i], list[j]] = [list[j], list[i]]
  setAnimation({ fps: fps.value, frames: list })
}

function setFps(value) {
  setAnimation({ fps: Math.min(FPS_RANGE.max, Math.max(FPS_RANGE.min, value)), frames: frames.value })
}

// ── Проигрывание ─────────────────────────────────────────────────

function togglePlay() {
  if (playing.value) stop()
  else start()
}

function start() {
  if (frames.value.length < 2) return
  playing.value = true
  emit('update:playing', true)
  timer = setInterval(() => {
    const next = frames.value[(index.value + 1) % frames.value.length]
    if (next) emit('update:frame', next.id)
  }, 1000 / Math.max(1, fps.value))
}

function stop() {
  clearInterval(timer)
  timer = null
  playing.value = false
  emit('update:playing', false)
}

function step(delta) {
  if (!frames.value.length) return
  const next = frames.value[(index.value + delta + frames.value.length) % frames.value.length]
  emit('update:frame', next.id)
}

// ── Миниатюры ────────────────────────────────────────────────────

/* Миниатюра кадра — тот же рендер сцены, что и на холсте, в общую для всех
   кадров рамку: иначе кадры «прыгали» бы по размеру и лента врала о движении. */
function drawThumbs() {
  const list = frames.value.slice(0, THUMB_LIMIT)
  const box = sceneBounds(orderedObjects(scene.value)) || { x: 0, y: 0, w: 100, h: 100 }
  const scale = Math.min(THUMB_W / (box.w + 24), THUMB_H / (box.h + 24))
  list.forEach((f, i) => {
    const canvas = thumbs.value[i]
    const ctx = canvas?.getContext('2d')
    if (!ctx) return
    ctx.setTransform(1, 0, 0, 1, 0, 0)
    ctx.clearRect(0, 0, THUMB_W, THUMB_H)
    renderScene(ctx, scene.value, {
      width: THUMB_W,
      height: THUMB_H,
      camera: {
        x: box.x + box.w / 2 - THUMB_W / (2 * scale),
        y: box.y + box.h / 2 - THUMB_H / (2 * scale),
        scale,
      },
      images: new Map(),
      frame: f.id,
    })
  })
}

function scheduleThumbs() {
  clearTimeout(thumbTimer)
  thumbTimer = setTimeout(async () => {
    await nextTick()
    drawThumbs()
  }, 300)
}

// ── Меню ─────────────────────────────────────────────────────────

function openFrameMenu(f, e) {
  e.preventDefault()
  menu.value = {
    visible: true,
    x: e.clientX,
    y: e.clientY,
    items: [
      { label: 'Дублировать кадр', icon: 'library_add', action: `dup:${f.id}` },
      { label: 'Очистить кадр', icon: 'cleaning_services', action: `clear:${f.id}` },
      { label: 'Сдвинуть влево', icon: 'chevron_left', action: `left:${f.id}` },
      { label: 'Сдвинуть вправо', icon: 'chevron_right', action: `right:${f.id}` },
      { divider: true },
      { label: 'Удалить кадр', icon: 'delete', danger: true, action: `del:${f.id}` },
    ],
  }
}

function openFpsMenu(e) {
  const rect = e.currentTarget.getBoundingClientRect()
  menu.value = {
    visible: true,
    x: rect.left,
    y: rect.top - 6,
    items: [2, 4, 6, 8, 12, 15, 24, 30, 60].map((v) => ({
      label: `${v} кадров/с`,
      icon: fps.value === v ? 'radio_button_checked' : 'radio_button_unchecked',
      action: `fps:${v}`,
    })),
  }
}

function openOnionMenu(e) {
  const rect = e.currentTarget.getBoundingClientRect()
  menu.value = {
    visible: true,
    x: rect.left,
    y: rect.top - 6,
    items: [0, 1, 2, 3].map((v) => ({
      label: v ? `${v} соседних кадра` : 'Выключить кальку',
      icon: props.onionDepth === v ? 'radio_button_checked' : 'radio_button_unchecked',
      action: `onion:${v}`,
    })),
  }
}

function onMenuSelect(action) {
  menu.value.visible = false
  const [kind, value] = action.split(':')
  switch (kind) {
    case 'dup': duplicateFrame(value); break
    case 'clear': clearFrame(value); break
    case 'del': removeFrame(value); break
    case 'left': moveFrame(value, -1); break
    case 'right': moveFrame(value, 1); break
    case 'fps': setFps(Number(value)); break
    case 'onion': emit('update:onionDepth', Number(value)); break
    default: break
  }
}

watch(() => props.scene, scheduleThumbs, { deep: true })
watch(frames, scheduleThumbs, { immediate: true })
// Частота меняется на ходу — перезапускаем таймер, иначе ролик идёт по старой.
watch(fps, () => { if (playing.value) { stop(); start() } })

onBeforeUnmount(() => {
  clearInterval(timer)
  clearTimeout(thumbTimer)
})
</script>

<template>
  <div class="ft">
    <div class="ft-bar">
      <button type="button" class="ft-icon" title="Предыдущий кадр" aria-label="Предыдущий кадр" @click="step(-1)">
        <span class="material-symbols-outlined">skip_previous</span>
      </button>
      <button
        type="button"
        class="ft-icon ft-icon--play"
        :title="playing ? 'Пауза' : 'Проиграть'"
        :aria-label="playing ? 'Пауза' : 'Проиграть'"
        @click="togglePlay"
      >
        <span class="material-symbols-outlined">{{ playing ? 'pause' : 'play_arrow' }}</span>
      </button>
      <button type="button" class="ft-icon" title="Следующий кадр" aria-label="Следующий кадр" @click="step(1)">
        <span class="material-symbols-outlined">skip_next</span>
      </button>

      <button type="button" class="ft-chip" title="Частота кадров" @click="openFpsMenu">
        {{ fps }} к/с
      </button>
      <button
        type="button"
        class="ft-chip"
        :class="{ 'is-on': onionDepth > 0 }"
        title="Калька: соседние кадры полупрозрачно"
        @click="openOnionMenu"
      >
        <span class="material-symbols-outlined">layers</span>
        {{ onionDepth || 'выкл' }}
      </button>

      <span class="ft-meta">
        Кадр {{ index + 1 }} из {{ frames.length }} · {{ duration }} с
        <template v-if="commonCount"> · общих объектов: {{ commonCount }}</template>
      </span>

      <template v-if="!readOnly">
        <button type="button" class="ft-icon" title="Новый кадр" aria-label="Новый кадр" @click="addFrame()">
          <span class="material-symbols-outlined">add</span>
        </button>
        <button type="button" class="ft-icon" title="Дублировать кадр" aria-label="Дублировать кадр" @click="duplicateFrame()">
          <span class="material-symbols-outlined">library_add</span>
        </button>
        <button
          type="button"
          class="ft-icon"
          title="Удалить кадр"
          aria-label="Удалить кадр"
          :disabled="frames.length < 2"
          @click="removeFrame()"
        >
          <span class="material-symbols-outlined">delete</span>
        </button>
      </template>

      <button
        v-if="!readOnly"
        type="button"
        class="ft-chip ft-chip--accent"
        :disabled="exporting"
        title="Сохранить анимацию видеофайлом"
        @click="emit('export')"
      >
        <span class="material-symbols-outlined">movie</span>
        {{ exporting ? 'Собираем…' : 'Видео' }}
      </button>
      <button type="button" class="ft-icon" title="Выйти из режима кадров" aria-label="Выйти из режима кадров" @click="emit('close')">
        <span class="material-symbols-outlined">close</span>
      </button>
    </div>

    <ul class="ft-list">
      <li
        v-for="(f, i) in frames"
        :key="f.id"
        class="ft-frame"
        :class="{ 'is-active': f.id === frame }"
        @click="emit('update:frame', f.id)"
        @contextmenu="openFrameMenu(f, $event)"
      >
        <canvas
          v-if="i < THUMB_LIMIT"
          :ref="(el) => { if (el) thumbs[i] = el }"
          class="ft-thumb"
          :width="THUMB_W"
          :height="THUMB_H"
        />
        <span v-else class="ft-thumb ft-thumb--plain">{{ i + 1 }}</span>
        <span class="ft-num">{{ i + 1 }}</span>
      </li>
      <li v-if="!readOnly && frames.length < MAX_FRAMES" class="ft-frame ft-frame--add" @click="addFrame(frames.length - 1)">
        <span class="material-symbols-outlined">add</span>
      </li>
    </ul>

    <ContextMenu
      :visible="menu.visible"
      :x="menu.x"
      :y="menu.y"
      :items="menu.items"
      @select="onMenuSelect"
      @close="menu.visible = false"
    />
  </div>
</template>

<style scoped>
.ft {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  pointer-events: auto;
  border: 1px solid var(--glass-edge);
  border-radius: var(--radius-lg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  background: var(--acrylic-bg);
  box-shadow: var(--shadow-2);
}

.ft-bar {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.ft-meta {
  flex: 1;
  min-width: 120px;
  padding: 0 6px;
  font-size: 12px;
  color: var(--color-text-muted);
  overflow-wrap: anywhere;
}

.ft-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  max-width: 32px;
  min-height: 32px;
  max-height: 32px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
}

.ft-icon:hover:not(:disabled) { background: var(--color-surface-variant); }
.ft-icon:disabled { opacity: 0.4; cursor: default; }
.ft-icon .material-symbols-outlined { font-size: 20px; }
.ft-icon--play { background: var(--color-primary); color: var(--color-on-primary); }
.ft-icon--play:hover:not(:disabled) { background: var(--color-primary); }

.ft-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 32px;
  padding: 0 10px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.ft-chip:hover:not(:disabled) { background: var(--color-surface-variant); }
.ft-chip:disabled { opacity: 0.6; cursor: default; }
.ft-chip.is-on { border-color: var(--color-primary); color: var(--color-primary); }
.ft-chip--accent { border-color: var(--color-primary); background: var(--color-primary); color: var(--color-on-primary); }
.ft-chip .material-symbols-outlined { font-size: 16px; }

.ft-list {
  display: flex;
  gap: 6px;
  margin: 0;
  padding: 2px;
  overflow-x: auto;
  list-style: none;
}

.ft-frame {
  position: relative;
  flex: 0 0 auto;
  border: 2px solid transparent;
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  cursor: pointer;
  line-height: 0;
}

.ft-frame.is-active { border-color: var(--color-primary); }
.ft-thumb { display: block; width: 84px; height: 56px; border-radius: 2px; }

.ft-thumb--plain {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 84px;
  height: 56px;
  color: var(--color-text-muted);
  font-size: 13px;
  line-height: 1;
}

.ft-num {
  position: absolute;
  left: 3px;
  bottom: 3px;
  padding: 0 4px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-variant);
  color: var(--color-text-muted);
  font-size: 10px;
  line-height: 14px;
}

.ft-frame--add {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 84px;
  height: 56px;
  border: 1px dashed var(--color-outline-dim);
  color: var(--color-text-muted);
}

@media (max-width: 768px) {
  .ft-meta { display: none; }
}
</style>
