<template>
  <div class="rm">
    <div class="rm-head">
      <h3 class="rm-title">Реестры компании</h3>
      <p class="rm-sub">Создавайте таблицы-справочники и настраивайте их поля. Сотрудники ведут в них записи.</p>
    </div>

    <div class="rm-body">
      <!-- Список реестров -->
      <aside class="rm-side">
        <div class="rm-side-head">
          <span>Список</span>
          <button class="rm-add-circle" title="Новый реестр" @click="openCreate">
            <span class="material-symbols-outlined">add</span>
          </button>
        </div>
        <div class="rm-side-list">
          <div v-if="store.loadingList" class="rm-side-note">Загрузка…</div>
          <div v-else-if="!store.registries.length" class="rm-side-note">Реестров пока нет</div>
          <button
            v-for="r in store.registries"
            :key="r.id"
            class="rm-side-item"
            :class="{ active: r.id === editId }"
            @click="selectRegistry(r)"
          >
            <span class="material-symbols-outlined">list_alt</span>
            <span class="rm-side-name">{{ r.name }}</span>
          </button>
        </div>
      </aside>

      <!-- Редактор выбранного реестра -->
      <section class="rm-detail">
        <template v-if="current">
          <div class="rm-detail-head">
            <input v-model="editName" class="ctl rm-name" placeholder="Название реестра" @input="dirty = true" />
            <button class="rm-icon-btn danger" title="Удалить реестр" @click="confirmDelete = true">
              <span class="material-symbols-outlined">delete</span>
            </button>
          </div>

          <div class="rm-fields">
            <div v-if="!editFields.length" class="rm-fields-empty">
              <span class="material-symbols-outlined">dashboard_customize</span>
              <p>В реестре пока нет полей</p>
              <button class="rm-btn-tonal" @click="openField(-1)">
                <span class="material-symbols-outlined">add</span> Добавить первое поле
              </button>
            </div>

            <ul v-else class="rm-field-rows">
              <li
                v-for="(f, i) in editFields"
                :key="f._k"
                class="rm-field-row"
                @click="openField(i)"
              >
                <span class="rm-reorder" @click.stop>
                  <button class="rm-reorder-btn" :disabled="i === 0" title="Выше" @click="move(i, -1)">
                    <span class="material-symbols-outlined">keyboard_arrow_up</span>
                  </button>
                  <button class="rm-reorder-btn" :disabled="i === editFields.length - 1" title="Ниже" @click="move(i, 1)">
                    <span class="material-symbols-outlined">keyboard_arrow_down</span>
                  </button>
                </span>

                <span class="rm-field-icon"><span class="material-symbols-outlined">{{ fieldIcon(f.type) }}</span></span>

                <span class="rm-field-main">
                  <span class="rm-field-label">{{ f.label || 'Без названия' }}</span>
                  <span class="rm-field-type">{{ fieldLabel(f.type) }}</span>
                </span>

                <span class="rm-field-meta">
                  <span class="rm-badge" :title="`Ширина ${f.col_span} · высота ${f.row_span}`">{{ f.col_span }}×{{ f.row_span }}</span>
                  <span class="rm-badge col" :class="{ off: !f.show_in_table }" :title="f.show_in_table ? 'Показывается колонкой в таблице' : 'Скрыто из таблицы'">
                    <span class="material-symbols-outlined">{{ f.show_in_table ? 'visibility' : 'visibility_off' }}</span>
                  </span>
                </span>

                <span class="rm-field-actions" @click.stop>
                  <button class="rm-icon-btn sm" title="Настроить" @click="openField(i)">
                    <span class="material-symbols-outlined">tune</span>
                  </button>
                  <button class="rm-icon-btn sm danger" title="Удалить поле" @click="removeField(i)">
                    <span class="material-symbols-outlined">close</span>
                  </button>
                </span>
              </li>
            </ul>
          </div>

          <div class="rm-detail-foot">
            <button v-if="editFields.length" class="rm-btn-tonal" @click="openField(-1)">
              <span class="material-symbols-outlined">add</span> Добавить поле
            </button>
            <span v-if="dirty" class="rm-dirty"><span class="rm-dot" /> Есть несохранённые изменения</span>
            <span class="rm-foot-spacer" />
            <button class="rm-btn-primary" :disabled="saving || !dirty" @click="save">
              <span v-if="saving" class="material-symbols-outlined spin">progress_activity</span>
              Сохранить
            </button>
          </div>
        </template>

        <div v-else class="rm-detail-empty">
          <span class="material-symbols-outlined">table_view</span>
          <p>Выберите реестр слева или создайте новый</p>
        </div>
      </section>
    </div>

    <!-- Настройка поля -->
    <AppDialog
      v-model="fieldOpen"
      :title="fieldIndex === -1 ? 'Новое поле' : 'Настройка поля'"
      :icon="draft.type ? fieldIcon(draft.type) : 'add_box'"
      size="md"
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Готово' }]"
      @cancel="fieldOpen = false"
      @confirm="applyField"
    >
      <div class="fd">
        <label class="fd-field">
          <span class="fd-label">Название</span>
          <input v-model="draft.label" class="ctl" placeholder="Например: Серийный номер" />
        </label>

        <label class="fd-field">
          <span class="fd-label">Тип</span>
          <Select
            v-model="draft.type" :options="typeOptions" option-label="label" option-value="value"
            @update:model-value="onDraftType"
          >
            <template #option="{ option }">
              <span class="fd-type-opt"><span class="material-symbols-outlined">{{ fieldIcon(option.value) }}</span>{{ option.label }}</span>
            </template>
          </Select>
        </label>

        <!-- Конфиг типа -->
        <label v-if="draft.type === 'text'" class="fd-check">
          <Checkbox v-model="draft.config.multiline" binary /> Многострочное поле
        </label>

        <label v-else-if="draft.type === 'number'" class="fd-field">
          <span class="fd-label">Шаблон номера (regex, необязательно)</span>
          <input v-model="draft.config.pattern" class="ctl" placeholder="напр. ^\d{3}-\d{4}$" />
          <span class="fd-hint">Если задан — значение должно совпадать с шаблоном.</span>
        </label>

        <div v-else-if="draft.type === 'select'" class="fd-field">
          <span class="fd-label">Варианты выбора</span>
          <div class="fd-options">
            <div v-for="(opt, oi) in draft.config.options" :key="oi" class="fd-option">
              <input :value="opt" class="ctl" placeholder="Вариант" @input="draft.config.options[oi] = $event.target.value" />
              <button class="rm-icon-btn sm danger" title="Удалить" @click="draft.config.options.splice(oi, 1)">
                <span class="material-symbols-outlined">close</span>
              </button>
            </div>
          </div>
          <button class="rm-btn-text" @click="draft.config.options.push('')">
            <span class="material-symbols-outlined">add</span> Добавить вариант
          </button>
          <label class="fd-check"><Checkbox v-model="draft.config.multiple" binary /> Разрешить выбор нескольких</label>
        </div>

        <div v-else-if="draft.type === 'datetime'" class="fd-field">
          <span class="fd-label">Что показывать</span>
          <div class="fd-inline">
            <label class="fd-check"><Checkbox v-model="draft.config.year" binary /> Год</label>
            <label class="fd-check"><Checkbox v-model="draft.config.month_day" binary /> Месяц и день</label>
            <label class="fd-check"><Checkbox v-model="draft.config.time" binary /> Время</label>
          </div>
        </div>

        <!-- Раскладка -->
        <div class="fd-grid">
          <div class="fd-field">
            <span class="fd-label">Ширина (колонок)</span>
            <div class="fd-seg">
              <button v-for="n in 3" :key="n" :class="{ active: draft.col_span === n }" @click="draft.col_span = n">{{ n }}</button>
            </div>
          </div>
          <div class="fd-field">
            <span class="fd-label">Высота (строк)</span>
            <div class="fd-seg">
              <button v-for="n in 3" :key="n" :class="{ active: draft.row_span === n }" @click="draft.row_span = n">{{ n }}</button>
            </div>
          </div>
        </div>

        <label class="fd-check"><Checkbox v-model="draft.show_in_table" binary /> Показывать колонкой в таблице</label>
      </div>
    </AppDialog>

    <!-- Создание реестра -->
    <AppDialog
      v-model="creating" title="Новый реестр" icon="add" :busy="saving"
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Создать' }]"
      @cancel="creating = false" @confirm="doCreate"
    >
      <input v-model="newName" class="ctl" placeholder="Название реестра" @keyup.enter="doCreate" />
    </AppDialog>

    <ConfirmDialog
      :visible="confirmDelete"
      header="Удалить реестр?"
      message="Реестр и все его записи будут удалены безвозвратно."
      confirm-label="Удалить" danger-confirm
      @confirm="doDelete" @cancel="confirmDelete = false"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import Select from 'primevue/select'
