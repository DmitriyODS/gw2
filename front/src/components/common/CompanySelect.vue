<template>
  <!-- Селектор компании. Три варианта:
         variant="pill"  — PrimeVue Select, компактный pill-style для шапок;
         variant="row"   — row-trigger + плавающая панель, для сайдбара;
         variant="form"  — триггер как .ctl + поповер, для форм (v-model).
       pill/row без v-model управляют companies.activeCompanyId глобально.
       При передаче v-model (controlled mode) компонент работает независимо. -->
  <template v-if="fixed">
    <div class="company-chip" :title="companyLabel">
      <span class="material-symbols-outlined company-icon">domain</span>
      <span class="company-chip-label">{{ companyLabel }}</span>
    </div>
  </template>

  <template v-else-if="variant === 'row'">
    <button
      ref="triggerEl"
      type="button"
      class="company-row"
      :class="{ open }"
      @click="toggle"
      :aria-expanded="open"
      :title="activeLabel || placeholder"
    >
      <span class="company-row-badge" aria-hidden="true">
        <span v-if="activeInitial">{{ activeInitial }}</span>
        <span v-else class="material-symbols-outlined">domain</span>
      </span>
      <span class="company-row-text">
        <span class="company-row-label">{{ activeLabel || placeholder }}</span>
        <span class="company-row-sub">Активная компания</span>
      </span>
      <span class="material-symbols-outlined company-row-chev">
        unfold_more
      </span>
    </button>

    <Teleport to="body">
      <transition name="company-pop">
        <div
          v-if="open"
          ref="popoverEl"
          class="company-popover"
          :style="popoverStyle"
          role="listbox"
          @mousedown.stop
        >
          <header class="company-popover-head">
            <span class="company-popover-title">Сменить компанию</span>
            <button
              class="company-popover-close"
              type="button"
              @click="close"
              title="Закрыть"
              aria-label="Закрыть"
            >
              <span class="material-symbols-outlined">close</span>
            </button>
          </header>

          <div v-if="rowList.length > 6" class="company-popover-search">
            <span class="material-symbols-outlined">search</span>
            <input
              ref="searchEl"
              v-model="query"
              type="text"
              placeholder="Поиск компании…"
              autocomplete="off"
            />
            <button
              v-if="query"
              class="company-popover-search-clear"
              type="button"
              @click="query = ''"
              title="Очистить"
              aria-label="Очистить"
            >
              <span class="material-symbols-outlined">close</span>
            </button>
          </div>

          <div class="company-popover-body">
            <button
              v-if="showAllOption"
              type="button"
              class="company-popover-item all"
              :class="{ active: effectiveValue == null }"
              @click="onPick(null)"
            >
              <span class="company-popover-badge" aria-hidden="true">
                <span class="material-symbols-outlined">public</span>
              </span>
              <span class="company-popover-text">
                <span class="company-popover-name">Все компании</span>
                <span class="company-popover-meta">Без фильтра — данные по всем</span>
              </span>
              <span
                v-if="effectiveValue == null"
                class="material-symbols-outlined company-popover-check"
              >check</span>
            </button>

            <div v-if="showAllOption" class="company-popover-sep" />

            <div v-if="!filteredCompanies.length" class="company-popover-empty">
              <span class="material-symbols-outlined">search_off</span>
              <span>{{ query ? 'Ничего не найдено' : 'Компании не загружены' }}</span>
            </div>

            <button
              v-for="c in filteredCompanies"
              :key="c.id"
              type="button"
              class="company-popover-item"
              :class="{ active: c.id === effectiveValue }"
              @click="onPick(c.id)"
            >
              <span class="company-popover-badge" aria-hidden="true">
                {{ initialOf(c.name) }}
              </span>
              <span class="company-popover-text">
                <span class="company-popover-name">{{ c.name }}</span>
                <span v-if="c.users_count != null" class="company-popover-meta">
                  {{ c.users_count }} {{ pluralUsers(c.users_count) }}
                </span>
              </span>
              <span
                v-if="c.id === effectiveValue"
                class="material-symbols-outlined company-popover-check"
              >check</span>
            </button>
          </div>
        </div>
      </transition>
    </Teleport>
  </template>

  <template v-else-if="variant === 'form'">
    <button
      ref="triggerEl"
      type="button"
      class="company-form-trigger"
      :class="{ open }"
      @click="toggle"
      :aria-expanded="open"
    >
      <span class="company-form-label" :class="{ 'is-placeholder': effectiveValue == null }">
        {{ effectiveValue != null ? labelOf(effectiveValue) : placeholder }}
      </span>
      <span class="material-symbols-outlined company-form-chev">expand_more</span>
    </button>

    <Teleport to="body">
      <transition name="company-pop">
        <div
          v-if="open"
          ref="popoverEl"
          class="company-popover"
          :style="popoverStyle"
          role="listbox"
          @mousedown.stop
        >
          <div v-if="companies.items.length > 6" class="company-popover-search">
            <span class="material-symbols-outlined">search</span>
            <input
              ref="searchEl"
              v-model="query"
              type="text"
              placeholder="Поиск компании…"
              autocomplete="off"
            />
            <button
              v-if="query"
              class="company-popover-search-clear"
              type="button"
              @click="query = ''"
              aria-label="Очистить"
            >
              <span class="material-symbols-outlined">close</span>
            </button>
          </div>

          <div class="company-popover-body">
            <button
              type="button"
              class="company-popover-item"
              :class="{ active: effectiveValue == null }"
              @click="onPick(null)"
            >
              <span class="company-popover-badge" aria-hidden="true">
                <span class="material-symbols-outlined">do_not_disturb_on</span>
              </span>
              <span class="company-popover-text">
                <span class="company-popover-name">{{ placeholder }}</span>
              </span>
              <span
                v-if="effectiveValue == null"
                class="material-symbols-outlined company-popover-check"
              >check</span>
            </button>

            <div v-if="filteredCompanies.length" class="company-popover-sep" />

            <div v-if="!filteredCompanies.length && query" class="company-popover-empty">
              <span class="material-symbols-outlined">search_off</span>
              <span>Ничего не найдено</span>
            </div>

            <button
              v-for="c in filteredCompanies"
              :key="c.id"
              type="button"
              class="company-popover-item"
              :class="{ active: c.id === effectiveValue }"
              @click="onPick(c.id)"
            >
              <span class="company-popover-badge" aria-hidden="true">
                {{ initialOf(c.name) }}
              </span>
              <span class="company-popover-text">
                <span class="company-popover-name">{{ c.name }}</span>
                <span v-if="c.users_count != null" class="company-popover-meta">
                  {{ c.users_count }} {{ pluralUsers(c.users_count) }}
                </span>
              </span>
              <span
                v-if="c.id === effectiveValue"
                class="material-symbols-outlined company-popover-check"
              >check</span>
            </button>
          </div>
        </div>
      </transition>
    </Teleport>
  </template>

  <template v-else>
    <Select
      :model-value="effectiveValue"
      :options="options"
      option-label="name"
      option-value="id"
      :placeholder="placeholder"
      :class="['company-select', { compact }]"
      show-clear
      :filter="companies.items.length > 6"
      filter-placeholder="Поиск компании…"
      scroll-height="320px"
      empty-message="Компании не загружены"
      empty-filter-message="Ничего не найдено"
      @update:model-value="onChange"
      @show="emit('show')"
      @hide="emit('hide')"
    >
      <template #value="slotProps">
        <span class="company-value">
          <span class="material-symbols-outlined company-icon">domain</span>
          <span class="company-value-label">
            {{ labelOf(slotProps.value) || placeholder }}
          </span>
        </span>
      </template>
      <template #option="slotProps">
        <span class="company-option">
          <span class="material-symbols-outlined company-icon">domain</span>
          <span class="company-option-label">{{ slotProps.option.name }}</span>
        </span>
      </template>
    </Select>
  </template>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Select from 'primevue/select'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'

