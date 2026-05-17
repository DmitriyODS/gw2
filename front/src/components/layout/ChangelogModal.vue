<template>
  <Teleport to="body">
    <div class="cl-overlay" @click.self="$emit('close')">
      <div class="cl-modal">

        <button class="cl-close" @click="$emit('close')" title="Закрыть">
          <span class="material-symbols-outlined">close</span>
        </button>

        <div v-if="loading" class="cl-loading">
          <span class="material-symbols-outlined spinning">progress_activity</span>
        </div>

        <div v-else-if="error" class="cl-error">
          <span class="material-symbols-outlined">error_outline</span>
          Не удалось загрузить список изменений
        </div>

        <template v-else>
          <div v-for="ver in versions" :key="ver.version">

            <!-- Hero-шапка -->
            <div class="cl-hero">
              <div class="cl-hero-meta">
                <span class="cl-badge">v{{ ver.version }}</span>
                <span class="cl-date">{{ formatDate(ver.date) }}</span>
              </div>
              <h2 class="cl-title">{{ ver.title }}</h2>
              <p v-if="ver.description" class="cl-desc">{{ ver.description }}</p>
            </div>

            <!-- Список изменений по группам -->
            <div class="cl-body">
              <div
                v-for="group in groupChanges(ver.changes)"
                :key="group.type"
                class="cl-group"
                :class="`cl-group--${group.type}`"
              >
                <div class="cl-group-head">
                  <span class="material-symbols-outlined cl-group-icon">{{ groupMeta[group.type]?.icon }}</span>
                  <span class="cl-group-label">{{ groupMeta[group.type]?.label }}</span>
                  <span class="cl-group-count">{{ group.items.length }}</span>
                </div>
                <ul class="cl-items">
                  <li v-for="(change, i) in group.items" :key="i" class="cl-item">
                    {{ change.text }}
                  </li>
                </ul>
              </div>
            </div>

          </div>
        </template>

      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { changelogApi } from '@/api/changelog.js'

defineEmits(['close'])

const loading = ref(true)
const error   = ref(false)
const versions = ref([])

const groupMeta = {
  new:      { icon: 'add_circle',              label: 'Добавили'  },
  improved: { icon: 'upgrade',                 label: 'Улучшили'  },
  fixed:    { icon: 'bug_report',              label: 'Исправили' },
  changed:  { icon: 'published_with_changes',  label: 'Изменили'  },
  removed:  { icon: 'remove_circle',           label: 'Убрали'    },
}

const GROUP_ORDER = ['new', 'improved', 'fixed', 'changed', 'removed']

function groupChanges(changes) {
  const map = {}
  for (const c of changes) {
    ;(map[c.type] ??= []).push(c)
  }
  return GROUP_ORDER.filter(t => map[t]).map(t => ({ type: t, items: map[t] }))
}

function formatDate(str) {
  if (!str) return ''
  return new Date(str).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' })
}

