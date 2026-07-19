<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="done_all"
    size="sm"
    title="Прочитали"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="loading" class="rb-empty"><BrandLoader :size="48" /></div>
    <div v-else-if="!readers.length" class="rb-empty">
      <span class="material-symbols-outlined">visibility_off</span>
      <p>Пока никто не прочитал</p>
    </div>
    <ul v-else class="rb-list">
      <li v-for="u in readers" :key="u.id" class="rb-item">
        <img class="rb-ava" :src="avatarOf(u)" :alt="u.fio" />
        <span class="rb-name">{{ u.fio }}</span>
      </li>
    </ul>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { messageReadBy } from '@/api/messenger.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  messageId: { type: Number, default: null },
})
defineEmits(['update:modelValue'])

const readers = ref([])
const loading = ref(false)

watch(() => props.modelValue, async (v) => {
  if (!v || !props.messageId) return
  loading.value = true
  readers.value = []
  try {
    const r = await messageReadBy(props.messageId)
    readers.value = r.readers || []
  } catch {
    readers.value = []
  } finally {
    loading.value = false
  }
})

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}
</script>

<style scoped>
.rb-empty { display: flex; flex-direction: column; align-items: center; gap: 6px; padding: 24px; color: var(--color-text-dim); }
.rb-empty .material-symbols-outlined { font-size: 32px; opacity: 0.6; }
.rb-list { list-style: none; padding: 0; margin: 0; max-height: 50dvh; overflow-y: auto; }
.rb-item { display: flex; align-items: center; gap: 12px; padding: 8px 4px; }
.rb-ava { width: 36px; height: 36px; border-radius: 50%; object-fit: cover; }
.rb-name { font-size: 14px; font-weight: 600; color: var(--color-text); }
</style>
