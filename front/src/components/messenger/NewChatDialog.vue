<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="edit_square"
    size="sm"
    title="Новый чат"
    subtitle="Найдите коллегу по фамилии или логину."
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="newchat-search">
      <span class="material-symbols-outlined">search</span>
      <input
        v-model="q"
        placeholder="Логин или фамилия"
        class="newchat-input"
        autofocus
      />
    </div>
    <div v-if="loading" class="newchat-empty">
      <ProgressSpinner style="width:32px;height:32px" />
    </div>
    <div v-else-if="!results.length" class="newchat-empty">
      <span class="material-symbols-outlined">person_search</span>
      <p>{{ q ? 'Никого не нашли' : 'Начните вводить' }}</p>
    </div>
    <ul v-else class="newchat-results">
      <li
        v-for="u in results"
        :key="u.id"
        class="newchat-item"
        @click="pick(u)"
      >
        <img class="newchat-avatar" :src="avatarOf(u)" :alt="u.fio" />
        <div class="newchat-info">
          <div class="newchat-name">{{ u.fio }}</div>
          <div class="newchat-meta">@{{ u.login }} · {{ u.post || u.role?.name }}</div>
        </div>
      </li>
    </ul>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import AppDialog from '@/components/common/AppDialog.vue'
import { getDirectory } from '@/api/users.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue', 'pick'])

const q = ref('')
const results = ref([])
const loading = ref(false)
let debounceTimer = null

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
    search()
  }
})

watch(q, () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(search, 200)
})

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function pick(u) {
  emit('pick', u)
  emit('update:modelValue', false)
}
</script>

<style scoped>
.newchat-search {
  position: relative;
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.newchat-search .material-symbols-outlined {
  position: absolute;
  left: 12px;
  color: var(--color-text-dim);
  font-size: 20px;
  pointer-events: none;
}

.newchat-input {
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

.newchat-input:focus { border-color: var(--color-primary); }

.newchat-results {
  list-style: none;
  padding: 0;
  margin: 0;
  max-height: 50vh;
  overflow-y: auto;
}

.newchat-item {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 10px 8px;
  cursor: pointer;
  border-radius: var(--radius-md);
}

.newchat-item:hover { background: var(--color-surface-low); }

.newchat-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.newchat-info { min-width: 0; }

.newchat-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.newchat-meta {
  font-size: 12px;
  color: var(--color-text-dim);
}

.newchat-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 28px 16px;
  color: var(--color-text-dim);
}

.newchat-empty .material-symbols-outlined { font-size: 40px; }
</style>
