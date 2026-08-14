<template>
  <AppDialog
    :model-value="modelValue"
    title="Печать QR-кодов"
    size="md"
    :busy="busy"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: 'Печать', icon: 'print', disabled: !fieldId || busy || !totalCodes },
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

        <div v-if="hasSelection" class="qp-scope">
          <label class="qp-radio">
            <input type="radio" value="selected" v-model="scope" />
            <span>Только выбранные записи ({{ selectedCount }})</span>
          </label>
          <label class="qp-radio">
            <input type="radio" value="all" v-model="scope" />
            <span>Все записи<template v-if="filtered"> (по фильтру списка)</template></span>
          </label>
        </div>

        <div v-if="loading" class="qp-empty">Готовим коды…</div>

        <template v-else-if="items.length">
          <!-- Сколько кодов печатать для каждой позиции: одной вещи нужен один
               ярлык, а партии — сколько штук в партии. -->
          <div class="qp-block">
            <div class="qp-head">
              <span class="qp-label">Что печатаем</span>
              <span class="qp-hint">{{ totalCodes }} код(ов) · {{ pages }} стр.</span>
            </div>
            <ul class="qp-items">
              <li v-for="it in items" :key="it.id" class="qp-item">
                <span class="qp-item-value">{{ it.value }}</span>
                <!-- Счётчик собран из своих кнопок: у спиннера PrimeVue значки
                     из PrimeIcons, а этот шрифт в проект не подключён — кнопки
                     выходили пустыми. -->
                <span class="qp-item-count">
                  <AppButton
                    variant="icon" size="sm" icon="remove"
                    title="Меньше" aria-label="Меньше"
                    :disabled="it.count <= 0"
                    @click="bump(it, -1)"
                  />
                  <input
                    class="qp-count-input"
                    type="text"
                    inputmode="numeric"
                    :value="it.count"
                    aria-label="Сколько кодов печатать"
                    @input="setCount(it, $event.target.value)"
                  />
                  <AppButton
                    variant="icon" size="sm" icon="add"
                    title="Больше" aria-label="Больше"
                    :disabled="it.count >= MAX_PER_ITEM"
                    @click="bump(it, 1)"
                  />
                </span>
              </li>
            </ul>
          </div>

          <!-- Предпросмотр листа: те же коды и подписи, что уйдут на печать. -->
          <div class="qp-block">
            <span class="qp-label">Предпросмотр листа</span>
            <div class="qp-sheet">
              <div v-for="(cell, i) in preview" :key="i" class="qp-cell">
                <img :src="cell.src" alt="" />
                <span class="qp-cap">{{ cell.value }}</span>
              </div>
            </div>
            <span v-if="totalCodes > preview.length" class="qp-hint">
              Показаны первые {{ preview.length }} из {{ totalCodes }}.
            </span>
          </div>
        </template>

        <p v-else class="qp-empty">
          Нет записей с заполненным значением этого поля — печатать нечего.
        </p>

        <p class="qp-note">
          Коды печатаются на листах A4 сеткой 4 × 6 — по 24 кода на страницу.
        </p>
      </template>
    </div>
  </AppDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import Select from 'primevue/select'
import QRCode from 'qrcode'
import AppButton from '@/components/ui/AppButton.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import { useNotificationsStore } from '@/stores/notifications.js'
import { hasQr, qrValue } from '@/utils/registryFields.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  registry: { type: Object, default: null },
  /* (params) => Promise<{items, total}> — страница записей. Раздел передаёт
     свою ручку, публичная страница — выборку по коду ссылки; сам диалог про
     это ничего не знает и потому годится обоим. */
  fetchPage: { type: Function, required: true },
  /* Записи, отмеченные галочками в таблице (может быть пусто → печатаем все).
     Именно ПЕРЕЧЕНЬ id, как и у выгрузки: раздел отдаёт массив, и Set здесь
     молча превращал «выбранные» в «все» — у массива нет ни .size, ни .has. */
  selectedIds: { type: Array, default: () => [] },
  /* Набор «выбрано всё по фильтру» из useRowSelection: {all:true, exclude:[…]}.
     Снятые галочки на клиент не приезжают перечнем, поэтому исключения
     применяем к выборке сами. */
  selection: { type: Object, default: () => ({}) },
  // Сколько записей выбрано — в режиме «всё» это total минус снятые.
  selectionCount: { type: Number, default: 0 },
  // Фильтр экрана — печать «всех» уважает его целиком: строку поиска,
  // выбранный подраздел и условия по колонкам.
  search: { type: String, default: '' },
  section: { type: String, default: '' },
  filters: { type: Array, default: () => [] },
  // Сортировка списка — коды печатаются в том же порядке, что видит человек.
  sort: { type: String, default: 'created_at' },
  order: { type: String, default: 'desc' },
})
const emit = defineEmits(['update:modelValue'])

