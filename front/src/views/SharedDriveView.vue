<template>
  <!-- Файл или папка, открытые публичной ссылкой: код в адресе и есть доступ,
       поэтому страница работает и без входа в приложение. -->
  <div class="shared">
    <BrandLoader v-if="loading" block :size="64" />

    <EmptyState
      v-else-if="error"
      icon="link_off"
      title="Ссылка недоступна"
      subtitle="Её могли отозвать, а файл — удалить."
    />

    <template v-else>
      <header class="shared-head">
        <BrandWordmark />
        <h1 class="shared-title">{{ title }}</h1>
      </header>

      <!-- Одиночный файл: сразу показываем и даём скачать. -->
      <AppCard v-if="file" class="shared-card">
        <div class="file-row">
          <span class="material-symbols-outlined file-icon">{{ fileIcon(file.mime, file.name) }}</span>
          <span class="file-main">
            <strong>{{ file.name }}</strong>
            <span class="file-meta">{{ formatBytes(file.size) }}</span>
          </span>
          <AppButton label="Скачать" icon="download" variant="filled" @click="download" />
        </div>

        <img v-if="isImage" :src="downloadHref" :alt="file.name" class="preview">
      </AppCard>

      <!-- Папка: перечисляем содержимое; вложенные папки открываются той же
           ссылкой — отдельных кодов на потомков не заводим. -->
      <AppCard v-else class="shared-card">
        <ul class="list">
          <li v-for="f in folders" :key="`f${f.id}`" class="row">
            <span class="material-symbols-outlined row-icon">folder</span>
            <span class="row-name">{{ f.name }}</span>
          </li>
          <li v-for="f in files" :key="`d${f.id}`" class="row">
            <span class="material-symbols-outlined row-icon">{{ fileIcon(f.mime, f.name) }}</span>
            <span class="row-name">{{ f.name }}</span>
            <span class="row-size">{{ formatBytes(f.size) }}</span>
          </li>
        </ul>
        <EmptyState v-if="!folders.length && !files.length" size="sm" icon="folder_open" title="Папка пуста" />
      </AppCard>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import BrandLoader from '@/components/common/BrandLoader.vue'
import BrandWordmark from '@/components/common/BrandWordmark.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import * as api from '@/api/drive.js'
import { fileIcon, fileKind } from '@/utils/fileTypes.js'
import { formatBytes } from '@/utils/money.js'

const route = useRoute()
const code = route.params.code

const loading = ref(true)
const error = ref(false)
const file = ref(null)
const folder = ref(null)
const folders = ref([])
const files = ref([])

const title = computed(() => file.value?.name || folder.value?.name || '')
const isImage = computed(() => file.value && fileKind(file.value.mime, file.value.name) === 'image')
const downloadHref = computed(() => api.sharedDownloadURL(code))

function download() {
  window.open(downloadHref.value, '_blank')
}

onMounted(async () => {
  try {
    const res = await api.getShared(code)
    file.value = res.file || null
    folder.value = res.folder || null
    if (folder.value) {
      const listing = await api.getSharedList(code)
      folders.value = listing.folders || []
      files.value = listing.files || []
    }
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.shared {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 860px;
  margin: 0 auto;
  padding: 24px 16px;
}

.shared-head {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.shared-title {
  margin: 0;
  font-size: 1.3rem;
  overflow-wrap: anywhere;
}

.file-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.file-icon {
  font-size: 32px;
  color: var(--color-primary);
}

.file-main {
  flex: 1;
  min-width: min(200px, 100%);
  display: flex;
  flex-direction: column;
  overflow-wrap: anywhere;
}

.file-meta {
  font-size: 0.85rem;
  color: var(--color-text-dim);
}

.preview {
  max-width: 100%;
  border-radius: var(--radius-md);
}

.list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
}

.row:hover { background: var(--color-surface-variant); }

.row-icon { color: var(--color-text-dim); }

.row-name {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
}

.row-size {
  color: var(--color-text-dim);
  font-size: 0.82rem;
  font-variant-numeric: tabular-nums;
}
</style>
