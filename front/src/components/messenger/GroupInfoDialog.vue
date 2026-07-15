<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="groups"
    size="md"
    title="О группе"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="gi-head">
      <button
        class="gi-avatar"
        type="button"
        :disabled="!canEditInfo"
        :title="canEditInfo ? 'Сменить аватар' : ''"
        @click="canEditInfo && (cropperOpen = true)"
      >
        <img v-if="conv?.avatar_path" :src="`/uploads/${conv.avatar_path}`" alt="" />
        <span v-else class="material-symbols-outlined">groups</span>
      </button>
      <div class="gi-head-info">
        <div v-if="!editingTitle" class="gi-title-row">
          <span class="gi-title">{{ conv?.title }}</span>
          <button v-if="canEditInfo" class="gi-icon-btn" title="Переименовать" @click="startEditTitle">
            <span class="material-symbols-outlined">edit</span>
          </button>
        </div>
        <div v-else class="gi-title-edit">
          <input v-model="titleDraft" class="gi-title-input" maxlength="120" @keydown.enter="saveTitle" />
          <button class="gi-icon-btn" @click="saveTitle"><span class="material-symbols-outlined">check</span></button>
          <button class="gi-icon-btn" @click="editingTitle = false"><span class="material-symbols-outlined">close</span></button>
        </div>
        <div class="gi-sub">{{ members.length }} участник{{ plural(members.length) }}</div>
      </div>
    </div>

    <!-- Быстрые действия -->
    <div class="gi-quick">
      <button class="gi-quick-btn" @click="toggleMute">
        <span class="material-symbols-outlined">{{ conv?.muted ? 'notifications' : 'notifications_off' }}</span>
        {{ conv?.muted ? 'Включить уведомления' : 'Выключить уведомления' }}
      </button>
    </div>

    <!-- Ссылка-приглашение -->
    <div v-if="canManageMembers" class="gi-invite">
      <div class="gi-section-title">Ссылка-приглашение</div>
      <div v-if="inviteUrl" class="gi-invite-row">
        <input class="gi-invite-input" :value="inviteUrl" readonly @focus="$event.target.select()" />
        <button class="gi-icon-btn" title="Скопировать" @click="copyInvite"><span class="material-symbols-outlined">content_copy</span></button>
        <button class="gi-icon-btn danger" title="Отозвать" @click="revokeInvite"><span class="material-symbols-outlined">link_off</span></button>
      </div>
      <button v-else class="gi-linkbtn" @click="createInvite">
        <span class="material-symbols-outlined">add_link</span> Создать ссылку
      </button>
    </div>

    <!-- Участники -->
    <div class="gi-section-title gi-members-head">
      Участники
      <button v-if="canManageMembers" class="gi-linkbtn small" @click="addOpen = true">
        <span class="material-symbols-outlined">person_add</span> Добавить
      </button>
    </div>
    <ul class="gi-members">
      <li v-for="m in members" :key="m.user?.id" class="gi-member">
        <img class="gi-member-ava" :src="avatarOf(m.user)" :alt="m.user?.fio" />
        <div class="gi-member-info">
          <div class="gi-member-name">{{ m.user?.fio }}<span v-if="m.user?.id === auth.userId" class="gi-you"> (вы)</span></div>
          <div class="gi-member-meta">@{{ m.user?.login }}</div>
        </div>
        <span v-if="m.role !== 'member'" class="gi-role" :class="m.role">{{ roleLabel(m.role) }}</span>
        <button
          v-if="canActOn(m)"
          class="gi-icon-btn"
          title="Действия"
          @click="openMemberMenu($event, m)"
        >
          <span class="material-symbols-outlined">more_vert</span>
        </button>
      </li>
    </ul>

    <template #footer-start>
      <div class="gi-footer-actions">
        <button class="gi-leave" @click="confirmLeave = true">
          <span class="material-symbols-outlined">logout</span> Выйти из группы
        </button>
        <button v-if="myRole === 'owner'" class="gi-leave" @click="confirmDelete = true">
          <span class="material-symbols-outlined">delete_forever</span> Удалить группу
        </button>
      </div>
    </template>

    <AddMembersDialog v-model="addOpen" :conversation-id="conversationId" :existing-ids="memberIds" />
    <AdminRightsDialog v-model="rightsOpen" :conversation-id="conversationId" :member="menuMember" />
    <AppDialog
      v-if="cropperOpen"
      model-value
      tone="primary"
      icon="account_circle"
      size="md"
      title="Аватар группы"
      @update:model-value="cropperOpen = false"
    >
      <AvatarCropper @cropped="onCropped" @cancel="cropperOpen = false" />
    </AppDialog>
    <ConfirmDialog
      :visible="confirmLeave"
      header="Выйти из группы"
      :message="myRole === 'owner' ? 'Вы владелец — при выходе владение перейдёт другому участнику. Выйти?' : 'Покинуть эту группу?'"
      confirm-label="Выйти"
      danger-confirm
      @confirm="leave"
      @cancel="confirmLeave = false"
    />
    <ConfirmDialog
      :visible="!!confirmRemove"
      header="Удалить участника"
      :message="`Удалить ${confirmRemove?.user?.fio || ''} из группы?`"
      confirm-label="Удалить"
      danger-confirm
      @confirm="doRemove"
      @cancel="confirmRemove = null"
    />
    <ConfirmDialog
      :visible="confirmDelete"
      header="Удалить группу"
      message="Группа исчезнет у всех участников вместе со всей перепиской. Это действие нельзя отменить."
      confirm-label="Удалить группу"
      danger-confirm
      @confirm="destroyGroup"
      @cancel="confirmDelete = false"
    />
    <ConfirmDialog
      :visible="!!confirmTransfer"
      header="Передать владение"
      :message="`Сделать ${confirmTransfer?.user?.fio || ''} владельцем группы? Вы станете администратором.`"
      confirm-label="Передать"
      @confirm="doTransfer"
      @cancel="confirmTransfer = null"
    />

    <!-- Меню действий над участником -->
    <Teleport to="body">
      <div v-if="menuOpen" class="gi-menu-mask" @click="menuOpen = false">
        <div class="gi-menu" :style="menuStyle" @click.stop>
          <button v-if="myRole === 'owner' && menuMember?.role === 'member'" class="gi-menu-item" @click="setRole(menuMember, 'admin')">
            <span class="material-symbols-outlined">shield_person</span> Назначить админом
          </button>
          <button v-if="myRole === 'owner' && menuMember?.role === 'admin'" class="gi-menu-item" @click="setRole(menuMember, 'member')">
            <span class="material-symbols-outlined">remove_moderator</span> Снять админа
          </button>
          <button v-if="myRole === 'owner' && menuMember?.role === 'admin'" class="gi-menu-item" @click="openRights(menuMember)">
            <span class="material-symbols-outlined">tune</span> Настроить права
          </button>
          <button v-if="myRole === 'owner'" class="gi-menu-item" @click="askTransfer(menuMember)">
            <span class="material-symbols-outlined">workspace_premium</span> Сделать владельцем
          </button>
          <button class="gi-menu-item danger" @click="askRemove(menuMember)">
            <span class="material-symbols-outlined">person_remove</span> Удалить из группы
          </button>
        </div>
      </div>
    </Teleport>
  </AppDialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import AddMembersDialog from './AddMembersDialog.vue'
