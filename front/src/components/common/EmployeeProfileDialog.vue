<template>
  <!-- Профиль сотрудника: градиентный hero + список контактов + действия.
       Один компонент для раздела «Сотрудники» и портала (клик по автору
       поста/комментария). Пользователь — объект из каталога getDirectory(). -->
  <Dialog
    :visible="modelValue"
    modal
    :draggable="false"
    :show-header="false"
    :dismissable-mask="true"
    :style="{ width: '460px', maxWidth: 'calc(100vw - 24px)' }"
    :pt="{
      root: { class: 'emp-dialog' },
      content: { style: 'overflow-x: hidden; padding: 0; background: transparent' },
      mask: { style: 'background: var(--color-scrim)', class: elevated ? 'emp-mask--elevated' : '' },
    }"
    @update:visible="$emit('update:modelValue', $event)"
  >
    <div v-if="user" class="emp-profile">
      <div class="profile-cover" aria-hidden="true"></div>
      <button class="profile-close" @click="close" aria-label="Закрыть">
        <span class="material-symbols-outlined">close</span>
      </button>

      <div class="profile-hero">
        <button class="profile-avatar-btn" @click="lightboxOpen = true" aria-label="Открыть фото">
          <span class="avatar avatar-xl" :class="presenceClass(user)">
            <img :src="avatarOf(user)" :alt="user.fio" />
          </span>
        </button>
        <h2 class="profile-name">
          {{ user.fio }}
          <span
            v-if="user.is_super_admin"
            class="root-badge inline"
            title="Супер-администратор платформы"
          >
            <span class="material-symbols-outlined">verified</span>
          </span>
        </h2>
        <div class="profile-tags">
          <RolePill :level="user.role?.level" :name="user.role?.name" />
          <span :class="['profile-status', { on: messenger.isOnline(user.id) }]">
            <span class="status-dot" />
            {{ statusOf(user) }}
          </span>
        </div>
      </div>

      <div class="profile-list">
        <div v-if="user.status_emoji || user.status_text" class="profile-row">
          <span class="row-ico" data-tone="tertiary">
            <span v-if="user.status_emoji" class="row-status-emoji">{{ user.status_emoji }}</span>
            <span v-else class="material-symbols-outlined">mood</span>
          </span>
          <span class="row-text">
            <span class="row-label">Статус</span>
            <span class="row-value">{{ user.status_text || user.status_emoji }}</span>
          </span>
        </div>
        <div v-if="user.post" class="profile-row">
          <span class="row-ico" data-tone="primary">
            <span class="material-symbols-outlined">badge</span>
          </span>
          <span class="row-text">
            <span class="row-label">Должность</span>
            <span class="row-value">{{ user.post }}</span>
          </span>
        </div>
        <div class="profile-row">
          <span class="row-ico" data-tone="secondary">
            <span class="material-symbols-outlined">alternate_email</span>
          </span>
          <span class="row-text">
            <span class="row-label">Логин</span>
            <span class="row-value">@{{ user.login }}</span>
          </span>
        </div>
        <a
          v-if="user.phone"
          class="profile-row link"
          :href="`tel:${user.phone}`"
        >
          <span class="row-ico" data-tone="tertiary">
            <span class="material-symbols-outlined">phone</span>
          </span>
          <span class="row-text">
            <span class="row-label">Телефон</span>
            <span class="row-value">{{ fmtPhone(user.phone) }}</span>
          </span>
          <span class="material-symbols-outlined row-chev">arrow_outward</span>
        </a>
        <a
          v-if="user.email"
          class="profile-row link"
          :href="`mailto:${user.email}`"
        >
          <span class="row-ico" data-tone="tertiary">
            <span class="material-symbols-outlined">mail</span>
          </span>
          <span class="row-text">
            <span class="row-label">Email</span>
            <span class="row-value">{{ user.email }}</span>
          </span>
          <span class="material-symbols-outlined row-chev">arrow_outward</span>
        </a>
        <div v-if="companyOf(user)" class="profile-row">
          <span class="row-ico" data-tone="primary">
            <span class="material-symbols-outlined">domain</span>
          </span>
          <span class="row-text">
            <span class="row-label">Компания</span>
            <span class="row-value">{{ companyOf(user) }}</span>
          </span>
        </div>
      </div>

      <div v-if="canViewActivity" class="profile-lead">
        <button class="btn-grad profile-activity" @click="openActivity">
          <span class="material-symbols-outlined">monitoring</span>
          Активность сотрудника
        </button>
      </div>

      <div v-if="user.id !== auth.user?.id" class="profile-actions">
        <button class="btn-glass" @click="writeTo(user)">
          <span class="material-symbols-outlined">chat</span>
          Написать
        </button>
        <button class="btn-glass" @click="callTo(user, 'video')">
          <span class="material-symbols-outlined">videocam</span>
          <span class="hide-narrow">Видео</span>
        </button>
        <button class="btn-glass" @click="callTo(user, 'audio')">
          <span class="material-symbols-outlined">call</span>
          <span class="hide-narrow">Аудио</span>
        </button>
      </div>
    </div>
  </Dialog>

  <ImageLightbox
    v-if="user"
    v-model="lightboxOpen"
    :src="avatarOf(user)"
    :caption="user.fio"
  />
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import Dialog from 'primevue/dialog'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useCallStore } from '@/stores/call.js'
import { formatLastSeen } from '@/utils/presence.js'
import ImageLightbox from '@/components/common/ImageLightbox.vue'
import RolePill from '@/components/common/RolePill.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  user: { type: Object, default: null },
  // Поднять над плавающими слоями (мини-мессенджер живёт на 10050).
  elevated: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const router = useRouter()
