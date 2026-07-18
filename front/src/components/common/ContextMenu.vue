<template>
  <Teleport to="body">
    <Transition name="ctxm">
      <div
        v-if="visible"
        ref="menuEl"
        class="ctxm"
        :style="menuStyle"
        role="menu"
        @click.stop
        @contextmenu.prevent
      >
        <slot name="header" />
        <template v-for="(item, i) in items" :key="i">
          <div v-if="item.divider" class="ctxm-divider" />
          <button
            class="ctxm-item"
            :class="{ danger: item.danger, 'has-sub': item.children, open: activeSub === i }"
            role="menuitem"
            @mouseenter="onEnter(item, i, $event)"
            @click="onItemClick(item, i, $event)"
          >
            <span v-if="item.icon" class="material-symbols-outlined">{{ item.icon }}</span>
            <span class="ctxm-label">{{ item.label }}</span>
            <span v-if="item.children" class="material-symbols-outlined ctxm-caret">chevron_right</span>
          </button>
        </template>
      </div>
    </Transition>

    <!-- Подменю (плавающая панель у активного пункта) -->
    <Transition name="ctxm">
      <div
        v-if="visible && activeSub !== null && subItems.length"
        ref="subEl"
        class="ctxm ctxm-sub"
        :style="subStyle"
        role="menu"
        @click.stop
        @mouseenter="keepSub = true"
      >
        <template v-for="(item, i) in subItems" :key="i">
          <div v-if="item.divider" class="ctxm-divider" />
          <button class="ctxm-item" :class="{ danger: item.danger }" role="menuitem" @click="pick(item)">
            <span v-if="item.icon" class="material-symbols-outlined">{{ item.icon }}</span>
            <span class="ctxm-label">{{ item.label }}</span>
          </button>
        </template>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  x: { type: Number, default: 0 },
  y: { type: Number, default: 0 },
  // Пункты: { label, icon?, action?, danger?, divider?, children?: [...] }.
  items: { type: Array, default: () => [] },
})
const emit = defineEmits(['select', 'close'])

const menuEl = ref(null)
const subEl = ref(null)
const pos = ref({ x: 0, y: 0 })
const activeSub = ref(null)
const subAnchor = ref({ x: 0, y: 0 })
const keepSub = ref(false)

const menuStyle = computed(() => ({ position: 'fixed', left: pos.value.x + 'px', top: pos.value.y + 'px', zIndex: 12000 }))
const subStyle = computed(() => ({ position: 'fixed', left: subAnchor.value.x + 'px', top: subAnchor.value.y + 'px', zIndex: 12001 }))
const subItems = computed(() => (activeSub.value !== null ? props.items[activeSub.value]?.children ?? [] : []))

watch(() => props.visible, async (v) => {
  if (!v) { activeSub.value = null; return }
  pos.value = { x: props.x, y: props.y }
  activeSub.value = null
  await nextTick()
  clamp(menuEl.value, pos, props.x, props.y)
})

function clamp(el, target, x, y) {
  if (!el) return
  const r = el.getBoundingClientRect()
  const pad = 8
  let nx = x
  let ny = y
  if (nx + r.width > window.innerWidth - pad) nx = window.innerWidth - r.width - pad
  if (ny + r.height > window.innerHeight - pad) ny = window.innerHeight - r.height - pad
  target.value = { x: Math.max(pad, nx), y: Math.max(pad, ny) }
}

async function openSub(index, e) {
  const rect = e.currentTarget.getBoundingClientRect()
  activeSub.value = index
  subAnchor.value = { x: rect.right - 4, y: rect.top }
  await nextTick()
  // Кламп подменю: если не влезает справа — открыть слева от пункта.
  const el = subEl.value
  if (!el) return
  const r = el.getBoundingClientRect()
  const pad = 8
  let nx = rect.right - 4
  let ny = rect.top
  if (nx + r.width > window.innerWidth - pad) nx = rect.left - r.width + 4
  if (ny + r.height > window.innerHeight - pad) ny = window.innerHeight - r.height - pad
  subAnchor.value = { x: Math.max(pad, nx), y: Math.max(pad, ny) }
}

function onEnter(item, i, e) {
  if (item.children) openSub(i, e)
  else activeSub.value = null
}
function onItemClick(item, i, e) {
  if (item.children) { openSub(i, e); return } // тач: клик раскрывает подменю
  pick(item)
}
function pick(item) {
  if (item.action) emit('select', item.action)
  emit('close')
}

function onDocDown(e) {
  if (!props.visible) return
  if (menuEl.value?.contains(e.target) || subEl.value?.contains(e.target)) return
  emit('close')
}
function onScroll(e) {
  if (!props.visible) return
  if (menuEl.value?.contains(e.target) || subEl.value?.contains(e.target)) return
  emit('close')
}
function onKey(e) { if (e.key === 'Escape' && props.visible) emit('close') }

onMounted(() => {
  document.addEventListener('mousedown', onDocDown, true)
  document.addEventListener('touchstart', onDocDown, { passive: true, capture: true })
  document.addEventListener('scroll', onScroll, true)
  document.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocDown, true)
  document.removeEventListener('touchstart', onDocDown, true)
  document.removeEventListener('scroll', onScroll, true)
  document.removeEventListener('keydown', onKey)
})
</script>

<style scoped>
.ctxm {
  min-width: 208px;
  max-width: min(86vw, 300px);
  max-height: min(50vh, 340px);
  overflow-y: auto;
  overscroll-behavior: contain;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md, 12px);
  padding: 6px;
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ctxm-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: none;
  background: transparent;
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  text-align: left;
  border-radius: var(--radius-sm, 8px);
  cursor: pointer;
}
.ctxm-item:hover, .ctxm-item.open { background: var(--color-surface-low); }
.ctxm-item.danger { color: var(--color-error); }
.ctxm-item.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.ctxm-item .material-symbols-outlined { font-size: 18px; }
.ctxm-label { flex: 1; min-width: 0; }
.ctxm-caret { color: var(--color-text-dim); margin-right: -4px; }
.ctxm-divider { height: 1px; background: var(--color-outline-dim); margin: 4px; }

.ctxm-enter-active, .ctxm-leave-active { transition: opacity 0.12s, transform 0.12s; transform-origin: top left; }
.ctxm-enter-from, .ctxm-leave-to { opacity: 0; transform: scale(0.97) translateY(-3px); }
</style>
