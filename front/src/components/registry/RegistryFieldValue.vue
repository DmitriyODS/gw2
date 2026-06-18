<template>
  <div class="rf-value">
    <!-- Картинка -->
    <template v-if="field.type === 'image'">
      <button v-if="src" class="rf-image" @click="lightbox = true">
        <img :src="src" :alt="value?.name || ''" />
      </button>
      <span v-else class="rf-empty">—</span>
      <ImageLightbox v-if="src" v-model="lightbox" :src="src" :caption="value?.name || ''" />
    </template>

    <!-- Файл -->
    <template v-else-if="field.type === 'file'">
      <a v-if="src" class="rf-file" :href="src" :download="value?.name || ''" target="_blank" rel="noopener">
        <span class="material-symbols-outlined">description</span>
        <span class="rf-file-name">{{ value?.name || 'Файл' }}</span>
      </a>
      <span v-else class="rf-empty">—</span>
    </template>

    <!-- Галочка -->
    <template v-else-if="field.type === 'checkbox'">
      <span class="rf-check" :class="{ on: !!value }">
        <span class="material-symbols-outlined">{{ value ? 'check_box' : 'check_box_outline_blank' }}</span>
        {{ value ? 'Да' : 'Нет' }}
      </span>
    </template>

    <!-- Список -->
    <template v-else-if="field.type === 'select'">
      <div v-if="selectChips.length" class="rf-chips">
        <span v-for="c in selectChips" :key="c" class="rf-chip">{{ c }}</span>
      </div>
      <span v-else class="rf-empty">—</span>
    </template>

    <!-- Ссылка -->
    <template v-else-if="field.type === 'link'">
      <div v-if="value" class="rf-link">
        <a :href="value" target="_blank" rel="noopener" class="rf-link-text">{{ value }}</a>
        <button class="rf-link-btn" title="Открыть" @click="openLink"><span class="material-symbols-outlined">open_in_new</span></button>
        <button class="rf-link-btn" title="Копировать" @click="copyLink"><span class="material-symbols-outlined">content_copy</span></button>
      </div>
      <span v-else class="rf-empty">—</span>
    </template>

    <!-- Дата/время -->
    <template v-else-if="field.type === 'datetime'">
      <span v-if="value">{{ formatDateTime(value, field.config || {}) }}</span>
      <span v-else class="rf-empty">—</span>
    </template>

    <!-- Текст / число -->
    <template v-else>
      <span v-if="value != null && value !== ''" class="rf-text">{{ value }}</span>
      <span v-else class="rf-empty">—</span>
    </template>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import ImageLightbox from '@/components/common/ImageLightbox.vue'
import { formatDateTime } from '@/utils/registryFields.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  field: { type: Object, required: true },
  value: { default: null },
})

const lightbox = ref(false)
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
.rf-value { font-size: 14px; color: var(--color-text); word-break: break-word; }
.rf-empty { color: var(--color-text-dim); }
.rf-text { white-space: pre-wrap; }

.rf-image {
  border: none;
  padding: 0;
  background: none;
  cursor: zoom-in;
  border-radius: var(--radius-md);
  overflow: hidden;
  display: inline-block;
  max-width: 100%;
}
.rf-image img { max-width: 100%; max-height: 220px; display: block; border-radius: var(--radius-md); }

.rf-file {
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
.rf-file:hover { background: var(--color-surface-high); }
.rf-file .material-symbols-outlined { flex-shrink: 0; }
.rf-file-name { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.rf-check { display: inline-flex; align-items: center; gap: 6px; color: var(--color-text-dim); }
.rf-check.on { color: var(--color-success); }

.rf-chips { display: flex; flex-wrap: wrap; gap: 6px; }
.rf-chip {
  padding: 3px 10px;
  border-radius: var(--radius-full);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 13px;
}

.rf-link { display: flex; align-items: center; gap: 6px; max-width: 100%; }
.rf-link-text { flex: 1; min-width: 0; color: var(--color-primary); text-decoration: none; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.rf-link-text:hover { text-decoration: underline; }
.rf-link-btn {
  width: 30px; height: 30px;
  display: grid; place-items: center;
  border: none; border-radius: var(--radius-full);
  background: var(--color-surface-low); color: var(--color-text-dim);
  cursor: pointer; flex-shrink: 0;
}
.rf-link-btn:hover { background: var(--color-surface-high); color: var(--color-primary); }
.rf-link-btn .material-symbols-outlined { font-size: 18px; }
</style>
