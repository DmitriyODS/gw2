<script setup>
/* Дерево слоёв доски: слои с их свойствами и ОБЪЕКТЫ внутри каждого —
   порядок перетаскиванием (в том числе между слоями), скрытие, блокировка,
   выделение по клику. То же дерево, что в Figma, только вместо вложенных
   групп — метки объектов (см. utils/boardScene.js).

   Порядок в списке ОБРАТНЫЙ порядку отрисовки: сверху то, что ближе к зрителю.

   Свойств у слоя больше, чем помещается в строку, поэтому редкие живут в меню
   строки: панель узкая, а прозрачность и наложение меняют не каждый день — в
   отличие от «показать/скрыть», которое должно быть под рукой. */
import { computed, ref } from 'vue'
import InputText from 'primevue/inputtext'
import ContextMenu from '@/components/common/ContextMenu.vue'
import {
  BLEND_MODES, newLayer, normalizeScene, objectIcon, objectLabel,
} from '@/utils/boardScene.js'

const props = defineProps({
  scene: { type: Object, required: true },
  activeLayer: { type: String, default: '' },
  // Выделение холста: подсвечиваем те же объекты, что и на доске.
  selection: { type: Array, default: () => [] },
  readOnly: { type: Boolean, default: false },
})

const emit = defineEmits(['update:layers', 'update:objects', 'update:activeLayer', 'select', 'close'])

const renaming = ref(null)      // id слоя, который переименовывают
const draft = ref('')
const collapsed = ref(new Set())
const menu = ref({ visible: false, x: 0, y: 0, id: '', kind: 'layer' })
// Что тащим и куда собираемся положить.
const dragging = ref(null)
const dropTarget = ref(null)

const scene = computed(() => normalizeScene(props.scene))
// Сверху панели — верхние слои: в списке порядок обратный порядку отрисовки.
const layers = computed(() => [...scene.value.layers].reverse())

/** Объекты слоя сверху вниз (верхний в отрисовке — первый в списке). */
function objectsOf(layerId) {
  return scene.value.objects.filter((o) => o.layer === layerId).reverse()
}

function applyLayers(next) {
  emit('update:layers', next)
}

function applyObjects(next) {
  emit('update:objects', next)
}

function patchLayer(id, fields) {
  applyLayers(scene.value.layers.map((l) => (l.id === id ? { ...l, ...fields } : l)))
}

function patchObject(id, fields) {
  applyObjects(scene.value.objects.map((o) => (o.id === id ? { ...o, ...fields } : o)))
}

// ── Слои ─────────────────────────────────────────────────────────

function addLayer() {
  const list = scene.value.layers
  const created = newLayer(`Слой ${list.length + 1}`)
  applyLayers([...list, created])
  emit('update:activeLayer', created.id)
}

function toggleLayer(id, field) {
  applyLayers(scene.value.layers.map((l) => (l.id === id ? { ...l, [field]: !l[field] } : l)))
}

/** Перестановка слоя в порядке отрисовки (вверх = ближе к зрителю). */
function moveLayer(id, delta) {
  const list = [...scene.value.layers]
  const i = list.findIndex((l) => l.id === id)
  const j = i + delta
  if (i < 0 || j < 0 || j >= list.length) return
  ;[list[i], list[j]] = [list[j], list[i]]
  applyLayers(list)
}

function startRename(l) {
  renaming.value = l.id
  draft.value = l.name
}

function commitRename() {
  const id = renaming.value
  renaming.value = null
  const name = draft.value.trim()
  if (!id || !name) return
  patchLayer(id, { name })
}

/** Удаление слоя вместе с его содержимым; последний слой не удаляем. */
function removeLayer(id) {
  const list = scene.value.layers
  if (list.length < 2) return
  const count = objectsOf(id).length
  if (count && !window.confirm(`Удалить слой вместе с содержимым (${count})?`)) return
  const next = list.filter((l) => l.id !== id)
  applyLayers(next)
  if (props.activeLayer === id) emit('update:activeLayer', next[next.length - 1].id)
}

