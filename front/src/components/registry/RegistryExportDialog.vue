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
           «всем записям» — выбранные строки. -->
      <AppTabs v-if="hasSelection" v-model="scope" :tabs="scopeTabs" variant="tint" full-width />

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

      <!-- Предпросмотр: тот же порядок колонок и то же представление значений,
           что уедут в файл. Строки берём с текущей страницы — показать все
           записи всё равно негде, а понять «что получится» хватает первых. -->
      <AppStack :gap="8">
        <div class="rx-head">
          <span class="rx-title">Как будет выглядеть файл</span>
          <span class="rx-note">{{ previewNote }}</span>
        </div>

        <div v-if="previewCols.length" class="rx-preview">
          <table class="rx-sheet">
            <thead>
              <tr>
                <th class="rx-sheet-num" />
                <th v-for="c in previewCols" :key="c.id">{{ c.label }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, i) in previewRows" :key="i">
                <td class="rx-sheet-num">{{ i + 1 }}</td>
                <td v-for="c in previewCols" :key="c.id">{{ row[c.id] }}</td>
              </tr>
            </tbody>
          </table>
          <p v-if="!previewRows.length" class="rx-empty">
            Под выгрузку не попало ни одной записи с этой страницы.
          </p>
        </div>
        <p v-else class="rx-empty">Отметьте хотя бы одно поле — тогда появится предпросмотр.</p>
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
import { fieldIcon, textValue } from '@/utils/registryFields.js'
import { saveBlob } from '@/utils/download.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  /** Поля, доступные для выгрузки (см. isExportable). */
  fields: { type: Array, default: () => [] },
  /** Отмеченные записи — с ними появляется выбор области выгрузки. */
  selectedIds: { type: Array, default: () => [] },
  /** Набор «выбрано всё по фильтру» из useRowSelection: {all:true, exclude:[…]}. */
  selection: { type: Object, default: () => ({}) },
  /** Сколько записей выбрано — в режиме «всё» это total минус снятые. */
  selectionCount: { type: Number, default: 0 },
  /** Фильтр экрана: выгрузка «всех» идёт ровно им — файл не должен
      расходиться с тем, что человек видит. */
  filter: { type: Object, default: () => ({}) },
  /** Записи текущей страницы — из них строится предпросмотр. */
  records: { type: Array, default: () => [] },
  /** Учётный реестр: в файл добавляется колонка состояния позиции. */
  accounting: { type: Boolean, default: false },
  /** Имя файла без расширения. */
  filename: { type: String, default: 'registry' },
  /** (params) => Promise<Response> — ручка выгрузки (своя / по коду ссылки). */
  request: { type: Function, required: true },
})

const emit = defineEmits(['update:modelValue', 'error'])

const scope = ref('all')
const chosen = ref(new Set())
const busy = ref(false)

/* Выбор приходит двумя видами: перечнем отмеченных id либо «всё по фильтру
   минус снятые» — второй экран на клиент id не тянет, поэтому он описывается
   самим набором selection. Оба вида дают область «Выбранные». */
const excluded = computed(() => new Set(props.selection?.exclude || []))
const selectAllMode = computed(() => !!props.selection?.all)
const selectedCount = computed(() => props.selectionCount || props.selectedIds.length)
const hasSelection = computed(() => props.selectedIds.length > 0 || selectAllMode.value)

// Открытие — всегда с чистого листа: поля отмечены все, область подсказана
// текущим выбором записей.
watch(() => props.modelValue, (open) => {
  if (!open) return
  scope.value = hasSelection.value ? 'selected' : 'all'
  chosen.value = new Set(props.fields.map((f) => f.id))
})

const scopeTabs = computed(() => [
  { value: 'all', label: props.filter?.search || props.filter?.section ? 'Все по фильтру' : 'Все записи' },
  { value: 'selected', label: `Выбранные (${selectedCount.value})` },
])

