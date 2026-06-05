<script setup>
import { ref, computed, onMounted, watch, onBeforeUnmount } from 'vue'
import { getDirectory, getDirectoryUser } from '@/api/users.js'
import { formatLastSeen } from '@/utils/presence.js'
import { useMessengerStore } from '@/stores/messenger.js'

const props = defineProps({
  modelValue: { type: [Number, null], default: null },
  placeholder: { type: String, default: 'Не назначен' },
  clearable: { type: Boolean, default: true },
  // Если задан — селект показывает только сотрудников этой компании.
  // У API нет такого фильтра; делаем клиентский (директория уже scope'нута бэком).
  excludeIds: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue', 'change'])

const messenger = useMessengerStore()
const open = ref(false)
const query = ref('')
const users = ref([])
const loading = ref(false)
const root = ref(null)
const selected = ref(null)

const filtered = computed(() => {
  const ex = new Set(props.excludeIds)
  let list = users.value.filter((u) => !ex.has(u.id))
  const q = query.value.trim().toLowerCase()
  if (q) {
    list = list.filter((u) =>
      (u.fio || '').toLowerCase().includes(q) ||
      (u.login || '').toLowerCase().includes(q)
    )
  }
  return list
})

async function load() {
  loading.value = true
  try {
    const data = await getDirectory()
    users.value = Array.isArray(data) ? data : (data?.items || [])
    await syncSelected()
  } finally {
    loading.value = false
  }
}

/* Подбираем «выбранного» пользователя по props.modelValue. Сначала ищем в уже
   загруженной директории; если нет (например, пользователь — Администратор
   системы и не входит в company-скоупленный каталог) — добираем одиночным
   запросом, чтобы корректно показать имя и аватар. */
async function syncSelected() {
  const id = props.modelValue
  if (id == null) { selected.value = null; return }
  const inList = users.value.find((u) => u.id === id)
  if (inList) { selected.value = inList; return }
  if (selected.value?.id === id) return
  try {
    selected.value = await getDirectoryUser(id)
  } catch {
    selected.value = null
  }
}

function avatarOf(u) {
  if (!u) return null
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function pick(u) {
  emit('update:modelValue', u.id)
  emit('change', u)
  selected.value = u
  open.value = false
  query.value = ''
}

function clear() {
  emit('update:modelValue', null)
  emit('change', null)
  selected.value = null
  open.value = false
}

function toggle() {
  open.value = !open.value
  if (open.value && !users.value.length) load()
}

function onClickOutside(e) {
  if (!root.value) return
  if (!root.value.contains(e.target)) open.value = false
}

onMounted(() => {
  document.addEventListener('mousedown', onClickOutside)
  if (props.modelValue != null) {
    // Сразу подгружаем профиль выбранного — чтобы триггер сразу показал имя
    // и аватар, не дожидаясь открытия выпадашки и полной загрузки директории.
    syncSelected()
  }
})
onBeforeUnmount(() => document.removeEventListener('mousedown', onClickOutside))

watch(() => props.modelValue, syncSelected)
</script>

<template>
  <div class="user-picker" ref="root">
    <button type="button" class="picker-control" @click="toggle">
      <template v-if="selected">
        <img :src="avatarOf(selected)" class="ava" alt="" />
        <span class="picker-name">{{ selected.fio }}</span>
      </template>
      <template v-else>
        <span class="material-symbols-outlined picker-placeholder-ico">person</span>
        <span class="picker-placeholder">{{ placeholder }}</span>
      </template>
      <span class="material-symbols-outlined picker-chevron">expand_more</span>
    </button>

    <transition name="picker-pop">
      <div v-if="open" class="picker-pop">
        <div class="picker-search">
          <span class="material-symbols-outlined search-ico">search</span>
          <input
            v-model="query"
            type="text"
            placeholder="Поиск сотрудника…"
            autofocus
          />
        </div>
        <div class="picker-list">
          <div v-if="loading" class="picker-status">Загрузка…</div>
          <div v-else-if="!filtered.length" class="picker-status">Ничего не найдено</div>
          <button
            v-for="u in filtered"
            :key="u.id"
            type="button"
            class="picker-item"
            :class="{ active: u.id === modelValue }"
            @click="pick(u)"
          >
            <img :src="avatarOf(u)" class="ava-sm" alt="" />
            <span class="item-main">
              <span class="item-fio">{{ u.fio }}</span>
              <span class="item-sub">
                <span v-if="messenger.isOnline?.(u.id)" class="online-dot" />
                {{ messenger.isOnline?.(u.id) ? 'в сети' : formatLastSeen(u.last_seen_at) }}
              </span>
            </span>
          </button>
        </div>
        <button v-if="clearable && modelValue != null" type="button" class="picker-clear" @click="clear">
          <span class="material-symbols-outlined">close</span>
          Снять назначение
        </button>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.user-picker { position: relative; }

/* Стилизация под PrimeVue Select (см. поле «Заказчик» в той же форме):
   та же высота, скругление, рамка и поведение hover. */
.picker-control {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 40px;
  padding: 4px 12px 4px 10px;
  border-radius: var(--radius-md, 10px);
  background: var(--color-surface);
  border: 1px solid var(--color-outline-variant);
  color: var(--color-on-surface);
  cursor: pointer;
  text-align: left;
  font: inherit;
  transition: background 0.15s, border-color 0.15s, box-shadow 0.15s;
}
.picker-control:hover {
  border-color: var(--color-primary);
}
.picker-control:focus-visible {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px color-mix(in oklch, var(--color-primary) 25%, transparent);
}

.ava { width: 26px; height: 26px; border-radius: 50%; object-fit: cover; }
.ava-sm { width: 28px; height: 28px; border-radius: 50%; object-fit: cover; }

.picker-name { font-weight: 600; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.picker-placeholder { flex: 1; color: var(--color-on-surface-variant); opacity: 0.85; }
.picker-placeholder-ico { font-size: 22px; opacity: 0.55; padding-left: 4px; }
.picker-chevron { font-size: 20px; opacity: 0.6; }

.picker-pop {
  position: absolute;
  z-index: 60;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  min-width: 260px;
  background: var(--color-surface);
  border-radius: var(--radius-md, 10px);
  box-shadow: var(--shadow-lg);
  border: 1px solid var(--color-outline-variant);
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.picker-search {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-variant);
  border-radius: var(--radius-sm, 8px);
}
.search-ico { font-size: 18px; opacity: 0.6; }
.picker-search input {
  background: transparent;
  border: none;
  outline: none;
  font: inherit;
  color: var(--color-on-surface);
  flex: 1;
  min-width: 0;
}

.picker-list {
  max-height: 280px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}
.picker-status {
  padding: 16px;
  text-align: center;
  color: var(--color-on-surface-variant);
  font-size: 13px;
}

.picker-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px;
  background: transparent;
  border: none;
  border-radius: var(--radius-md, 12px);
  cursor: pointer;
  text-align: left;
  font: inherit;
  color: var(--color-on-surface);
}
.picker-item:hover { background: var(--color-surface-high); }
.picker-item.active { background: var(--color-primary-container); color: var(--color-on-primary-container); }

.item-main { display: flex; flex-direction: column; gap: 2px; min-width: 0; flex: 1; }
.item-fio { font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.item-sub {
  font-size: 12px;
  opacity: 0.75;
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.online-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-success);
}

.picker-clear {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
  padding: 8px 12px;
  background: transparent;
  border: none;
  border-radius: var(--radius-md, 12px);
  color: var(--color-error);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}
.picker-clear:hover { background: color-mix(in oklab, var(--color-error) 8%, transparent); }

.picker-pop-enter-active, .picker-pop-leave-active {
  transition: opacity 0.16s, transform 0.16s;
  transform-origin: top center;
}
.picker-pop-enter-from, .picker-pop-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
}
</style>
