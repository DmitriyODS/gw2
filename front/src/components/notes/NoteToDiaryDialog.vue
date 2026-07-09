<template>
  <AppDialog
    :model-value="modelValue"
    title="Пункт в ежедневник"
    icon="book"
    size="sm"
    :busy="saving"
    :actions="diaries.length
      ? [{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Добавить' }]
      : [{ kind: 'cancel', label: 'Закрыть' }]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
    @confirm="save"
  >
    <div v-if="loading" class="ntd-hint">Загрузка ежедневников…</div>
    <div v-else-if="!diaries.length" class="ntd-hint">
      У вас пока нет ежедневников — создайте его в разделе «Ежедневники».
    </div>
    <template v-else>
      <label class="ntd-label">Ежедневник</label>
      <select v-model="diaryId" class="ntd-input">
        <option v-for="d in diaries" :key="d.id" :value="d.id">{{ d.name }}</option>
      </select>

      <label class="ntd-label">День</label>
      <input v-model="date" type="date" class="ntd-input" />

      <label class="ntd-label">Название</label>
      <input v-model="title" type="text" class="ntd-input" maxlength="300" placeholder="Название записи" />

      <label class="ntd-label">Описание</label>
      <textarea v-model="description" class="ntd-input ntd-area" rows="4" placeholder="Описание (необязательно)" />
    </template>
  </AppDialog>
</template>

<script setup>
// Создание записи ежедневника из выделенного текста заметки: первая строка —
// название, весь фрагмент (если он длиннее) — описание. Ежедневники — только
// свои (tab=mine, они кросс-компанийные — активная компания не нужна).
import { ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { createEntry, getDiaries } from '@/api/diaries.js'
import { dayKey } from '@/stores/diaries.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  text: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const notif = useNotificationsStore()

const loading = ref(false)
const saving = ref(false)
const diaries = ref([])
const diaryId = ref(null)
const date = ref(dayKey(new Date()))
const title = ref('')
const description = ref('')

watch(() => props.modelValue, async (open) => {
  if (!open) return
  const firstLine = props.text.split('\n').map((s) => s.trim()).find(Boolean) || ''
  title.value = firstLine.slice(0, 300)
  description.value = props.text.trim() !== firstLine ? props.text.trim() : ''
  date.value = dayKey(new Date())
  loading.value = true
  try {
    const { diaries: list } = await getDiaries('mine')
    diaries.value = list || []
    if (!diaries.value.some((d) => d.id === diaryId.value)) diaryId.value = diaries.value[0]?.id ?? null
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить ежедневники')
    diaries.value = []
  } finally {
    loading.value = false
  }
})

async function save() {
  if (!diaryId.value) return
  const t = title.value.trim()
  if (!t) { notif.error('Укажите название записи'); return }
  saving.value = true
  try {
    await createEntry(diaryId.value, {
      entry_date: date.value,
      start_min: null,
      end_min: null,
      title: t,
      description: description.value.trim(),
    })
    notif.success('Запись добавлена в ежедневник')
    emit('update:modelValue', false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось добавить запись')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.ntd-hint { padding: 10px 0; color: var(--color-text-dim); font-size: 14px; }

.ntd-label {
  display: block;
  margin: 12px 0 4px;
  color: var(--color-text-dim);
  font-size: 12.5px;
  font-weight: 700;
}
.ntd-label:first-of-type { margin-top: 0; }

.ntd-input {
  width: 100%;
  height: 40px;
  padding: 0 12px;
  border: 1px solid var(--color-outline-variant);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
}
.ntd-input:focus { outline: none; border-color: var(--color-primary); }

.ntd-area {
  height: auto;
  padding: 10px 12px;
  resize: vertical;
  line-height: 1.5;
}
</style>
