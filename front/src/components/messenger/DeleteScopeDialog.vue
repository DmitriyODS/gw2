<template>
  <Dialog
    :visible="modelValue"
    @update:visible="$emit('update:modelValue', $event)"
    modal
    :draggable="false"
    :show-header="false"
    :style="{ width: '380px', maxWidth: '92vw' }"
    :pt="{ root: { class: 'delete-scope-dialog' } }"
  >
    <div class="ds">
      <div class="ds-icon">
        <span class="material-symbols-outlined">delete</span>
      </div>
      <h3 class="ds-title">{{ title }}</h3>
      <p class="ds-text">{{ text }}</p>

      <label v-if="canForAll" class="ds-check" :class="{ active: forAll }">
        <input type="checkbox" v-model="forAll" />
        <span class="ds-check-box">
          <span class="material-symbols-outlined">{{ forAll ? 'check_box' : 'check_box_outline_blank' }}</span>
        </span>
        <span class="ds-check-label">Удалить также у {{ otherName || 'собеседника' }}</span>
      </label>

      <div class="ds-actions">
        <button class="btn-text" @click="cancel">Отмена</button>
        <button class="btn-filled-error" @click="confirm">
          <span class="material-symbols-outlined">delete</span>
          Удалить
        </button>
      </div>
    </div>
  </Dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import Dialog from 'primevue/dialog'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  title: { type: String, default: 'Удалить?' },
  text: { type: String, default: '' },
  canForAll: { type: Boolean, default: true },
  otherName: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'confirm'])

const forAll = ref(false)

watch(() => props.modelValue, (v) => {
  if (v) forAll.value = false
})

function cancel() {
  emit('update:modelValue', false)
}

function confirm() {
  emit('confirm', { scope: forAll.value ? 'all' : 'me' })
  emit('update:modelValue', false)
}
</script>

<style scoped>
.ds {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 24px 24px 20px;
}

.ds-icon {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12px;
}

.ds-icon .material-symbols-outlined {
  font-size: 28px;
  font-variation-settings: 'FILL' 1, 'wght' 500, 'GRAD' 0, 'opsz' 32;
}

.ds-title {
  margin: 0 0 6px;
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text);
}

.ds-text {
  margin: 0 0 16px;
  font-size: 14px;
  line-height: 1.45;
  color: var(--color-text-dim);
}

.ds-check {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  background: var(--color-surface-low);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  cursor: pointer;
  user-select: none;
  width: 100%;
  margin-bottom: 18px;
  transition: background 0.15s, border-color 0.15s;
}

.ds-check.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.ds-check input { display: none; }

.ds-check-box {
  display: inline-flex;
  align-items: center;
  color: var(--color-primary);
}

.ds-check-box .material-symbols-outlined {
  font-size: 22px;
}

.ds-check-label {
  font-size: 14px;
  font-weight: 500;
  text-align: left;
}

.ds-actions {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
  width: 100%;
}

.btn-text {
  background: none;
  border: none;
  color: var(--color-primary);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  padding: 10px 16px;
  border-radius: var(--radius-full);
  cursor: pointer;
}

.btn-text:hover { background: var(--color-surface-low); }

/* M3 filled error button */
.btn-filled-error {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: none;
  background: var(--color-error);
  color: var(--color-on-error);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  padding: 10px 18px 10px 14px;
  border-radius: var(--radius-full);
  cursor: pointer;
  position: relative;
  isolation: isolate;
  overflow: hidden;
  transition: box-shadow 0.18s ease;
}

.btn-filled-error::before {
  content: '';
  position: absolute;
  inset: 0;
  background: currentColor;
  opacity: 0;
  transition: opacity 0.18s ease;
  z-index: -1;
}

.btn-filled-error:hover::before { opacity: 0.1; }
.btn-filled-error:active::before { opacity: 0.18; }
.btn-filled-error:hover { box-shadow: var(--shadow-sm); }

.btn-filled-error .material-symbols-outlined {
  font-size: 18px;
}
</style>
