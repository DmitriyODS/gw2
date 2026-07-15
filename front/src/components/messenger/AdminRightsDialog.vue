<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="admin_panel_settings"
    size="sm"
    :title="`Права: ${member?.user?.fio || ''}`"
    subtitle="Что может делать администратор в этой группе."
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: 'Сохранить', icon: 'check', disabled: busy },
    ]"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="save"
    @cancel="$emit('update:modelValue', false)"
  >
    <label class="ar-row">
      <span>Управлять участниками</span>
      <input type="checkbox" v-model="manageMembers" />
    </label>
    <label class="ar-row">
      <span>Менять название и аватар</span>
      <input type="checkbox" v-model="editInfo" />
    </label>
    <label class="ar-row">
      <span>Закреплять сообщения</span>
      <input type="checkbox" v-model="pinMessages" />
    </label>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  conversationId: { type: Number, required: true },
  member: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue'])

const messenger = useMessengerStore()
const notify = useNotificationsStore()

const manageMembers = ref(true)
const editInfo = ref(true)
const pinMessages = ref(true)
const busy = ref(false)

watch(() => props.modelValue, (v) => {
  if (v && props.member) {
    manageMembers.value = props.member.can_manage_members
    editInfo.value = props.member.can_edit_info
    pinMessages.value = props.member.can_pin_messages
    busy.value = false
  }
})

async function save() {
  busy.value = true
  try {
    await messenger.setMemberRightsAction(props.conversationId, props.member.user.id, {
      manage_members: manageMembers.value,
      edit_info: editInfo.value,
      pin_messages: pinMessages.value,
    })
    emit('update:modelValue', false)
  } catch (e) {
    notify.error(e?.message || 'Не удалось сохранить права')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.ar-row {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 12px 4px; border-bottom: 1px solid var(--color-outline-dim);
  font-size: 14px; color: var(--color-text); cursor: pointer;
}
.ar-row:last-child { border-bottom: none; }
.ar-row input { width: 18px; height: 18px; accent-color: var(--color-primary); }
</style>
