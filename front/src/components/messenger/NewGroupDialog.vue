<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="group_add"
    size="sm"
    title="Новая группа"
    subtitle="Название, участники — и можно общаться."
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: 'Создать', icon: 'check', disabled: !canCreate || creating },
    ]"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="create"
    @cancel="$emit('update:modelValue', false)"
  >
    <div class="ng-head">
      <button class="ng-avatar" type="button" :title="avatarPreview ? 'Сменить аватар' : 'Загрузить аватар'" @click="cropperOpen = true">
        <img v-if="avatarPreview" :src="avatarPreview" alt="" />
        <span v-else class="material-symbols-outlined">add_a_photo</span>
      </button>
      <input
        v-model="title"
        class="ng-title-input"
        placeholder="Название группы"
        maxlength="120"
        @keydown.enter.prevent="create"
      />
    </div>

    <div v-if="selected.length" class="ng-chips">
      <span v-for="u in selected" :key="u.id" class="member-chip">
        <img class="member-chip-ava" :src="avatarOf(u)" :alt="u.fio" />
        <span class="member-chip-name">{{ u.fio }}</span>
        <button type="button" class="member-chip-x" :aria-label="`Убрать ${u.fio}`" @click="toggle(u)">
          <span class="material-symbols-outlined">close</span>
        </button>
      </span>
    </div>

    <div class="ng-search">
      <span class="material-symbols-outlined">search</span>
      <input v-model="q" placeholder="Добавить участников — по фамилии или логину" class="ng-search-input" />
    </div>

    <div v-if="loading && !results.length" class="ng-empty">
      <ProgressSpinner style="width:28px;height:28px" />
    </div>
    <ul v-else class="ng-results">
      <li
        v-for="u in results"
        :key="u.id"
        class="ng-item"
        :class="{ picked: isPicked(u) }"
        @click="toggle(u)"
      >
        <img class="ng-item-ava" :src="avatarOf(u)" :alt="u.fio" />
        <div class="ng-item-info">
          <div class="ng-item-name">{{ u.fio }}</div>
          <div class="ng-item-meta">@{{ u.login }}</div>
        </div>
        <span class="material-symbols-outlined ng-check">{{ isPicked(u) ? 'check_circle' : 'radio_button_unchecked' }}</span>
      </li>
    </ul>

    <AppDialog
      v-if="cropperOpen"
      model-value
      tone="primary"
      icon="account_circle"
      size="md"
      title="Аватар группы"
      mask-class="ng-cropper-mask"
      @update:model-value="cropperOpen = false"
    >
      <AvatarCropper @cropped="onCropped" @cancel="cropperOpen = false" />
    </AppDialog>
  </AppDialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import AppDialog from '@/components/common/AppDialog.vue'
import AvatarCropper from '@/components/settings/AvatarCropper.vue'
import { useContactPicker } from '@/composables/useContactPicker.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { uploadAttachment } from '@/api/messenger.js'

const props = defineProps({ modelValue: { type: Boolean, default: false } })
const emit = defineEmits(['update:modelValue', 'created'])

const messenger = useMessengerStore()
const notify = useNotificationsStore()
const { q, results, loading, reset } = useContactPicker()

const title = ref('')
const selected = ref([])
const avatarPreview = ref(null)
const avatarAttachmentId = ref(null)
const creating = ref(false)
const cropperOpen = ref(false)

const canCreate = computed(() => title.value.trim().length > 0 && selected.value.length > 0)

watch(() => props.modelValue, (v) => {
  if (v) {
    title.value = ''
    selected.value = []
    avatarPreview.value = null
    avatarAttachmentId.value = null
    creating.value = false
    reset()
  }
})

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}
function isPicked(u) {
  return selected.value.some((s) => s.id === u.id)
}
function toggle(u) {
  if (isPicked(u)) selected.value = selected.value.filter((s) => s.id !== u.id)
  else selected.value = [...selected.value, u]
}

// Кроппер (тот же, что для аватарки профиля) отдаёт готовый JPEG-blob.
async function onCropped(blob) {
  cropperOpen.value = false
  try {
    const file = new File([blob], 'group-avatar.jpg', { type: blob.type || 'image/jpeg' })
    const att = await uploadAttachment(file)
    avatarPreview.value = att.thumb_url || att.url
    avatarAttachmentId.value = att.id
  } catch {
    notify.error('Не удалось загрузить аватар')
  }
}

async function create() {
  if (!canCreate.value || creating.value) return
  creating.value = true
  try {
    const id = await messenger.createGroup({
      title: title.value.trim(),
      memberIds: selected.value.map((u) => u.id),
      avatarAttachmentId: avatarAttachmentId.value,
    })
    emit('created', id)
    emit('update:modelValue', false)
  } catch (err) {
    notify.error(err?.message || 'Не удалось создать группу')
  } finally {
    creating.value = false
  }
}
</script>

<style scoped>
.ng-head { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
.ng-avatar {
  width: 52px; height: 52px; flex-shrink: 0; border-radius: 50%;
  border: 1px dashed var(--color-outline-dim); background: var(--color-surface-low);
  color: var(--color-text-dim); display: grid; place-items: center; cursor: pointer; overflow: hidden;
}
.ng-avatar img { width: 100%; height: 100%; object-fit: cover; }
.ng-title-input {
  flex: 1; min-width: 0; padding: 10px 12px; border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md); background: var(--color-surface-low); color: var(--color-text);
  font: inherit; font-size: 15px; font-weight: 600; outline: none;
}
.ng-title-input:focus { border-color: var(--color-primary); }

.ng-chips {
  display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 10px;
  max-height: 96px; overflow-y: auto;
}
/* Аккуратный input-chip участника (Material 3): мини-аватар, тонкая заливка,
   имя с обрезкой — не растягивает форму на длинных ФИО. */
.member-chip {
  display: inline-flex; align-items: center; gap: 6px;
  max-width: 100%;
  padding: 3px 6px 3px 3px;
  border-radius: var(--radius-full);
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

.ng-search { position: relative; display: flex; align-items: center; margin-bottom: 8px; }
.ng-search .material-symbols-outlined { position: absolute; left: 12px; color: var(--color-text-dim); font-size: 20px; pointer-events: none; }
.ng-search-input {
  width: 100%; padding: 10px 12px 10px 40px; border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md); background: var(--color-surface-low); color: var(--color-text);
  font: inherit; font-size: 14px; outline: none;
}
.ng-search-input:focus { border-color: var(--color-primary); }

.ng-results { list-style: none; padding: 0; margin: 0; max-height: 42dvh; overflow-y: auto; }
.ng-item { display: flex; gap: 12px; align-items: center; padding: 8px; cursor: pointer; border-radius: var(--radius-md); }
.ng-item:hover { background: var(--color-surface-low); }
.ng-item.picked { background: color-mix(in oklab, var(--color-primary) 10%, transparent); }
.ng-item-ava { width: 38px; height: 38px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.ng-item-info { min-width: 0; flex: 1; }
.ng-item-name { font-size: 14px; font-weight: 600; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.ng-item-meta { font-size: 12px; color: var(--color-text-dim); }
.ng-check { color: var(--color-primary); }
.ng-item:not(.picked) .ng-check { color: var(--color-outline-dim); }
.ng-empty { display: flex; justify-content: center; padding: 24px; }
</style>