import Checkbox from 'primevue/checkbox'
import AppDialog from '@/components/common/AppDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import * as api from '@/api/registries.js'
import { useRegistriesStore } from '@/stores/registries.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { FIELD_TYPES, defaultConfig, fieldIcon, fieldLabel } from '@/utils/registryFields.js'

const store = useRegistriesStore()
const notif = useNotificationsStore()

const typeOptions = FIELD_TYPES.map((f) => ({ label: f.label, value: f.value || f.type }))

const editId = ref(null)
const editName = ref('')
const editFields = ref([])
const dirty = ref(false)
const saving = ref(false)
const creating = ref(false)
const newName = ref('')
const confirmDelete = ref(false)

let keySeq = 0
const current = computed(() => store.registries.find((r) => r.id === editId.value) || null)

function selectRegistry(r) {
  editId.value = r.id
  editName.value = r.name
  editFields.value = (r.fields || []).map((f) => normalizeField(f))
  dirty.value = false
}

function normalizeField(f) {
  return {
    _k: ++keySeq,
    id: f.id || 0,
    label: f.label || '',
    type: f.type || 'text',
    config: { ...defaultConfig(f.type || 'text'), ...(f.config || {}) },
    col_span: clampSpan(f.col_span),
    row_span: clampSpan(f.row_span),
    show_in_table: f.show_in_table !== false,
  }
}
function clampSpan(v) { return Math.min(3, Math.max(1, v || 1)) }

