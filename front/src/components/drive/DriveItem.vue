<template>
  <!-- Элемент диска: папка или файл. Одна разметка на оба вида раскладки —
       плитки и список отличаются только стилями. -->
  <component
    :is="layout === 'grid' ? 'article' : 'div'"
    class="item"
    :class="[`is-${layout}`, { 'is-folder': kind === 'folder', 'is-selected': selected }]"
    role="button"
    tabindex="0"
    :aria-selected="selected"
    @click="onClick"
    @keydown.enter.prevent="$emit('open')"
    @contextmenu.prevent="$emit('menu', $event)"
  >
    <!-- Отметка выбора: видна при наведении и у выбранных. Клик по ней НЕ
         открывает — иначе набрать несколько файлов мышью было бы нельзя, а
         про Ctrl-клик надо ещё догадаться. -->
    <button
      class="item-check"
      :class="{ shown: selected }"
      type="button"
      :aria-label="selected ? 'Снять выделение' : 'Выделить'"
      :aria-pressed="selected"
      @click.stop="$emit('select', { item, kind, additive: true })"
    >
      <span class="material-symbols-outlined">
        {{ selected ? 'check_circle' : 'radio_button_unchecked' }}
      </span>
    </button>

    <span class="item-icon" :data-kind="kindOf">
      <img v-if="thumb" :src="thumb" class="item-thumb" alt="" loading="lazy">
      <span v-else class="material-symbols-outlined">{{ icon }}</span>
    </span>

    <span class="item-main">
      <span class="item-name" :title="item.name">{{ item.name }}</span>
      <span class="item-meta">
        <template v-if="kind === 'file'">{{ formatBytes(item.size) }} · </template>
        {{ formatDate(item.updated_at || item.created_at) }}
        <template v-if="item.owner_name"> · {{ item.owner_name }}</template>
      </span>
    </span>

    <span class="item-flags">
      <span v-if="item.starred" class="material-symbols-outlined flag star" title="В избранном">star</span>
      <span v-if="item.shared" class="material-symbols-outlined flag" title="Доступ открыт">group</span>
    </span>

    <button
      class="item-more"
      type="button"
      aria-label="Действия"
      @click.stop="$emit('menu', $event)"
    >
      <span class="material-symbols-outlined">more_vert</span>
    </button>
  </component>
</template>

<script setup>
import { computed } from 'vue'
import { formatBytes } from '@/utils/money.js'
import { fileIcon, fileKind } from '@/utils/fileTypes.js'

const props = defineProps({
  item: { type: Object, required: true },
  kind: { type: String, default: 'file' }, // file | folder
  layout: { type: String, default: 'grid' },
  trash: { type: Boolean, default: false },
  selected: { type: Boolean, default: false },
})

const emit = defineEmits(['open', 'menu', 'select'])

const kindOf = computed(() => (props.kind === 'folder' ? 'folder' : fileKind(props.item.mime, props.item.name)))
const icon = computed(() => (props.kind === 'folder' ? 'folder' : fileIcon(props.item.mime, props.item.name)))

// Превью показываем только картинкам и только плитками: в списке оно не видно,
// а грузить его на каждую строку — лишний трафик.
const thumb = computed(() => (
  props.layout === 'grid' && kindOf.value === 'image' && props.item.url ? props.item.url : ''
))

/* Один клик открывает: папку — внутрь, файл — на просмотр. Клик с Ctrl/Cmd
   (или Shift) не открывает, а отмечает — так набирается несколько файлов для
   общего действия. */
function onClick(e) {
  if (e.ctrlKey || e.metaKey || e.shiftKey) {
    emit('select', { item: props.item, kind: props.kind, additive: true })
    return
  }
  emit('select', { item: props.item, kind: props.kind, additive: false })
  emit('open')
}

function formatDate(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  return Number.isNaN(d.getTime())
    ? ''
    : d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short', year: '2-digit' })
}
</script>

<style scoped>
.item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background 0.15s ease;
}

.item:hover,
.item:focus-visible {
  background: color-mix(in oklch, var(--color-primary) 8%, transparent);
  outline: none;
}

/* Выбранное отмечаем заливкой и кромкой: рамкой-обводкой плитка бы «прыгала»
   на пиксель. */
.item.is-selected {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  box-shadow: inset 0 0 0 1px color-mix(in oklch, var(--color-primary) 45%, transparent);
}

.item.is-selected .item-meta,
.item.is-selected .item-more { color: inherit; opacity: 0.8; }

.item.is-grid {
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
  padding: 12px;
  border: 1px solid var(--acrylic-border);
  background: var(--glass-bg), var(--acrylic-card-bg);
}

.item-icon {
  display: grid;
  place-items: center;
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-variant);
  color: var(--color-text-dim);
  overflow: hidden;
}

.item.is-grid .item-icon {
  width: 100%;
  height: 96px;
}

.item-icon[data-kind='folder'] { color: var(--color-primary); }
.item-icon[data-kind='image'] { color: var(--color-success); }
.item-icon[data-kind='video'] { color: var(--color-tertiary); }
.item-icon[data-kind='archive'] { color: var(--color-warning, var(--color-tertiary)); }

.item-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.item-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.item-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 500;
}

.item-meta {
  font-size: 0.78rem;
  color: var(--color-text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-flags {
  display: flex;
  gap: 4px;
  flex: none;
}

.item.is-grid .item-flags {
  position: absolute;
  inset-block-start: 14px;
  inset-inline-end: 44px;
}

.flag {
  font-size: 18px;
  color: var(--color-text-dim);
}

.flag.star { color: var(--color-warning, var(--color-tertiary)); }

.item-more {
  display: grid;
  place-items: center;
  flex: none;
  width: 32px;
  min-width: 32px;
  max-width: 32px;
  height: 32px;
  min-height: 32px;
  max-height: 32px;
  border: none;
  border-radius: 50%;
  background: none;
  color: var(--color-text-dim);
  cursor: pointer;
}

.item-more:hover { background: var(--color-surface-variant); }

.item-check {
  display: grid;
  place-items: center;
  flex: none;
  width: 28px;
  min-width: 28px;
  max-width: 28px;
  height: 28px;
  min-height: 28px;
  max-height: 28px;
  border: none;
  border-radius: 50%;
  background: none;
  color: var(--color-text-dim);
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.15s ease;
}

/* Показываем отметку, когда до неё дотянулись или уже выбрали; на тач-экранах
   наведения нет, поэтому там она видна всегда. */
.item:hover .item-check,
.item:focus-within .item-check,
.item-check.shown,
.item-check:focus-visible { opacity: 1; }

.item-check.shown { color: var(--color-primary); }
.item.is-selected .item-check { color: inherit; }

@media (hover: none) {
  .item-check { opacity: 1; }
}

.item.is-grid .item-check {
  position: absolute;
  inset-block-start: 14px;
  inset-inline-start: 14px;
  background: var(--acrylic-card-bg);
}

.item.is-grid {
  position: relative;
}

.item.is-grid .item-more {
  position: absolute;
  inset-block-start: 14px;
  inset-inline-end: 10px;
}
</style>
