<template>
  <Dialog
    :visible="modelValue"
    @update:visible="onClose"
    :header="isEdit ? 'Редактирование компании' : 'Новая компания'"
    modal
    :style="{ width: '560px' }"
    :pt="{ content: { style: 'overflow: visible' } }"
  >
    <div class="form-body">
      <div class="field">
        <label class="lbl">Название <span class="req">*</span></label>
        <input
          v-model.trim="form.name"
          class="ctl"
          maxlength="255"
          placeholder="ООО «Ромашка»"
          :class="{ invalid: !!errors.name }"
        />
        <div v-if="errors.name" class="err">{{ errors.name }}</div>
      </div>

      <div class="field">
        <label class="lbl">Описание</label>
        <textarea
          v-model.trim="form.description"
          class="ctl ctl-area"
          rows="2"
          placeholder="Несколько слов о компании (необязательно)"
        />
      </div>

      <div class="field">
        <label class="lbl">Руководитель</label>
        <select v-model="form.director_id" class="ctl">
          <option :value="null">— не выбран —</option>
          <option v-for="u in directors" :key="u.id" :value="u.id">
            {{ u.fio }} <template v-if="u.login">({{ u.login }})</template>
          </option>
        </select>
        <div class="hint">
          Корневой Руководитель компании. Его не могут разжаловать другие Руководители — только Администратор.
        </div>
      </div>

      <div class="field">
        <label class="lbl">Настройки</label>
        <div class="switch-list">
          <label class="switch-row">
            <span class="switch-text">
              <span class="material-symbols-rounded">view_kanban</span>
              <span>
                <strong>Этапы задач</strong>
                <small>Канбан-режим, цветные теги этапов в карточках</small>
              </span>
            </span>
            <input type="checkbox" v-model="form.settings.uses_stages" class="switch" />
          </label>
          <label class="switch-row">
            <span class="switch-text">
              <span class="material-symbols-rounded">link</span>
              <span>
                <strong>Интеграция YouGile</strong>
                <small>Поле «Ссылка на YouGile» в форме задачи</small>
              </span>
            </span>
            <input type="checkbox" v-model="form.settings.uses_yougile" class="switch" />
          </label>
          <label class="switch-row">
            <span class="switch-text">
              <span class="material-symbols-rounded">call</span>
              <span>
                <strong>Аудио/видео-звонки</strong>
                <small>Кнопки звонка в мессенджере и профилях</small>
              </span>
            </span>
            <input type="checkbox" v-model="form.settings.uses_calls" class="switch" />
          </label>
        </div>
      </div>

      <div v-if="serverError" class="form-err">{{ serverError }}</div>
    </div>

    <template #footer>
      <button class="btn-text" :disabled="saving" @click="onClose">Отмена</button>
      <button class="btn-filled" :disabled="!canSave || saving" @click="save">
        <span v-if="saving" class="material-symbols-rounded spin">progress_activity</span>
        {{ isEdit ? 'Сохранить' : 'Создать' }}
      </button>
    </template>
  </Dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import Dialog from 'primevue/dialog'
import { getCompanyDirectory } from '@/api/companies.js'

