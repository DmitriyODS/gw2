<template>
  <!-- Селектор активной компании для Администратора системы. У обычных ролей
       компания фиксирована (по auth.companyId), и этот компонент рендерит
       статичный чип. -->
  <div v-if="!fixed" class="company-select" :class="{ compact }">
    <span class="material-symbols-outlined company-icon">domain</span>
    <select
      :value="companies.activeCompanyId ?? ''"
      class="company-native-select"
      @change="onChange"
      :aria-label="'Компания'"
    >
      <option value="">Все компании</option>
      <option v-for="c in companies.items" :key="c.id" :value="c.id">
        {{ c.name }}
      </option>
    </select>
    <span class="material-symbols-outlined chevron">expand_more</span>
  </div>

  <div v-else class="company-chip" :title="companyLabel">
    <span class="material-symbols-outlined company-icon">domain</span>
    <span class="company-chip-label">{{ companyLabel }}</span>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'

const props = defineProps({
  compact: { type: Boolean, default: false },
})

const auth = useAuthStore()
const companies = useCompaniesStore()

const fixed = computed(() => auth.companyId != null)
const companyLabel = computed(() => auth.companyName || 'Без компании')

onMounted(() => {
  if (!fixed.value) companies.load()
})

function onChange(e) {
  const v = e.target.value
  companies.setActive(v === '' ? null : Number(v))
}
</script>

<style scoped>
.company-select {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  position: relative;
  height: 36px;
  padding: 0 12px 0 10px;
  border-radius: var(--radius-full, 999px);
  background: var(--color-surface-high, var(--gw-surface));
  border: 1px solid var(--color-outline-variant, var(--gw-border));
  color: var(--color-text);
  font-weight: 600;
  font-size: 13px;
  transition: background 0.15s, border-color 0.15s;
  cursor: pointer;
}

.company-select:hover {
  background: var(--color-primary-container);
  border-color: var(--color-primary);
  color: var(--color-on-primary-container);
}

.company-select.compact { height: 32px; padding: 0 10px 0 8px; font-size: 12px; }

.company-icon, .chevron { font-size: 18px; line-height: 1; opacity: 0.7; pointer-events: none; }

.company-native-select {
  appearance: none;
  -webkit-appearance: none;
  background: transparent;
  border: none;
  outline: none;
  font: inherit;
  color: inherit;
  padding-right: 4px;
  cursor: pointer;
  max-width: 240px;
}

.company-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  padding: 0 14px;
  border-radius: var(--radius-full, 999px);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-weight: 600;
  font-size: 13px;
  max-width: 240px;
}

.company-chip-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