const props = defineProps({
  modelValue: { default: undefined }, // если передан — controlled mode (не трогает companies.activeCompanyId)
  compact: { type: Boolean, default: false },
  placeholder: { type: String, default: 'Все компании' },
  variant: { type: String, default: 'pill' }, // 'pill' | 'row'
})

const emit = defineEmits(['show', 'hide', 'update:modelValue'])

const auth = useAuthStore()
const companies = useCompaniesStore()

// Многокомпанийный обычный пользователь — переключает активную компанию из
// своих членств (auth.companies) через switchCompany (перевыпуск токена).
const isMulti = computed(() => auth.isMultiCompany)
// Платформенный супер-админ — локально выбирает компанию для платформенных
// экранов (через companies.setActive, без перевыпуска токена).
const isSuper = computed(() => auth.isSuperAdmin)
// Неизменяемый чип — у обычного пользователя ровно с одной активной компанией.
const fixed = computed(() => !isSuper.value && auth.companyId != null && !isMulti.value)
const companyLabel = computed(() => auth.companyName || 'Без компании')
const options = computed(() => companies.items)

// Список для row-поповера: у многокомпанийного — его членства, у супер-админа —
// все компании (с опцией «Все компании»).
const rowList = computed(() => {
  if (isMulti.value) {
    return auth.companies.map((c) => ({ id: c.company_id, name: c.company_name, is_active: c.is_active }))
  }
  return companies.items
})
// «Все компании» (null) — только супер-админу, не многокомпанийному пользователю.
const showAllOption = computed(() => isSuper.value)