const auth = useAuthStore()
const companies = useCompaniesStore()
const messenger = useMessengerStore()
const callStore = useCallStore()

const lightboxOpen = ref(false)

// «Активность» видит только руководитель компании (роль Администратор) — над
// другими сотрудниками. Бэкенд повторно гардирует доступ.
const canViewActivity = computed(() =>
  props.user && props.user.id !== auth.user?.id && auth.roleLevel >= 3)

function openActivity() {
  const id = props.user?.id
  close()
  if (id) router.push(`/employees/${id}/activity`)
}

watch(() => props.modelValue, (open) => {
  if (!open) lightboxOpen.value = false
})

function close() {
  emit('update:modelValue', false)
}

function statusOf(u) {
  if (messenger.isOnline(u.id)) return 'в сети'
  return formatLastSeen(messenger.lastSeenOf(u.id, u.last_seen_at))
}

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function presenceClass(u) {
  return {
    online: messenger.isOnline(u.id),
    offline: !messenger.isOnline(u.id),
  }
}

function companyOf(u) {
  if (u.company?.name) return u.company.name
  const c = companies.items.find(c => c.id === u.company_id)
  return c?.name || null
}

function fmtPhone(p) {
  if (!p || !p.startsWith('+7') || p.length !== 12) return p
  const d = p.slice(2)
  return `+7 (${d.slice(0, 3)}) ${d.slice(3, 6)}-${d.slice(6, 8)}-${d.slice(8, 10)}`
}

async function writeTo(u) {
  close()
  const cid = await messenger.openWith(u.id)
  router.push(`/messenger/${cid}`)
}

async function callTo(u, media) {
  close()
  try { await callStore.startCall({ userIds: [u.id], media }) }
  catch { /* ошибка обрабатывается в store */ }
}
</script>

<style scoped>
.emp-profile {
  display: flex;
  flex-direction: column;
  background: var(--color-surface);
  width: 100%;
  box-sizing: border-box;
  position: relative;
}
.profile-close {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 2;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: none;
  background: color-mix(in oklch, var(--color-surface) 60%, transparent);
  color: var(--color-text-dim);
  display: grid;
  place-items: center;
  cursor: pointer;
  -webkit-backdrop-filter: blur(8px);
  backdrop-filter: blur(8px);
  transition: background .12s, color .12s;
}
.profile-close:hover {
  background: var(--color-surface);
  color: var(--color-text);
}
.profile-close .material-symbols-outlined { font-size: 20px; }

/* Пастельная обложка, затухающая книзу — в стиле карточки в разделе аккаунта. */
.profile-cover {
  position: absolute;
  inset: 0 0 auto;
  height: 150px;
  background:
    radial-gradient(120% 140% at 85% 0%,
      color-mix(in oklch, var(--color-tertiary-container) 40%, transparent) 0%,
      transparent 60%),
    linear-gradient(120deg,
      color-mix(in oklch, var(--color-primary-container) 55%, var(--color-surface)),
      color-mix(in oklch, var(--color-secondary-container) 55%, var(--color-surface)));
  -webkit-mask-image: linear-gradient(to bottom, black 30%, transparent 100%);
  mask-image: linear-gradient(to bottom, black 30%, transparent 100%);
  pointer-events: none;
}

