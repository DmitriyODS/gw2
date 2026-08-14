<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      ref="panel"
      class="cf"
      :style="style"
      @click.stop
    >
      <span class="cf-title">{{ field?.label }}</span>

      <Select
        v-model="op"
        :options="ops"
        option-label="label"
        option-value="value"
        class="cf-op"
      />

      <!-- Список выбора: отмечаем варианты, подходит любой из отмеченных. -->
      <MultiSelect
        v-if="op === 'any'"
        v-model="values"
        :options="options"
        display="chip"
        placeholder="Варианты"
        class="cf-value"
      />
      <!-- Флажок: сравнение с одним из двух состояний. -->
      <Select
        v-else-if="field?.type === 'checkbox' && needsValue"
        v-model="single"
        :options="checkboxOptions"
        option-label="label"
        option-value="value"
        class="cf-value"
      />
      <template v-else-if="op === 'between'">
        <InputText v-model="from" class="cf-value" placeholder="От" />
        <InputText v-model="to" class="cf-value" placeholder="До" />
      </template>
      <InputText
        v-else-if="needsValue"
        v-model="single"
        class="cf-value"
        placeholder="Значение"
        @keydown.enter="apply"
      />

      <div class="cf-actions">
        <AppButton size="sm" variant="text" label="Сбросить" @click="reset" />
        <AppButton size="sm" variant="filled" label="Применить" @click="apply" />
      </div>
    </div>
  </Teleport>
</template>

<script setup>
/* Фильтр одной колонки таблицы. Всплывает у воронки в заголовке, поэтому
   телепортируется в body и позиционируется экранными координатами — как и
   остальные попапы у курсора (контекстные меню, выбор эмодзи).

   Набор сравнений зависит от типа поля (utils/registryFields.js:filterOps), а
   их коды — зеркало разбора на сервере. */
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import InputText from 'primevue/inputtext'
import MultiSelect from 'primevue/multiselect'
import Select from 'primevue/select'
import AppButton from '@/components/ui/AppButton.vue'
import { checkboxText, filterOps, opNeedsValue } from '@/utils/registryFields.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  field: { type: Object, default: null },
  /** Уже наложенное условие по этой колонке (null — фильтра нет). */
  filter: { type: Object, default: null },
  /** Элемент-воронка, у которого разворачиваем панель. */
  anchor: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue', 'apply'])

const panel = ref(null)
const op = ref('contains')
const single = ref('')
const from = ref('')
const to = ref('')
const values = ref([])
const style = ref({})

const ops = computed(() => filterOps(props.field?.type))
const needsValue = computed(() => opNeedsValue(op.value))
const options = computed(() => props.field?.config?.options || [])

const checkboxOptions = computed(() => [
  { value: 'true', label: checkboxText(props.field, true) },
  { value: 'false', label: checkboxText(props.field, false) },
])

watch(() => props.modelValue, async (open) => {
  if (!open) {
    document.removeEventListener('pointerdown', onOutside, true)
    return
  }
  const f = props.filter
  op.value = f?.op || ops.value[0]?.value || 'contains'
  single.value = f?.values?.[0] || ''
  from.value = f?.values?.[0] || ''
  to.value = f?.values?.[1] || ''
  values.value = op.value === 'any' ? [...(f?.values || [])] : []

  await nextTick()
  place()
  document.addEventListener('pointerdown', onOutside, true)
})

// Панель разворачивается у воронки и не вылезает за край окна: у правых колонок
// она иначе оказывалась бы за границей экрана.
function place() {
  const rect = props.anchor?.getBoundingClientRect?.()
  const width = panel.value?.offsetWidth || 260
  if (!rect) {
    style.value = { top: '80px', left: '50%', transform: 'translateX(-50%)' }
    return
  }
  const left = Math.min(Math.max(8, rect.left), window.innerWidth - width - 8)
  style.value = { top: `${rect.bottom + 6}px`, left: `${left}px` }
}

function onOutside(e) {
  if (panel.value && !panel.value.contains(e.target)) close()
}

function close() {
  emit('update:modelValue', false)
}

function apply() {
  if (!needsValue.value) {
    emit('apply', { op: op.value, values: [] })
    return
  }
  if (op.value === 'any') {
    emit('apply', values.value.length ? { op: op.value, values: [...values.value] } : null)
    return
  }
  if (op.value === 'between') {
    const pair = [from.value.trim(), to.value.trim()]
    emit('apply', pair.every(Boolean) ? { op: op.value, values: pair } : null)
    return
  }
  const v = String(single.value ?? '').trim()
  emit('apply', v ? { op: op.value, values: [v] } : null)
}

function reset() {
  emit('apply', null)
}

onBeforeUnmount(() => document.removeEventListener('pointerdown', onOutside, true))
</script>

<style scoped>
.cf {
  position: fixed;
  z-index: 1200;
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 260px;
  padding: 12px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg), var(--acrylic-card-bg);
  -webkit-backdrop-filter: blur(14px);
  backdrop-filter: blur(14px);
  box-shadow: var(--shadow-lg);
}

.cf-title { font-size: 13px; font-weight: 600; overflow-wrap: anywhere; }
.cf-op, .cf-value { width: 100%; }
.cf-actions { display: flex; justify-content: flex-end; gap: 6px; }
</style>
