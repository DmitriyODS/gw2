<template>
  <AppDialog
    :model-value="modelValue"
    :title="title"
    icon="event"
    size="md"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="cd">
      <p v-if="!entries.length" class="cd-empty">На этот день записей нет.</p>
      <ul v-else class="cd-list">
        <li v-for="e in entries" :key="e.id" class="cd-row">
          <button class="cd-main" @click="$emit('open-entry', e)">
            <span class="cd-time">{{ hhmm(e.event_at) }}</span>
            <span class="cd-body">
              <span class="cd-title">{{ entryTitle(calendar, e) }}</span>
              <span v-for="cf in cardFields(calendar, e)" :key="cf.field.id" class="cd-sub">
                <span class="cd-field-label">{{ cf.field.label }}:</span> {{ cf.value }}
              </span>
            </span>
            <span class="material-symbols-outlined cd-chev">chevron_right</span>
          </button>
          <button v-if="!readonly" class="cd-del" title="Удалить" @click="askDelete(e)">
            <span class="material-symbols-outlined">delete</span>
          </button>
        </li>
      </ul>
    </div>

    <template #footer>
      <button class="cd-btn-text" @click="$emit('update:modelValue', false)">Закрыть</button>
      <button v-if="!readonly" class="cd-btn-filled" @click="$emit('add')">
        <span class="material-symbols-outlined">add</span> Добавить запись
      </button>
    </template>

    <ConfirmDialog
      :visible="confirm != null"
      header="Удалить запись?"
      message="Запись будет удалена безвозвратно."
      confirm-label="Удалить" danger-confirm
      @confirm="doDelete" @cancel="confirm = null"
    />
  </AppDialog>
</template>

<script setup>
import { computed, ref } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { useCalendarsStore } from '@/stores/calendars.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { cardFields, entryTitle, hhmm } from '@/utils/calendarFields.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  calendar: { type: Object, default: null },
  date: { type: [Date, String, Number], default: null },
  entries: { type: Array, default: () => [] },
  readonly: { type: Boolean, default: false },
})
defineEmits(['update:modelValue', 'open-entry', 'add'])

const store = useCalendarsStore()
const notif = useNotificationsStore()

const title = computed(() => {
  if (!props.date) return 'День'
  const d = new Date(props.date)
  if (isNaN(d.getTime())) return 'День'
  const s = d.toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
  return s.charAt(0).toUpperCase() + s.slice(1)
})

const confirm = ref(null)
function askDelete(e) { confirm.value = e }
async function doDelete() {
  const e = confirm.value
  confirm.value = null
  if (!e) return
  try {
    await store.deleteEntry(e.id)
    notif.success('Запись удалена')
  } catch (err) {
    notif.error(err?.message || 'Не удалось удалить запись')
  }
}
</script>

<style scoped>
.cd { display: flex; flex-direction: column; gap: 8px; }
.cd-empty { margin: 8px 0; color: var(--color-text-dim); text-align: center; }
.cd-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.cd-row { display: flex; align-items: stretch; gap: 6px; }
.cd-main {
  flex: 1; min-width: 0; display: flex; align-items: center; gap: 12px; text-align: left;
  padding: 10px 12px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md);
  background: var(--color-surface); cursor: pointer;
}
.cd-main:hover { background: var(--color-surface-high); border-color: var(--color-outline); }
.cd-time {
  flex-shrink: 0; min-width: 48px; font-size: 15px; font-weight: 700; color: var(--color-primary);
  font-variant-numeric: tabular-nums;
}
.cd-body { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.cd-title { font-size: 14px; font-weight: 600; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cd-sub { font-size: 12px; color: var(--color-text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cd-field-label { font-weight: 600; color: var(--color-text); }
.cd-chev { flex-shrink: 0; color: var(--color-text-dim); }
.cd-del {
  flex-shrink: 0; width: 42px; display: grid; place-items: center;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md);
  background: var(--color-surface); color: var(--color-error); cursor: pointer;
}
.cd-del:hover { background: var(--color-error-container, var(--color-surface-high)); }

.cd-btn-text {
  border: none; background: none; cursor: pointer;
  padding: 10px 16px; border-radius: var(--radius-full);
  color: var(--color-text-dim); font-weight: 600; font-size: 14px;
}
.cd-btn-text:hover { background: var(--color-surface-high); color: var(--color-text); }
.cd-btn-filled {
  display: inline-flex; align-items: center; gap: 6px;
  border: none; cursor: pointer;
  padding: 10px 18px; border-radius: var(--radius-full);
  background: var(--color-primary); color: var(--color-on-primary);
  font-weight: 600; font-size: 14px;
}
</style>