import AdminRightsDialog from './AdminRightsDialog.vue'
import AvatarCropper from '@/components/settings/AvatarCropper.vue'
import { useMessengerStore } from '@/stores/messenger.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { uploadAttachment } from '@/api/messenger.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  conversationId: { type: Number, required: true },
})
const emit = defineEmits(['update:modelValue', 'left'])

const messenger = useMessengerStore()
const auth = useAuthStore()
const notify = useNotificationsStore()

const conv = computed(() => messenger.conversationById.get(props.conversationId) || null)
const members = computed(() => messenger.groupMembers(props.conversationId))
const memberIds = computed(() => members.value.map((m) => m.user?.id))
const myMember = computed(() => members.value.find((m) => m.user?.id === auth.userId) || null)
const myRole = computed(() => myMember.value?.role || conv.value?.my_role || 'member')
const canEditInfo = computed(() => myRole.value === 'owner' || (myRole.value === 'admin' && myMember.value?.can_edit_info))
const canManageMembers = computed(() => myRole.value === 'owner' || (myRole.value === 'admin' && myMember.value?.can_manage_members))

const editingTitle = ref(false)
const titleDraft = ref('')
const cropperOpen = ref(false)
const addOpen = ref(false)
const rightsOpen = ref(false)
const confirmLeave = ref(false)
const confirmDelete = ref(false)
const confirmRemove = ref(null)
const confirmTransfer = ref(null)
const inviteUrl = ref('')

const menuOpen = ref(false)
const menuMember = ref(null)
const menuStyle = ref({})

watch(() => props.modelValue, (v) => {
  if (v) {
    editingTitle.value = false
    inviteUrl.value = conv.value?.invite_code ? inviteLinkFor(conv.value.invite_code) : ''
    messenger.fetchGroup(props.conversationId).catch(() => {})
  }
})

