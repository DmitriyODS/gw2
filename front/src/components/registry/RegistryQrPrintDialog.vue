<template>
  <AppDialog
    :model-value="modelValue"
    title="Печать QR-кодов"
    size="md"
    :busy="busy"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: 'Печать', icon: 'print', disabled: !fieldId || busy },
    ]"
    @update:model-value="close"
    @cancel="close(false)"
    @confirm="doPrint"
  >
    <div class="qp">
      <p v-if="!qrFields.length" class="qp-empty">
        В этом реестре нет полей с QR-кодом. Включите «Показывать QR-код значения»
        в настройках текстового или числового поля.
      </p>

      <template v-else>
        <div class="qp-field">
          <span class="qp-label">Поле для QR-кода</span>
          <Select
            v-model="fieldId"
            :options="qrFields" option-label="label" option-value="id"
            placeholder="Выберите поле"
          />
          <span class="qp-hint">Подписью под кодом печатается значение этого поля.</span>
        </div>

        <div v-if="selectedIds.size" class="qp-scope">
          <label class="qp-radio">
            <input type="radio" value="selected" v-model="scope" />
            <span>Только выбранные записи ({{ selectedIds.size }})</span>
          </label>
          <label class="qp-radio">
            <input type="radio" value="all" v-model="scope" />
            <span>Все записи<template v-if="search"> (по фильтру поиска)</template></span>
          </label>
        </div>

        <p class="qp-note">
          Коды печатаются на листах A4 сеткой 4 × 6 — по 24 кода на страницу.
          Записи с пустым значением поля пропускаются.
        </p>
      </template>
    </div>
  </AppDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import Select from 'primevue/select'
import QRCode from 'qrcode'
import AppDialog from '@/components/ui/AppDialog.vue'
import { getRecords } from '@/api/registries.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { hasQr, qrValue } from '@/utils/registryFields.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  registry: { type: Object, default: null },
  // Записи, отмеченные галочками в таблице (может быть пусто → печатаем все).
  selectedIds: { type: Object, default: () => new Set() },
  // Текущая строка поиска — печать «всех» уважает фильтр списка.
  search: { type: String, default: '' },
  // Сортировка списка — коды печатаются в том же порядке, что видит человек.
  sort: { type: String, default: 'created_at' },
  order: { type: String, default: 'desc' },
})
const emit = defineEmits(['update:modelValue'])

const notif = useNotificationsStore()

const busy = ref(false)
const fieldId = ref(null)
const scope = ref('all')

const qrFields = computed(() => (props.registry?.fields || []).filter(hasQr))

watch(() => props.modelValue, (open) => {
  if (!open) return
  fieldId.value = qrFields.value[0]?.id ?? null
  scope.value = props.selectedIds.size ? 'selected' : 'all'
})

function close(v) {
  if (v) return
  emit('update:modelValue', false)
}

const MAX_RECORDS = 500
// Потолок страницы на сервере: просить больше бессмысленно — вернётся 200.
const PAGE_SIZE = 200

// В списке видна лишь текущая страница, поэтому записи для печати догружаем
// САМИ и СТРАНИЦАМИ: одним запросом реестр не забрать (сервер зажимает размер
// страницы), а выбранная запись может лежать хоть в конце реестра.
// Отмеченные записи ищем по всему реестру, без строки поиска: галочки могли
// быть проставлены до того, как список отфильтровали.
async function collectRecords() {
  const onlySelected = scope.value === 'selected' && props.selectedIds.size > 0
  const params = {
    search: onlySelected ? '' : props.search,
    sort: props.sort, order: props.order,
    per_page: PAGE_SIZE,
  }
  const out = []
  for (let page = 1; ; page += 1) {
    const data = await getRecords(props.registry.id, { ...params, page })
    const items = data.items ?? []
    out.push(...(onlySelected ? items.filter((r) => props.selectedIds.has(r.id)) : items))
    const enough = onlySelected
      ? out.length >= props.selectedIds.size
      : out.length >= MAX_RECORDS
    if (enough || items.length < PAGE_SIZE || page * PAGE_SIZE >= (data.total ?? 0)) break
  }
  return out.slice(0, MAX_RECORDS)
}

