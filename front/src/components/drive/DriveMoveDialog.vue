<template>
  <!-- Куда переместить: тот же обзор папок, что и в разделе, но без файлов —
       выбирают ведь папку. Заходим внутрь кликом, поднимаемся крошками. -->
  <AppDialog
    :model-value="true"
    title="Переместить"
    :subtitle="subtitle"
    size="sm"
    :busy="moving"
    :actions="[
      { kind: 'cancel', label: 'Отмена', disabled: moving },
      { kind: 'confirm', label: 'Переместить сюда', disabled: moving },
    ]"
    @confirm="apply"
    @cancel="$emit('close')"
    @update:model-value="(v) => !v && $emit('close')"
  >
    <Breadcrumbs :items="path" root-label="Мой диск" root-icon="cloud" @navigate="onCrumb" />

    <BrandLoader v-if="loading" block :size="48" :min-height="140" />

    <ul v-else-if="folders.length" class="mv-list">
      <li v-for="f in folders" :key="f.id">
        <button type="button" class="mv-item" :disabled="blocked.has(f.id)" @click="open(f)">
          <span class="material-symbols-outlined">folder</span>
          <span class="mv-name">{{ f.name }}</span>
          <span v-if="blocked.has(f.id)" class="mv-hint">нельзя</span>
          <span v-else class="material-symbols-outlined mv-chev">chevron_right</span>
        </button>
      </li>
    </ul>

    <p v-else class="mv-empty">Здесь нет вложенных папок — можно переместить прямо сюда.</p>
  </AppDialog>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import Breadcrumbs from '@/components/common/Breadcrumbs.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import * as api from '@/api/drive.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  /** Что переносим: [{ item, kind }]. */
  items: { type: Array, required: true },
})

const emit = defineEmits(['close', 'moved'])

const notif = useNotificationsStore()
const loading = ref(true)
const moving = ref(false)
const folders = ref([])
const path = ref([])
const current = ref(null) // null — корень

const subtitle = computed(() => (props.items.length === 1
  ? props.items[0].item.name
  : `Выбрано: ${props.items.length}`))

// Папку нельзя положить в саму себя: такой пункт показываем, но не даём выбрать.
const blocked = computed(() => new Set(
  props.items.filter((i) => i.kind === 'folder').map((i) => i.item.id),
))

async function load(folderId) {
  loading.value = true
  try {
    const data = await api.browse({ folder_id: folderId })
    folders.value = data.folders || []
    path.value = data.path || []
    current.value = data.folder?.id ?? null
  } catch (e) {
    notif.error(e.message || 'Не удалось открыть папку')
  } finally {
    loading.value = false
  }
}

function open(folder) {
  if (blocked.value.has(folder.id)) return
  load(folder.id)
}

function onCrumb(index) {
  load(index < 0 ? null : path.value[index]?.id ?? null)
}

async function apply() {
  moving.value = true
  try {
    for (const { item, kind } of props.items) {
      if (kind === 'folder') await api.moveFolder(item.id, current.value)
      else await api.moveFile(item.id, current.value)
    }
    emit('moved')
    emit('close')
  } catch (e) {
    notif.error(e.message || 'Не удалось переместить')
  } finally {
    moving.value = false
  }
}

onMounted(() => load(null))
</script>

<style scoped>
.mv-list {
  margin: 8px 0 0;
  padding: 0;
  list-style: none;
  max-height: 260px;
  overflow-y: auto;
}

.mv-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: none;
  color: var(--color-text);
  cursor: pointer;
  text-align: left;
}

.mv-item:hover:not(:disabled) { background: var(--color-surface-variant); }
.mv-item:disabled { opacity: 0.45; cursor: default; }

.mv-name {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
}

.mv-hint,
.mv-chev { color: var(--color-text-dim); font-size: 0.82rem; }

.mv-empty {
  margin: 12px 0 0;
  color: var(--color-text-dim);
}
</style>
