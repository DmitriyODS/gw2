<template>
  <div class="unit-item" :class="{ edited: unit.is_edited }">
    <div
      class="unit-bar"
      :class="{ active: isActive }"
    ></div>

    <div class="unit-main" @click="isExpanded = !isExpanded">
      <div class="unit-info">
        <span class="unit-name">{{ unit.name }}</span>
        <span v-if="unit.unit_type?.name" class="unit-type-badge">{{ unit.unit_type.name }}</span>
      </div>
      <span class="unit-duration">{{ formatDuration(unit.datetime_start, unit.datetime_end) }}</span>
      <span class="material-symbols-outlined chevron-icon">
        {{ isExpanded ? 'expand_less' : 'expand_more' }}
      </span>
    </div>

    <transition name="expand">
      <div v-if="isExpanded" class="unit-details">
        <div class="unit-user">{{ unit.user?.fio || '—' }}</div>
        <div class="unit-dates">
          <span>Начат: {{ formatFullDate(unit.datetime_start) }}</span>
          <span class="dates-sep">|</span>
          <span>{{ unit.datetime_end ? `Окончен: ${formatFullDate(unit.datetime_end)}` : 'В работе' }}</span>
        </div>
        <div class="unit-actions">
          <button v-if="canEdit" class="icon-btn" @click.stop="$emit('edit', unit)" title="Редактировать">
            <span class="material-symbols-outlined">edit</span>
          </button>
          <button v-if="canDelete" class="icon-btn danger" @click.stop="$emit('delete', unit)" title="Удалить">
            <span class="material-symbols-outlined">delete</span>
          </button>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useAuthStore } from '@/stores/auth.js'

const props = defineProps({
  unit: {
    type: Object,
    required: true
  }
})

defineEmits(['edit', 'delete'])

const { isAtLeast } = usePermission()
const authStore = useAuthStore()

const isExpanded = ref(false)

const isActive = computed(() => !props.unit.datetime_end)

const isOwnUnit = computed(() => props.unit.user_id === authStore.user?.id)

const canEdit = computed(() => isOwnUnit.value || isAtLeast(ROLES.MANAGER))

const canDelete = computed(() => isOwnUnit.value || isAtLeast(ROLES.MANAGER))

function formatDuration(start, end) {
  const ms = end ? new Date(end) - new Date(start) : Date.now() - new Date(start)
  const totalMin = Math.floor(ms / 60000)
  const h = Math.floor(totalMin / 60)
  const m = totalMin % 60
  return h > 0 ? `${h} ч ${m} мин` : `${totalMin} мин`
}

function formatFullDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
.unit-item {
  position: relative;
  display: flex;
  flex-direction: column;
  border-radius: 12px;
  background: var(--color-surface);
  border: 1px solid var(--gw-border);
  overflow: hidden;
  margin-bottom: 8px;
}

.unit-item.edited {
  background: var(--color-warning-container);
  border-color: color-mix(in oklch, var(--color-warning) 40%, var(--color-outline-dim));
}

.unit-bar {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 5px;
  background: var(--gw-border);
}

.unit-bar.active {
  background: var(--gw-accent);
}

.unit-main {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px 12px 20px;
  cursor: pointer;
  user-select: none;
}

.unit-main:hover {
  background: color-mix(in oklch, var(--color-primary) 4%, transparent);
}

.unit-info {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.unit-name {
  font-size: 14px;
  font-weight: 700;
  color: var(--gw-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.unit-type-badge {
  border: 1px solid var(--gw-border);
  border-radius: 20px;
  padding: 2px 10px;
  font-size: 12px;
  color: var(--gw-text-secondary);
  white-space: nowrap;
  flex-shrink: 0;
}

.unit-duration {
  font-size: 13px;
  color: var(--gw-text-secondary);
  white-space: nowrap;
  flex-shrink: 0;
}

.chevron-icon {
  font-size: 20px;
  color: var(--gw-text-secondary);
  flex-shrink: 0;
}

.unit-details {
  padding: 10px 14px 14px 20px;
  border-top: 1px solid var(--gw-border);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.unit-user {
  font-size: 13px;
  font-weight: 600;
  color: var(--gw-text);
}

.unit-dates {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--gw-text-secondary);
  flex-wrap: wrap;
}

.dates-sep {
  opacity: 0.4;
}

.unit-actions {
  display: flex;
  gap: 6px;
  justify-content: flex-end;
  margin-top: 2px;
}

.icon-btn {
  background: none;
  border: 1px solid var(--gw-border);
  border-radius: 6px;
  padding: 4px 8px;
  cursor: pointer;
  color: var(--gw-text-secondary);
  display: flex;
  align-items: center;
  transition: background 0.12s, color 0.12s;
}

.icon-btn:hover {
  background: var(--gw-primary-light);
  color: var(--gw-primary);
  border-color: var(--gw-primary);
}

.icon-btn.danger:hover {
  background: var(--color-error-container);
  color: var(--color-error);
  border-color: color-mix(in oklch, var(--color-error) 30%, var(--color-outline-dim));
}

.icon-btn .material-symbols-outlined {
  font-size: 16px;
}

/* Анимация разворачивания */
.expand-enter-active,
.expand-leave-active {
  transition: all 0.2s ease;
  overflow: hidden;
}

.expand-enter-from,
.expand-leave-to {
  max-height: 0;
  opacity: 0;
}

.expand-enter-to,
.expand-leave-from {
  max-height: 200px;
  opacity: 1;
}
</style>
