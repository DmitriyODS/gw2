<template>
  <!-- Универсальная таблица проекта: PrimeVue DataTable, обёрнутый в
       flex-контейнер с внутренним скроллом (scrollHeight=flex), стилизованный
       только семантическими токенами. Колонки задаются через слот по умолчанию
       (используются обычные <Column> из PrimeVue). -->
  <div class="app-table" :class="{ 'app-table-bordered': bordered }">
    <DataTable
      :value="value"
      :loading="loading"
      :data-key="dataKey"
      :sort-field="sortField"
      :sort-order="sortOrder"
      :scrollable="true"
      scroll-height="flex"
      :row-hover="rowHover"
      :striped-rows="false"
      :show-gridlines="false"
      :pt="ptOverrides"
      size="small"
      @row-click="onRowClick"
      @sort="onSort"
      @row-reorder="onRowReorder"
      :row-class="rowClass"
      :reorderable-columns="false"
    >
      <template #empty>
        <slot name="empty">
          <div class="app-table-empty">
            <span class="material-symbols-outlined">inbox</span>
            <span>{{ emptyMessage }}</span>
          </div>
        </slot>
      </template>
      <template #loading>
        <div class="app-table-loading">
          <ProgressSpinner style="width: 36px; height: 36px" stroke-width="3" />
        </div>
      </template>
      <slot />
    </DataTable>
  </div>
</template>

<script setup>
import DataTable from 'primevue/datatable'
import ProgressSpinner from 'primevue/progressspinner'

const props = defineProps({
  value: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  dataKey: { type: String, default: 'id' },
  sortField: { type: String, default: null },
  sortOrder: { type: Number, default: null },
  emptyMessage: { type: String, default: 'Ничего не найдено' },
  rowHover: { type: Boolean, default: true },
  bordered: { type: Boolean, default: true },
  rowClass: { type: Function, default: null },
})

const emit = defineEmits(['row-click', 'sort', 'row-reorder', 'update:sortField', 'update:sortOrder'])

function onRowClick(e) { emit('row-click', e) }
function onSort(e) {
  emit('update:sortField', e.sortField)
  emit('update:sortOrder', e.sortOrder)
  emit('sort', e)
}
function onRowReorder(e) { emit('row-reorder', e) }

const ptOverrides = {
  wrapper: { class: 'app-table-wrapper' },
}
</script>

<style scoped>
.app-table {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--acrylic-card-bg);
  border-radius: var(--radius-xl);
  overflow: hidden;
}
.app-table-bordered {
  border: 1px solid var(--color-outline-dim);
}

.app-table :deep(.p-datatable) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: transparent;
  color: var(--color-text);
}
.app-table :deep(.p-datatable-wrapper),
.app-table :deep(.app-table-wrapper) {
  flex: 1;
  min-height: 0;
  overflow: auto;
  scrollbar-color: var(--color-outline) transparent;
  scrollbar-width: thin;
}
.app-table :deep(.p-datatable-wrapper::-webkit-scrollbar) {
  width: 10px;
  height: 10px;
}
.app-table :deep(.p-datatable-wrapper::-webkit-scrollbar-thumb) {
  background: var(--color-outline-dim);
  border-radius: var(--radius-full);
  border: 2px solid var(--color-surface);
}
.app-table :deep(.p-datatable-wrapper::-webkit-scrollbar-thumb:hover) {
  background: var(--color-outline);
}

.app-table :deep(.p-datatable-table) {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
}

.app-table :deep(.p-datatable-thead) {
  position: sticky;
  top: 0;
  z-index: 2;
}
.app-table :deep(.p-datatable-thead > tr > th) {
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding: 24px 22px !important;
  height: auto;
  min-height: 64px;
  border: none;
  border-bottom: 1px solid var(--color-outline-dim);
  text-align: left;
  white-space: nowrap;
  transition: background .14s, color .14s;
  box-sizing: border-box;
}
.app-table :deep(.p-datatable-thead > tr > th:first-child) {
  padding-left: 26px !important;
}
.app-table :deep(.p-datatable-thead > tr > th:last-child) {
  padding-right: 26px !important;
}
/* PrimeVue 4 оборачивает содержимое заголовка во внутренние блоки —
   обнуляем их paddings, чтобы наши паддинги на <th> были единственным
   источником вертикальных отступов. */