function removeField(i) { editFields.value.splice(i, 1); dirty.value = true }
function move(i, dir) {
  const j = i + dir
  if (j < 0 || j >= editFields.value.length) return
  const arr = editFields.value
  ;[arr[i], arr[j]] = [arr[j], arr[i]]
  dirty.value = true
}

// ── Диалог настройки поля ──
const fieldOpen = ref(false)
const fieldIndex = ref(-1)
const draft = reactive({ _k: 0, id: 0, label: '', type: 'text', config: {}, col_span: 1, row_span: 1, show_in_table: true })

function openField(i) {
  fieldIndex.value = i
  const src = i === -1
    ? { id: 0, label: '', type: 'text', config: defaultConfig('text'), col_span: 1, row_span: 1, show_in_table: true }
    : editFields.value[i]
  Object.assign(draft, {
    _k: src._k ?? 0,
    id: src.id || 0,
    label: src.label || '',
    type: src.type || 'text',
    config: deepConfig(src.type, src.config),
    col_span: clampSpan(src.col_span),
    row_span: clampSpan(src.row_span),
    show_in_table: src.show_in_table !== false,
  })
  fieldOpen.value = true
}
function deepConfig(type, config) {
  const base = { ...defaultConfig(type), ...(config || {}) }
  if (Array.isArray(base.options)) base.options = [...base.options]
  return base
}
function onDraftType() { Object.assign(draft, { config: defaultConfig(draft.type) }) }

