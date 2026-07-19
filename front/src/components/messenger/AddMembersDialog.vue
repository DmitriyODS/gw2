<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="person_add"
    size="sm"
    title="Добавить участников"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: 'Добавить', icon: 'check', disabled: !selected.length || busy },
    ]"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="submit"
    @cancel="$emit('update:modelValue', false)"
  >
    <div v-if="selected.length" class="am-chips">
      <span v-for="u in selected" :key="u.id" class="member-chip">
        <img class="member-chip-ava" :src="avatarOf(u)" :alt="u.fio" />
        <span class="member-chip-name">{{ u.fio }}</span>
        <button type="button" class="member-chip-x" :aria-label="`Убрать ${u.fio}`" @click="toggle(u)">
          <span class="material-symbols-outlined">close</span>
        </button>
      </span>
    </div>
    <div class="am-search">
      <span class="material-symbols-outlined">search</span>
      <input v-model="q" placeholder="По фамилии или логину" class="am-search-input" />
    </div>
    <div v-if="loading && !results.length" class="am-empty"><BrandLoader :size="48" /></div>
    <ul v-else class="am-results">
      <li
        v-for="u in candidates"
        :key="u.id"
        class="am-item"
        :class="{ picked: isPicked(u) }"
        @click="toggle(u)"
      >
        <img class="am-ava" :src="avatarOf(u)" :alt="u.fio" />
        <div class="am-info">
          <div class="am-name">{{ u.fio }}</div>
          <div class="am-meta">@{{ u.login }}</div>
        </div>
        <span class="material-symbols-outlined am-check">{{ isPicked(u) ? 'check_circle' : 'radio_button_unchecked' }}</span>
      </li>
    </ul>
  </AppDialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { useContactPicker } from '@/composables/useContactPicker.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  conversationId: { type: Number, required: true },
  existingIds: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue'])

const messenger = useMessengerStore()
const notify = useNotificationsStore()
const { q, results, loading, reset } = useContactPicker()

const selected = ref([])
const busy = ref(false)

const candidates = computed(() => {
  const ex = new Set(props.existingIds)
  return results.value.filter((u) => !ex.has(u.id))
})

watch(() => props.modelValue, (v) => {
  if (v) { selected.value = []; busy.value = false; reset() }
})

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}
function isPicked(u) { return selected.value.some((s) => s.id === u.id) }
function toggle(u) {
  if (isPicked(u)) selected.value = selected.value.filter((s) => s.id !== u.id)
  else selected.value = [...selected.value, u]
}

async function submit() {
  if (!selected.value.length || busy.value) return
  busy.value = true
  try {
    await messenger.addGroupMembersAction(props.conversationId, selected.value.map((u) => u.id))
    emit('update:modelValue', false)
  } catch (e) {
    notify.error(e?.message || 'Не удалось добавить')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.am-chips { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 10px; max-height: 96px; overflow-y: auto; }
.member-chip {
  display: inline-flex; align-items: center; gap: 6px; max-width: 100%;
  padding: 3px 6px 3px 3px; border-radius: var(--radius-full);
  background: color-mix(in oklab, var(--color-primary) 12%, var(--color-surface-low));
  border: 1px solid var(--color-outline-dim);
  font-size: 13px; font-weight: 500; color: var(--color-text);
}
.member-chip-ava { width: 22px; height: 22px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.member-chip-name { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.member-chip-x {
  flex-shrink: 0; display: inline-flex; align-items: center; justify-content: center;
  width: 18px; height: 18px; min-height: 0; padding: 0; border: none; border-radius: 50%;
  background: transparent; color: var(--color-text-dim); cursor: pointer;
}
.member-chip-x:hover { background: var(--color-surface-high); color: var(--color-text); }
.member-chip-x .material-symbols-outlined { font-size: 15px; }
.am-search { position: relative; display: flex; align-items: center; margin-bottom: 8px; }
.am-search .material-symbols-outlined { position: absolute; left: 12px; color: var(--color-text-dim); font-size: 20px; pointer-events: none; }
.am-search-input { width: 100%; padding: 10px 12px 10px 40px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--color-surface-low); color: var(--color-text); font: inherit; font-size: 14px; outline: none; }
.am-search-input:focus { border-color: var(--color-primary); }
.am-results { list-style: none; padding: 0; margin: 0; max-height: 44dvh; overflow-y: auto; }
.am-item { display: flex; gap: 12px; align-items: center; padding: 8px; cursor: pointer; border-radius: var(--radius-md); }
.am-item:hover { background: var(--color-surface-low); }
.am-item.picked { background: color-mix(in oklab, var(--color-primary) 10%, transparent); }
.am-ava { width: 38px; height: 38px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.am-info { min-width: 0; flex: 1; }
.am-name { font-size: 14px; font-weight: 600; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.am-meta { font-size: 12px; color: var(--color-text-dim); }
.am-check { color: var(--color-primary); }
.am-item:not(.picked) .am-check { color: var(--color-outline-dim); }
.am-empty { display: flex; justify-content: center; padding: 24px; }
</style>