onMounted(async () => {
  try {
    const data = await changelogApi.get()
    versions.value = data.versions ?? []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
/* ── Оверлей ─────────────────────────────────────────────── */
.cl-overlay {
  position: fixed;
  inset: 0;
  background: var(--color-scrim);
  backdrop-filter: blur(6px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}

/* ── Модалка ─────────────────────────────────────────────── */
.cl-modal {
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  width: 600px;
  max-width: 100%;
  max-height: 88vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}

/* ── Кнопка закрытия ─────────────────────────────────────── */
.cl-close {
  position: absolute;
  top: 14px;
  right: 14px;
  z-index: 2;
  width: 36px;
  height: 36px;
  border: none;
  background: color-mix(in oklch, var(--color-primary-container) 60%, transparent);
  color: var(--color-on-primary-container);
  cursor: pointer;
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
}
.cl-close:hover {
  background: var(--color-primary-container);
}
.cl-close .material-symbols-outlined { font-size: 20px; }

/* ── Hero-шапка ──────────────────────────────────────────── */
.cl-hero {
  background: var(--color-primary-container);
  padding: 32px 28px 26px;
  flex-shrink: 0;
}

.cl-hero-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}

.cl-badge {
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 12px;
  font-weight: 800;
  padding: 4px 14px;
  border-radius: var(--radius-full);
  letter-spacing: 0.6px;
  text-transform: uppercase;
}

.cl-date {
  font-size: 13px;
  color: var(--color-on-primary-container);
  opacity: 0.65;
}

.cl-title {
  font-size: 22px;
  font-weight: 800;
  color: var(--color-on-primary-container);
  line-height: 1.3;
  margin-bottom: 12px;
}

.cl-desc {
  font-size: 14px;
  color: var(--color-on-primary-container);
  opacity: 0.8;
  line-height: 1.65;
}

/* ── Тело с группами ─────────────────────────────────────── */
.cl-body {
  overflow-y: auto;
  padding: 20px 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* ── Группа изменений ────────────────────────────────────── */
.cl-group {
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: var(--shadow-sm);
}

.cl-group-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
}

.cl-group-icon {
  font-size: 18px;
  flex-shrink: 0;
}

.cl-group-label {
  flex: 1;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.8px;
  text-transform: uppercase;
}

.cl-group-count {
  font-size: 11px;
  font-weight: 700;
  min-width: 22px;
  height: 22px;
  padding: 0 7px;
  border-radius: var(--radius-full);
  background: color-mix(in oklch, currentColor 18%, transparent);
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Цвета групп */
.cl-group--new .cl-group-head {
  background: var(--color-success-container);
  color: var(--color-on-success-container);
}
.cl-group--new .cl-items {
  background: color-mix(in oklch, var(--color-success-container) 35%, var(--color-surface));
  border-left: 3px solid var(--color-success);
}

.cl-group--improved .cl-group-head {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.cl-group--improved .cl-items {
  background: color-mix(in oklch, var(--color-tertiary-container) 35%, var(--color-surface));
  border-left: 3px solid var(--color-tertiary);
}

.cl-group--fixed .cl-group-head {
  background: var(--color-warning-container);
  color: var(--color-on-warning-container);
}
.cl-group--fixed .cl-items {
  background: color-mix(in oklch, var(--color-warning-container) 35%, var(--color-surface));
  border-left: 3px solid var(--color-warning);
}

.cl-group--changed .cl-group-head {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.cl-group--changed .cl-items {
  background: color-mix(in oklch, var(--color-secondary-container) 35%, var(--color-surface));
  border-left: 3px solid var(--color-secondary);
}

.cl-group--removed .cl-group-head {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}
.cl-group--removed .cl-items {
  background: color-mix(in oklch, var(--color-error-container) 35%, var(--color-surface));
  border-left: 3px solid var(--color-error);
}

/* ── Пункты изменений ────────────────────────────────────── */
.cl-items {
  list-style: none;
  padding: 0;
}

.cl-item {
  padding: 10px 16px;
  font-size: 14px;
  color: var(--color-text);
  line-height: 1.6;
  border-top: 1px solid var(--color-outline-dim);
}
.cl-item:first-child {
  border-top: none;
}

/* ── Состояния загрузки / ошибки ─────────────────────────── */
.cl-loading,
.cl-error {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 60px 24px;
  color: var(--color-text-dim);
  font-size: 14px;
}

.spinning {
  animation: spin 1s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

/* ── Мобильный: bottom sheet ─────────────────────────────── */
@media (max-width: 600px) {
  .cl-overlay {
    padding: 0;
    align-items: flex-end;
  }

  .cl-modal {
    width: 100%;
    max-height: 92vh;
    border-radius: var(--radius-xl) var(--radius-xl) 0 0;
  }

  .cl-hero {
    padding: 24px 20px 20px;
  }

  .cl-title {
    font-size: 18px;
  }

  .cl-body {
    padding: 16px 16px 20px;
  }
}
</style>
