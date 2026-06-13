<template>
  <AppDialog
    :model-value="modelValue"
    tone="success"
    icon="add_call"
    size="sm"
    title="Пригласить в звонок"
    subtitle="Выберите коллег, которым позвонить присоединиться."
    mask-class="above-callview"
    dialog-class="above-callview"
    :actions="dialogActions"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="confirm"
  >
    <div class="inv-search">
      <span class="material-symbols-outlined">search</span>
      <input
        v-model="q"
        placeholder="Кого позвать — имя или логин"
        class="inv-input"
        autofocus
      />
    </div>

    <div v-if="loading" class="inv-empty">
      <ProgressSpinner style="width:32px;height:32px" />
    </div>
    <div v-else-if="!items.length" class="inv-empty">
      <span class="material-symbols-outlined">person_search</span>
      <p>{{ q ? 'Никого не нашли' : 'Все уже в звонке или начните вводить' }}</p>
    </div>
    <ul v-else class="inv-list">
      <li
        v-for="u in items"
        :key="u.id"
        class="inv-item"
        :class="{ selected: selectedIds.has(u.id) }"
        @click="toggle(u.id)"
      >
        <img class="inv-avatar" :src="avatarOf(u)" :alt="u.fio" />
        <div class="inv-info">
          <div class="inv-name">{{ u.fio }}</div>
          <div class="inv-meta">@{{ u.login }} · {{ u.post || u.role?.name }}</div>
        </div>
        <span class="inv-check">
          <span class="material-symbols-outlined">
            {{ selectedIds.has(u.id) ? 'check_circle' : 'radio_button_unchecked' }}
          </span>
        </span>
      </li>
    </ul>
  </AppDialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import AppDialog from '@/components/common/AppDialog.vue'
import { getDirectory } from '@/api/users.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  // id уже участвующих в звонке — их не показываем в списке.
  excludeIds: { type: Array, default: () => [] },
})

const emit = defineEmits(['update:modelValue', 'confirm'])

const q = ref('')
const results = ref([])
const loading = ref(false)
const selectedIds = ref(new Set())
let debounceTimer = null

const items = computed(() => {
  const ex = new Set(props.excludeIds)
  return results.value.filter(u => !ex.has(u.id))
})

const dialogActions = computed(() => [
  { kind: 'cancel', label: 'Отмена' },
  {
    kind: 'confirm',
    label: 'Позвать' + (selectedIds.value.size ? ` (${selectedIds.value.size})` : ''),
    icon: 'add_call',
    tone: 'success',
    disabled: !selectedIds.value.size,
  },
])

async function search() {
  loading.value = true
  try {
    results.value = await getDirectory(q.value.trim(), /* excludeSelf */ true)
  } finally {
    loading.value = false
  }
}

watch(() => props.modelValue, (v) => {
  if (v) {
    q.value = ''
    selectedIds.value = new Set()
    search()
  }
})

watch(q, () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(search, 200)
})

function toggle(id) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = next
}

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function confirm() {
  if (!selectedIds.value.size) return
  emit('confirm', { userIds: [...selectedIds.value] })
  emit('update:modelValue', false)
}
</script>

<style scoped>
.inv-search {
  position: relative;
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.inv-search .material-symbols-outlined {
  position: absolute;
  left: 12px;
  color: var(--color-text-dim);
  font-size: 20px;
  pointer-events: none;
}

.inv-input {
  width: 100%;
  padding: 10px 12px 10px 40px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
  outline: none;
}

.inv-input:focus { border-color: var(--color-primary); }

.inv-list {
  list-style: none;
  padding: 0;
  margin: 0 0 4px;
  max-height: 42dvh;
  overflow-y: auto;
}

.inv-item {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 10px 8px;
  cursor: pointer;
  border-radius: var(--radius-md);
}

.inv-item:hover { background: var(--color-surface-low); }
.inv-item.selected { background: var(--color-primary-container); }

.inv-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.inv-info { min-width: 0; flex: 1; }

.inv-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.inv-item.selected .inv-name { color: var(--color-on-primary-container); }

.inv-meta {
  font-size: 12px;
  color: var(--color-text-dim);
}

.inv-check {
  display: inline-flex;
  align-items: center;
  color: var(--color-text-dim);
  flex-shrink: 0;
}

.inv-item.selected .inv-check { color: var(--color-primary); }

.inv-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 28px 16px;
  color: var(--color-text-dim);
}

.inv-empty .material-symbols-outlined { font-size: 40px; }
</style>

<style>
/* CallView имеет z-index 11500, его mini-окно 12500 — нашу маску надо
   поднять выше, иначе invite-диалог окажется под видеоплитками. */
.app-dialog-mask.above-callview {
  z-index: 13000 !important;
}
.app-dialog-root.above-callview {
  z-index: 13001 !important;
}
</style>
