<template>
  <!-- Селектор активной компании: пилюля «портфель · компания · ▾» и поповер
       со списком (панель задач телефона, подвал меню «Пуск»). Без v-model
       переключает активную компанию глобально, с v-model — работает
       независимо (controlled mode). -->
  <template v-if="fixed">
    <!-- Выбирать не из чего (одна компания) — тот же вид, но без стрелки. -->
    <div v-bind="$attrs" class="company-button is-static" :title="companyLabel">
      <span class="material-symbols-outlined company-button-ico">business_center</span>
      <span class="company-button-label">{{ companyLabel }}</span>
    </div>
  </template>

  <template v-else>
    <button
      v-bind="$attrs"
      ref="triggerEl"
      type="button"
      class="company-button"
      :class="{ open }"
      @click="toggle"
      :aria-expanded="open"
      :title="activeLabel || placeholder"
    >
      <span class="material-symbols-outlined company-button-ico">business_center</span>
      <span class="company-button-label">{{ activeLabel || placeholder }}</span>
      <span class="material-symbols-outlined company-button-chev">expand_more</span>
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

</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  modelValue: { default: undefined }, // если передан — controlled mode (не трогает companies.activeCompanyId)
  placeholder: { type: String, default: 'Все компании' },
})

const emit = defineEmits(['update:modelValue'])

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

// Список для поповера: у многокомпанийного — его членства, у супер-админа —
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

/* ---------- поповер выбора ---------- */
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
  // Триггер может стоять у нижней кромки экрана (подвал меню «Пуск») — тогда
  // раскрываемся вверх, а не за край.
  const height = popoverEl.value?.offsetHeight || 0
  const below = rect.bottom + gap
  const top = height && below + height > window.innerHeight - 12
    ? Math.max(12, rect.top - gap - height)
    : below
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

// При уходе на другой раздел/изменении layout схлопываем поповер.
watch(() => auth.companyId, (v) => {
  if (v != null) close()
})
</script>

<style scoped>
/* Пилюля-триггер: портфель, название компании и стрелка. */
.company-button {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  /* Длинное название обрезается многоточием и не распирает соседей: пилюля
     стоит в тесных строках (подвал «Пуска», панель статусов). */
  min-width: 0;
  max-width: 100%;
  height: 52px;
  padding: 0 14px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full, 999px);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit;
  font-size: 15px;
  font-weight: 500;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.company-button.is-static { cursor: default; }

.company-button:hover:not(.is-static),
.company-button.open {
  border-color: color-mix(in oklch, var(--color-primary) 34%, var(--acrylic-border));
  background: color-mix(in oklch, var(--color-primary) 7%, var(--glass-bg));
}

.company-button-ico { font-size: 22px; flex-shrink: 0; }

.company-button-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.company-button-chev {
  flex-shrink: 0;
  font-size: 22px;
  color: var(--color-text-dim);
  transition: rotate 0.2s ease;
}

.company-button.open .company-button-chev { rotate: 180deg; }

.company-popover {
  position: fixed;
  z-index: 2000;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border-radius: var(--radius-xl, 18px);
  border: 1px solid var(--acrylic-border);
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
  height: 28px; min-height: 0;
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
  height: 22px; min-height: 0;
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
  background: var(--grad-primary-soft);
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
  background: var(--grad-primary);
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
