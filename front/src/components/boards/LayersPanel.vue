<script setup>
/* Панель слоёв доски: порядок отрисовки снизу вверх, видимость, блокировка,
   переименование. Активный слой — тот, в который попадают новые объекты. */
import { computed, ref } from 'vue'
import InputText from 'primevue/inputtext'
import { newLayer, normalizeScene } from '@/utils/boardScene.js'

const props = defineProps({
  scene: { type: Object, required: true },
  activeLayer: { type: String, default: '' },
  readOnly: { type: Boolean, default: false },
})

const emit = defineEmits(['update:layers', 'update:activeLayer', 'close'])

const renaming = ref(null)   // id слоя, который переименовывают
const draft = ref('')

// Сверху панели — верхние слои: в списке порядок обратный порядку отрисовки.
const layers = computed(() => [...normalizeScene(props.scene).layers].reverse())
const objectsByLayer = computed(() => {
  const counts = {}
  for (const o of normalizeScene(props.scene).objects) {
    counts[o.layer] = (counts[o.layer] || 0) + 1
  }
  return counts
})

function apply(next) {
  emit('update:layers', next)
}

function addLayer() {
  const list = normalizeScene(props.scene).layers
  const created = newLayer(`Слой ${list.length + 1}`)
  apply([...list, created])
  emit('update:activeLayer', created.id)
}

function toggle(id, field) {
  apply(normalizeScene(props.scene).layers.map((l) => (l.id === id ? { ...l, [field]: !l[field] } : l)))
}

/** Перестановка слоя в порядке отрисовки (вверх = ближе к зрителю). */
function move(id, delta) {
  const list = [...normalizeScene(props.scene).layers]
  const i = list.findIndex((l) => l.id === id)
  const j = i + delta
  if (i < 0 || j < 0 || j >= list.length) return
  ;[list[i], list[j]] = [list[j], list[i]]
  apply(list)
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
  apply(normalizeScene(props.scene).layers.map((l) => (l.id === id ? { ...l, name } : l)))
}

/** Удаление слоя вместе с его содержимым; последний слой не удаляем. */
function remove(id) {
  const list = normalizeScene(props.scene).layers
  if (list.length < 2) return
  const count = objectsByLayer.value[id] || 0
  if (count && !window.confirm(`Удалить слой вместе с содержимым (${count})?`)) return
  const next = list.filter((l) => l.id !== id)
  apply(next)
  if (props.activeLayer === id) emit('update:activeLayer', next[next.length - 1].id)
}
</script>

<template>
  <aside class="lp">
    <header class="lp-head">
      <span class="material-symbols-outlined">layers</span>
      <h3 class="lp-title">Слои</h3>
      <button v-if="!readOnly" type="button" class="lp-icon" title="Новый слой" @click="addLayer">
        <span class="material-symbols-outlined">add</span>
      </button>
      <button type="button" class="lp-icon" title="Скрыть панель" @click="emit('close')">
        <span class="material-symbols-outlined">close</span>
      </button>
    </header>

    <ul class="lp-list">
      <li
        v-for="(l, i) in layers"
        :key="l.id"
        class="lp-item"
        :class="{ 'is-active': l.id === activeLayer, 'is-hidden': !l.visible }"
        @click="emit('update:activeLayer', l.id)"
      >
        <button
          type="button"
          class="lp-icon"
          :title="l.visible ? 'Скрыть слой' : 'Показать слой'"
          :disabled="readOnly"
          @click.stop="toggle(l.id, 'visible')"
        >
          <span class="material-symbols-outlined">{{ l.visible ? 'visibility' : 'visibility_off' }}</span>
        </button>
        <button
          type="button"
          class="lp-icon"
          :title="l.locked ? 'Разблокировать слой' : 'Заблокировать слой'"
          :disabled="readOnly"
          @click.stop="toggle(l.id, 'locked')"
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
          {{ l.name }}
          <span class="lp-count">{{ objectsByLayer[l.id] || 0 }}</span>
        </button>

        <template v-if="!readOnly">
          <button type="button" class="lp-icon" title="Выше" :disabled="i === 0" @click.stop="move(l.id, 1)">
            <span class="material-symbols-outlined">keyboard_arrow_up</span>
          </button>
          <button
            type="button"
            class="lp-icon"
            title="Ниже"
            :disabled="i === layers.length - 1"
            @click.stop="move(l.id, -1)"
          >
            <span class="material-symbols-outlined">keyboard_arrow_down</span>
          </button>
          <button
            type="button"
            class="lp-icon lp-icon--danger"
            title="Удалить слой"
            :disabled="layers.length < 2"
            @click.stop="remove(l.id)"
          >
            <span class="material-symbols-outlined">delete</span>
          </button>
        </template>
      </li>
    </ul>

    <p class="lp-hint">Новые объекты попадают в выбранный слой. Заблокированный слой не реагирует на клики.</p>
  </aside>
</template>

<style scoped>
.lp {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 260px;
  max-height: 100%;
  padding: 10px;
  border: 1px solid var(--glass-edge);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
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

.lp-item {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.lp-item:hover { background: var(--color-surface-variant); }
.lp-item.is-active { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.lp-item.is-hidden .lp-name { opacity: 0.5; text-decoration: line-through; }

.lp-name {
  display: flex;
  align-items: center;
  gap: 6px;
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

.lp-count { font-size: 11px; opacity: 0.6; }
.lp-input { flex: 1; min-width: 0; font-size: 13px; }

.lp-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 28px;
  max-width: 28px;
  min-height: 28px;
  max-height: 28px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.lp-icon:hover:not(:disabled) { background: var(--color-surface); }
.lp-icon:disabled { opacity: 0.35; cursor: default; }
.lp-icon--danger:hover:not(:disabled) { color: var(--color-error); }
.lp-icon .material-symbols-outlined { font-size: 18px; }

.lp-hint { margin: 0; font-size: 11px; color: var(--color-text-muted); line-height: 1.4; }
</style>
