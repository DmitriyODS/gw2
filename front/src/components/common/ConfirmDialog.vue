<template>
  <Dialog
    :visible="visible"
    @update:visible="$emit('cancel')"
    :header="header"
    modal
    style="width:420px"
  >
    <p class="confirm-message">{{ message }}</p>
    <template #footer>
      <button class="btn-secondary" @click="$emit('cancel')">Отмена</button>
      <button
        class="btn-primary"
        :class="{ danger: dangerConfirm }"
        @click="$emit('confirm')"
      >
        {{ confirmLabel || 'Подтвердить' }}
      </button>
    </template>
  </Dialog>
</template>

<script setup>
import Dialog from 'primevue/dialog'

defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  header: {
    type: String,
    default: 'Подтверждение'
  },
  message: {
    type: String,
    default: ''
  },
  confirmLabel: {
    type: String,
    default: 'Подтвердить'
  },
  dangerConfirm: {
    type: Boolean,
    default: false
  }
})

defineEmits(['confirm', 'cancel'])
</script>

<style scoped>
.confirm-message {
  margin: 0 0 8px;
  color: var(--gw-text);
  line-height: 1.5;
  font-size: 15px;
}

:deep(.p-dialog-footer) {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 16px;
}

.btn-secondary {
  background: transparent;
  color: var(--gw-text-secondary);
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  padding: 8px 18px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.btn-secondary:hover {
  background: var(--gw-bg);
  color: var(--gw-text);
}

.btn-primary {
  background: var(--gw-primary);
  color: var(--color-on-primary);
  border: none;
  border-radius: 8px;
  padding: 8px 18px;
  font-size: 14px;
  cursor: pointer;
  font-weight: 600;
  transition: background 0.15s;
}

.btn-primary:hover {
  background: var(--gw-primary-hover);
}

.btn-primary.danger {
  background: var(--color-error);
  color: var(--color-on-error);
}

.btn-primary.danger:hover {
  background: var(--color-error-hover);
}
</style>