const notif = useNotificationsStore()

const busy = ref(false)
const loading = ref(false)
const fieldId = ref(null)
const scope = ref('all')
/* items — что печатаем: значение поля и СКОЛЬКО копий кода для него нужно.
   Одной вещи хватает одного ярлыка, а партии одинаковых — по ярлыку на штуку,
   поэтому количество задаётся для каждой позиции отдельно. */
const items = ref([])
const preview = ref([])

const qrFields = computed(() => (props.registry?.fields || []).filter(hasQr))
const filtered = computed(() =>
  !!(props.search || props.section || (props.filters || []).length))

// Set строим сами — так диалог переживает и массив, и Set в пропе.
const pickedIds = computed(() => new Set(props.selectedIds || []))
const selectAllMode = computed(() => !!props.selection?.all)
const selectedCount = computed(() => props.selectionCount || pickedIds.value.size)
const hasSelection = computed(() => pickedIds.value.size > 0 || selectAllMode.value)

// Сколько копий одной позиции разумно заказать за раз.
const MAX_PER_ITEM = 99

function bump(item, delta) {
  item.count = Math.min(MAX_PER_ITEM, Math.max(0, (item.count || 0) + delta))
}

// Ноль — законное значение: так позицию исключают из печати, не удаляя её из
// списка. Мусор в поле трактуем как ноль, а не как «оставить прежнее».
function setCount(item, raw) {
  const n = parseInt(String(raw).replace(/\D/g, ''), 10)
  item.count = Number.isFinite(n) ? Math.min(MAX_PER_ITEM, n) : 0
}

const totalCodes = computed(() => items.value.reduce((n, it) => n + (it.count || 0), 0))
// Лист A4 сеткой 4 × 6 — 24 кода на страницу.
const pages = computed(() => Math.max(1, Math.ceil(totalCodes.value / 24)))

watch(() => props.modelValue, (open) => {
  if (!open) {
    items.value = []
    preview.value = []
    return
  }
  fieldId.value = qrFields.value[0]?.id ?? null
  scope.value = hasSelection.value ? 'selected' : 'all'
  reload()
})

// Смена поля или области — другой набор кодов: пересобираем список.
watch([fieldId, scope], () => {
  if (props.modelValue) reload()
})

async function reload() {
  const field = qrFields.value.find((f) => f.id === fieldId.value)
  if (!field) {
    items.value = []
    return
  }
  loading.value = true
  try {
    const records = await collectRecords()
    const key = String(field.id)
    items.value = records
      .map((r) => ({ id: r.id, value: qrValue(r.data?.[key]), count: 1 }))
      .filter((it) => it.value)
    await renderPreview()
  } catch (e) {
    notif.error(e?.message || 'Не удалось подготовить коды')
    items.value = []
  } finally {
    loading.value = false
  }
}

// Предпросмотр — первые PREVIEW_MAX кодов с учётом заданных количеств: рисовать
// все пятьсот незачем, а понять «что выйдет» хватает первой страницы.
const PREVIEW_MAX = 12
let previewSeq = 0

async function renderPreview() {
  const seq = ++previewSeq
  const cells = []
  for (const it of items.value) {
    for (let i = 0; i < (it.count || 0) && cells.length < PREVIEW_MAX; i++) {
      cells.push(it.value)
    }
    if (cells.length >= PREVIEW_MAX) break
  }
  const drawn = await Promise.all(cells.map(async (value) => ({ value, src: await qrImage(value) })))
  if (seq === previewSeq) preview.value = drawn
}

// Количество меняют часто — перерисовку предпросмотра придерживаем, иначе
// каждый щелчок по «плюсу» гонит новую партию отрисовок.
let previewTimer = null
watch(items, () => {
  clearTimeout(previewTimer)
  previewTimer = setTimeout(renderPreview, 250)
}, { deep: true })

