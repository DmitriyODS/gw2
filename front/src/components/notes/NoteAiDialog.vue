<template>
  <AppDialog
    :model-value="modelValue"
    :title="label || 'ИИ-обработка'"
    icon="auto_awesome"
    size="md"
    :closable="!loading"
    :actions="[]"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="loading" class="nad-loading">
      <span class="material-symbols-outlined nad-spin">progress_activity</span>
      <span>ИИ обрабатывает текст…</span>
    </div>

    <template v-else-if="error">
      <p class="nad-error">{{ error }}</p>
      <div class="nad-actions">
        <button class="nad-btn" @click="$emit('update:modelValue', false)">Закрыть</button>
        <button class="nad-btn primary" @click="$emit('retry')">
          <span class="material-symbols-outlined">refresh</span> Повторить
        </button>
      </div>
    </template>

    <template v-else>
      <div class="nad-result">{{ result }}</div>
      <div class="nad-actions">
        <button class="nad-btn" @click="$emit('update:modelValue', false)">Отмена</button>
        <button class="nad-btn" title="Скопировать результат" @click="copyResult">
          <span class="material-symbols-outlined">content_copy</span> Копировать
        </button>
        <button class="nad-btn" @click="$emit('apply', 'below')">
          <span class="material-symbols-outlined">add_row_below</span>
          {{ isContinue ? 'Вставить' : 'Вставить ниже' }}
        </button>
        <button v-if="!isContinue" class="nad-btn primary" @click="$emit('apply', 'replace')">
          <span class="material-symbols-outlined">swap_horiz</span> Заменить
        </button>
      </div>
    </template>
  </AppDialog>
</template>

<script setup>
// Результат ИИ-операции над выделенным текстом заметки: превью + выбор, как
// применить (заменить выделенное / вставить ниже / скопировать). Ничего не
// меняет в документе само — решает пользователь.
import AppDialog from '@/components/common/AppDialog.vue'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  label: { type: String, default: '' },
  loading: { type: Boolean, default: false },
  error: { type: String, default: '' },
  result: { type: String, default: '' },
  // «Продолжить текст» заменять нечем — только вставка после выделения.
  isContinue: { type: Boolean, default: false },
})

defineEmits(['update:modelValue', 'apply', 'retry'])

const notif = useNotificationsStore()

async function copyResult() {
  try {
    await navigator.clipboard.writeText(props.result)
    notif.success('Скопировано в буфер обмена')
  } catch {
    notif.error('Не удалось скопировать')
  }
}
</script>

<style scoped>
.nad-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 28px 0;
  color: var(--color-text-dim);
  font-size: 14px;
}
.nad-spin { animation: nadspin 1s linear infinite; color: var(--color-primary); }
@keyframes nadspin { to { transform: rotate(360deg); } }

.nad-error { margin: 4px 0 12px; color: var(--color-error); font-size: 14px; }

.nad-result {
  max-height: 44vh;
  overflow-y: auto;
  padding: 12px 14px;
  border-radius: var(--radius-md);
  background: var(--color-surface-high);
  color: var(--color-text);
  font-size: 14.5px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.nad-actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
}
.nad-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 38px;
  padding: 0 14px;
  border: 1px solid var(--color-outline-variant);
  border-radius: var(--radius-full);
  background: none;
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}
.nad-btn:hover { background: var(--color-surface-low); }
.nad-btn .material-symbols-outlined { font-size: 18px; }
.nad-btn.primary {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: var(--color-on-primary);
}
.nad-btn.primary:hover { filter: brightness(1.06); }
</style>