// controlled mode: если props.modelValue передан — используем его, иначе —
// активная компания (для многокомпанийного — из токена auth.companyId,
// для супер-админа — выбранная локально companies.activeCompanyId).
const isControlled = computed(() => props.modelValue !== undefined)
const effectiveValue = computed(() => {
  if (isControlled.value) return props.modelValue
  if (isMulti.value) return auth.companyId
  return companies.activeCompanyId
})

const activeLabel = computed(() => {
  if (isMulti.value) return auth.companyName
  return companies.activeCompany?.name ?? null
})

const activeInitial = computed(() => initialOf(activeLabel.value))

function initialOf(name) {
  if (!name) return ''
  const t = name.trim()
  if (!t) return ''
  return t[0].toUpperCase()
}

function pluralUsers(n) {
  const m10 = n % 10
  const m100 = n % 100
  if (m10 === 1 && m100 !== 11) return 'сотрудник'
  if ([2, 3, 4].includes(m10) && ![12, 13, 14].includes(m100)) return 'сотрудника'
  return 'сотрудников'
}

function labelOf(id) {
  if (id == null) return null
  const c = companies.items.find((x) => x.id === id)
  return c?.name ?? null
}

/* ---------- variant='row' state & popover ---------- */
const open = ref(false)
const query = ref('')
const triggerEl = ref(null)
const popoverEl = ref(null)
const searchEl = ref(null)
const popoverStyle = ref({})

const filteredCompanies = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return rowList.value
  return rowList.value.filter((c) => (c.name || '').toLowerCase().includes(q))
})

function computePosition() {
  const el = triggerEl.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const gap = 8
  const width = Math.max(rect.width, 320)
  const maxLeft = window.innerWidth - width - 12
  const left = Math.min(Math.max(12, rect.left), Math.max(12, maxLeft))
  const top = rect.bottom + gap
  popoverStyle.value = {
    top: `${top}px`,
    left: `${left}px`,
    width: `${width}px`,
  }
}

function toggle() {
  if (open.value) close()
  else openPopover()
}

async function openPopover() {
  open.value = true
  emit('show')
  await nextTick()
  computePosition()
  if (searchEl.value) {
    searchEl.value.focus()
  }
  window.addEventListener('resize', computePosition)
  window.addEventListener('scroll', computePosition, true)
  document.addEventListener('mousedown', onDocMouseDown, true)
  document.addEventListener('keydown', onDocKeydown, true)
}

