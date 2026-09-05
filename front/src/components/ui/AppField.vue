<template>
  <div class="field" :class="{ inline }">
    <label class="field-label" :for="controlId">
      {{ label }}<span v-if="required" class="field-req" aria-hidden="true">*</span>
    </label>
    <slot :id="controlId" />
    <span v-if="error" class="field-error">{{ error }}</span>
    <span v-else-if="hint" class="field-hint">{{ hint }}</span>
  </div>
</template>

<script setup>
/* Поле формы: метка сверху, управление под ней, подпись или ошибка снизу.

   AppRow ставит управление СПРАВА от текста и годится строке настройки; в
   формах поле занимает всю ширину, и метка живёт над ним. Раньше каждый раздел
   описывал эту пару у себя — вместе с геометрией поля, потому что глобальный
   `.ctl` задаёт только фон и рамку, а отступы оставляет компоненту. Отсюда и
   правило: управление внутрь кладём готовое (InputText, Select, Textarea,
   DatePicker), а не голый input с классом.

   Слот получает готовый id — его можно повесить на поле, чтобы клик по метке
   попадал в него. */
import { computed, useId } from 'vue'

const props = defineProps({
  label: { type: String, default: '' },
  /** Подпись под полем: поясняет, а не ругается. */
  hint: { type: String, default: '' },
  /** Текст ошибки — вытесняет подпись. */
  error: { type: String, default: '' },
  required: { type: Boolean, default: false },
  /** Метка и поле в одну строку (узкие поля тулбаров). */
  inline: { type: Boolean, default: false },
  /** Свой id управления; по умолчанию — сгенерированный. */
  id: { type: String, default: '' },
})

const generated = useId()
const controlId = computed(() => props.id || generated)
</script>

<style scoped>
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.field.inline {
  flex-direction: row;
  align-items: center;
  gap: 10px;
}

.field-label {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--color-text);
}
.field-req { margin-left: 2px; color: var(--color-error); }

.field-hint, .field-error {
  font-size: 0.8rem;
  overflow-wrap: anywhere;
}
.field-hint { color: var(--color-text-dim); }
.field-error { color: var(--color-error); }

/* Поле занимает всю ширину строки: у PrimeVue ширина по содержимому, и без
   этого инпуты в форме получались разной длины. */
.field :deep(.p-inputtext),
.field :deep(.p-select),
.field :deep(.p-textarea),
.field :deep(.p-inputnumber),
.field :deep(.p-datepicker) {
  width: 100%;
}
.field.inline :deep(.p-inputtext),
.field.inline :deep(.p-select),
.field.inline :deep(.p-inputnumber) {
  width: auto;
}
</style>
