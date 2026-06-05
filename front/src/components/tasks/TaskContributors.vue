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
    <div v-else class="contributors-list">
      <span v-for="u in list" :key="u.id" class="contributor" :title="u.fio">
        <img :src="avatarOf(u)" :alt="u.fio" />
      </span>
    </div>
  </div>
</template>

<style scoped>
.contributors { display: flex; flex-direction: column; gap: 6px; }
.contributors-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-primary);
}
.contributors-empty { font-size: 12px; color: var(--color-on-surface-variant); }
.contributors-list { display: flex; align-items: center; flex-wrap: wrap; gap: 2px; }
.contributor {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  overflow: hidden;
  border: 2px solid var(--color-surface);
  margin-left: -6px;
  box-shadow: 0 0 0 1px var(--color-outline-dim);
}
.contributor:first-child { margin-left: 0; }
.contributor img { width: 100%; height: 100%; object-fit: cover; display: block; }
</style>
