<template>
  <div class="cpanel">
    <header class="cpanel-head">
      <span class="cpanel-title">Участники</span>
      <span class="cpanel-count">{{ callStore.participantCount }}</span>
      <button class="cpanel-close" title="Закрыть" @click="callStore.sidePanel = null">
        <span class="material-symbols-outlined">close</span>
      </button>
    </header>

    <div class="cpanel-body">
      <div class="p-row">
        <div class="p-avatar">
          <img v-if="myAvatar" :src="myAvatar" alt="" />
          <span v-else class="material-symbols-outlined">person</span>
        </div>
        <div class="p-info">
          <div class="p-name">{{ myName }} <span class="p-you">(вы)</span></div>
          <div v-if="isInitiator(myUserId)" class="p-sub">организатор</div>
        </div>
        <span class="material-symbols-outlined p-state" :class="{ off: !callStore.audioEnabled }">
          {{ callStore.audioEnabled ? 'mic' : 'mic_off' }}
        </span>
        <span class="material-symbols-outlined p-state" :class="{ off: !callStore.videoEnabled }">
          {{ callStore.videoEnabled ? 'videocam' : 'videocam_off' }}
        </span>
      </div>

      <div
        v-for="p in callStore.participantList"
        :key="p.identity"
        class="p-row"
        :class="{ speaking: p.speaking, pending: p.pending }"
      >
        <div class="p-avatar">
          <img v-if="avatarOf(p)" :src="avatarOf(p)" alt="" />
          <span v-else class="material-symbols-outlined">person</span>
        </div>
        <div class="p-info">
          <div class="p-name">{{ p.name }}</div>
          <div v-if="p.pending" class="p-sub">ждём ответа…</div>
          <div v-else-if="p.guest" class="p-sub guest">гость по ссылке</div>
          <div v-else-if="isInitiator(p.userId)" class="p-sub">организатор</div>
        </div>
        <template v-if="!p.pending">
          <span class="material-symbols-outlined p-state" :class="{ off: !p.audio }">
            {{ p.audio ? 'mic' : 'mic_off' }}
          </span>
          <span class="material-symbols-outlined p-state" :class="{ off: !p.video }">
            {{ p.video ? 'videocam' : 'videocam_off' }}
          </span>
        </template>
      </div>
    </div>

    <footer v-if="!callStore.guest" class="cpanel-foot">
      <button class="foot-btn" @click="$emit('invite')">
        <span class="material-symbols-outlined">person_add</span>
        Пригласить
      </button>
      <button class="foot-btn tonal" @click="$emit('copy-link')">
        <span class="material-symbols-outlined">link</span>
        Ссылка
      </button>
    </footer>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useCallStore } from '@/stores/call.js'
import { useAuthStore } from '@/stores/auth.js'

defineEmits(['invite', 'copy-link'])

const callStore = useCallStore()
const authStore = useAuthStore()

const myUserId = computed(() => callStore.guest ? null : authStore.user?.id)
const myName = computed(() => callStore.guest
  ? (callStore.guestName || 'Вы')
  : (authStore.user?.fio || 'Вы'))

const myAvatar = computed(() => {
  if (callStore.guest) return null
  const u = authStore.user
  if (!u) return null
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
})

function isInitiator(userId) {
  return userId != null && callStore.call?.initiator_id === userId
}

function avatarOf(p) {
  if (p.avatarPath) return `/uploads/${p.avatarPath}`
  if (p.userId) return `/api/users/${p.userId}/identicon`
  return null
}
</script>

<style scoped>
.cpanel {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.cpanel-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--acrylic-border);
  flex-shrink: 0;
}

.cpanel-title { font-weight: 700; font-size: 15px; }

.cpanel-count {
  padding: 1px 10px;
  border-radius: 999px;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 12px;
  font-weight: 700;
}

.cpanel-close {
  margin-left: auto;
  width: 32px;
  height: 32px; min-height: 0;
  border-radius: 50%;
  border: 0;
  background: transparent;
  color: var(--color-text);
  display: grid;
  place-items: center;
  cursor: pointer;
}

.cpanel-close:hover { background: var(--glass-hover-bg); }
.cpanel-close .material-symbols-outlined { font-size: 18px; }

.cpanel-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.p-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 14px;
}

.p-row.speaking { background: color-mix(in oklch, var(--color-primary-container) 55%, transparent); }
.p-row.pending { opacity: 0.65; }

.p-avatar {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: grid;
  place-items: center;
  overflow: hidden;
  flex-shrink: 0;
}

.p-avatar img { width: 100%; height: 100%; object-fit: cover; }
.p-avatar .material-symbols-outlined { font-size: 22px; }

.p-info { flex: 1; min-width: 0; }

.p-name {
  font-size: 14px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.p-you { color: var(--color-text-dim); font-weight: 500; }

.p-sub { font-size: 12px; color: var(--color-text-dim); }
.p-sub.guest { color: var(--color-tertiary); font-weight: 600; }

.p-state {
  font-size: 18px;
  color: var(--color-text-dim);
  flex-shrink: 0;
}

.p-state.off { color: var(--color-error); }

.cpanel-foot {
  display: flex;
  gap: 8px;
  padding: 12px 16px calc(12px + env(safe-area-inset-bottom, 0px));
  border-top: 1px solid var(--acrylic-border);
  flex-shrink: 0;
}

.foot-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 12px;
  border: 0;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  font-family: inherit;
}

.foot-btn.tonal {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}

.foot-btn .material-symbols-outlined { font-size: 18px; }
.foot-btn:hover { filter: brightness(1.05); }
</style>