.app-table :deep(.p-datatable-thead .p-datatable-column-header-content),
.app-table :deep(.p-datatable-thead .p-column-header-content) {
  padding: 0;
  margin: 0;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.app-table :deep(.p-datatable-thead .p-datatable-column-title),
.app-table :deep(.p-datatable-thead .p-column-title) {
  padding: 0;
  line-height: 1.2;
}
.app-table :deep(.p-datatable-thead > tr > th.p-sortable-column:hover) {
  background: color-mix(in oklch, var(--color-surface-high) 70%, var(--color-primary));
  color: var(--color-text);
}
.app-table :deep(.p-datatable-thead > tr > th.p-highlight) {
  color: var(--color-primary);
}
.app-table :deep(.p-sortable-column-icon) {
  color: var(--color-text-dim);
  margin-left: 4px;
  font-size: 14px;
}
.app-table :deep(.p-sortable-column.p-highlight .p-sortable-column-icon) {
  color: var(--color-primary);
}

.app-table :deep(.p-datatable-tbody > tr) {
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  outline: none;
  transition: background .14s;
}
.app-table :deep(.p-datatable-tbody > tr > td) {
  padding: 18px 22px !important;
  border: none;
  border-bottom: 1px solid var(--color-outline-dim);
  font-size: 13.5px;
  vertical-align: middle;
  box-sizing: border-box;
}
.app-table :deep(.p-datatable-tbody > tr > td:first-child) {
  padding-left: 26px !important;
}
.app-table :deep(.p-datatable-tbody > tr > td:last-child) {
  padding-right: 26px !important;
}
.app-table :deep(.p-datatable-tbody > tr:last-child > td) {
  border-bottom: none;
}
.app-table :deep(.p-datatable-tbody > tr.p-row-hover),
.app-table :deep(.p-datatable-tbody > tr:hover) {
  background: var(--color-surface-high);
}
.app-table :deep(.p-datatable-tbody > tr.p-highlight) {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.app-table :deep(.p-datatable-tbody > tr:focus-visible) {
  box-shadow: inset 2px 0 0 var(--color-primary);
}

.app-table :deep(.p-datatable-tbody > tr.row-disabled > td) {
  opacity: 0.55;
}

.app-table :deep(.p-datatable-tbody > tr.row-clickable) {
  cursor: pointer;
}

/* PrimeVue 4 row reorder handle: даём ему вид «drag-grip» из проекта. */
.app-table :deep(.p-datatable-reorderable-row-handle),
.app-table :deep(.p-row-reorder-icon) {
  cursor: grab;
  color: var(--color-text-dim);
  font-size: 18px;
  opacity: 0.5;
  transition: opacity .12s;
}
.app-table :deep(.p-datatable-tbody > tr:hover .p-datatable-reorderable-row-handle),
.app-table :deep(.p-datatable-tbody > tr:hover .p-row-reorder-icon) {
  opacity: 1;
}

/* Loading-плёнка PrimeVue. */
.app-table :deep(.p-datatable-loading-overlay) {
  background: color-mix(in oklch, var(--color-surface) 70%, transparent);
  -webkit-backdrop-filter: blur(2px);
  backdrop-filter: blur(2px);
}

.app-table-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 48px 20px;
  color: var(--color-text-dim);
  font-size: 14px;
}
.app-table-empty .material-symbols-outlined {
  font-size: 36px;
  opacity: 0.6;
}
.app-table-loading {
  display: grid;
  place-items: center;
  padding: 40px 0;
}
</style>
