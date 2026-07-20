<template>
  <div class="fv-value">
    <!-- Картинка -->
    <template v-if="field.type === 'image'">
      <button v-if="src" class="fv-image" @click="lightbox = true">
        <img :src="src" :alt="value?.name || ''" />
      </button>
      <span v-else class="fv-empty">—</span>
      <ImageLightbox v-if="src" v-model="lightbox" :src="src" :caption="value?.name || ''" />
    </template>

    <!-- Файл -->
    <template v-else-if="field.type === 'file'">
      <a v-if="src" class="fv-file" :href="src" :download="value?.name || ''" target="_blank" rel="noopener">
        <span class="material-symbols-outlined">description</span>
        <span class="fv-file-name">{{ value?.name || 'Файл' }}</span>
      </a>
      <span v-else class="fv-empty">—</span>
    </template>

    <!-- Галочка -->
    <template v-else-if="field.type === 'checkbox'">
      <span class="fv-check" :class="{ on: !!value }">
        <span class="material-symbols-outlined">{{ value ? 'check_box' : 'check_box_outline_blank' }}</span>
        {{ value ? 'Да' : 'Нет' }}
      </span>
    </template>

    <!-- Список -->
    <template v-else-if="field.type === 'select'">
      <div v-if="selectChips.length" class="fv-chips">
        <span v-for="c in selectChips" :key="c" class="fv-chip">{{ c }}</span>
      </div>
      <span v-else class="fv-empty">—</span>
    </template>

    <!-- Ссылка -->
    <template v-else-if="field.type === 'link'">
      <div v-if="value" class="fv-link">
        <a :href="value" target="_blank" rel="noopener" class="fv-link-text">{{ value }}</a>
        <button class="fv-link-btn" title="Открыть" @click="openLink"><span class="material-symbols-outlined">open_in_new</span></button>
        <button class="fv-link-btn" title="Копировать" @click="copyLink"><span class="material-symbols-outlined">content_copy</span></button>
      </div>
      <span v-else class="fv-empty">—</span>
    </template>

    <!-- Дата/время -->
    <template v-else-if="field.type === 'datetime'">
      <span v-if="value">{{ formatDateTime(value, field.config || {}) }}</span>
      <span v-else class="fv-empty">—</span>
    </template>

    <!-- Текст / число -->
    <template v-else>
      <div v-if="value != null && value !== ''" class="fv-textline">
        <span class="fv-text">{{ value }}</span>
        <button v-if="qrCode" class="fv-link-btn" title="Показать QR-код" @click="qrOpen = true">
          <span class="material-symbols-outlined">qr_code_2</span>
        </button>
      </div>
      <span v-else class="fv-empty">—</span>

      <AppDialog
        v-if="qrCode"
        v-model="qrOpen"
        :title="field.label"
        icon="qr_code_2"
        size="sm"
        :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
        @cancel="qrOpen = false"
      >
        <div class="fv-qr">
          <QrImage :value="qrCode" :size="240" />
          <span class="fv-qr-value">{{ qrCode }}</span>
        </div>
      </AppDialog>
    </template>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import ImageLightbox from '@/components/common/ImageLightbox.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import QrImage from '@/components/common/QrImage.vue'
import { formatDateTime, hasQr, qrValue } from '@/utils/registryFields.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  field: { type: Object, required: true },
  value: { default: null },
})

const lightbox = ref(false)
const qrOpen = ref(false)
const qrCode = computed(() => (hasQr(props.field) ? qrValue(props.value) : ''))
const src = computed(() => (props.value?.path ? `/uploads/${props.value.path}` : ''))
const selectChips = computed(() => {
  const v = props.value
  if (Array.isArray(v)) return v
  return v ? [v] : []
})

function openLink() { window.open(props.value, '_blank', 'noopener') }
async function copyLink() {
  try {
    await navigator.clipboard.writeText(props.value)
    useNotificationsStore().success('Ссылка скопирована')
  } catch { /* ignore */ }
}
</script>

<style scoped>
.fv-value { font-size: 14px; color: var(--color-text); word-break: break-word; }
.fv-empty { color: var(--color-text-dim); }
.fv-text { white-space: pre-wrap; }
.fv-textline { display: flex; align-items: flex-start; gap: 6px; }
.fv-textline .fv-text { flex: 1; min-width: 0; }
.fv-textline .fv-link-btn { flex-shrink: 0; }
.fv-qr { display: flex; flex-direction: column; align-items: center; gap: 12px; }
.fv-qr-value { font-size: 14px; font-weight: 600; color: var(--color-text); text-align: center; word-break: break-all; }

.fv-image {
  border: none;
  padding: 0;
  background: none;
  cursor: zoom-in;
  border-radius: var(--radius-md);
  overflow: hidden;
  display: inline-block;
  max-width: 100%;
}
.fv-image img { max-width: 100%; max-height: 220px; display: block; border-radius: var(--radius-md); }

.fv-file {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  max-width: 100%;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  color: var(--color-primary);
  text-decoration: none;
}
.fv-file:hover { background: var(--color-surface-high); }
.fv-file .material-symbols-outlined { flex-shrink: 0; }
.fv-file-name { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.fv-check { display: inline-flex; align-items: center; gap: 6px; color: var(--color-text-dim); }
.fv-check.on { color: var(--color-success); }

.fv-chips { display: flex; flex-wrap: wrap; gap: 6px; }
.fv-chip {
  padding: 3px 10px;
  border-radius: var(--radius-full);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 13px;
}

.fv-link { display: flex; align-items: center; gap: 6px; max-width: 100%; }
.fv-link-text { flex: 1; min-width: 0; color: var(--color-primary); text-decoration: none; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.fv-link-text:hover { text-decoration: underline; }
.fv-link-btn {
  width: 30px; height: 30px;
  display: grid; place-items: center;
  border: none; border-radius: var(--radius-full);
  background: var(--color-surface-low); color: var(--color-text-dim);
  cursor: pointer; flex-shrink: 0;
}
.fv-link-btn:hover { background: var(--color-surface-high); color: var(--color-primary); }
.fv-link-btn .material-symbols-outlined { font-size: 18px; }
</style>
