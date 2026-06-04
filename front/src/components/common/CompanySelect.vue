<template>
  <!-- Селектор активной компании для Администратора системы. У обычных ролей
       компания фиксирована (по auth.companyId), и этот компонент рендерит
       статичный чип. Стилистически — единый PrimeVue Select, как и все
       остальные выпадашки в проекте. -->
  <Select
    v-if="!fixed"
    :model-value="companies.activeCompanyId"
    :options="options"
    option-label="name"
    option-value="id"
    :placeholder="placeholder"
    :class="['company-select', { compact }]"
    show-clear
    :filter="companies.items.length > 6"
    filter-placeholder="Поиск компании…"
    scroll-height="320px"
    empty-message="Компании не загружены"
    empty-filter-message="Ничего не найдено"
    @update:model-value="onChange"
    @show="emit('show')"
    @hide="emit('hide')"
  >
    <template #value="slotProps">
      <span class="company-value">
        <span class="material-symbols-outlined company-icon">domain</span>
        <span class="company-value-label">
          {{ labelOf(slotProps.value) || placeholder }}
        </span>
      </span>
    </template>
    <template #option="slotProps">
      <span class="company-option">
        <span class="material-symbols-outlined company-icon">domain</span>
        <span class="company-option-label">{{ slotProps.option.name }}</span>
      </span>
    </template>
  </Select>

  <div v-else class="company-chip" :title="companyLabel">
    <span class="material-symbols-outlined company-icon">domain</span>
    <span class="company-chip-label">{{ companyLabel }}</span>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import Select from 'primevue/select'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'

const props = defineProps({
  compact: { type: Boolean, default: false },
  placeholder: { type: String, default: 'Все компании' },
})

// show/hide прокидываем наверх, чтобы родитель мог удержать своё состояние,
// пока открыта overlay-выпадашка (см. AppSidebar — он сворачивается на
// mouseleave, но не должен делать этого, пока курсор в выпадашке).
const emit = defineEmits(['show', 'hide'])

const auth = useAuthStore()
const companies = useCompaniesStore()

const fixed = computed(() => auth.companyId != null)
const companyLabel = computed(() => auth.companyName || 'Без компании')

const options = computed(() => companies.items)

function labelOf(id) {
  if (id == null) return null
  const c = companies.items.find((x) => x.id === id)
  return c?.name ?? null
}

onMounted(() => {
  if (!fixed.value) companies.load()
})

function onChange(value) {
  // null приходит при show-clear → «Все компании»
  companies.setActive(value ?? null)
}
</script>

<style scoped>
/* Внутренние подписи value/option — общие для обоих слотов. */
.company-value,
.company-option {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.company-icon {
  font-size: 18px;
  opacity: 0.75;
  flex-shrink: 0;
}

.company-value-label,
.company-option-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* PrimeVue Select под единый стиль выпадашек проекта (как dept-select). */
:deep(.company-select.p-select) {
  font-size: 13px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  border-color: transparent;
  min-width: 200px;
  max-width: 280px;
  font-weight: 600;
}

:deep(.company-select.p-select:hover) {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

:deep(.company-select .p-select-label) {
  padding: 8px 12px;
  display: flex;
  align-items: center;
}

:deep(.company-select.compact.p-select) {
  font-size: 12px;
  min-width: 180px;
  max-width: 240px;
}

:deep(.company-select.compact .p-select-label) {
  padding: 6px 10px;
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
