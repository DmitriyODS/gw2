<template>
  <AppDialog
    :model-value="modelValue"
    title="Экспорт в XLSX"
    size="md"
    :busy="busy"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: 'Экспортировать', icon: 'download', disabled: !chosen.size },
    ]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
    @confirm="run"
  >
    <AppStack :gap="16">
      <!-- Область выгрузки появляется, только когда есть что противопоставить
           «всем записям» — отмеченные строки. -->
      <AppTabs v-if="selectedIds.length" v-model="scope" :tabs="scopeTabs" variant="tint" full-width />

      <AppStack :gap="8">
        <div class="rx-head">
          <span class="rx-title">Поля для выгрузки</span>
          <AppStack row :gap="4">
            <AppButton variant="text" size="sm" label="Выбрать всё" @click="selectAll" />
            <AppButton variant="text" size="sm" label="Снять всё" @click="clearAll" />
          </AppStack>
        </div>

        <div class="rx-fields">
          <label v-for="f in fields" :key="f.id" class="rx-row">
            <Checkbox :model-value="chosen.has(f.id)" binary @update:model-value="toggle(f.id)" />
            <span class="material-symbols-outlined">{{ fieldIcon(f.type) }}</span>
            <span class="rx-name">{{ f.label }}</span>
          </label>
          <p v-if="!fields.length" class="rx-empty">
            В этом реестре нет полей, доступных для экспорта (картинки и файлы не выгружаются).
          </p>
        </div>
      </AppStack>
    </AppStack>
  </AppDialog>
</template>

<script setup>
/* Выгрузка записей в xlsx — одна и та же форма у раздела и у публичной
   ссылки. Различается только запрос (свои записи или записи по коду ссылки),
   поэтому он приходит пропом: сам файл собирает сервер, здесь — выбор полей,
   области и сохранение полученного файла. */
import { computed, ref, watch } from 'vue'
import Checkbox from 'primevue/checkbox'
import AppButton from '@/components/ui/AppButton.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import { fieldIcon } from '@/utils/registryFields.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  /** Поля, доступные для выгрузки (см. isExportable). */
  fields: { type: Array, default: () => [] },
  /** Отмеченные записи — с ними появляется выбор области выгрузки. */
  selectedIds: { type: Array, default: () => [] },
  /** Текущий поиск: им ограничена выгрузка «всех записей». */
  search: { type: String, default: '' },
  /** Имя файла без расширения. */
  filename: { type: String, default: 'registry' },
  /** (params) => Promise<Response> — ручка выгрузки (своя / по коду ссылки). */
  request: { type: Function, required: true },
})

const emit = defineEmits(['update:modelValue', 'error'])

const scope = ref('all')
const chosen = ref(new Set())
const busy = ref(false)

// Открытие — всегда с чистого листа: поля отмечены все, область подсказана
// текущим выбором записей.
watch(() => props.modelValue, (open) => {
  if (!open) return
  scope.value = props.selectedIds.length ? 'selected' : 'all'
  chosen.value = new Set(props.fields.map((f) => f.id))
})

const scopeTabs = computed(() => [
  { value: 'all', label: props.search ? 'Все по фильтру' : 'Все записи' },
  { value: 'selected', label: `Выбранные (${props.selectedIds.length})` },
])

function toggle(id) {
  const next = new Set(chosen.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  chosen.value = next
}
function selectAll() { chosen.value = new Set(props.fields.map((f) => f.id)) }
function clearAll() { chosen.value = new Set() }

async function run() {
  if (!chosen.value.size) return
  busy.value = true
  try {
    const params = { fields: [...chosen.value] }
    if (scope.value === 'selected' && props.selectedIds.length) params.ids = [...props.selectedIds]
    else params.search = props.search
    const resp = await props.request(params)
    if (!resp.ok) {
      let msg = 'Не удалось выгрузить'
      try { msg = (await resp.json()).message || msg } catch { /* тело не json */ }
      throw new Error(msg)
    }
    const url = URL.createObjectURL(await resp.blob())
    const a = document.createElement('a')
    a.href = url
    a.download = `${props.filename || 'registry'}.xlsx`
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
    emit('update:modelValue', false)
  } catch (e) {
    emit('error', e?.message || 'Не удалось выгрузить')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.rx-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.rx-title {
  font-size: 13px;
  font-weight: 700;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

.rx-fields { display: flex; flex-direction: column; gap: 2px; max-height: 320px; overflow-y: auto; }

.rx-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 8px;
  border-radius: var(--radius-md);
  font-size: 14px;
  cursor: pointer;
}

.rx-row:hover { background: var(--color-surface-high); }
.rx-row .material-symbols-outlined { font-size: 20px; color: var(--color-text-dim); }
.rx-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.rx-empty { margin: 0; font-size: 14px; color: var(--color-text-dim); }
</style>