function applyField() {
  if (!draft.label.trim()) { notif.error('Укажите название поля'); return }
  const field = {
    _k: draft._k || ++keySeq,
    id: draft.id || 0,
    label: draft.label.trim(),
    type: draft.type,
    config: cleanConfig(draft),
    col_span: draft.col_span,
    row_span: draft.row_span,
    show_in_table: draft.show_in_table,
  }
  if (fieldIndex.value === -1) editFields.value.push(field)
  else editFields.value[fieldIndex.value] = field
  dirty.value = true
  fieldOpen.value = false
}
function cleanConfig(f) {
  if (f.type === 'select') {
    return { options: (f.config.options || []).map((s) => s.trim()).filter(Boolean), multiple: !!f.config.multiple }
  }
  if (f.type === 'number') return { pattern: (f.config.pattern || '').trim() }
  if (f.type === 'text') return { multiline: !!f.config.multiline }
  if (f.type === 'datetime') return { year: !!f.config.year, month_day: !!f.config.month_day, time: !!f.config.time }
  return {}
}

// ── Сохранение / CRUD реестра ──
async function save() {
  if (!editName.value.trim()) { notif.error('Укажите название реестра'); return }
  for (const f of editFields.value) {
    if (!f.label.trim()) { notif.error('У каждого поля должно быть название'); return }
  }
  saving.value = true
  try {
    if (editName.value.trim() !== current.value.name) {
      await api.updateRegistry(editId.value, editName.value.trim())
    }
    await api.replaceFields(editId.value, editFields.value.map((f) => ({
      id: f.id || 0, label: f.label.trim(), type: f.type, config: f.config,
      col_span: f.col_span, row_span: f.row_span, show_in_table: f.show_in_table,
    })))
    notif.success('Реестр сохранён')
    await store.fetchRegistries()
    const again = store.registries.find((r) => r.id === editId.value)
    if (again) selectRegistry(again)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить реестр')
  } finally {
    saving.value = false
  }
}

function openCreate() { newName.value = ''; creating.value = true }
async function doCreate() {
  if (!newName.value.trim()) return
  saving.value = true
  try {
    const reg = await api.createRegistry(newName.value.trim())
    creating.value = false
    await store.fetchRegistries()
    const created = store.registries.find((r) => r.id === reg.id)
    if (created) selectRegistry(created)
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать реестр')
  } finally {
    saving.value = false
  }
}

async function doDelete() {
  confirmDelete.value = false
  try {
    await api.deleteRegistry(editId.value)
    editId.value = null
    await store.fetchRegistries()
    notif.success('Реестр удалён')
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить реестр')
  }
}

onMounted(() => store.fetchRegistries())
</script>

<style scoped>
/* Заполняет всю область вкладки (.settings-fill) и держит ВНУТРЕННЮЮ прокрутку:
   шапка/панели зафиксированы, скроллится только список реестров и список полей. */
.rm { flex: 1; min-height: 0; width: 100%; display: flex; flex-direction: column; gap: 16px; }
/* Глобальный input.ctl задаёт фон/рамку, но не padding — добавляем. */
.ctl { width: 100%; padding: 10px 12px; font: inherit; appearance: none; }

.rm-head { flex: none; display: flex; flex-direction: column; gap: 4px; }
.rm-title { margin: 0; font-size: 17px; font-weight: 700; color: var(--color-text); }
.rm-sub { margin: 0; font-size: 13px; color: var(--color-text-dim); max-width: 560px; }

.rm-body { flex: 1; min-height: 0; display: flex; gap: 16px; align-items: stretch; }

