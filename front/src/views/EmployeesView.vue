<template>
  <div class="employees-view">
    <header class="employees-header">
      <h1>Сотрудники</h1>
      <div class="employees-search">
        <span class="material-symbols-outlined">search</span>
        <InputText
          v-model="search"
          placeholder="Поиск по логину или фамилии"
          class="search-input"
        />
      </div>
    </header>

    <div v-if="loading" class="employees-empty">
      <ProgressSpinner />
    </div>
    <div v-else-if="!filtered.length" class="employees-empty">
      <span class="material-symbols-outlined">person_off</span>
      <p>Никого не нашли</p>
    </div>
    <div v-else class="employees-grid">
      <button
        v-for="user in filtered"
        :key="user.id"
        class="employee-card"
        @click="openProfile(user)"
      >
        <div class="employee-avatar-wrap">
          <img class="employee-avatar" :src="avatarOf(user)" :alt="user.fio" />
          <span v-if="messenger.isOnline(user.id)" class="online-dot" title="В сети"></span>
        </div>
        <div class="employee-name">{{ user.fio }}</div>
        <div class="employee-post">{{ user.post || '—' }}</div>
        <div class="employee-role">{{ user.role?.name }}</div>
        <div class="employee-status" :class="{ 'is-online': messenger.isOnline(user.id) }">
          {{ statusOf(user) }}
        </div>
      </button>
    </div>

    <Dialog
      v-model:visible="profileOpen"
      modal
      :draggable="false"
      :show-header="false"
      :pt="{ root: { class: 'employee-dialog' } }"
    >
      <div v-if="selected" class="employee-profile">
        <div class="profile-avatar-wrap">
          <img class="profile-avatar" :src="avatarOf(selected)" :alt="selected.fio" />
          <span v-if="messenger.isOnline(selected.id)" class="online-dot profile-dot" title="В сети"></span>
        </div>
        <h2 class="profile-name">{{ selected.fio }}</h2>
        <div class="profile-status" :class="{ 'is-online': messenger.isOnline(selected.id) }">
          {{ statusOf(selected) }}
        </div>
        <div class="profile-post">{{ selected.post || '—' }}</div>
        <div class="profile-role">{{ selected.role?.name }}</div>
        <div class="profile-login">@{{ selected.login }}</div>
        <div class="profile-actions">
          <button
            v-if="selected.id !== authStore.user?.id"
            class="btn-primary profile-write"
            @click="writeTo(selected)"
          >
            <span class="material-symbols-outlined">chat</span>
            Написать
          </button>
          <button class="btn-secondary" @click="profileOpen = false">Закрыть</button>
        </div>
      </div>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getDirectory } from '@/api/users.js'
import { useAuthStore } from '@/stores/auth.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { formatLastSeen } from '@/utils/presence.js'
import InputText from 'primevue/inputtext'
import Dialog from 'primevue/dialog'
import ProgressSpinner from 'primevue/progressspinner'

const router = useRouter()
const authStore = useAuthStore()
const messenger = useMessengerStore()

const users = ref([])
const loading = ref(false)
const search = ref('')
const profileOpen = ref(false)
const selected = ref(null)

async function load() {
  loading.value = true
  try {
    users.value = await getDirectory()
  } finally {
    loading.value = false
  }
}

// Статус: «в сети» для онлайн, иначе точное время последнего захода.
// last_seen из стора (живой, приходит по сокету) приоритетнее, чем из каталога.
function statusOf(user) {
  if (messenger.isOnline(user.id)) return 'в сети'
  return formatLastSeen(messenger.lastSeenOf(user.id, user.last_seen_at))
}

onMounted(() => {
  load()
  // Свежий снимок онлайн-статусов (на случай если сокет ещё не присылал presence).
  messenger.fetchPresence()
})

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return users.value
  return users.value.filter(u =>
    u.fio.toLowerCase().includes(q) || u.login.toLowerCase().includes(q)
  )
})

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function openProfile(user) {
  selected.value = user
  profileOpen.value = true
}