function close() {
  if (!open.value) return
  open.value = false
  query.value = ''
  emit('hide')
  window.removeEventListener('resize', computePosition)
  window.removeEventListener('scroll', computePosition, true)
  document.removeEventListener('mousedown', onDocMouseDown, true)
  document.removeEventListener('keydown', onDocKeydown, true)
}

function onDocMouseDown(e) {
  const t = e.target
  if (triggerEl.value && triggerEl.value.contains(t)) return
  if (popoverEl.value && popoverEl.value.contains(t)) return
  close()
}

function onDocKeydown(e) {
  if (e.key === 'Escape') close()
}

function onPick(id) {
  if (isControlled.value) {
    emit('update:modelValue', id ?? null)
    close()
    return
  }
  if (isMulti.value) {
    // Перевыпуск токена под выбранную компанию; данные перезагрузятся по watch.
    if (id != null && id !== auth.companyId) auth.switchCompany(id).catch(() => {})
    close()
    return
  }
  companies.setActive(id)
  close()
}

/* ---------- common ---------- */
onMounted(() => {
  // Список компаний из API (платформенный эндпоинт) нужен только супер-админу;
  // многокомпанийный пользователь берёт свои компании из auth.companies.
  if (isSuper.value) companies.load()
})

onBeforeUnmount(() => {
  if (open.value) close()
})

function onChange(value) {
  if (isControlled.value) {
    emit('update:modelValue', value ?? null)
  } else {
    companies.setActive(value ?? null)
  }
}

// При уходе на другой раздел/изменении layout схлопываем поповер.
watch(() => auth.companyId, (v) => {
  if (v != null) close()
})
</script>

<style scoped>
.company-value,
.company-option {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.company-icon {
  font-size: 18px;
  opacity: 0.75;
  flex-shrink: 0;
}

.company-value-label,
.company-option-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:deep(.company-select.p-select) {
  font-size: 13px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  border-color: transparent;
  min-width: 200px;
  max-width: 280px;
  font-weight: 600;
}

:deep(.company-select.p-select:hover) {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

:deep(.company-select .p-select-label) {
  padding: 8px 12px;
  display: flex;
  align-items: center;
}

:deep(.company-select.compact.p-select) {
  font-size: 12px;
  min-width: 180px;
  max-width: 240px;
}

:deep(.company-select.compact .p-select-label) {
  padding: 6px 10px;
}

.company-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  padding: 0 14px;
  border-radius: var(--radius-full, 999px);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-weight: 600;
  font-size: 13px;
  max-width: 240px;
}

.company-chip-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ---------- variant='row' ---------- */
.company-row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 8px 10px;
  border: 1px solid transparent;
  border-radius: var(--radius-lg, 14px);
  background: var(--color-surface-high);
  color: var(--color-text);
  cursor: pointer;
  text-align: left;
  font: inherit;
  transition: background 0.15s, border-color 0.15s, box-shadow 0.15s;
}