.profile-hero {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 36px 22px 22px;
  gap: 10px;
  color: var(--color-text);
}
.profile-avatar-btn {
  appearance: none;
  border: none;
  background: transparent;
  padding: 0;
  cursor: zoom-in;
  margin-bottom: 4px;
}
.profile-name {
  margin: 0;
  font-size: 22px;
  font-weight: 800;
  line-height: 1.2;
  letter-spacing: -0.01em;
  color: var(--color-text);
  word-break: break-word;
  overflow-wrap: anywhere;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  justify-content: center;
  flex-wrap: wrap;
}
.profile-tags {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 4px;
}
.profile-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 600;
  background: color-mix(in oklch, var(--color-text) 8%, transparent);
  color: var(--color-text-dim);
}
.profile-status .status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-outline-dim);
}
.profile-status.on {
  background: color-mix(in oklch, var(--color-success) 22%, transparent);
}
.profile-status.on .status-dot { background: var(--color-success); }

.profile-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 16px;
  background: var(--color-surface);
}
.profile-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: var(--radius-lg);
  text-decoration: none;
  color: var(--color-text);
  background: var(--color-surface-low);
  transition: background .12s;
}
.profile-row.link { cursor: pointer; }
.profile-row.link:hover { background: var(--color-surface-high); }
.row-ico {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.row-ico[data-tone="primary"] {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.row-ico[data-tone="secondary"] {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.row-ico[data-tone="tertiary"] {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.row-ico .material-symbols-outlined { font-size: 20px; }
.row-status-emoji { font-size: 19px; line-height: 1; }

.row-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.row-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-dim);
}
.row-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.row-chev {
  font-size: 18px;
  color: var(--color-text-dim);
  flex-shrink: 0;
}

.profile-lead {
  padding: 4px 16px 12px;
}
.profile-activity {
  width: 100%;
  justify-content: center;
}
.profile-activity .material-symbols-outlined { font-size: 19px; }

.profile-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 0 16px 16px;
}
.profile-actions > * {
  flex: 1 1 120px;
  justify-content: center;
}

/* ============ Кнопки ============ */
.btn-filled, .btn-tonal {
  appearance: none;
  border: none;
  cursor: pointer;
  border-radius: var(--radius-full);
  padding: 10px 18px;
  font: inherit;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: background .12s, color .12s, border-color .12s, box-shadow .12s, transform .12s;
}
.btn-filled {
  background: var(--color-primary);
  color: var(--color-on-primary);
  box-shadow: var(--shadow-sm);
}
.btn-filled:hover { background: var(--color-primary-hover); }
.btn-filled .material-symbols-outlined { font-size: 18px; }
.btn-tonal {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.btn-tonal.tertiary {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.btn-tonal:hover { filter: brightness(.96); }
.btn-tonal .material-symbols-outlined { font-size: 18px; }

/* ============ Аватар с presence-ring ============ */
.avatar {
  position: relative;
  display: inline-grid;
  place-items: center;
  flex-shrink: 0;
  border-radius: 50%;
  isolation: isolate;
}
.avatar img {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  object-fit: cover;
  display: block;
}
.avatar-xl { width: 116px; height: 116px; }

.avatar::before {
  content: '';
  position: absolute;
  inset: -5px;
  border-radius: 50%;
  border: 4px solid var(--color-outline-dim);
  z-index: -1;
  transition: border-color .18s, box-shadow .18s;
}
.avatar.online::before {
  border-color: var(--color-success);
  box-shadow: 0 0 0 2px color-mix(in oklch, var(--color-success) 22%, transparent);
}

.root-badge {
  display: inline-grid;
  place-items: center;
  color: var(--color-tertiary);
  flex-shrink: 0;
}
.root-badge.inline {
  width: 22px;
  height: 22px;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  border-radius: 50%;
  margin-left: 4px;
}
.root-badge.inline .material-symbols-outlined { font-size: 14px; font-variation-settings: 'FILL' 1; }

@media (max-width: 768px) {
  .hide-narrow { display: none; }
  .profile-list { padding: 12px; }
  .profile-actions { padding-left: 12px; padding-right: 12px; }
}
</style>

<style>
/* Профиль, открытый из плавающего мини-мессенджера (z-index 10050), должен
   лечь выше его панели. */
.p-dialog-mask.emp-mask--elevated {
  z-index: 10060 !important;
}
</style>