/* ── Левая колонка ── */
.rm-side {
  width: 240px; flex-shrink: 0;
  display: flex; flex-direction: column; min-height: 0;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg); overflow: hidden;
}
.rm-side-head {
  flex: none;
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 14px; font-weight: 700; font-size: 14px; color: var(--color-text);
  border-bottom: 1px solid var(--color-outline-dim);
}
.rm-add-circle {
  width: 30px; height: 30px; display: grid; place-items: center;
  border: none; border-radius: var(--radius-full);
  background: var(--color-primary); color: var(--color-on-primary); cursor: pointer;
}
.rm-add-circle:hover { filter: brightness(1.05); }
.rm-add-circle .material-symbols-outlined { font-size: 20px; }
.rm-side-list { flex: 1; min-height: 0; overflow-y: auto; padding: 8px; display: flex; flex-direction: column; gap: 2px; }
.rm-side-note { padding: 20px 12px; color: var(--color-text-dim); font-size: 13px; text-align: center; }
.rm-side-item {
  display: flex; align-items: center; gap: 10px; width: 100%;
  padding: 10px 12px; border: none; background: none; cursor: pointer;
  border-radius: var(--radius-md); color: var(--color-text-dim);
  text-align: left; font-size: 14px; font-weight: 600;
}
.rm-side-item:hover { background: var(--color-surface-high); color: var(--color-text); }
.rm-side-item.active { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.rm-side-item .material-symbols-outlined { font-size: 20px; flex-shrink: 0; }
.rm-side-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/* ── Правая колонка ── */
.rm-detail {
  flex: 1; min-width: 0; min-height: 0;
  display: flex; flex-direction: column;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg); overflow: hidden;
}
.rm-detail-head {
  flex: none; display: flex; align-items: center; gap: 10px;
  padding: 12px 14px; border-bottom: 1px solid var(--color-outline-dim);
}
.rm-name { flex: 1; font-weight: 700; }