function toggleCollapse(id) {
  const next = new Set(collapsed.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  collapsed.value = next
}

// ── Объекты ──────────────────────────────────────────────────────

function selectObject(o, e) {
  if (e.ctrlKey || e.metaKey) {
    const next = props.selection.includes(o.id)
      ? props.selection.filter((id) => id !== o.id)
      : [...props.selection, o.id]
    emit('select', next)
    return
  }
  emit('select', [o.id])
  emit('update:activeLayer', o.layer)
}

function removeObject(id) {
  applyObjects(scene.value.objects.filter((o) => o.id !== id))
}

/* Перетаскивание: объект встаёт ПЕРЕД целью в списке (то есть выше неё в
   отрисовке) и переезжает в её слой. Порядок в сцене общий на все слои,
   поэтому вставка считается по индексу цели в общем массиве. */
function onDragStart(o) {
  if (props.readOnly) return
  dragging.value = o.id
}

function onDragOver(target, kind) {
  if (!dragging.value || props.readOnly) return
  dropTarget.value = { id: target.id, kind }
}

function onDrop(target, kind) {
  const dragId = dragging.value
  dragging.value = null
  dropTarget.value = null
  if (!dragId || props.readOnly || dragId === target.id) return

  const list = [...scene.value.objects]
  const from = list.findIndex((o) => o.id === dragId)
  if (from < 0) return
  const [moved] = list.splice(from, 1)

  if (kind === 'layer') {
    // Брошен на сам слой — кладём наверх этого слоя.
    const lastOfLayer = list.reduce((acc, o, i) => (o.layer === target.id ? i : acc), -1)
    list.splice(lastOfLayer + 1, 0, { ...moved, layer: target.id })
    applyObjects(list)
    return
  }
  const to = list.findIndex((o) => o.id === target.id)
  if (to < 0) return
  // В списке объект над целью, значит в массиве он идёт ПОСЛЕ неё.
  list.splice(to + 1, 0, { ...moved, layer: target.layer })
  applyObjects(list)
}

// ── Меню строки ──────────────────────────────────────────────────

function openLayerMenu(l, e) {
  const rect = e.currentTarget.getBoundingClientRect()
  menu.value = { visible: true, x: rect.left - 150, y: rect.bottom + 4, id: l.id, kind: 'layer' }
}

function openObjectMenu(o, e) {
  e.preventDefault()
  menu.value = { visible: true, x: e.clientX, y: e.clientY, id: o.id, kind: 'object' }
}

const menuLayer = computed(() => scene.value.layers.find((l) => l.id === menu.value.id) || null)
const menuObject = computed(() => scene.value.objects.find((o) => o.id === menu.value.id) || null)
// Нижнему слою обтравляться не по чему — пункт ему не показываем.
const canClip = computed(() => scene.value.layers.findIndex((l) => l.id === menu.value.id) > 0)

const menuItems = computed(() => {
  if (menu.value.kind === 'object') {
    const o = menuObject.value
    if (!o) return []
    return [
      { label: o.hidden ? 'Показать' : 'Скрыть', icon: o.hidden ? 'visibility' : 'visibility_off', action: 'obj-visible' },
      { label: o.locked ? 'Разблокировать' : 'Заблокировать', icon: o.locked ? 'lock_open' : 'lock', action: 'obj-lock' },
      { label: o.maskObject ? 'Снять маску' : 'Использовать как маску', icon: 'photo_filter', action: 'obj-mask' },
      { divider: true },
      {
        label: 'Перенести в слой',
        icon: 'layers',
        children: scene.value.layers.filter((l) => l.id !== o.layer)
          .map((l) => ({ label: l.name, icon: 'layers', action: `obj-layer:${l.id}` })),
      },
      { divider: true },
      { label: 'Удалить', icon: 'delete', danger: true, action: 'obj-delete' },
    ]
  }
  const l = menuLayer.value
  if (!l) return []
  return [
    {
      label: `Прозрачность ${Math.round(l.opacity * 100)}%`,
      icon: 'opacity',
      children: [100, 75, 50, 25, 10].map((p) => ({
        label: `${p}%`,
        icon: Math.round(l.opacity * 100) === p ? 'radio_button_checked' : 'radio_button_unchecked',
        action: `opacity:${p}`,
      })),
    },
    {
      label: 'Режим наложения',
      icon: 'gradient',
      children: BLEND_MODES.map((b) => ({
        label: b.label,
        icon: l.blend === b.key ? 'radio_button_checked' : 'radio_button_unchecked',
        action: `blend:${b.key}`,
      })),
    },
    ...(canClip.value ? [{
      label: l.clip ? 'Снять обтравку' : 'Обтравить по нижнему',
      icon: 'content_cut',
      action: 'clip',
    }] : []),
    ...(l.mask ? [{ label: 'Снять маску слоя', icon: 'layers_clear', action: 'mask-clear' }] : []),
    { divider: true },
    { label: 'Дублировать слой', icon: 'library_add', action: 'duplicate' },
  ]
})

function onMenuSelect(action) {
  menu.value.visible = false
  const [kind, value] = action.split(':')
  const o = menuObject.value
  const l = menuLayer.value
  if (kind === 'obj-layer' && o) { patchObject(o.id, { layer: value }); return }
  if (kind === 'obj-visible' && o) { patchObject(o.id, { hidden: !o.hidden }); return }
  if (kind === 'obj-lock' && o) { patchObject(o.id, { locked: !o.locked }); return }
  if (kind === 'obj-mask' && o) { patchObject(o.id, { maskObject: !o.maskObject }); return }
  if (kind === 'obj-delete' && o) { removeObject(o.id); return }
  if (!l) return
  if (kind === 'opacity') patchLayer(l.id, { opacity: Number(value) / 100 })
  else if (kind === 'blend') patchLayer(l.id, { blend: value })
  else if (kind === 'clip') patchLayer(l.id, { clip: !l.clip })
  else if (kind === 'mask-clear') patchLayer(l.id, { mask: null })
  else if (kind === 'duplicate') duplicateLayer(l)
}

/* Дубликат слоя копирует только сам слой: объекты остаются на месте — их
   переносит перетаскивание в дереве, иначе одна кнопка порождала бы вторую
   копию всего нарисованного без спроса. */
function duplicateLayer(l) {
  const list = scene.value.layers
  const created = { ...newLayer(`${l.name} — копия`), opacity: l.opacity, blend: l.blend }
  const index = list.findIndex((x) => x.id === l.id)
  applyLayers([...list.slice(0, index + 1), created, ...list.slice(index + 1)])
  emit('update:activeLayer', created.id)
}

// Значки-пометки строки слоя: показывают, что с ним что-то сделано.
function marks(l) {
  const out = []
  if (l.clip) out.push({ icon: 'content_cut', title: 'Обтравлен по нижнему слою' })
  if (l.mask) out.push({ icon: 'photo_filter', title: 'Маска слоя' })
  if (l.blend !== 'normal') out.push({ icon: 'gradient', title: 'Режим наложения' })
  if (l.opacity < 1) out.push({ icon: 'opacity', title: `Прозрачность ${Math.round(l.opacity * 100)}%` })
  return out
}
</script>

<template>
  <aside class="lp">
    <header class="lp-head">
      <span class="material-symbols-outlined">layers</span>
      <h3 class="lp-title">Слои</h3>
      <button v-if="!readOnly" type="button" class="lp-icon" title="Новый слой" aria-label="Новый слой" @click="addLayer">
        <span class="material-symbols-outlined">add</span>
      </button>
      <button type="button" class="lp-icon" title="Скрыть панель" aria-label="Скрыть панель" @click="emit('close')">
        <span class="material-symbols-outlined">close</span>
      </button>
    </header>

    <ul class="lp-list">
      <template v-for="(l, i) in layers" :key="l.id">
        <li
          class="lp-item"
          :class="{
            'is-active': l.id === activeLayer,
            'is-hidden': !l.visible,
            'is-clip': l.clip,
            'is-drop': dropTarget?.id === l.id,
          }"
          @click="emit('update:activeLayer', l.id)"
          @dragover.prevent="onDragOver(l, 'layer')"
          @drop.prevent="onDrop(l, 'layer')"
        >
          <button
            type="button"
            class="lp-icon lp-caret"
            :title="collapsed.has(l.id) ? 'Показать объекты' : 'Свернуть'"
            @click.stop="toggleCollapse(l.id)"
          >
            <span class="material-symbols-outlined">{{ collapsed.has(l.id) ? 'chevron_right' : 'expand_more' }}</span>
          </button>
          <button
            type="button"
            class="lp-icon"
            :title="l.visible ? 'Скрыть слой' : 'Показать слой'"
            :disabled="readOnly"
            @click.stop="toggleLayer(l.id, 'visible')"
          >
            <span class="material-symbols-outlined">{{ l.visible ? 'visibility' : 'visibility_off' }}</span>
          </button>
          <button
            type="button"
            class="lp-icon"
            :title="l.locked ? 'Разблокировать слой' : 'Заблокировать слой'"
            :disabled="readOnly"
            @click.stop="toggleLayer(l.id, 'locked')"
          >
            <span class="material-symbols-outlined">{{ l.locked ? 'lock' : 'lock_open' }}</span>
          </button>

          <InputText
            v-if="renaming === l.id"
            v-model="draft"
            class="lp-input"
            autofocus
            @blur="commitRename"
            @keydown.enter="commitRename"
            @keydown.esc="renaming = null"
            @click.stop
          />
          <button v-else type="button" class="lp-name" :disabled="readOnly" @click.stop="startRename(l)">
            <span class="lp-text">{{ l.name }}</span>
            <span v-for="m in marks(l)" :key="m.icon" class="material-symbols-outlined lp-mark" :title="m.title">
              {{ m.icon }}
            </span>
            <span class="lp-count">{{ objectsOf(l.id).length }}</span>
          </button>

          <template v-if="!readOnly">
            <button type="button" class="lp-icon" title="Выше" :disabled="i === 0" @click.stop="moveLayer(l.id, 1)">
              <span class="material-symbols-outlined">keyboard_arrow_up</span>
            </button>
            <button
              type="button"
              class="lp-icon"
              title="Ниже"
              :disabled="i === layers.length - 1"
              @click.stop="moveLayer(l.id, -1)"
            >
              <span class="material-symbols-outlined">keyboard_arrow_down</span>
            </button>
            <button type="button" class="lp-icon" title="Свойства слоя" @click.stop="openLayerMenu(l, $event)">
              <span class="material-symbols-outlined">more_vert</span>
            </button>
            <button
              type="button"
              class="lp-icon lp-icon--danger"
              title="Удалить слой"
              :disabled="layers.length < 2"
              @click.stop="removeLayer(l.id)"
            >
              <span class="material-symbols-outlined">delete</span>
            </button>
          </template>
        </li>

        <li
          v-for="o in (collapsed.has(l.id) ? [] : objectsOf(l.id))"
          :key="o.id"
          class="lp-obj"
          :class="{
            'is-selected': selection.includes(o.id),
            'is-hidden': o.hidden,
            'is-drop': dropTarget?.id === o.id,
          }"
          :draggable="!readOnly"
          @click="selectObject(o, $event)"
          @contextmenu="openObjectMenu(o, $event)"
          @dragstart="onDragStart(o)"
          @dragover.prevent="onDragOver(o, 'object')"
          @drop.prevent="onDrop(o, 'object')"
          @dragend="dragging = null; dropTarget = null"
        >
          <span class="material-symbols-outlined lp-obj-icon">{{ objectIcon(o) }}</span>
          <span class="lp-text">{{ objectLabel(o) }}</span>
          <span v-if="o.bool" class="material-symbols-outlined lp-mark" title="Булева группа">join_full</span>
          <span v-if="o.maskObject" class="material-symbols-outlined lp-mark" title="Маска">photo_filter</span>
          <template v-if="!readOnly">
            <button
              type="button"
              class="lp-icon"
              :title="o.hidden ? 'Показать' : 'Скрыть'"
              @click.stop="patchObject(o.id, { hidden: !o.hidden })"
            >
              <span class="material-symbols-outlined">{{ o.hidden ? 'visibility_off' : 'visibility' }}</span>
            </button>
            <button
              type="button"
              class="lp-icon"
              :title="o.locked ? 'Разблокировать' : 'Заблокировать'"
              @click.stop="patchObject(o.id, { locked: !o.locked })"
            >
              <span class="material-symbols-outlined">{{ o.locked ? 'lock' : 'lock_open' }}</span>
            </button>
          </template>
        </li>
      </template>
    </ul>

    <p class="lp-hint">
      Объекты перетаскиваются внутри слоя и между слоями. Пиксельный ластик стирает
      только в своём слое, а обтравленный слой виден там, где непрозрачен слой под ним.
    </p>

    <ContextMenu
      :visible="menu.visible"
      :x="menu.x"
      :y="menu.y"
      :items="menuItems"
      @select="onMenuSelect"
      @close="menu.visible = false"
    />
  </aside>
