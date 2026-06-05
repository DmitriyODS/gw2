<template>
  <AppDialog
    model-value
    tone="warning"
    icon="hourglass_top"
    size="md"
    :title="'Задачи засиделись'"
    :subtitle="`${countLabel} висят дольше недели. Загляните — может, пора закрыть или передвинуть срок?`"
    :actions="[
      { kind: 'cancel', label: 'Позже' },
      { kind: 'confirm', label: 'Разобрать задачи' },
    ]"
    @update:model-value="(v) => !v && $emit('close')"
    @cancel="$emit('close')"
    @confirm="goToTasks"
  >
    <ul class="st-list">
      <li
        v-for="t in tasks"
        :key="t.id"
        class="st-item"
        @click="open(t)"
      >
        <div class="st-item-main">
          <span class="st-item-name">{{ t.name }}</span>
          <span v-if="t.department?.name" class="st-item-dept">{{ t.department.name }}</span>
        </div>
        <div class="st-item-side">
          <span class="st-days">{{ daysLabel(t.days_pending) }}</span>
          <span class="material-symbols-outlined st-arrow">chevron_right</span>
        </div>
      </li>
    </ul>
  </AppDialog>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import AppDialog from '@/components/common/AppDialog.vue'

const props = defineProps({
  tasks: { type: Array, required: true },
})
const emit = defineEmits(['close'])
const router = useRouter()

const countLabel = computed(() => {
  const n = props.tasks.length
  const mod10 = n % 10, mod100 = n % 100
  let word = 'задач'
  if (mod10 === 1 && mod100 !== 11) word = 'задача'
  else if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) word = 'задачи'
  return `${n} ${word}`
})

function daysLabel(days) {
  const d = Math.max(7, days || 0)
  const mod10 = d % 10, mod100 = d % 100
  let word = 'дней'
  if (mod10 === 1 && mod100 !== 11) word = 'день'
  else if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) word = 'дня'
  return `${d} ${word}`
}

function open(task) {
  emit('close')
  router.push({ path: '/tasks', query: { open: task.id } })
}

function goToTasks() {
  emit('close')
  router.push('/tasks')
}
</script>

<style scoped>
.st-list {
  list-style: none;
  margin: 4px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.st-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 12px 12px 16px;
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  border-left: 3px solid var(--color-warning);
  cursor: pointer;
  transition: background 0.15s, transform 0.12s;
}

.st-item:hover {
  background: var(--color-surface-high);
  transform: translateX(2px);
}

.st-item-main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }

.st-item-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.st-item-dept {
  font-size: 12px;
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.st-item-side {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.st-days {
  font-size: 12px;
  font-weight: 700;
  color: var(--color-on-warning-container);
  background: var(--color-warning-container);
  padding: 3px 10px;
  border-radius: var(--radius-full);
  white-space: nowrap;
}

.st-arrow {
  font-size: 20px;
  color: var(--color-text-dim);
}
</style>