async function doPrint() {
  const field = qrFields.value.find((f) => f.id === fieldId.value)
  if (!field) return
  busy.value = true
  try {
    const records = await collectRecords()
    const key = String(field.id)
    const values = records
      .map((r) => qrValue(r.data?.[key]))
      .filter(Boolean)
    if (!values.length) {
      notif.error('Нет записей с заполненным значением этого поля')
      return
    }
    if (records.length >= MAX_RECORDS) {
      notif.warn(`За раз печатается не больше ${MAX_RECORDS} кодов — остальные записи отфильтруйте и напечатайте отдельно.`)
    }
    const cells = await Promise.all(values.map(async (v) => ({
      value: v,
      src: await QRCode.toDataURL(v, {
        margin: 0,
        width: 400,
        errorCorrectionLevel: 'M',
        color: { dark: '#000000', light: '#ffffff' },
      }),
    })))
    printSheet(field.label, cells)
    emit('update:modelValue', false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось подготовить печать')
  } finally {
    busy.value = false
  }
}

// Печать во ВРЕМЕННОМ iframe, а не в новом окне: popup-блокировщики окно
// глушат, а iframe печатает и в мобильном WebView обёрток.
function printSheet(fieldLabel, cells) {
  const esc = (s) => String(s).replace(/[&<>"]/g, (c) => (
    { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]
  ))
  const title = `${props.registry?.name || 'Реестр'} — ${fieldLabel}`
  const body = cells.map((c) => `
    <div class="cell">
      <img src="${c.src}" alt="" />
      <div class="cap">${esc(c.value)}</div>
    </div>`).join('')

  // Печатный лист — самостоятельный документ: токены темы здесь неприменимы
  // (QR обязан быть чёрным на белом, иначе сканеры его не читают).
  const html = `<!doctype html><html lang="ru"><head><meta charset="utf-8">
<title>${esc(title)}</title>
<style>
  @page { size: A4 portrait; margin: 10mm; }
  * { box-sizing: border-box; }
  body { margin: 0; font-family: Arial, Helvetica, sans-serif; color: #000; background: #fff; }
  .grid { display: grid; grid-template-columns: repeat(4, 1fr); grid-auto-rows: calc(277mm / 6); }
  .cell {
    display: flex; flex-direction: column; align-items: center; justify-content: center;
    gap: 2mm; padding: 2mm; page-break-inside: avoid; break-inside: avoid; overflow: hidden;
  }
  .cell img { width: 30mm; height: 30mm; display: block; }
  .cap { font-size: 8pt; line-height: 1.15; text-align: center; word-break: break-all; max-height: 10mm; overflow: hidden; }
</style></head><body><div class="grid">${body}</div></body></html>`

  const frame = document.createElement('iframe')
  frame.setAttribute('aria-hidden', 'true')
  frame.style.cssText = 'position:fixed;right:0;bottom:0;width:0;height:0;border:0;'
  document.body.appendChild(frame)
  const doc = frame.contentDocument
  doc.open()
  doc.write(html)
  doc.close()

  const run = () => {
    frame.contentWindow.focus()
    frame.contentWindow.print()
    // Убираем iframe только после диалога печати — иначе задание отменится.
    setTimeout(() => frame.remove(), 60000)
  }
  if (frame.contentWindow.document.readyState === 'complete') run()
  else frame.onload = run
}
</script>

<style scoped>
.qp { display: flex; flex-direction: column; gap: 16px; }
.qp-field { display: flex; flex-direction: column; gap: 6px; }
.qp-label { font-size: 13px; font-weight: 600; color: var(--color-text-dim); }
.qp-hint, .qp-note { margin: 0; font-size: 12px; color: var(--color-text-dim); line-height: 1.5; }
.qp :deep(.p-select) { width: 100%; }
.qp-empty { margin: 0; font-size: 14px; color: var(--color-text-dim); line-height: 1.5; }
.qp-scope {
  display: flex; flex-direction: column; gap: 8px; padding: 12px;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md);
  background: var(--color-surface-low);
}
.qp-radio { display: flex; align-items: center; gap: 10px; font-size: 14px; color: var(--color-text); cursor: pointer; }
.qp-radio input { width: 18px; height: 18px; accent-color: var(--color-primary); }
</style>