</template>

<style scoped>
.lp {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 300px;
  max-height: 100%;
  padding: 10px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  /* Плотная подложка, как у панели свойств: под панелью рисунок. */
  background: var(--color-surface);
  box-shadow: var(--shadow-2);
}

.lp-head { display: flex; align-items: center; gap: 6px; }
.lp-title { flex: 1; margin: 0; font-size: 14px; font-weight: 600; }

.lp-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  list-style: none;
}

.lp-item,
.lp-obj {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.lp-item:hover,
.lp-obj:hover { background: var(--color-surface-variant); }
.lp-item.is-active { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.lp-item.is-hidden .lp-name { opacity: 0.5; text-decoration: line-through; }
/* Обтравленный слой сдвинут вправо — как «приклеенный» к слою под ним. */
.lp-item.is-clip { padding-left: 12px; }
.lp-item.is-drop,
.lp-obj.is-drop { box-shadow: inset 0 2px 0 var(--color-primary); }

.lp-obj { padding-left: 22px; font-size: 12px; }
.lp-obj.is-selected { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.lp-obj.is-hidden { opacity: 0.5; }
.lp-obj-icon { font-size: 16px; opacity: 0.75; }
.lp-obj .lp-text { flex: 1; }

.lp-name {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
  min-width: 0;
  padding: 6px 4px;
  border: none;
  background: transparent;
  color: inherit;
  font-size: 13px;
  text-align: left;
  cursor: text;
}

.lp-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.lp-mark { font-size: 14px; opacity: 0.7; }
.lp-count { margin-left: auto; font-size: 11px; opacity: 0.6; }
.lp-input { flex: 1; min-width: 0; font-size: 13px; }

.lp-icon {
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
  color: inherit;
  cursor: pointer;
}

.lp-icon:hover:not(:disabled) { background: var(--color-surface); }
.lp-icon:disabled { opacity: 0.35; cursor: default; }
.lp-icon--danger:hover:not(:disabled) { color: var(--color-error); }
.lp-icon .material-symbols-outlined { font-size: 17px; }
.lp-caret .material-symbols-outlined { font-size: 18px; }

.lp-hint { margin: 0; font-size: 11px; color: var(--color-text-muted); line-height: 1.4; }
</style>
