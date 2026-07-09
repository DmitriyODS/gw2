<template>
  <!-- Роль есть только у члена компании; у прочих (например, в платформенном
       списке супер-админа) роли нет — пилюлю не показываем. -->
  <span v-if="level || name" class="role-pill" :class="`lvl-${level || 1}`">
    {{ name || ROLE_NAMES[level] || 'Сотрудник' }}
  </span>
</template>

<script setup>
import { ROLE_NAMES } from '@/composables/usePermission.js'

defineProps({
  level: { type: Number, default: null },
  name: { type: String, default: '' },
})
</script>

<style scoped>
.role-pill {
  display: inline-flex;
  align-items: center;
  padding: 3px 12px;
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  line-height: 1.4;
}
.role-pill.lvl-1 { background: var(--color-surface-high); color: var(--color-text-dim); }
.role-pill.lvl-2 { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.role-pill.lvl-3 { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }
</style>
