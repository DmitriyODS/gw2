<template>
  <div class="stats-widget">
    <div class="widget-header">
      <h3>{{ title }}</h3>
      <button
        v-if="exportFn"
        class="export-btn"
        @click="handleExport"
        title="Скачать XLSX"
      >
        <span class="material-symbols-outlined">download</span>
      </button>
    </div>
    <div class="widget-body">
      <slot />
    </div>
  </div>
</template>

<script setup>
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  title: {
    type: String,
    required: true
  },
  exportFn: {
    type: Function,
    default: null
  }
})

const notif = useNotificationsStore()

async function handleExport() {
  if (!props.exportFn) return
  try {
    const response = await props.exportFn()
    let blob
    if (response instanceof Blob) {
      blob = response
    } else if (response && typeof response.blob === 'function') {
      blob = await response.blob()
    } else {
      blob = new Blob([response])
    }
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `export_${Date.now()}.xlsx`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (e) {
    notif.error(e.message || 'Ошибка экспорта')
  }
}
</script>

<style scoped>
.stats-widget {
  background: var(--gw-surface);
  border: 1px solid var(--gw-border);
  border-radius: var(--gw-radius);
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  box-shadow: var(--gw-card-shadow);
}

.widget-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.widget-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--gw-text);
}

.export-btn {
  background: none;
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  width: 34px;
  height: 34px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--gw-text-secondary);
  transition: background 0.15s, color 0.15s, border-color 0.15s;
  flex-shrink: 0;
}

.export-btn:hover {
  background: var(--gw-primary);
  border-color: var(--gw-primary);
  color: var(--color-on-primary);
}

.export-btn .material-symbols-outlined {
  font-size: 18px;
}

.widget-body {
  flex: 1;
}
</style>