.company-row:hover {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.company-row.open {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  border-color: color-mix(in oklch, var(--color-primary) 30%, transparent);
  box-shadow: var(--shadow-sm);
}

.company-row-badge {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border-radius: var(--radius-md, 10px);
  display: grid;
  place-items: center;
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-weight: 700;
  font-size: 14px;
  letter-spacing: 0.2px;
}

.company-row-badge .material-symbols-outlined {
  font-size: 18px;
}

.company-row-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.company-row-label {
  font-size: 13.5px;
  font-weight: 700;
  line-height: 1.2;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.company-row-sub {
  font-size: 11px;
  font-weight: 500;
  line-height: 1.2;
  opacity: 0.7;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.company-row-chev {
  font-size: 20px;
  opacity: 0.6;
  flex-shrink: 0;
  transition: transform 0.18s;
}

.company-row.open .company-row-chev {
  transform: rotate(180deg);
}

/* ---------- variant='form' ---------- */
.company-form-trigger {
  appearance: none;
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface-high);
  color: var(--color-on-surface);
  padding: 10px 12px;
  border-radius: var(--radius-md, 12px);
  font: inherit;
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.company-form-trigger:hover {
  border-color: var(--color-outline);
}

.company-form-trigger.open,
.company-form-trigger:focus-visible {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px color-mix(in oklab, var(--color-primary) 18%, transparent);
}

.company-form-label {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.company-form-label.is-placeholder {
  color: var(--color-on-surface-variant);
  opacity: 0.6;
}

.company-form-chev {
  font-size: 20px;
  color: var(--color-on-surface-variant);
  flex-shrink: 0;
  transition: transform 0.18s;
}

.company-form-trigger.open .company-form-chev {
  transform: rotate(180deg);
}
</style>

<style>
/* Поповер живёт в body (Teleport), поэтому стили — глобальные. */
.company-popover {
  position: fixed;
  z-index: 2000;
  background: var(--color-surface);
  border-radius: var(--radius-xl, 18px);
  border: 1px solid var(--color-outline-dim);
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  max-height: min(70vh, 520px);
  overflow: hidden;
  font-size: 13px;
}

.company-popover-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px 8px;
}

.company-popover-title {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.6px;
  color: var(--color-text-dim);
}

.company-popover-close {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: transparent;
  border: none;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.company-popover-close:hover {
  background: var(--color-surface-high);
  color: var(--color-text);
}

.company-popover-close .material-symbols-outlined { font-size: 18px; }

.company-popover-search {
  position: relative;
  display: flex;
  align-items: center;
  padding: 0 14px 10px;
}

.company-popover-search > .material-symbols-outlined {
  position: absolute;
  left: 26px;
  font-size: 18px;
  color: var(--color-text-dim);
  pointer-events: none;
}

.company-popover-search input {
  flex: 1;
  height: 36px;
  padding: 0 32px 0 36px;
  border-radius: var(--radius-full, 999px);
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface-low);
  color: var(--color-text);
  font: inherit;
  font-size: 13px;
  outline: none;
  transition: border-color 0.15s, background 0.15s;
}

.company-popover-search input:focus {
  border-color: var(--color-primary);
  background: var(--color-surface);
}

.company-popover-search-clear {
  position: absolute;
  right: 22px;
  display: grid;
  place-items: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: transparent;
  border: none;
  color: var(--color-text-dim);
  cursor: pointer;
}

.company-popover-search-clear:hover {
  background: var(--color-surface-high);
  color: var(--color-text);
}

.company-popover-search-clear .material-symbols-outlined { font-size: 14px; }

.company-popover-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 4px 8px 8px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.company-popover-sep {
  height: 1px;
  background: var(--color-outline-dim);
  margin: 6px 6px;
}

.company-popover-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 8px;
  border-radius: var(--radius-md, 10px);
  border: none;
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  text-align: left;
  font: inherit;
  transition: background 0.12s;
}

.company-popover-item:hover {
  background: var(--color-surface-high);
}

.company-popover-item.active {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.company-popover-badge {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border-radius: var(--radius-md, 10px);
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  font-weight: 700;
  font-size: 13px;
}

.company-popover-badge .material-symbols-outlined { font-size: 18px; }

.company-popover-item.all .company-popover-badge {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}

.company-popover-item.active .company-popover-badge {
  background: var(--color-primary);
  color: var(--color-on-primary);
}

.company-popover-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.company-popover-name {
  font-size: 13.5px;
  font-weight: 600;
  line-height: 1.2;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.company-popover-meta {
  font-size: 11px;
  font-weight: 500;
  line-height: 1.2;
  opacity: 0.7;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.company-popover-check {
  font-size: 20px;
  flex-shrink: 0;
}

.company-popover-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 24px 12px;
  color: var(--color-text-dim);
  font-size: 12.5px;
}

.company-popover-empty .material-symbols-outlined { font-size: 28px; opacity: 0.6; }

/* Транзишн появления поповера */
.company-pop-enter-from,
.company-pop-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.98);
}

.company-pop-enter-active,
.company-pop-leave-active {
  transition: opacity 0.16s ease, transform 0.16s cubic-bezier(0.2, 0, 0, 1);
  transform-origin: top left;
}

@media (max-width: 600px) {
  .company-popover {
    max-width: calc(100vw - 24px);
  }
}
</style>