function qrImage(value) {
  return QRCode.toDataURL(value, {
    margin: 0,
    width: 400,
    errorCorrectionLevel: 'M',
    color: { dark: '#000000', light: '#ffffff' },
  })
}

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
// Отмеченные по одной ищем по всему реестру, без фильтра списка: галочки могли
// быть проставлены до того, как список отфильтровали. Режим «выбрано всё»
// описывается ТЕМ ЖЕ фильтром экрана — из его выдачи убираем снятые галочки.
async function collectRecords() {
  const onlySelected = scope.value === 'selected' && hasSelection.value
  const ids = onlySelected && pickedIds.value.size ? pickedIds.value : null
  const excluded = onlySelected && !ids ? new Set(props.selection?.exclude || []) : null
  const params = ids
    ? { sort: props.sort, order: props.order, per_page: PAGE_SIZE }
    : {
        search: props.search,
        section: props.section,
        filters: props.filters,
        sort: props.sort, order: props.order,
        per_page: PAGE_SIZE,
      }
  const out = []
  for (let page = 1; ; page += 1) {
    const data = await props.fetchPage({ ...params, page })
    const list = data.items ?? []
    if (ids) out.push(...list.filter((r) => ids.has(r.id)))
    else if (excluded) out.push(...list.filter((r) => !excluded.has(r.id)))
    else out.push(...list)
    const enough = ids ? out.length >= ids.size : out.length >= MAX_RECORDS
    if (enough || list.length < PAGE_SIZE || page * PAGE_SIZE >= (data.total ?? 0)) break
  }
  return out.slice(0, MAX_RECORDS)
}

async function doPrint() {
  const field = qrFields.value.find((f) => f.id === fieldId.value)
  if (!field || !totalCodes.value) return
  busy.value = true
  try {
    // Каждая позиция даёт СТОЛЬКО кодов, сколько для неё заказали.
    const values = []
    for (const it of items.value) {
      for (let i = 0; i < (it.count || 0); i++) values.push(it.value)
    }
    if (values.length > MAX_CODES) {
      notif.warn(`За раз печатается не больше ${MAX_CODES} кодов — остальное напечатайте отдельно.`)
      values.length = MAX_CODES
    }
    const cells = await Promise.all(values.map(async (v) => ({ value: v, src: await qrImage(v) })))
    printSheet(field.label, cells)
    emit('update:modelValue', false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось подготовить печать')
  } finally {
    busy.value = false
  }
}

// Потолок одного задания печати: дальше браузер захлёбывается на data-URI.
const MAX_CODES = 1000

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

.qp-block { display: flex; flex-direction: column; gap: 8px; }
.qp-head { display: flex; align-items: baseline; justify-content: space-between; gap: 10px; }

/* Список позиций: значение слева, счётчик копий справа. Своя прокрутка — их
   бывают сотни, а диалог расти до бесконечности не должен. */
.qp-items {
  display: flex; flex-direction: column; gap: 6px;
  max-height: 210px; overflow-y: auto;
  margin: 0; padding: 0; list-style: none;
}

.qp-item {
  display: flex; align-items: center; gap: 10px;
  padding: 6px 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
}

.qp-item-value { flex: 1; min-width: 0; font-size: 13px; overflow-wrap: anywhere; }
.qp-item-count { flex: none; display: inline-flex; align-items: center; gap: 4px; }

.qp-count-input {
  width: 44px;
  height: 30px;
  padding: 0 4px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-sm);
  background: var(--color-surface-low);
  color: var(--color-text);
  font: inherit;
  font-size: 13px;
  text-align: center;
}

/* Предпросмотр листа: коды чёрным по белому, как они и напечатаются — тема
   приложения здесь неприменима, сканеру нужен контраст. */
.qp-sheet {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(76px, 1fr));
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: #fff;
}

.qp-cell { display: flex; flex-direction: column; align-items: center; gap: 3px; }
.qp-cell img { width: 100%; max-width: 64px; aspect-ratio: 1; display: block; }

.qp-cap {
  max-width: 100%;
  font-size: 9px;
  line-height: 1.15;
  text-align: center;
  color: #000;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