async function writeTo(user) {
  profileOpen.value = false
  const convId = await messenger.openWith(user.id)
  router.push(`/messenger/${convId}`)
}

watch(profileOpen, (open) => { if (!open) selected.value = null })
</script>

<style scoped>
.employees-view {
  padding: 24px;
  max-width: 1280px;
  margin: 0 auto;
}

.employees-header {
  display: flex;
  align-items: center;
  gap: 24px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}

.employees-header h1 {
  font-size: 22px;
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.employees-search {
  flex: 1;
  min-width: 240px;
  position: relative;
  display: flex;
  align-items: center;
}

.employees-search .material-symbols-outlined {
  position: absolute;
  left: 12px;
  color: var(--color-text-dim);
  font-size: 20px;
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding-left: 40px !important;
  border-radius: var(--radius-md);
}

.employees-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
}

.employee-card {
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  padding: 20px 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, transform 0.15s, box-shadow 0.15s;
  color: var(--color-text);
}

.employee-card:hover {
  border-color: var(--color-primary);
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.employee-avatar-wrap {
  position: relative;
  margin-bottom: 12px;
}

.employee-avatar {
  width: 88px;
  height: 88px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--color-outline-dim);
  display: block;
}

.online-dot {
  position: absolute;
  right: 4px;
  bottom: 4px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--color-success);
  border: 3px solid var(--color-surface);
}

.employee-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text);
  margin-bottom: 2px;
}

.employee-post {
  font-size: 13px;
  color: var(--color-text-dim);
  margin-bottom: 6px;
}

.employee-role {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.4px;
  color: var(--color-primary);
}

.employee-status {
  margin-top: 6px;
  font-size: 11px;
  color: var(--color-text-dim);
}

.employee-status.is-online {
  color: var(--color-success);
  font-weight: 600;
}

.employees-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 80px 16px;
  color: var(--color-text-dim);
}

.employees-empty .material-symbols-outlined {
  font-size: 56px;
}

.employee-profile {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 32px 24px 24px;
  min-width: 320px;
}

.profile-avatar-wrap {
  position: relative;
  margin-bottom: 16px;
}

.profile-avatar {
  width: 128px;
  height: 128px;
  border-radius: 50%;
  object-fit: cover;
  border: 3px solid var(--color-primary);
  display: block;
}

.profile-dot {
  right: 8px;
  bottom: 8px;
  width: 20px;
  height: 20px;
}

.profile-status {
  font-size: 13px;
  color: var(--color-text-dim);
  margin-bottom: 8px;
}

.profile-status.is-online {
  color: var(--color-success);
  font-weight: 600;
}

.profile-name {
  font-size: 20px;
  font-weight: 600;
  margin: 0 0 4px;
  color: var(--color-text);
}

.profile-post {
  font-size: 14px;
  color: var(--color-text-dim);
  margin-bottom: 6px;
}

.profile-role {
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.4px;
  color: var(--color-primary);
  margin-bottom: 6px;
}

.profile-login {
  font-size: 13px;
  color: var(--color-text-dim);
  margin-bottom: 20px;
}

.profile-actions {
  display: flex;
  gap: 12px;
  width: 100%;
  justify-content: center;
}

.profile-write {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.btn-primary {
  background: var(--color-primary);
  color: var(--color-on-primary);
  border: none;
  padding: 10px 18px;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-primary:hover { background: var(--color-primary-hover); }

.btn-secondary {
  background: transparent;
  color: var(--color-text);
  border: 1px solid var(--color-outline);
  padding: 10px 18px;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.btn-secondary:hover {
  background: var(--color-surface-low);
}

@media (max-width: 768px) {
  .employees-view {
    padding: 16px 12px 80px;
  }
  .employees-grid {
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 12px;
  }
  .employee-avatar {
    width: 72px;
    height: 72px;
  }
  .employee-profile {
    min-width: unset;
    padding: 24px 16px 16px;
  }
}
</style>
