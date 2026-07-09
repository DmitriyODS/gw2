<template>
  <AppDialog
    :model-value="modelValue"
    title="Поделиться заметкой" icon="share" size="md"
    :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
    @cancel="close" @update:model-value="(v) => !v && close()"
  >
    <div class="ns">
      <p class="ns-note">
        По ссылке заметку увидит любой — даже без входа в систему. Режим
        выбирается при создании: «чтение» или «чтение и редактирование».
        Ссылку можно отозвать в любой момент.
      </p>

      <div class="ns-create">
        <button class="ns-btn" :disabled="busy" @click="createLink('view')">
          <span class="material-symbols-outlined">visibility</span> Только чтение
        </button>
        <button class="ns-btn" :disabled="busy" @click="createLink('edit')">
          <span class="material-symbols-outlined">edit</span> Чтение и редактирование
        </button>
      </div>

      <div v-if="loading" class="ns-empty">Загрузка…</div>
      <ul v-else-if="shares.length" class="ns-shares">
        <li v-for="s in shares" :key="s.id" class="ns-share">
          <span class="chip-tint" :class="s.access === 'edit' ? 'chip-tint--warning' : 'chip-tint--primary'">
            <span class="material-symbols-outlined">{{ s.access === 'edit' ? 'edit' : 'visibility' }}</span>
            {{ s.access === 'edit' ? 'Правка' : 'Чтение' }}
          </span>
          <input class="ns-url" :value="shareUrl(s.code)" readonly @focus="$event.target.select()" />
          <button class="ns-icon" title="Копировать" @click="copy(s.code)">
            <span class="material-symbols-outlined">content_copy</span>
          </button>
          <a class="ns-icon" :href="shareUrl(s.code)" target="_blank" rel="noopener" title="Открыть">
            <span class="material-symbols-outlined">open_in_new</span>
          </a>
          <button class="ns-icon danger" title="Отозвать" @click="revoke(s.id)">
            <span class="material-symbols-outlined">delete</span>
          </button>
        </li>
      </ul>
      <p v-else class="ns-empty">Ссылок пока нет.</p>
    </div>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import * as api from '@/api/notes.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  noteId: { type: [Number, String, null], default: null },
})
const emit = defineEmits(['update:modelValue'])

const notif = useNotificationsStore()
const shares = ref([])
const loading = ref(false)
const busy = ref(false)

function shareUrl(code) { return `${location.origin}/note/${code}` }

watch(() => props.modelValue, (open) => { if (open && props.noteId != null) load() })

async function load() {
  loading.value = true
  try {
    const data = await api.getShares(props.noteId)
    shares.value = data.shares ?? []
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить ссылки')
  } finally {
    loading.value = false
  }
}

async function createLink(access) {
  busy.value = true
  try {
    const s = await api.createShare(props.noteId, access)
    shares.value.unshift(s)
    await copy(s.code)
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать ссылку')
  } finally {
    busy.value = false
  }
}

async function revoke(id) {
  try {
    await api.revokeShare(props.noteId, id)
    shares.value = shares.value.filter((s) => s.id !== id)
  } catch (e) {
    notif.error(e?.message || 'Не удалось отозвать ссылку')
  }
}

async function copy(code) {
  try { await navigator.clipboard.writeText(shareUrl(code)); notif.success('Ссылка скопирована') } catch { /* ignore */ }
}

function close() { emit('update:modelValue', false) }
</script>

<style scoped>
.ns { display: flex; flex-direction: column; gap: 14px; }
.ns-note { margin: 0; font-size: 13px; color: var(--color-text-dim); line-height: 1.5; }
.ns-create { display: flex; gap: 8px; flex-wrap: wrap; }
.ns-btn {
  display: inline-flex; align-items: center; gap: 6px; height: 38px; padding: 0 16px;
  border: none; border-radius: var(--radius-full); background: var(--color-primary); color: var(--color-on-primary);
  font-weight: 600; font-size: 13.5px; cursor: pointer;
}
.ns-btn .material-symbols-outlined { font-size: 18px; }
.ns-btn:disabled { opacity: 0.6; }
.ns-empty { margin: 0; font-size: 13px; color: var(--color-text-dim); }
.ns-shares { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.ns-share { display: flex; align-items: center; gap: 6px; }
.ns-share .chip-tint { flex-shrink: 0; }
.ns-url { flex: 1; min-width: 0; height: 38px; padding: 0 12px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--color-surface-low); color: var(--color-text); font-size: 13px; }
.ns-icon { flex-shrink: 0; width: 36px; height: 36px; display: grid; place-items: center; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); background: var(--color-surface); color: var(--color-text-dim); cursor: pointer; }
.ns-icon:hover { background: var(--color-surface-high); color: var(--color-text); }
.ns-icon.danger { color: var(--color-error); }
</style>
