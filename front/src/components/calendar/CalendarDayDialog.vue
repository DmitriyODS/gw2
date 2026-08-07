<template>
  <AppDialog
    :model-value="modelValue"
    :title="title"
    :subtitle="entries.length ? `Записей: ${entries.length}` : ''"
    size="md"
    :actions="actions"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="$emit('add')"
  >
    <EmptyState
      v-if="!entries.length"
      size="sm"
      icon="event_busy"
      title="На этот день записей нет"
      :subtitle="readonly ? '' : 'Добавьте первую — она появится здесь.'"
    />

    <AppStack v-else :gap="8">
      <AppRow
        v-for="e in entries"
        :key="e.id"
        :title="entryTitle(calendar, e)"
        clickable
        inline
        @click="$emit('open-entry', e)"
      >
        <template #lead>
          <span class="cd-time">{{ hhmm(e.event_at) }}</span>
        </template>

        <template v-if="cardFields(calendar, e).length" #hint>
          <span v-for="cf in cardFields(calendar, e)" :key="cf.field.id" class="cd-sub">
            <span class="cd-field-label">{{ cf.field.label }}:</span> {{ cf.value }}
          </span>
        </template>

        <AppButton
          v-if="!readonly"
          variant="icon"
          size="sm"
          tone="danger"
          icon="delete"
          title="Удалить"
          aria-label="Удалить запись"
          @click.stop="askDelete(e)"
        />
      </AppRow>
    </AppStack>

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
import AppButton from '@/components/ui/AppButton.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
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

const actions = computed(() => [
  { kind: 'cancel', label: 'Закрыть' },
  ...(props.readonly ? [] : [{ kind: 'confirm', label: 'Добавить запись', icon: 'add' }]),
])

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
/* Время — «якорь» строки: моноширинные цифры, чтобы столбик не плясал. */
.cd-time {
  min-width: 46px;
  font-size: 15px;
  font-weight: 700;
  color: var(--color-primary);
  font-variant-numeric: tabular-nums;
}

.cd-sub {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cd-field-label { font-weight: 600; color: var(--color-text); }
</style>