.rm-fields { flex: 1; min-height: 0; overflow-y: auto; padding: 10px 12px; }
.rm-field-rows { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.rm-field-row {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 12px; cursor: pointer;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
  transition: border-color .12s, background .12s;
}
.rm-field-row:hover { background: var(--color-surface-high); border-color: var(--color-outline); }

.rm-reorder { display: flex; flex-direction: column; gap: 2px; flex-shrink: 0; }
.rm-reorder-btn {
  width: 24px; height: 18px; display: grid; place-items: center;
  border: none; border-radius: var(--radius-sm); background: transparent;
  color: var(--color-text-dim); cursor: pointer;
}
.rm-reorder-btn:disabled { opacity: 0.25; cursor: default; }
.rm-reorder-btn:not(:disabled):hover { background: var(--color-surface-low); color: var(--color-text); }
.rm-reorder-btn .material-symbols-outlined { font-size: 18px; }

.rm-field-icon {
  width: 36px; height: 36px; flex-shrink: 0; display: grid; place-items: center;
  border-radius: var(--radius-md); background: var(--color-primary-container); color: var(--color-on-primary-container);
}
.rm-field-icon .material-symbols-outlined { font-size: 20px; }

.rm-field-main { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.rm-field-label { font-size: 14px; font-weight: 600; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.rm-field-type { font-size: 12px; color: var(--color-text-dim); }

.rm-field-meta { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.rm-badge {
  display: inline-flex; align-items: center; justify-content: center; gap: 2px;
  min-width: 34px; height: 24px; padding: 0 8px;
  border-radius: var(--radius-full); background: var(--color-surface-low);
  color: var(--color-text-dim); font-size: 12px; font-weight: 700;
}
.rm-badge.col { min-width: 24px; padding: 0 6px; }
.rm-badge.col .material-symbols-outlined { font-size: 16px; }
.rm-badge.col.off { color: var(--color-error); }

.rm-field-actions { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }

.rm-fields-empty {
  height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 10px; color: var(--color-text-dim);
}
.rm-fields-empty .material-symbols-outlined { font-size: 44px; }
.rm-fields-empty p { margin: 0; font-size: 14px; }

.rm-detail-foot {
  flex: none; display: flex; align-items: center; gap: 12px;
  padding: 12px 14px; border-top: 1px solid var(--color-outline-dim);
}
.rm-foot-spacer { flex: 1; }
.rm-dirty { display: inline-flex; align-items: center; gap: 6px; font-size: 12px; color: var(--color-text-dim); }
.rm-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--color-warning, var(--color-primary)); }

.rm-detail-empty {
  flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 8px; color: var(--color-text-dim);
}
.rm-detail-empty .material-symbols-outlined { font-size: 48px; }
.rm-detail-empty p { margin: 0; font-size: 14px; }

/* ── Диалог поля ── */
.fd { display: flex; flex-direction: column; gap: 16px; }
.fd-field { display: flex; flex-direction: column; gap: 6px; }
.fd-label { font-size: 13px; font-weight: 600; color: var(--color-text-dim); }
.fd-hint { font-size: 12px; color: var(--color-text-dim); }
.fd :deep(.p-select) { width: 100%; }
.fd-type-opt { display: inline-flex; align-items: center; gap: 8px; }
.fd-type-opt .material-symbols-outlined { font-size: 18px; color: var(--color-text-dim); }
.fd-check { display: inline-flex; align-items: center; gap: 8px; font-size: 14px; color: var(--color-text); cursor: pointer; }
.fd-inline { display: flex; flex-wrap: wrap; gap: 16px; }
.fd-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.fd-options { display: flex; flex-direction: column; gap: 8px; }
.fd-option { display: flex; align-items: center; gap: 8px; }

.fd-seg { display: inline-flex; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); overflow: hidden; width: max-content; }
.fd-seg button {
  width: 44px; height: 38px; border: none; background: var(--acrylic-card-bg);
  color: var(--color-text-dim); cursor: pointer; font-weight: 700; font-size: 14px;
  border-right: 1px solid var(--color-outline-dim);
}
.fd-seg button:last-child { border-right: none; }
.fd-seg button.active { background: var(--color-primary); color: var(--color-on-primary); }

/* ── Кнопки ── */
.rm-icon-btn {
  width: 36px; height: 36px; flex-shrink: 0; display: grid; place-items: center;
  border: none; border-radius: var(--radius-full);
  background: var(--color-surface-low); color: var(--color-text-dim); cursor: pointer;
}
.rm-icon-btn:hover { background: var(--color-surface-high); color: var(--color-text); }
.rm-icon-btn.sm { width: 32px; height: 32px; }
.rm-icon-btn.sm .material-symbols-outlined { font-size: 18px; }
.rm-icon-btn.danger { color: var(--color-error); }

.rm-btn-primary {
  display: inline-flex; align-items: center; gap: 6px; height: 38px; padding: 0 18px;
  border: none; border-radius: var(--radius-full);
  background: var(--color-primary); color: var(--color-on-primary);
  font-weight: 600; font-size: 14px; cursor: pointer;
}
.rm-btn-primary:disabled { opacity: 0.5; cursor: default; }
.rm-btn-tonal {
  display: inline-flex; align-items: center; gap: 6px; height: 38px; padding: 0 16px;
  border: none; border-radius: var(--radius-full);
  background: var(--color-primary-container); color: var(--color-on-primary-container);
  font-weight: 600; font-size: 14px; cursor: pointer;
}
.rm-btn-text {
  display: inline-flex; align-items: center; gap: 4px; align-self: flex-start;
  border: none; background: none; cursor: pointer; color: var(--color-primary); font-weight: 600; font-size: 14px;
}
.spin { animation: rmspin 1s linear infinite; }
@keyframes rmspin { to { transform: rotate(360deg); } }

@media (max-width: 768px) {
  .rm-body { flex-direction: column; }
  .rm-side { width: 100%; max-height: 30vh; }
  .fd-grid { grid-template-columns: 1fr; }
}
</style>