function toggle(id) {
  const next = new Set(chosen.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  chosen.value = next
}
/* ── Предпросмотр ──
   Колонки идут в порядке РЕЕСТРА (как в файле), а не в порядке отмечания, и
   значения проходят через тот же textValue, что и таблица, — зеркало серверного
   exportValue. Совпадения «символ в символ» здесь не требуется: предпросмотр
   отвечает на вопрос «что и в каком порядке окажется в файле». */
const previewCols = computed(() => {
  const cols = props.fields.filter((f) => chosen.value.has(f.id))
    .map((f) => ({ id: String(f.id), label: f.label, field: f }))
  if (cols.length && props.accounting) {
    cols.push({ id: '__state', label: 'Состояние', field: null })
  }
  return cols
})

const PREVIEW_ROWS_MAX = 6

// Что именно попадёт в файл: выбранные записи либо всё по фильтру экрана.
const previewSource = computed(() => {
  if (scope.value !== 'selected' || !hasSelection.value) return props.records
  if (!props.selectedIds.length) return props.records.filter((r) => !excluded.value.has(r.id))
  const picked = new Set(props.selectedIds)
  return props.records.filter((r) => picked.has(r.id))
})

const previewRows = computed(() => previewSource.value.slice(0, PREVIEW_ROWS_MAX).map((rec) => {
  const row = {}
  for (const c of previewCols.value) {
    row[c.id] = c.field
      ? textValue(c.field, rec.data?.[String(c.field.id)])
      : stateText(rec.issue)
  }
  return row
}))

const previewNote = computed(() => {
  const total = previewSource.value.length
  if (!total) return ''
  return total > PREVIEW_ROWS_MAX
    ? `первые ${PREVIEW_ROWS_MAX} строк из ${total} на этой странице`
    : `строк на этой странице: ${total}`
})

// Зеркало issueText на сервере: состояние позиции такой же колонкой отчёта.
function stateText(issue) {
  if (!issue) return 'В наличии'
  const who = issue.issued_to || issue.holder_name
  if (!issue.due_at) return `Выдано без срока (${who})`
  const overdue = Math.ceil((Date.now() - new Date(issue.due_at).getTime()) / 86400000)
  return overdue > 0
    ? `Просрочено на ${overdue} дн. (${who})`
    : `Выдано до ${new Date(issue.due_at).toLocaleDateString('ru-RU')} (${who})`
}

function selectAll() { chosen.value = new Set(props.fields.map((f) => f.id)) }
function clearAll() { chosen.value = new Set() }

async function run() {
  if (!chosen.value.size) return
  busy.value = true
  try {
    const params = {
      fields: [...chosen.value],
      selection: scope.value === 'selected'
        ? (props.selectedIds.length ? { ids: [...props.selectedIds] } : props.selection)
        : { all: true },
      filter: props.filter,
    }
    const resp = await props.request(params)
    if (!resp.ok) {
      let msg = 'Не удалось выгрузить'
      try { msg = (await resp.json()).message || msg } catch { /* тело не json */ }
      throw new Error(msg)
    }
    await saveBlob(await resp.blob(), `${props.filename || 'registry'}.xlsx`)
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
.rx-note { font-size: 12px; color: var(--color-text-dim); }

/* Предпросмотр листа: своя прокрутка по горизонтали — колонок бывает много, а
   раздел уезжать вбок не должен. */
.rx-preview {
  overflow-x: auto;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
}

.rx-sheet { border-collapse: collapse; width: 100%; font-size: 12.5px; }

.rx-sheet th,
.rx-sheet td {
  padding: 6px 10px;
  border-right: 1px solid var(--acrylic-border);
  border-bottom: 1px solid var(--acrylic-border);
  text-align: left;
  white-space: nowrap;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rx-sheet th { background: var(--color-surface-high); font-weight: 700; }
.rx-sheet tr:last-child td { border-bottom: none; }
.rx-sheet th:last-child, .rx-sheet td:last-child { border-right: none; }

/* Колонка номеров строк — как в самом Excel: она задаёт масштаб происходящего. */
.rx-sheet-num {
  width: 34px;
  text-align: center !important;
  color: var(--color-text-dim);
  background: var(--color-surface-high);
}

.rx-empty { margin: 0; font-size: 14px; color: var(--color-text-dim); }
</style>
