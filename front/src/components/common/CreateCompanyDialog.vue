<template>
  <AppDialog
    :model-value="modelValue"
    title="Создать компанию"
    subtitle="Вы станете её администратором. После создания мы переключим вас на новую компанию."
    icon="add_business"
    tone="primary"
    size="sm"
    :busy="saving"
    :closable="!saving"
    @update:model-value="onVisible"
  >
    <form class="cc-form" @submit.prevent="submit">
      <div class="cc-field">
        <label>Название</label>
        <input
          ref="nameEl"
          v-model.trim="name"
          type="text"
          class="cc-input"
          maxlength="120"
          placeholder="Например: Groove Studio"
          :disabled="saving"
        />
      </div>
      <div class="cc-field">
        <label>Описание <span class="cc-optional">— необязательно</span></label>
        <textarea
          v-model.trim="description"
          class="cc-textarea"
          rows="3"
          maxlength="500"
          placeholder="Чем занимается компания"
          :disabled="saving"
        ></textarea>
      </div>
      <p v-if="error" class="cc-error">{{ error }}</p>
    </form>

    <template #footer>
      <div class="cc-footer">
        <button class="cc-cancel" type="button" :disabled="saving" @click="onVisible(false)">
          Отмена
        </button>
        <button class="cc-submit" type="button" :disabled="!name || saving" @click="submit">
          <span v-if="saving" class="cc-spinner" aria-hidden="true" />
          <span v-else class="material-symbols-outlined">add_business</span>
          Создать
        </button>
      </div>
    </template>
  </AppDialog>
</template>

<script setup>
import { nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import AppDialog from '@/components/common/AppDialog.vue'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const router = useRouter()
const auth = useAuthStore()
const companies = useCompaniesStore()
const notify = useNotificationsStore()

const name = ref('')
const description = ref('')
const error = ref('')
const saving = ref(false)
const nameEl = ref(null)

watch(() => props.modelValue, async (open) => {
  if (!open) return
  name.value = ''
  description.value = ''
  error.value = ''
  await nextTick()
  nameEl.value?.focus()
})

function onVisible(v) {
  if (saving.value) return
  emit('update:modelValue', v)
}

async function submit() {
  if (!name.value || saving.value) return
  saving.value = true
  error.value = ''
  try {
    const payload = { name: name.value }
    if (description.value) payload.description = description.value
    const company = await companies.create(payload)
    // Переключаемся на новую компанию (перевыпуск токена с активной компанией) —
    // создатель сразу начинает в ней работать как администратор.
    await auth.switchCompany(company.id)
    emit('update:modelValue', false)
    notify.success(`Компания «${company.name}» создана`)
    // На страницу управления компанией — добавить участников и настроить.
    router.push(`/companies/${company.id}`)
  } catch (e) {
    error.value = e?.message || 'Не удалось создать компанию'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.cc-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.cc-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cc-field label {
  font-size: 12px;
  font-weight: 700;
  color: var(--color-primary);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.cc-optional {
  font-weight: 600;
  color: var(--color-text-dim);
  text-transform: none;
  letter-spacing: normal;
}

.cc-input,
.cc-textarea {
  width: 100%;
  box-sizing: border-box;
  border: 1.5px solid var(--color-outline);
  background: transparent;
  color: var(--color-text);
  font-family: inherit;
  font-size: 15px;
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.cc-input {
  height: 48px;
  border-radius: var(--radius-full);
  padding: 0 18px;
}

.cc-textarea {
  border-radius: var(--radius-lg, 16px);
  padding: 12px 16px;
  resize: vertical;
}

.cc-input:focus,
.cc-textarea:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 15%, transparent);
}

.cc-input:disabled,
.cc-textarea:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.cc-error {
  margin: 0;
  font-size: 13px;
  color: var(--color-on-error-container);
  padding: 8px 14px;
  background: var(--color-error-container);
  border-radius: var(--radius-md, 12px);
}

.cc-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  width: 100%;
}

.cc-cancel {
  border: none;
  background: none;
  color: var(--color-primary);
  font-size: 14px;
  font-weight: 600;
  padding: 10px 16px;
  border-radius: var(--radius-full);
  cursor: pointer;
}

.cc-cancel:hover:not(:disabled) { background: var(--color-surface-high); }
.cc-cancel:disabled { opacity: 0.5; cursor: not-allowed; }

.cc-submit {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 14px;
  font-weight: 600;
  padding: 10px 20px;
  cursor: pointer;
}

.cc-submit:disabled { opacity: 0.45; cursor: not-allowed; }
.cc-submit .material-symbols-outlined { font-size: 18px; }

.cc-spinner {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 2px solid color-mix(in oklch, var(--color-on-primary) 40%, transparent);
  border-top-color: var(--color-on-primary);
  animation: cc-spin 0.7s linear infinite;
}

@keyframes cc-spin {
  to { transform: rotate(360deg); }
}
</style>
