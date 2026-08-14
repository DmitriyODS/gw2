<template>
  <AppDialog
    :model-value="modelValue"
    :title="title"
    size="lg"
    :busy="saving"
    @update:model-value="onClose"
  >
    <div v-if="registry?.fields?.length" class="rec-form">
      <div class="rec-grid">
        <div
          v-for="f in registry.fields"
          :key="f.id"
          class="rec-cell"
          :style="cellStyle(f)"
        >
          <div class="rec-label">{{ f.label }}</div>
          <FieldInput
            v-if="editing"
            :field="f"
            :model-value="form[String(f.id)] ?? null"
            :upload="upload"
            @update:model-value="form[String(f.id)] = $event"
          />
          <FieldValue v-else :field="f" :value="record?.data?.[String(f.id)] ?? null" />
        </div>
      </div>
    </div>
    <p v-else class="rec-empty">В этом реестре пока нет полей.</p>

    <template #footer>
      <template v-if="editing">
        <AppButton variant="text" label="Отмена" :disabled="saving" @click="cancelEdit" />
        <AppButton
          variant="filled"
          :label="isNew ? 'Создать' : 'Сохранить'"
          :loading="saving"
          @click="submit"
        />
      </template>
      <template v-else>
        <AppButton variant="text" label="Закрыть" @click="onClose(false)" />
        <!-- Учётный реестр: выдача позиции — такое же действие над записью, как
             её правка, и добираться до него через контекстное меню списка,
             открыв карточку, было неоткуда. -->
        <AppButton
          v-if="canManageStock"
          variant="glass"
          icon="inventory"
          label="Управлять"
          @click="$emit('manage', record)"
        />
        <AppButton
          v-if="!readonly"
          variant="filled"
          icon="edit"
          label="Редактировать"
          @click="editing = true"
        />
      </template>
    </template>
  </AppDialog>
</template>

<script setup>
/* Карточка записи: просмотр и правка. Сохранение и загрузку файлов делает
   вызывающий (`save`/`upload`) — раздел ходит своими ручками, публичная
   страница по ссылке своими, а сама карточка про это ничего не знает и потому
   годится обеим. */
import { computed, reactive, ref, watch } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import FieldInput from '@/components/common/FieldInput.vue'
import FieldValue from '@/components/common/FieldValue.vue'
import { useNotificationsStore } from '@/stores/notifications.js'
import { GRID_COLS } from '@/utils/registryFields.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  registry: { type: Object, default: null },
  record: { type: Object, default: null }, // null → создание новой записи
  readonly: { type: Boolean, default: false }, // просмотр по ссылке уровня view
  /* Открыть сразу в правке: из контекстного меню записи выбирают «изменить», и
     лишний клик по «Редактировать» в карточке там ни к чему. */
  startEditing: { type: Boolean, default: false },
  /** (data, record|null) => Promise — сохранение; record === null → создание. */
  save: { type: Function, default: null },
  /* (file) => Promise<{path,name,mime,size,thumb?}> — загрузка файла поля.
     Умолчания у неё нет НАМЕРЕННО: раздел и публичная страница грузят разными
     ручками, а раздел вдобавок обязан назвать реестр — иначе серверу некуда
     положить файл и нечью квоту тратить. */
  upload: { type: Function, required: true },
})
const emit = defineEmits(['update:modelValue', 'saved', 'manage'])

/* Выдача есть только у учётного реестра и только у сохранённой записи: новой
   записи ещё нечего выдавать, а смотрящему — незачем. */
const canManageStock = computed(() =>
  !!props.registry?.accounting && !props.readonly && !!props.record?.id)

const notif = useNotificationsStore()

const editing = ref(false)
const saving = ref(false)
const form = reactive({})
const isNew = ref(false)

watch(() => props.modelValue, (open) => {
  if (!open) return
  isNew.value = !props.record && !props.readonly
  // В readonly — всегда просмотр, что бы ни просили снаружи.
  editing.value = !props.readonly && (isNew.value || props.startEditing)
  for (const k of Object.keys(form)) delete form[k]
  const data = props.record?.data || {}
  for (const [k, v] of Object.entries(data)) form[k] = v
})

const title = ref('')
watch([() => props.modelValue, () => props.record, editing], () => {
  title.value = isNew.value ? 'Новая запись' : (editing.value ? 'Редактирование записи' : 'Запись')
})

// Карточка делится на четверти — ровно так, как её собрали в конструкторе
// структуры (domain.GridCols); потолок берём оттуда, а не числом на месте.
function cellStyle(f) {
  return {
    gridColumn: `span ${Math.min(GRID_COLS, Math.max(1, f.col_span || 1))}`,
    gridRow: `span ${Math.max(1, f.row_span || 1)}`,
  }
}

function onClose(v) {
  if (v) return
  emit('update:modelValue', false)
}
function cancelEdit() {
  if (isNew.value) { emit('update:modelValue', false); return }
  editing.value = false
}

async function submit() {
  if (!props.save) return
  saving.value = true
  try {
    await props.save({ ...form }, isNew.value ? null : props.record)
    notif.success(isNew.value ? 'Запись добавлена' : 'Запись сохранена')
    emit('saved')
    emit('update:modelValue', false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить запись')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
/* Раскладка считается от ширины САМОГО диалога, а не экрана: он живёт окном
   рабочего стола, и медиазапрос про размер экрана здесь ничего не значит. */
.rec-form { container-type: inline-size; }

.rec-grid {
  display: grid;
  /* Четыре четверти — та же сетка, что и в конструкторе структуры.
     minmax(0, 1fr): «1fr» не уже своего содержимого, и длинное значение в поле
     раздувало бы колонку, выталкивая диалог за край. */
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}
.rec-cell { min-width: 0; display: flex; flex-direction: column; gap: 6px; }
.rec-label { font-size: 12px; font-weight: 600; color: var(--color-text-dim); text-transform: uppercase; letter-spacing: 0.02em; }
.rec-empty { color: var(--color-text-dim); text-align: center; padding: 24px 0; }

/* Пороги — по ширине ТЕЛА диалога, а не окна. У размера lg тело выходит около
   670px, поэтому порог в 680px срабатывал всегда: сетка молча сводилась к двум
   колонкам, и раскладка, собранная в настройках реестра, не применялась
   никогда. Ниже 560px четверти уже нечитаемы — вот там и сворачиваем. */
@container (max-width: 560px) {
  .rec-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .rec-cell { grid-column: span 2 !important; }
}

@container (max-width: 380px) {
  .rec-grid { grid-template-columns: 1fr; }
  .rec-cell { grid-column: span 1 !important; }
}

/* Дубль на медиазапросах — заводской WebView старых Android не знает
   @container (см. build.target). */
@media (max-width: 680px) {
  .rec-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .rec-cell { grid-column: span 2 !important; }
}

@media (max-width: 420px) {
  .rec-grid { grid-template-columns: 1fr; }
  .rec-cell { grid-column: span 1 !important; }
}

</style>
