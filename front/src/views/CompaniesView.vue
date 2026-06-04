<template>
  <div class="cmp-view">
    <header class="cmp-header">
      <h1 class="cmp-title">Компании</h1>
      <p class="cmp-subtitle">Управление компаниями платформы — раздел дорабатывается во 2-м этапе v3.0.</p>
    </header>

    <div v-if="loading" class="cmp-loading">
      <ProgressSpinner />
    </div>
    <div v-else class="cmp-table-wrap">
      <table class="cmp-table">
        <thead>
          <tr>
            <th>Название</th>
            <th>Дата создания</th>
            <th>Сотрудников</th>
            <th>Задач</th>
            <th>Руководитель</th>
            <th>Статус</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in companies.items" :key="c.id">
            <td><strong>{{ c.name }}</strong></td>
            <td>{{ fmtDate(c.created_at) }}</td>
            <td>{{ c.employees_count }}</td>
            <td>{{ c.tasks_count }}</td>
            <td>{{ c.director?.fio || '—' }}</td>
            <td>
              <span class="status-chip" :class="{ on: c.is_active }">
                {{ c.is_active ? 'Активна' : 'Отключена' }}
              </span>
            </td>
          </tr>
          <tr v-if="!companies.items.length">
            <td colspan="6" class="cmp-empty">Нет компаний</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { onMounted, computed } from 'vue'
import { useCompaniesStore } from '@/stores/companies.js'
import ProgressSpinner from 'primevue/progressspinner'

const companies = useCompaniesStore()
const loading = computed(() => companies.loading && !companies.loaded)

onMounted(() => companies.load(true))

function fmtDate(s) {
  if (!s) return '—'
  const d = new Date(s)
  return d.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
}
</script>

<style scoped>
.cmp-view { padding: 24px; max-width: 1200px; margin: 0 auto; }

.cmp-header { margin-bottom: 24px; }
.cmp-title { font-size: 28px; font-weight: 700; margin: 0 0 6px; color: var(--color-text); }
.cmp-subtitle { font-size: 14px; color: var(--gw-text-secondary); margin: 0; }

.cmp-loading { display: grid; place-items: center; min-height: 200px; }

.cmp-table-wrap {
  background: var(--color-surface-high, var(--gw-surface));
  border-radius: var(--radius-lg, 16px);
  overflow: hidden;
  border: 1px solid var(--color-outline-variant, var(--gw-border));
}

.cmp-table { width: 100%; border-collapse: collapse; }

.cmp-table thead {
  background: var(--color-surface-container, var(--gw-bg));
}
.cmp-table th, .cmp-table td {
  padding: 14px 16px;
  text-align: left;
  font-size: 14px;
  border-bottom: 1px solid var(--color-outline-variant, var(--gw-border));
}
.cmp-table th { font-weight: 600; color: var(--gw-text-secondary); }
.cmp-table tbody tr:last-child td { border-bottom: none; }
.cmp-table tbody tr:hover { background: var(--gw-primary-light); }

.cmp-empty { text-align: center; padding: 32px; color: var(--gw-text-secondary); }

.status-chip {
  display: inline-flex;
  padding: 4px 10px;
  border-radius: var(--radius-full, 999px);
  font-size: 12px;
  font-weight: 600;
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}
.status-chip.on { background: var(--color-success-container, var(--color-primary-container)); color: var(--color-on-primary-container); }
</style>