function plural(n) { const d = n % 10, dd = n % 100; if (d === 1 && dd !== 11) return ''; if (d >= 2 && d <= 4 && (dd < 10 || dd >= 20)) return 'а'; return 'ов' }
function avatarOf(u) { return u?.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u?.id}/identicon` }
function roleLabel(role) { return role === 'owner' ? 'Владелец' : role === 'admin' ? 'Админ' : '' }
function inviteLinkFor(code) { return `${window.location.origin}/group/${code}` }

function canActOn(m) {
  if (m.user?.id === auth.userId || m.role === 'owner') return false
  if (myRole.value === 'owner') return true
  // Админ с правом управлять — только обычных участников.
  return canManageMembers.value && m.role === 'member'
}

function startEditTitle() { titleDraft.value = conv.value?.title || ''; editingTitle.value = true }
async function saveTitle() {
  const t = titleDraft.value.trim()
  if (!t) return
  try { await messenger.renameGroupAction(props.conversationId, t) } catch (e) { notify.error(e?.message || 'Ошибка') }
  editingTitle.value = false
}

// Кроппер (как для аватарки профиля) отдаёт готовый JPEG-blob.
async function onCropped(blob) {
  cropperOpen.value = false
  try {
    const file = new File([blob], 'group-avatar.jpg', { type: blob.type || 'image/jpeg' })
    const att = await uploadAttachment(file)
    await messenger.setGroupAvatarAction(props.conversationId, att.id)
  } catch { notify.error('Не удалось обновить аватар') }
}

async function toggleMute() {
  try { await messenger.setGroupMuteAction(props.conversationId, !conv.value?.muted) } catch (e) { notify.error(e?.message || 'Ошибка') }
}

async function createInvite() {
  try { inviteUrl.value = inviteLinkFor(await messenger.groupInviteLinkAction(props.conversationId)) }
  catch (e) { notify.error(e?.message || 'Ошибка') }
}
async function copyInvite() {
  try { await navigator.clipboard.writeText(inviteUrl.value); notify.success('Ссылка скопирована') }
  catch { notify.error('Не удалось скопировать') }
}
async function revokeInvite() {
  try { await messenger.revokeGroupInviteLinkAction(props.conversationId); inviteUrl.value = '' }
  catch (e) { notify.error(e?.message || 'Ошибка') }
}

function openMemberMenu(ev, m) {
  menuMember.value = m
  const r = ev.currentTarget.getBoundingClientRect()
  menuStyle.value = { position: 'fixed', top: `${r.bottom + 4}px`, left: `${Math.max(8, r.right - 220)}px`, zIndex: 12000 }
  menuOpen.value = true
}
async function setRole(m, role) {
  menuOpen.value = false
  try { await messenger.setMemberRoleAction(props.conversationId, m.user.id, role) } catch (e) { notify.error(e?.message || 'Ошибка') }
}
function openRights(m) { menuOpen.value = false; menuMember.value = m; rightsOpen.value = true }
function askTransfer(m) { menuOpen.value = false; confirmTransfer.value = m }
async function doTransfer() {
  const m = confirmTransfer.value; confirmTransfer.value = null
  if (!m) return
  try { await messenger.transferOwnershipAction(props.conversationId, m.user.id) } catch (e) { notify.error(e?.message || 'Ошибка') }
}
function askRemove(m) { menuOpen.value = false; confirmRemove.value = m }
async function doRemove() {
  const m = confirmRemove.value; confirmRemove.value = null
  if (!m) return
  try { await messenger.removeGroupMemberAction(props.conversationId, m.user.id) } catch (e) { notify.error(e?.message || 'Ошибка') }
}
async function leave() {
  confirmLeave.value = false
  try {
    await messenger.leaveGroupAction(props.conversationId)
    emit('left')
    emit('update:modelValue', false)
  } catch (e) { notify.error(e?.message || 'Ошибка') }
}

// Роспуск группы (только владелец) — исчезает у всех участников.
async function destroyGroup() {
  confirmDelete.value = false
  try {
    await messenger.deleteConversationAction(props.conversationId, 'all')
    emit('left')
    emit('update:modelValue', false)
  } catch (e) { notify.error(e?.message || 'Не удалось удалить группу') }
}
</script>

<style scoped>
.gi-head { display: flex; align-items: center; gap: 14px; margin-bottom: 16px; }
.gi-avatar {
  width: 64px; height: 64px; flex-shrink: 0; border-radius: 50%; overflow: hidden;
  border: none; background: var(--color-primary-container); color: var(--color-on-primary-container);
  display: grid; place-items: center; cursor: pointer;
}
.gi-avatar:disabled { cursor: default; }
.gi-avatar img { width: 100%; height: 100%; object-fit: cover; }
.gi-avatar .material-symbols-outlined { font-size: 32px; font-variation-settings: 'FILL' 1; }
.gi-head-info { min-width: 0; flex: 1; }
.gi-title-row { display: flex; align-items: center; gap: 6px; }
.gi-title { font-size: 18px; font-weight: 700; color: var(--color-text); }
.gi-title-edit { display: flex; align-items: center; gap: 4px; }
.gi-title-input { flex: 1; padding: 6px 10px; border: 1px solid var(--color-primary); border-radius: var(--radius-md); background: var(--color-surface-low); color: var(--color-text); font: inherit; font-size: 16px; font-weight: 600; outline: none; }
.gi-sub { font-size: 13px; color: var(--color-text-dim); margin-top: 2px; }

.gi-icon-btn { border: none; background: transparent; color: var(--color-text-dim); cursor: pointer; display: inline-flex; padding: 6px; border-radius: 50%; }
.gi-icon-btn:hover { background: var(--color-surface-high); color: var(--color-text); }
.gi-icon-btn.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.gi-icon-btn .material-symbols-outlined { font-size: 18px; }

.gi-quick { margin-bottom: 12px; }
.gi-quick-btn { display: inline-flex; align-items: center; gap: 8px; width: 100%; padding: 10px 12px; border: none; border-radius: var(--radius-md); background: var(--color-surface-low); color: var(--color-text); font: inherit; font-size: 14px; cursor: pointer; }
.gi-quick-btn:hover { background: var(--color-surface-high); }
.gi-quick-btn .material-symbols-outlined { font-size: 20px; color: var(--color-text-dim); }

.gi-section-title { font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.04em; color: var(--color-text-dim); margin: 12px 0 8px; }
.gi-members-head { display: flex; align-items: center; justify-content: space-between; }
.gi-invite { margin-bottom: 4px; }
.gi-invite-row { display: flex; align-items: center; gap: 4px; }
.gi-invite-input { flex: 1; padding: 8px 10px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--color-surface-low); color: var(--color-text); font: inherit; font-size: 13px; outline: none; }
.gi-linkbtn { display: inline-flex; align-items: center; gap: 6px; border: none; background: transparent; color: var(--color-primary); font: inherit; font-size: 14px; font-weight: 600; cursor: pointer; padding: 6px 0; }
.gi-linkbtn.small { font-size: 13px; }
.gi-linkbtn .material-symbols-outlined { font-size: 18px; }

.gi-members { list-style: none; padding: 0; margin: 0; max-height: 34dvh; overflow-y: auto; }
.gi-member { display: flex; align-items: center; gap: 12px; padding: 8px 4px; }
.gi-member-ava { width: 40px; height: 40px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.gi-member-info { min-width: 0; flex: 1; }
.gi-member-name { font-size: 14px; font-weight: 600; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.gi-you { color: var(--color-text-dim); font-weight: 400; }
.gi-member-meta { font-size: 12px; color: var(--color-text-dim); }
.gi-role { font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: var(--radius-full); background: var(--color-surface-high); color: var(--color-text-dim); }
.gi-role.owner { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.gi-role.admin { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }

.gi-footer-actions { display: flex; flex-wrap: wrap; gap: 4px 16px; }
.gi-leave { display: inline-flex; align-items: center; gap: 6px; border: none; background: transparent; color: var(--color-error); font: inherit; font-size: 14px; font-weight: 600; cursor: pointer; padding: 8px 0; }
.gi-leave .material-symbols-outlined { font-size: 18px; }

.gi-menu-mask { position: fixed; inset: 0; z-index: 11999; }
.gi-menu { min-width: 210px; background: var(--acrylic-card-bg); border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); padding: 6px; box-shadow: var(--shadow-lg); display: flex; flex-direction: column; gap: 2px; }
.gi-menu-item { display: flex; align-items: center; gap: 10px; padding: 10px 12px; border: none; background: transparent; color: var(--color-text); font: inherit; font-size: 14px; text-align: left; border-radius: var(--radius-sm); cursor: pointer; }
.gi-menu-item:hover { background: var(--color-surface-low); }
.gi-menu-item.danger { color: var(--color-error); }
.gi-menu-item.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.gi-menu-item .material-symbols-outlined { font-size: 18px; }
</style>
