<template>
  <Teleport to="body">
    <div class="cl-overlay" @click.self="$emit('close')">
      <div class="cl-modal">

        <!-- Хедер (не скроллится) -->
        <div class="cl-header">
          <span class="material-symbols-outlined cl-header-icon">new_releases</span>
          <span class="cl-header-title">Что нового</span>
          <span v-if="currentVersion" class="cl-current-version">Версия: {{ currentVersion }}</span>
          <button class="cl-close" @click="$emit('close')" title="Закрыть">
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>

        <!-- Скроллируемое тело -->
        <div class="cl-scroll">
          <div v-if="loading" class="cl-loading">
            <span class="material-symbols-outlined spinning">progress_activity</span>
          </div>

          <div v-else-if="error" class="cl-error">
            <span class="material-symbols-outlined">error_outline</span>
            Не удалось загрузить изменения
          </div>

          <template v-else>
            <div v-for="ver in versions" :key="ver.version" class="cl-version">

              <!-- Заголовок версии -->
              <div class="cl-version-top">
                <span class="cl-badge">v{{ ver.version }}</span>
                <span class="cl-date">{{ formatDate(ver.date) }}</span>
              </div>
              <h2 class="cl-title">{{ ver.title }}</h2>
              <p v-if="ver.description" class="cl-desc">{{ ver.description }}</p>

              <!-- Группы -->
              <div class="cl-groups">
                <div
                  v-for="group in groupChanges(ver.changes)"
                  :key="group.type"
                  class="cl-group"
                >
                  <!-- Цветной чип-заголовок группы -->
                  <div class="cl-chip" :class="`cl-chip--${group.type}`">
                    <span class="material-symbols-outlined cl-chip-icon">{{ groupMeta[group.type]?.icon }}</span>
                    <span class="cl-chip-label">{{ groupMeta[group.type]?.label }}</span>
                    <span class="cl-chip-count">{{ group.items.length }}</span>
                  </div>

                  <!-- Пункты -->
                  <ul class="cl-items">
                    <li
                      v-for="(change, i) in group.items"
                      :key="i"
                      class="cl-item"
                      :class="`cl-item--${group.type}`"
                    >
                      {{ change.text }}
                    </li>
                  </ul>
                </div>
              </div>

            </div>
          </template>
        </div>

      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { changelogApi } from '@/api/changelog.js'

defineEmits(['close'])

const loading  = ref(true)
const error    = ref(false)
const versions = ref([])

const currentVersion = computed(() => versions.value[0]?.version ?? null)

const groupMeta = {
  new:      { icon: 'add_circle',             label: 'Добавили'  },
  improved: { icon: 'upgrade',                label: 'Улучшили'  },
  fixed:    { icon: 'bug_report',             label: 'Исправили' },
  changed:  { icon: 'published_with_changes', label: 'Изменили'  },
  removed:  { icon: 'remove_circle',          label: 'Убрали'    },
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
/* ── Оверлей ──────────────────────────────────────────────────── */
.cl-overlay {
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

/* ── Модалка ──────────────────────────────────────────────────── */
.cl-modal {
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  width: 560px;
  max-width: 100%;
  max-height: 86vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* ── Хедер (не скроллится) ────────────────────────────────────── */
.cl-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 18px 20px 16px;
  border-bottom: 1px solid var(--color-outline-dim);
  flex-shrink: 0;
}

.cl-header-icon {
  font-size: 22px;
  color: var(--color-primary);
}

.cl-header-title {
  flex: 1;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text);
}

.cl-current-version {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-dim);
  background: var(--color-surface-high);
  padding: 3px 10px;
  border-radius: var(--radius-full);
  white-space: nowrap;
  margin-right: 4px;
}

.cl-close {
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
.cl-close:hover {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.cl-close .material-symbols-outlined { font-size: 20px; }

/* ── Скролл-область ───────────────────────────────────────────── */
.cl-scroll {
  flex: 1;
  min-height: 0;          /* ← критично для скролла внутри flex */
  overflow-y: auto;
  padding: 24px 24px 32px;
}

/* ── Версия ───────────────────────────────────────────────────── */
.cl-version-top {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.cl-badge {
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 11px;
  font-weight: 800;
  padding: 3px 12px;
  border-radius: var(--radius-full);
  letter-spacing: 0.5px;
  text-transform: uppercase;
}

.cl-date {
  font-size: 13px;
  color: var(--color-text-dim);
}

.cl-title {
  font-size: 26px;
  font-weight: 800;
  color: var(--color-text);
  line-height: 1.25;
  letter-spacing: -0.3px;
  margin-bottom: 12px;
}

.cl-desc {
  font-size: 14px;
  color: var(--color-text-dim);
  line-height: 1.7;
  padding-bottom: 4px;
}

/* ── Группы ───────────────────────────────────────────────────── */
.cl-groups {
  display: flex;
  flex-direction: column;
  gap: 20px;
  margin-top: 24px;
}

.cl-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* ── Чип-заголовок ────────────────────────────────────────────── */
.cl-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px 5px 8px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 700;
  align-self: flex-start;
}
.cl-chip-icon { font-size: 16px; }
.cl-chip-label { letter-spacing: 0.3px; }
.cl-chip-count {
  font-size: 11px;
  font-weight: 800;
  background: color-mix(in oklch, currentColor 20%, transparent);
  min-width: 20px;
  height: 20px;
  padding: 0 5px;
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-left: 2px;
}

.cl-chip--new      { background: var(--color-success-container);   color: var(--color-on-success-container);   }
.cl-chip--improved { background: var(--color-tertiary-container);  color: var(--color-on-tertiary-container);  }
.cl-chip--fixed    { background: var(--color-warning-container);   color: var(--color-on-warning-container);   }
.cl-chip--changed  { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.cl-chip--removed  { background: var(--color-error-container);     color: var(--color-on-error-container);     }

/* ── Пункты изменений ─────────────────────────────────────────── */
.cl-items {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 1px;
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--color-outline-dim);   /* gap между строками через bg контейнера */
}

.cl-item {
  font-size: 14px;
  color: var(--color-text);
  line-height: 1.6;
  padding: 10px 14px 10px 16px;
  background: var(--color-surface-low);
  position: relative;
}

/* Цветная левая полоска через псевдоэлемент */
.cl-item::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  border-radius: 0 2px 2px 0;
}

.cl-item--new::before      { background: var(--color-success);   }
.cl-item--improved::before { background: var(--color-tertiary);  }
.cl-item--fixed::before    { background: var(--color-warning);   }
.cl-item--changed::before  { background: var(--color-secondary); }
.cl-item--removed::before  { background: var(--color-error);     }

/* ── Загрузка / ошибка ────────────────────────────────────────── */
.cl-loading,
.cl-error {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 60px 0;
  color: var(--color-text-dim);
  font-size: 14px;
}

.spinning { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

/* ── Мобильный: bottom sheet ──────────────────────────────────── */
@media (max-width: 600px) {
  .cl-overlay {
    padding: 0;
    align-items: flex-end;
  }

  .cl-modal {
    width: 100%;
    max-height: 90vh;
    border-radius: var(--radius-xl) var(--radius-xl) 0 0;
  }

  .cl-title {
    font-size: 20px;
  }

  .cl-scroll {
    padding: 20px 16px 28px;
  }
}
</style>
