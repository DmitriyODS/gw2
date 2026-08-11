<template>
  <div class="rr">
    <!-- Узкая панель: сортировка контролом — заголовков колонок там нет. -->
    <div v-if="narrow && fields.length" class="rr-sortbar">
      <span class="material-symbols-outlined">sort</span>
      <Select
        class="rr-sortbar-select"
        :model-value="sort"
        :options="sortOptions"
        option-label="label"
        option-value="value"
        @update:model-value="pickSort"
      />
      <AppButton
        variant="icon"
        size="sm"
        :icon="order === 'asc' ? 'arrow_upward' : 'arrow_downward'"
        :title="order === 'asc' ? 'По возрастанию' : 'По убыванию'"
        aria-label="Направление сортировки"
        @click="flipOrder"
      />
    </div>

    <!-- Массовые действия над выбранным: у раздела своё, у публичной ссылки своё. -->
    <slot name="selection" />

    <div class="rr-box">
      <div v-if="!narrow" class="rr-scroll">
        <table class="rr-table">
          <thead>
            <tr>
              <th class="rr-th-check">
                <Checkbox
                  :model-value="allSelected"
                  binary
                  :disabled="!records.length"
                  @update:model-value="$emit('toggle-all')"
                />
              </th>
              <th
                v-for="f in fields"
                :key="f.id"
                :class="{ sortable: isSortable(f.type) }"
                @click="isSortable(f.type) && headerSort(String(f.id))"
              >
                <span class="rr-th-inner">
                  {{ f.label }}
                  <span v-if="sort === String(f.id)" class="material-symbols-outlined rr-sort">
                    {{ order === 'asc' ? 'arrow_upward' : 'arrow_downward' }}
                  </span>
                </span>
              </th>
              <th class="rr-th-date sortable" @click="headerSort('created_at')">
                <span class="rr-th-inner">
                  Создано
                  <span v-if="sort === 'created_at'" class="material-symbols-outlined rr-sort">
                    {{ order === 'asc' ? 'arrow_upward' : 'arrow_downward' }}
                  </span>
                </span>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="rec in records"
              :key="rec.id"
              class="rr-row"
              :class="{ selected: selected.has(rec.id) }"
              @click="$emit('open', rec)"
            >
              <td class="rr-td-check" @click.stop>
                <Checkbox
                  :model-value="selected.has(rec.id)"
                  binary
                  @update:model-value="$emit('toggle', rec.id)"
                />
              </td>
              <td v-for="f in fields" :key="f.id">
                <button
                  v-if="f.type === 'image' && thumbSrc(rec.data?.[String(f.id)])"
                  class="rr-thumb"
                  type="button"
                  title="Открыть картинку"
                  @click.stop="openImage(f, rec)"
                >
                  <img
                    :src="thumbSrc(rec.data?.[String(f.id)])"
                    :alt="rec.data?.[String(f.id)]?.name || ''"
                    loading="lazy"
                    decoding="async"
                  />
                </button>
                <span
                  v-else-if="f.type === 'stock'"
                  class="rr-stock"
                  :class="{ taken: !!rec.data?.[String(f.id)]?.taken }"
                >{{ textValue(f, rec.data?.[String(f.id)]) }}</span>
                <span v-else class="rr-cell">{{ textValue(f, rec.data?.[String(f.id)]) }}</span>
              </td>
              <td class="rr-td-date">{{ shortDate(rec.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Узкая панель: карточки записей вместо таблицы -->
      <AppStack v-else class="rr-cards" :gap="10">
        <label v-if="records.length" class="rr-cards-selall">
          <Checkbox :model-value="allSelected" binary @update:model-value="$emit('toggle-all')" />
          <span>Выбрать все на странице</span>
        </label>
        <AppCard
          v-for="rec in records"
          :key="rec.id"
          :tone="selected.has(rec.id) ? 'primary' : 'neutral'"
          clickable
          :gap="8"
          @click="$emit('open', rec)"
        >
          <div class="rr-card-head">
            <span class="rr-card-check" @click.stop>
              <Checkbox
                :model-value="selected.has(rec.id)"
                binary
                @update:model-value="$emit('toggle', rec.id)"
              />
            </span>
            <span class="rr-card-title">{{ cardTitle(rec) }}</span>
            <span class="material-symbols-outlined rr-card-chev">chevron_right</span>
          </div>
          <div v-if="bodyFields.length" class="rr-card-body">
            <div
              v-for="f in bodyFields"
              :key="f.id"
              class="rr-card-row"
              :class="{ image: f.type === 'image' && thumbSrc(rec.data?.[String(f.id)]) }"
            >
              <span class="rr-card-label">{{ f.label }}</span>
              <button
                v-if="f.type === 'image' && thumbSrc(rec.data?.[String(f.id)])"
                class="rr-thumb card"
                type="button"
                title="Открыть картинку"
                @click.stop="openImage(f, rec)"
              >
                <img
                  :src="thumbSrc(rec.data?.[String(f.id)])"
                  :alt="rec.data?.[String(f.id)]?.name || ''"
                  loading="lazy"
                  decoding="async"
                />
              </button>
              <span
                v-else-if="f.type === 'stock'"
                class="rr-stock"
                :class="{ taken: !!rec.data?.[String(f.id)]?.taken }"
              >{{ textValue(f, rec.data?.[String(f.id)]) }}</span>
              <span v-else class="rr-card-val">{{ textValue(f, rec.data?.[String(f.id)]) || '—' }}</span>
            </div>
          </div>
        </AppCard>
      </AppStack>

      <div v-if="loading" class="rr-overlay">
        <BrandLoader :size="48" />
      </div>
      <EmptyState
        v-else-if="!records.length"
        class="rr-empty"
        icon="inbox"
        tone="soft"
        :title="search ? 'Ничего не найдено' : 'Записей пока нет'"
        :subtitle="search ? 'Попробуйте другой запрос.' : emptyHint"
      />
    </div>

    <!-- Клик по превью открывает картинку целиком, не трогая карточку записи. -->
    <ImageLightbox v-model="lightbox" :src="lightboxSrc" :caption="lightboxCaption" />
  </div>
</template>

<script setup>
/* Записи реестра: широко — таблица с сортировкой по заголовкам, тесно —
   карточки и сортировка отдельным контролом. Один и тот же вид показывают
   раздел «Реестры» и публичная страница внешней ссылки, поэтому он живёт
   здесь, а данные и запросы остаются у них: сюда приходят готовые записи,
   отсюда уходят намерения (открыть, выбрать, пересортировать).

   Узкую раскладку задаёт вызывающий: она про ширину ПАНЕЛИ (раздел живёт
   окном рабочего стола), а не про ширину экрана. */
import { computed, ref } from 'vue'
import Checkbox from 'primevue/checkbox'
import Select from 'primevue/select'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppStack from '@/components/ui/AppStack.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ImageLightbox from '@/components/common/ImageLightbox.vue'
import { fileUrl, isSortable, textValue, thumbUrl } from '@/utils/registryFields.js'

const props = defineProps({
  /** Видимые поля в порядке реестра (см. useRegistryColumns). */
  fields: { type: Array, default: () => [] },
  records: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  /** Ключ сортировки: 'created_at' либо строковый id поля. */
  sort: { type: String, default: 'created_at' },
  order: { type: String, default: 'desc' },
  /** Выбранные записи — множество id (см. useRowSelection). */
  selected: { type: Set, default: () => new Set() },
  allSelected: { type: Boolean, default: false },
  /** Тесная панель: карточки вместо таблицы. */
  narrow: { type: Boolean, default: false },
  /** Активный поисковый запрос — от него зависит текст пустого состояния. */
  search: { type: String, default: '' },
  emptyHint: { type: String, default: '' },
})

const emit = defineEmits(['update:sort', 'open', 'toggle', 'toggle-all'])

/* Сортировку решаем здесь целиком: клик по заголовку той же колонки
   переворачивает порядок, выбор в узком контроле его сохраняет. Наружу уходит
   готовая пара — вызывающему остаётся перечитать первую страницу. */
function headerSort(key) {
  if (props.sort === key) emit('update:sort', { sort: key, order: props.order === 'asc' ? 'desc' : 'asc' })
  else emit('update:sort', { sort: key, order: 'asc' })
}
function pickSort(key) {
  emit('update:sort', { sort: key, order: props.order })
}
function flipOrder() {
  emit('update:sort', { sort: props.sort, order: props.order === 'asc' ? 'desc' : 'asc' })
}

const sortOptions = computed(() => {
  const opts = [{ value: 'created_at', label: 'Дате создания' }]
  for (const f of props.fields) {
    if (isSortable(f.type)) opts.push({ value: String(f.id), label: f.label })
  }
  return opts
})

/* На карточке первое видимое поле — заголовок, остальные — тело. Картинка
   заголовком быть не может: в строке от неё осталось бы имя файла, а показать
   её самоё гораздо полезнее. */
const titleField = computed(() => props.fields.find((f) => f.type !== 'image') || null)
const bodyFields = computed(() => props.fields.filter((f) => f !== titleField.value))

function cardTitle(rec) {
  const f = titleField.value
  const v = f ? textValue(f, rec.data?.[String(f.id)]) : ''
  return v || `Запись #${rec.id}`
}

function shortDate(v) {
  if (!v) return ''
  const d = new Date(v)
  return isNaN(d.getTime()) ? '' : d.toLocaleDateString('ru-RU')
}

/* Превью картинок прямо в строке: до него о содержимом поля говорило только имя
   файла. В таблице открывается лайтбокс с оригиналом — карточку записи ради
   одной картинки открывать незачем. */
const lightbox = ref(false)
const lightboxSrc = ref('')
const lightboxCaption = ref('')

const thumbSrc = (value) => thumbUrl(value)

function openImage(field, rec) {
  const value = rec.data?.[String(field.id)]
  lightboxSrc.value = fileUrl(value)
  lightboxCaption.value = value?.name || field.label
  lightbox.value = true
}
</script>

<style scoped>
.rr {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

/* ── Сортировка в узкой раскладке ── */
.rr-sortbar {
  flex: none;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-bottom: 1px solid var(--color-outline-dim);
  color: var(--color-text-dim);
}

.rr-sortbar > .material-symbols-outlined { font-size: 20px; }
.rr-sortbar-select { flex: 1; min-width: 0; }

/* ── Таблица: собственный скролл, sticky-шапка ── */
.rr-box { position: relative; flex: 1; min-height: 0; display: flex; }
.rr-scroll { position: relative; flex: 1; min-height: 0; overflow: auto; }

.rr-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

/* Строки уезжают под шапку — ей нужно плотное стекло с блюром. */
.rr-table thead th {
  position: sticky;
  top: 0;
  z-index: 1;
  padding: 12px 14px;
  border-bottom: 1px solid var(--color-outline-dim);
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  text-align: left;
  font-weight: 700;
  color: var(--color-text);
  white-space: nowrap;
  user-select: none;
}

.rr-table thead th.sortable { cursor: pointer; }
.rr-table thead th.sortable:hover { color: var(--color-primary); }
.rr-th-inner { display: inline-flex; align-items: center; gap: 4px; }
.rr-sort { font-size: 16px; }
.rr-th-check, .rr-td-check { width: 48px; text-align: center; padding-left: 16px; padding-right: 0; }
.rr-th-date, .rr-td-date { width: 130px; white-space: nowrap; color: var(--color-text-dim); }

.rr-row { cursor: pointer; }
.rr-row:hover { background: var(--color-surface-high); }
.rr-row.selected { background: var(--color-primary-container); }

.rr-table tbody td {
  padding: 11px 14px;
  border-bottom: 1px solid var(--color-outline-dim);
  color: var(--color-text);
}

.rr-cell {
  display: block;
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Наличие: состояние строки видно, не читая текст. */
.rr-stock {
  display: inline-block;
  padding: 2px 10px;
  border-radius: var(--radius-full);
  background: var(--color-success-container);
  color: var(--color-on-success-container);
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}

.rr-stock.taken { background: var(--color-warning-container); color: var(--color-on-warning-container); }

/* Превью картинки. Размер задан всеми тремя свойствами: глобальный мобильный
   `button { min-height: 36px }` иначе растянул бы плитку. */
.rr-thumb {
  display: block;
  width: 44px; min-width: 44px;
  height: 44px; min-height: 44px;
  padding: 0;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-sm);
  background: var(--color-surface-low);
  overflow: hidden;
  cursor: zoom-in;
}

.rr-thumb:hover { border-color: var(--color-primary); }
.rr-thumb img { width: 100%; height: 100%; object-fit: cover; display: block; }

/* В карточке (телефон, узкая панель) превью занимает ВСЮ ширину карточки и
   становится прямоугольным: снимки почти всегда горизонтальные, а места вширь
   там достаточно — подпись поля уходит над картинкой. */
.rr-card-row.image {
  flex-direction: column;
  align-items: stretch;
  gap: 6px;
}

.rr-card-row.image .rr-card-label { width: auto; max-width: none; }

.rr-thumb.card {
  width: 100%; min-width: 0;
  height: 160px; min-height: 160px;
}

.rr-overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: color-mix(in oklch, var(--color-surface) 60%, transparent);
}

.rr-empty { position: absolute; inset: 0; pointer-events: none; }

/* ── Карточки записей (узкая раскладка) ── */
.rr-cards { flex: 1; min-height: 0; overflow-y: auto; padding: 12px; }
.rr-cards > :deep(*) { flex-shrink: 0; }

.rr-cards-selall {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 4px 0;
  font-size: 13px;
  color: var(--color-text-dim);
}

.rr-card-head { display: flex; align-items: center; gap: 10px; }
.rr-card-check { flex: none; display: inline-flex; }

.rr-card-title {
  flex: 1;
  min-width: 0;
  font-size: 15px;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rr-card-chev { flex: none; color: var(--color-text-dim); }
.rr-card-body { display: flex; flex-direction: column; gap: 6px; }
.rr-card-row { display: flex; gap: 10px; font-size: 14px; }

.rr-card-label {
  flex: none;
  width: 40%;
  max-width: 160px;
  color: var(--color-text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rr-card-val { flex: 1; min-width: 0; word-break: break-word; }
</style>
