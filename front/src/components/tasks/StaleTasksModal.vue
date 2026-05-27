<template>
  <Teleport to="body">
    <div class="st-overlay" @click.self="$emit('close')">
      <div class="st-modal">

        <div class="st-header">
          <div class="st-header-icon">
            <span class="material-symbols-outlined">hourglass_top</span>
          </div>
          <div class="st-header-text">
            <h2 class="st-title">Задачи засиделись</h2>
            <p class="st-sub">
              {{ countLabel }} висят дольше недели. Загляните — может, пора закрыть или передвинуть срок?
            </p>
          </div>
          <button class="st-close" @click="$emit('close')" title="Закрыть">
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>

        <div class="st-scroll">
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
        </div>

        <div class="st-footer">
          <button class="st-btn-text" @click="$emit('close')">Позже</button>
          <button class="st-btn-filled" @click="goToTasks">Разобрать задачи</button>
        </div>

      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'

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
.st-overlay {
  position: fixed;
  inset: 0;
  background: var(--color-scrim);
  backdrop-filter: blur(4px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}

.st-modal {
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  width: 480px;
  max-width: 100%;
  max-height: 84vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.st-header {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 22px 20px 18px;
  border-bottom: 1px solid var(--color-outline-dim);
  flex-shrink: 0;
}

.st-header-icon {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--color-warning-container);
  color: var(--color-on-warning-container);
  display: flex;
  align-items: center;
  justify-content: center;
}

.st-header-icon .material-symbols-outlined {
  font-size: 26px;
  font-variation-settings: 'FILL' 1, 'wght' 500, 'GRAD' 0, 'opsz' 24;
}

.st-header-text { flex: 1; min-width: 0; }

.st-title {
  margin: 0 0 4px;
  font-size: 19px;
  font-weight: 700;
  color: var(--color-text);
  letter-spacing: -0.2px;
}

.st-sub {
  margin: 0;
  font-size: 13.5px;
  line-height: 1.5;
  color: var(--color-text-dim);
}

.st-close {
  width: 36px;
  height: 36px;
  border: none;
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  cursor: pointer;
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, color 0.15s;
  flex-shrink: 0;
}
.st-close:hover {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.st-close .material-symbols-outlined { font-size: 20px; }

.st-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 12px;
}

.st-list {
  list-style: none;
  margin: 0;
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

.st-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 14px 20px 18px;
  border-top: 1px solid var(--color-outline-dim);
  flex-shrink: 0;
}

.st-btn-text {
  border: none;
  background: transparent;
  color: var(--color-primary);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  padding: 10px 18px;
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: background 0.15s;
}
.st-btn-text:hover { background: var(--color-surface-high); }

.st-btn-filled {
  border: none;
  background: var(--color-primary);
  color: var(--color-on-primary);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  padding: 10px 20px;
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: background 0.15s;
}
.st-btn-filled:hover { background: var(--color-primary-hover); }

@media (max-width: 600px) {
  .st-overlay { padding: 0; align-items: flex-end; }
  .st-modal {
    width: 100%;
    max-height: 88vh;
    border-radius: var(--radius-xl) var(--radius-xl) 0 0;
  }
}
</style>
