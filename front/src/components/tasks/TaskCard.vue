<template>
  <div
    class="task-card"
    :class="{ favorite: task.is_favorite, archived: task.is_archived }"
    @click.stop="$emit('click', task)"
  >
    <div class="card-header">
      <span class="dept-badge">{{ task.department?.name || '—' }}</span>
      <button class="favorite-btn" @click.stop="$emit('toggle-favorite', task)" :title="task.is_favorite ? 'Убрать из избранного' : 'Добавить в избранное'">
        <span class="material-symbols-outlined" :class="{ filled: task.is_favorite }">
          {{ task.is_favorite ? 'favorite' : 'favorite_border' }}
        </span>
      </button>
    </div>

    <h3 class="task-name">{{ task.name }}</h3>

    <div class="task-meta">
      <div v-if="task.deadline" class="meta-row">
        <span class="material-symbols-outlined">event</span>
        Сделать до: {{ formatDate(task.deadline) }}
      </div>
      <div class="meta-row">
        <span class="material-symbols-outlined">inbox</span>
        Поступила: {{ formatDate(task.received_at) }}
      </div>
    </div>

    <div class="card-footer">
      <div v-if="task.has_units" class="units-indicator" title="Есть юниты">
        <span class="material-symbols-outlined">timer</span>
      </div>
      <div v-if="task.active_users?.length" class="active-users">
        <div
          v-for="user in task.active_users.slice(0, 4)"
          :key="user.id"
          class="active-avatar"
          :title="user.fio"
        >
          <img
            :src="user.avatar_path ? `/uploads/${user.avatar_path}` : `/api/users/${user.id}/identicon`"
            :alt="user.fio"
          />
        </div>
        <div v-if="task.active_users.length > 4" class="active-avatar active-avatar-more">
          +{{ task.active_users.length - 4 }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  task: {
    type: Object,
    required: true
  }
})

defineEmits(['click', 'toggle-favorite'])

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
}
</script>

<style scoped>
.task-card {
  background: var(--gw-surface);
  border-radius: var(--gw-radius);
  box-shadow: var(--gw-card-shadow);
  padding: 16px;
  min-width: 200px;
  cursor: pointer;
  transition: box-shadow 0.15s, transform 0.1s;
  display: flex;
  flex-direction: column;
  gap: 10px;
  border: 1px solid transparent;
}

.task-card:hover {
  box-shadow: var(--shadow-lg);
  transform: translateY(-2px);
}

.task-card.favorite {
  background: var(--color-secondary-container);
  border-color: var(--color-secondary);
}

.task-card.archived {
  background: var(--color-surface-high);
  border-color: var(--color-outline-dim);
  opacity: 0.85;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.dept-badge {
  display: inline-block;
  background: var(--gw-primary-light);
  color: var(--gw-primary);
  border-radius: 20px;
  padding: 3px 10px;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 160px;
}

.favorite-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 4px;
  color: var(--gw-text-secondary);
  border-radius: 6px;
  display: flex;
  align-items: center;
  transition: color 0.15s;
  flex-shrink: 0;
}

.favorite-btn:hover {
  color: var(--color-secondary);
}

.favorite-btn .material-symbols-outlined {
  font-size: 20px;
}

.favorite-btn .material-symbols-outlined.filled {
  color: var(--color-secondary);
  font-variation-settings: 'FILL' 1;
}

.task-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--gw-text);
  margin: 0;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.task-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.meta-row {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--gw-text-secondary);
}

.meta-row .material-symbols-outlined {
  font-size: 14px;
  flex-shrink: 0;
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 4px;
}

.units-indicator {
  display: flex;
  align-items: center;
}

.units-indicator .material-symbols-outlined {
  font-size: 16px;
  color: var(--gw-accent);
}

.active-users {
  display: flex;
  align-items: center;
}

.active-avatar {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid var(--gw-surface);
  overflow: hidden;
  margin-left: -5px;
  flex-shrink: 0;
  box-shadow: 0 0 0 1px var(--gw-border);
}

.active-avatar:first-child {
  margin-left: 0;
}

.active-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.active-avatar-more {
  background: var(--gw-primary-light);
  color: var(--gw-primary);
  font-size: 9px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: visible;
}
</style>
