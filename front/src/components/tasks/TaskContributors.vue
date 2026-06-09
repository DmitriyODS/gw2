<script setup>
import { ref, onMounted } from 'vue'
import { useTasksStore } from '@/stores/tasks.js'

const props = defineProps({
  taskId: { type: Number, required: true },
})

const tasks = useTasksStore()
const loading = ref(false)
const list = ref([])

async function load() {
  loading.value = true
  try {
    list.value = await tasks.loadContributors(props.taskId)
  } catch {
    list.value = []
  } finally {
    loading.value = false
  }
}

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

onMounted(load)
</script>

<template>
  <div class="contributors">
    <div class="contributors-label">Работали над задачей</div>
    <div v-if="loading" class="contributors-empty">…</div>
    <div v-else-if="!list.length" class="contributors-empty">Пока никто</div>
    <div v-else class="contributors-chips">
      <span v-for="u in list" :key="u.id" class="contributor-chip" :title="u.fio">
        <img class="chip-avatar" :src="avatarOf(u)" :alt="u.fio" />
        <span class="chip-name">{{ u.fio }}</span>
      </span>
    </div>
  </div>
</template>

<style scoped>
.contributors {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.contributors-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-primary);
}

.contributors-empty {
  font-size: 12px;
  color: var(--color-text-dim);
}

.contributors-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

/* M3 input-chip: аватар + ФИО */
.contributor-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  max-width: 100%;
  padding: 4px 12px 4px 4px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  border: 1px solid var(--color-outline-dim);
  color: var(--color-text);
  transition: background 0.15s, border-color 0.15s;
}

.contributor-chip:hover {
  background: var(--color-surface-highest);
  border-color: var(--color-outline);
}

.chip-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  display: block;
}

.chip-name {
  font-size: 13px;
  font-weight: 600;
  line-height: 1.2;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
