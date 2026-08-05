<template>
  <!-- Товар автора: что продаём, за сколько и что именно получит покупатель
       (payload — рецепт темы из конструктора). -->
  <AppDialog
    :model-value="visible"
    :title="product ? 'Изменить товар' : 'Выставить товар'"
    :actions="ACTIONS"
    @update:model-value="$emit('update:visible', $event)"
    @confirm="save"
  >
    <div class="form">
      <label class="field">
        <span class="field-label">Что продаём</span>
        <Dropdown v-model="form.kind" :options="KINDS" option-label="label" option-value="value" />
      </label>

      <label class="field">
        <span class="field-label">Название</span>
        <InputText v-model="form.title" maxlength="120" placeholder="Например: «Тёплый графит»" />
      </label>

      <label class="field">
        <span class="field-label">Описание</span>
        <Textarea v-model="form.description" rows="3" auto-resize />
      </label>

      <label class="field">
        <span class="field-label">Цена, руб.</span>
        <InputNumber v-model="form.priceRub" :min="0" :max="100000" />
      </label>

      <label v-if="form.kind === 'theme'" class="field">
        <span class="field-label">Тема из конструктора</span>
        <Dropdown
          v-model="form.themeName"
          :options="themeOptions"
          option-label="label"
          option-value="value"
          placeholder="Выберите свою тему"
        />
        <span class="field-label">
          Покупатель получит эту палитру целиком. Свои темы создаются в
          «Настройки → Темы и оформление».
        </span>
      </label>

      <p v-if="error" class="form-error">{{ error }}</p>
      <p class="field-label">
        После сохранения отправьте товар на проверку — на витрину он попадёт
        после одобрения администратором платформы.
      </p>
    </div>
  </AppDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Textarea from 'primevue/textarea'
import Dropdown from 'primevue/dropdown'
import AppDialog from '@/components/ui/AppDialog.vue'
import * as api from '@/api/billing.js'
import { useThemeStore } from '@/stores/theme.js'

const props = defineProps({
  visible: { type: Boolean, default: false },
  product: { type: Object, default: null },
})

const emit = defineEmits(['update:visible', 'saved'])

const theme = useThemeStore()

const KINDS = [
  { value: 'theme', label: 'Тема оформления' },
  { value: 'wallpaper', label: 'Обои рабочего стола' },
  { value: 'gradient', label: 'Градиент' },
  { value: 'pet_skin', label: 'Скин питомца' },
  { value: 'pet_decor', label: 'Декор домика' },
  { value: 'other', label: 'Другое' },
]

const form = ref(emptyForm())
const error = ref('')

const themeOptions = computed(() =>
  theme.customThemes.map((t) => ({ value: t.name, label: t.name })))

const ACTIONS = [
  { kind: 'cancel', label: 'Отмена' },
  { kind: 'confirm', label: 'Сохранить', icon: 'check' },
]

watch(() => props.visible, (open) => {
  if (!open) return
  error.value = ''
  form.value = props.product
    ? {
      kind: props.product.kind,
      title: props.product.title,
      description: props.product.description,
      priceRub: Math.round((props.product.price || 0) / 100),
      themeName: props.product.payload?.theme_name || '',
    }
    : emptyForm()
})

function emptyForm() {
  return { kind: 'theme', title: '', description: '', priceRub: 199, themeName: '' }
}

function payload() {
  if (form.value.kind !== 'theme' || !form.value.themeName) return {}
  const found = theme.customThemes.find((t) => t.name === form.value.themeName)
  return found ? { theme_name: found.name, vars: found.vars } : {}
}

async function save() {
  if (!form.value.title.trim()) {
    error.value = 'Дайте товару название'
    return
  }
  const body = {
    kind: form.value.kind,
    title: form.value.title.trim(),
    description: form.value.description,
    price: Math.round((form.value.priceRub || 0) * 100),
    payload: payload(),
  }
  try {
    if (props.product) await api.updateMyProduct(props.product.id, body)
    else await api.createMyProduct(body)
    emit('update:visible', false)
    emit('saved')
  } catch (e) {
    error.value = e?.data?.message || 'Не удалось сохранить товар'
  }
}
</script>

<style scoped>
.form { display: flex; flex-direction: column; gap: 12px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.field-label { font-size: 0.85rem; color: var(--color-text-dim); }
.form-error { margin: 0; font-size: 0.85rem; color: var(--color-error); }
</style>
