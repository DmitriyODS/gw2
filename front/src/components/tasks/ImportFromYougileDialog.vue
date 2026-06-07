<template>
  <Dialog
    :visible="visible"
    @update:visible="(v) => v || $emit('close')"
    modal
    :closable="!busy"
    header="Импорт из YouGile"
    :style="{ width: '520px', maxWidth: 'calc(100vw - 24px)' }"
  >
    <div class="imp-body">
      <div class="field">
        <label class="lbl">Ссылка на карточку YouGile <span class="required">*</span></label>
        <InputText
          v-model="form.url"
          placeholder="https://ru.yougile.com/team/.../#tasks?task=…"
          class="w-full"
        />
        <div class="hint">
          Вставьте ссылку прямо из адресной строки открытой карточки в YouGile.
        </div>
      </div>

      <div class="field">
        <label class="lbl">Заказчик (отдел) <span class="required">*</span></label>
        <Select
          v-model="form.department_id"
          :options="departments"
          option-label="name"
          option-value="id"
          placeholder="Выберите отдел"
          class="w-full"
          filter
          filterPlaceholder="Поиск отдела..."
        />
      </div>

      <div class="field">
        <label class="lbl">Ответственный</label>
        <Select
          v-model="form.responsible_user_id"
          :options="responsibles"
          option-label="fio"
          option-value="id"
          placeholder="По умолчанию — вы"
          class="w-full"
          :loading="responsiblesLoading"
          filter
          filterPlaceholder="Поиск сотрудника..."
          show-clear
        />
      </div>

      <label class="check-row">
        <input type="checkbox" v-model="form.pull_deadline" />
        <span>Подтянуть дедлайн из YouGile, если задан</span>
      </label>
    </div>

    <template #footer>
      <button class="btn-text" :disabled="busy" @click="$emit('close')">Отмена</button>
      <button class="btn-filled" :disabled="busy || !canSubmit" @click="onImport">
        <span class="material-symbols-outlined">download</span>
        {{ busy ? 'Импорт…' : 'Импортировать' }}
      </button>
    </template>
  </Dialog>
</template>

<script setup>
import { reactive, ref, computed, onMounted } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAuthStore } from '@/stores/auth.js'
import { importYougileTask } from '@/api/yougile.js'
import { getDepartments } from '@/api/departments.js'
import { getDirectory } from '@/api/users.js'

defineProps({ visible: { type: Boolean, default: false } })
const emit = defineEmits(['close', 'imported'])

const notif = useNotificationsStore()
const authStore = useAuthStore()

const form = reactive({
  url: '',
  department_id: null,
  responsible_user_id: null,
  pull_deadline: true,
})

const departments = ref([])
const responsibles = ref([])
const responsiblesLoading = ref(false)
const busy = ref(false)

const canSubmit = computed(() => !!form.url.trim() && !!form.department_id)

async function loadDepartments() {
  try {
    const data = await getDepartments()
    departments.value = Array.isArray(data) ? data : (data.departments ?? data.items ?? [])
  } catch {
    departments.value = []
  }
}

async function loadResponsibles() {
  responsiblesLoading.value = true
  try {
    const list = await getDirectory()
    responsibles.value = Array.isArray(list) ? list : (list.items ?? [])
  } catch {
    responsibles.value = []
  } finally {
    responsiblesLoading.value = false
  }
}

async function onImport() {
  busy.value = true
  try {
    const task = await importYougileTask({
      url: form.url.trim(),
      department_id: form.department_id,
      responsible_user_id: form.responsible_user_id || null,
      pull_deadline: form.pull_deadline,
    })
    notif.success('Карточка импортирована из YouGile')
    emit('imported', task)
  } catch (e) {
    notif.error(e?.data?.message || e?.message || 'Не удалось импортировать')
  } finally {
    busy.value = false
  }
}

onMounted(() => {
  loadDepartments()
  loadResponsibles()
})
</script>

<style scoped>
.imp-body { display: flex; flex-direction: column; gap: 14px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.lbl { font-size: 13px; font-weight: 600; color: var(--color-on-surface-variant); }
.required { color: var(--color-error); }
.hint { font-size: 12px; color: var(--color-text-dim); }
.check-row { display: inline-flex; align-items: center; gap: 8px; font-size: 14px; cursor: pointer; }
.check-row input { width: 18px; height: 18px; accent-color: var(--color-primary); }

.btn-filled, .btn-text {
  display: inline-flex; align-items: center; gap: 8px;
  height: 40px; padding: 0 18px; border-radius: 20px;
  font: inherit; font-weight: 600; cursor: pointer; border: 1px solid transparent;
}
.btn-filled { background: var(--color-primary); color: var(--color-on-primary); }
.btn-filled:hover:not(:disabled) { background: color-mix(in oklch, var(--color-primary) 90%, black); }
.btn-text { background: transparent; color: var(--color-text); }
.btn-text:hover:not(:disabled) { background: var(--color-surface-high); }
button:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
