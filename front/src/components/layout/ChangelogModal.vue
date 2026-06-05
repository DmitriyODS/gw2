<template>
  <AppDialog
    model-value
    tone="primary"
    icon="new_releases"
    size="lg"
    title="Что нового"
    :subtitle="currentVersion ? `Текущая версия: ${currentVersion}` : ''"
    @update:model-value="(v) => !v && $emit('close')"
  >
    <div v-if="loading" class="cl-loading">
      <span class="material-symbols-outlined spinning">progress_activity</span>
    </div>

    <div v-else-if="error" class="cl-error">
      <span class="material-symbols-outlined">error_outline</span>
      Не удалось загрузить изменения
    </div>

    <template v-else>
      <div v-for="ver in versions" :key="ver.version" class="cl-version">
        <div class="cl-version-top">
          <span class="cl-badge">v{{ ver.version }}</span>
          <span class="cl-date">{{ formatDate(ver.date) }}</span>
        </div>
        <h2 class="cl-title">{{ ver.title }}</h2>
        <p v-if="ver.description" class="cl-desc">{{ ver.description }}</p>

        <div class="cl-groups">
          <div
            v-for="group in groupsOf(ver)"
            :key="group.type"
            class="cl-group"
          >
            <div class="cl-chip" :class="`cl-chip--${group.type}`">
              <span class="material-symbols-outlined cl-chip-icon">{{ groupMeta[group.type]?.icon }}</span>
              <span class="cl-chip-label">{{ groupMeta[group.type]?.label }}</span>
              <span class="cl-chip-count">{{ group.items.length }}</span>
            </div>

            <ul class="cl-items">
              <li
                v-for="(text, i) in group.items"
                :key="i"
                class="cl-item"
                :class="`cl-item--${group.type}`"
              >
                {{ text }}
              </li>
            </ul>
          </div>
        </div>
      </div>
    </template>
  </AppDialog>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { changelogApi } from '@/api/changelog.js'

defineEmits(['close'])

const loading  = ref(true)
const error    = ref(false)
const versions = ref([])

const currentVersion = computed(() => versions.value[0]?.version ?? null)

const groupMeta = {
  added:    { icon: 'add_circle', label: 'Добавили'  },
  improved: { icon: 'upgrade',    label: 'Улучшили'  },
  fixed:    { icon: 'bug_report', label: 'Исправили' },
}

const GROUP_ORDER = ['added', 'improved', 'fixed']

function groupsOf(ver) {
  return GROUP_ORDER
    .filter(t => Array.isArray(ver[t]) && ver[t].length)
    .map(t => ({ type: t, items: ver[t] }))
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
.cl-version + .cl-version {
  margin-top: 32px;
  padding-top: 28px;
  border-top: 1px solid var(--color-outline-dim);
}

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
  font-size: 24px;
  font-weight: 800;
  color: var(--color-text);
  line-height: 1.25;
  letter-spacing: -0.3px;
  margin: 0 0 12px;
}

.cl-desc {
  font-size: 14px;
  color: var(--color-text-dim);
  line-height: 1.7;
  margin: 0 0 4px;
}

.cl-groups {
  display: flex;
  flex-direction: column;
  gap: 18px;
  margin-top: 18px;
}

.cl-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

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

.cl-chip--added    { background: var(--color-success-container);  color: var(--color-on-success-container);  }
.cl-chip--improved { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }
.cl-chip--fixed    { background: var(--color-warning-container);  color: var(--color-on-warning-container);  }

.cl-items {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--color-outline-dim);
}

.cl-item {
  font-size: 14px;
  color: var(--color-text);
  line-height: 1.6;
  padding: 10px 14px 10px 16px;
  background: var(--color-surface-low);
  position: relative;
}

.cl-item::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  border-radius: 0 2px 2px 0;
}

.cl-item--added::before    { background: var(--color-success);  }
.cl-item--improved::before { background: var(--color-tertiary); }
.cl-item--fixed::before    { background: var(--color-warning);  }

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

@media (max-width: 600px) {
  .cl-title { font-size: 20px; }
}
</style>
