<template>
  <Teleport to="body">
    <div class="cl-overlay" @click.self="$emit('close')">
      <div class="cl-modal">
        <div class="cl-header">
          <h2 class="cl-title">Что нового</h2>
          <button class="cl-close" @click="$emit('close')" title="Закрыть">
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>

        <div class="cl-body">
          <div v-if="loading" class="cl-loading">
            <span class="material-symbols-outlined spinning">progress_activity</span>
          </div>

          <div v-else-if="error" class="cl-error">
            Не удалось загрузить список изменений
          </div>

          <template v-else>
            <div
              v-for="ver in versions"
              :key="ver.version"
              class="cl-version"
            >
              <div class="cl-version-header">
                <span class="cl-badge">v{{ ver.version }}</span>
                <span class="cl-date">{{ formatDate(ver.date) }}</span>
                <span class="cl-version-title">{{ ver.title }}</span>
              </div>

              <ul class="cl-changes">
                <li
                  v-for="(change, i) in ver.changes"
                  :key="i"
                  class="cl-change"
                  :class="`cl-change--${change.type}`"
                >
                  <span class="cl-change-icon material-symbols-outlined">
                    {{ icons[change.type] || 'fiber_manual_record' }}
                  </span>
                  <span class="cl-change-text">{{ change.text }}</span>
                </li>
              </ul>
            </div>
          </template>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { changelogApi } from '@/api/changelog.js'

defineEmits(['close'])

const loading = ref(true)
const error = ref(false)
const versions = ref([])

const icons = {
  new:      'add_circle',
  improved: 'upgrade',
  fixed:    'bug_report',
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
.cl-overlay {
  position: fixed;
  inset: 0;
  background: var(--color-scrim);
  backdrop-filter: blur(4px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cl-modal {
  background: var(--gw-surface);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xl);
  width: 520px;
  max-width: 95vw;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.cl-header {
  display: flex;
  align-items: center;
  padding: 20px 24px 16px;
  border-bottom: 1px solid var(--gw-border);
  gap: 12px;
}

.cl-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--gw-text);
  flex: 1;
}

.cl-close {
  width: 36px;
  height: 36px;
  border: none;
  background: transparent;
  color: var(--gw-text-secondary);
  cursor: pointer;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, color 0.15s;
}

.cl-close:hover {
  background: var(--gw-primary-light);
  color: var(--gw-primary);
}

.cl-body {
  overflow-y: auto;
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 28px;
}

.cl-loading,
.cl-error {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 0;
  color: var(--gw-text-secondary);
  font-size: 14px;
  gap: 8px;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.cl-version-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}

.cl-badge {
  background: var(--gw-primary);
  color: var(--color-on-primary);
  font-size: 12px;
  font-weight: 700;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  letter-spacing: 0.3px;
}

.cl-date {
  font-size: 12px;
  color: var(--gw-text-secondary);
}

.cl-version-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--gw-text);
}

.cl-changes {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cl-change {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 14px;
  color: var(--gw-text);
  line-height: 1.5;
}

.cl-change-icon {
  font-size: 16px;
  flex-shrink: 0;
  margin-top: 1px;
}

.cl-change--new    .cl-change-icon { color: var(--color-primary); }
.cl-change--improved .cl-change-icon { color: var(--color-secondary); }
.cl-change--fixed  .cl-change-icon { color: var(--color-error); }
</style>