const props = defineProps({
  modelValue: { type: Boolean, required: true },
  company: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue', 'save'])

const isEdit = computed(() => !!props.company?.id)

const form = ref(_blank())
const errors = ref({})
const serverError = ref('')
const saving = ref(false)
const directors = ref([])

function _blank() {
  return {
    name: '',
    description: '',
    director_id: null,
    settings: { uses_stages: false, uses_yougile: false, uses_calls: true },
  }
}

watch(() => props.modelValue, (v) => {
  if (!v) return
  errors.value = {}
  serverError.value = ''
  if (props.company) {
    form.value = {
      name: props.company.name || '',
      description: props.company.description || '',
      director_id: props.company.director?.id ?? props.company.director_id ?? null,
      settings: {
        uses_stages: !!props.company.settings?.uses_stages,
        uses_yougile: !!props.company.settings?.uses_yougile,
        uses_calls: props.company.settings?.uses_calls !== false,
      },
    }
  } else {
    form.value = _blank()
  }
  loadDirectors()
}, { immediate: false })

async function loadDirectors() {
  // Для редактирования — сотрудники этой компании;
  // для создания — все видимые без фильтра по компании.
  try {
    const cid = props.company?.id ?? null
    const users = await getCompanyDirectory(cid)
    directors.value = users || []
  } catch {
    directors.value = []
  }
}

const canSave = computed(() => form.value.name.trim().length >= 1)

function validate() {
  errors.value = {}
  if (!form.value.name.trim()) errors.value.name = 'Введите название'
  return Object.keys(errors.value).length === 0
}

async function save() {
  if (!validate()) return
  serverError.value = ''
  saving.value = true
  try {
    const payload = {
      name: form.value.name.trim(),
      description: form.value.description.trim() || null,
      director_id: form.value.director_id || null,
      settings: { ...form.value.settings },
    }
    emit('save', { payload, isEdit: isEdit.value, id: props.company?.id ?? null })
  } finally {
    saving.value = false
  }
}

function onClose() {
  if (saving.value) return
  emit('update:modelValue', false)
}

defineExpose({
  showError(message) { serverError.value = message; saving.value = false },
  finish() { saving.value = false },
})
</script>

<style scoped>
.form-body { display: flex; flex-direction: column; gap: 16px; padding: 4px 0 8px; }

.field { display: flex; flex-direction: column; gap: 6px; }
.lbl { font-size: 13px; font-weight: 600; color: var(--color-on-surface-variant); }
.req { color: var(--color-error); }

.ctl {
  appearance: none;
  width: 100%;
  border: 1px solid var(--color-outline-variant);
  background: var(--color-surface);
  color: var(--color-on-surface);
  padding: 10px 12px;
  border-radius: var(--radius-md, 12px);
  font: inherit;
  transition: border-color .15s, box-shadow .15s;
}
.ctl:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px color-mix(in oklab, var(--color-primary) 18%, transparent);
}
.ctl.invalid { border-color: var(--color-error); }
.ctl-area { resize: vertical; min-height: 56px; }

select.ctl {
  background: var(--color-surface) url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='10' height='6'><path d='M0 0l5 6 5-6z' fill='%23999'/></svg>") no-repeat right 12px center;
  padding-right: 32px;
}

.hint { font-size: 12px; color: var(--color-on-surface-variant); line-height: 1.4; }
.err { font-size: 12px; color: var(--color-error); }
.form-err {
  padding: 10px 12px;
  border-radius: var(--radius-md, 12px);
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  font-size: 14px;
}

.switch-list { display: flex; flex-direction: column; gap: 6px; }
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  background: var(--color-surface-container);
  border-radius: var(--radius-md, 12px);
  cursor: pointer;
  transition: background .12s;
}
.switch-row:hover { background: var(--color-surface-high); }
.switch-text {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.switch-text .material-symbols-rounded {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md, 12px);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 20px;
  flex: none;
}
.switch-text strong { display: block; font-size: 14px; color: var(--color-on-surface); }
.switch-text small { display: block; font-size: 12px; color: var(--color-on-surface-variant); }

.switch {
  appearance: none;
  width: 40px;
  height: 22px;
  border-radius: 999px;
  background: var(--color-surface-high);
  position: relative;
  cursor: pointer;
  outline: none;
  transition: background .15s;
  flex: none;
}
.switch::after {
  content: '';
  position: absolute;
  top: 3px;
  left: 3px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--color-on-surface-variant);
  transition: transform .18s, background .18s;
}
.switch:checked { background: var(--color-primary); }
.switch:checked::after {
  transform: translateX(18px);
  background: var(--color-on-primary);
}

.btn-text, .btn-filled {
  appearance: none;
  border: none;
  cursor: pointer;
  border-radius: var(--radius-full, 999px);
  padding: 8px 18px;
  font: inherit;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.btn-text { background: transparent; color: var(--color-on-surface-variant); }
.btn-text:hover { background: var(--color-surface-container); color: var(--color-on-surface); }
.btn-filled { background: var(--color-primary); color: var(--color-on-primary); }
.btn-filled:hover { filter: brightness(.94); }
.btn-filled:disabled, .btn-text:disabled { opacity: .55; cursor: not-allowed; }

.spin { animation: spin 1s linear infinite; font-size: 18px; }
@keyframes spin { to { transform: rotate(360deg); } }

:deep(.p-dialog-footer) { display: flex; justify-content: flex-end; gap: 8px; padding-top: 14px; }
</style>
