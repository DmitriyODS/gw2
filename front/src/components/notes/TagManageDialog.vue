<template>
  <AppDialog
    :model-value="modelValue"
    title="Управление тегами" icon="sell" size="md"
    :actions="[{ kind: 'cancel', label: 'Готово' }]"
    @cancel="close" @update:model-value="(v) => !v && close()"
  >
    <div class="tm">
      <div class="tm-scroll">
        <ul v-if="store.tags.length" class="tm-list">
        <li v-for="t in store.tags" :key="t.id" class="tm-row">
          <button
            class="tm-dot"
            :style="dotStyle(t)"
            :title="'Цвет тега'"
            @click="expanded = expanded === t.id ? null : t.id"
          />
          <input
            class="tm-name"
            :value="t.name"
            maxlength="60"
            @change="rename(t, $event.target.value)"
            @keydown.enter="$event.target.blur()"
          />
          <button class="tm-del" title="Удалить тег" @click="askDelete(t)">
            <span class="material-symbols-outlined">delete</span>
          </button>
          <div v-if="expanded === t.id" class="tm-palette">
            <ColorSwatchPicker :model-value="t.color || ''" @update:model-value="(c) => recolor(t, c)" />
          </div>
        </li>
        </ul>
        <p v-else class="tm-empty">Тегов пока нет — создайте первый ниже.</p>
      </div>

      <form class="tm-create" @submit.prevent="create">
        <ColorSwatchPicker v-model="newColor" aria-label="Цвет нового тега" />
        <div class="tm-create-row">
          <input v-model="newName" class="tm-name" maxlength="60" placeholder="Новый тег" />
          <button class="btn-grad" type="submit" :disabled="!newName.trim() || creating">
            <span class="material-symbols-outlined">add</span>
          </button>
        </div>
      </form>
    </div>

    <ConfirmDialog
      :visible="!!toDelete"
      header="Удалить тег?"
      :message="`Тег «${toDelete?.name}» будет удалён. Заметки останутся — просто потеряют эту метку.`"
      confirm-label="Удалить"
      danger-confirm
      @confirm="confirmDelete"
      @cancel="toDelete = null"
    />
  </AppDialog>
</template>

<script setup>
import { ref } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import ColorSwatchPicker from '@/components/common/ColorSwatchPicker.vue'
import { useNotesStore } from '@/stores/notes.js'
import { useNotificationsStore } from '@/stores/notifications.js'

defineProps({ modelValue: { type: Boolean, default: false } })
const emit = defineEmits(['update:modelValue'])

const store = useNotesStore()
const notif = useNotificationsStore()
const expanded = ref(null)
const newName = ref('')
const newColor = ref('')
const creating = ref(false)
const toDelete = ref(null)

function dotStyle(t) {
  if (!t.color) return { background: 'var(--color-surface-high)' }
  return { background: `var(--tag-${t.color}-surface)`, borderColor: `var(--tag-${t.color}-border)` }
}

async function create() {
  const name = newName.value.trim()
  if (!name) return
  creating.value = true
  try {
    await store.createTag(name, newColor.value)
    newName.value = ''
    newColor.value = ''
  } catch (e) { notif.error(e?.message || 'Не удалось создать тег') } finally { creating.value = false }
}

async function rename(t, value) {
  const name = value.trim()
  if (!name || name === t.name) return
  try { await store.renameTag(t.id, name, t.color || '') }
  catch (e) { notif.error(e?.message || 'Не удалось переименовать') }
}

async function recolor(t, color) {
  expanded.value = null
  if ((t.color || '') === color) return
  try { await store.renameTag(t.id, t.name, color) }
  catch (e) { notif.error(e?.message || 'Не удалось изменить цвет') }
}

function askDelete(t) { toDelete.value = t }
async function confirmDelete() {
  const t = toDelete.value
  toDelete.value = null
  if (!t) return
  try { await store.removeTag(t.id) }
  catch (e) { notif.error(e?.message || 'Не удалось удалить тег') }
}

function close() { emit('update:modelValue', false) }
</script>

<style scoped>
/* Список тегов прокручивается САМ (фикс. высота), форма добавления пришпилена
   снизу и не уезжает вместе со списком; окно не растёт на весь экран. */
.tm { display: flex; flex-direction: column; min-height: 0; }
.tm-scroll { flex: 1 1 auto; min-height: 0; max-height: 46vh; overflow-y: auto; overscroll-behavior: contain; padding-right: 4px; }
.tm-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.tm-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
/* Явные min/max по ОБЕИМ осям — иначе глобальный мобильный button{min-height:36px}
   вытягивает кружок в овал. */
.tm-dot { width: 24px; height: 24px; min-width: 24px; min-height: 24px; max-width: 24px; max-height: 24px; border-radius: 50%; border: 1.5px solid var(--color-outline-dim); flex-shrink: 0; cursor: pointer; padding: 0; }
.tm-name { flex: 1; min-width: 0; height: 38px; padding: 0 12px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--color-surface); color: var(--color-text); font: inherit; outline: none; }
.tm-name:focus { border-color: var(--color-primary); }
.tm-del { flex-shrink: 0; width: 38px; height: 38px; display: grid; place-items: center; border: none; border-radius: var(--radius-md); background: transparent; color: var(--color-error); cursor: pointer; }
.tm-del:hover { background: color-mix(in oklch, var(--color-error) 12%, transparent); }
.tm-palette { flex-basis: 100%; padding: 8px 0 4px 32px; }
.tm-empty { margin: 0; font-size: 13px; color: var(--color-text-dim); }
.tm-create { flex-shrink: 0; display: flex; flex-direction: column; gap: 10px; margin-top: 14px; padding-top: 12px; border-top: 1px solid var(--color-outline-dim); }
.tm-create-row { display: flex; gap: 8px; }
.tm-create-row .btn-grad { width: 44px; padding: 0; display: grid; place-items: center; }
</style>
