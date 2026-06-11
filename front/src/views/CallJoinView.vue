<template>
  <div class="join-page">
    <!-- Лендинг-карточка (пока не вошли в звонок) -->
    <div v-if="callStore.phase === 'idle'" class="join-card">
      <img src="/logo.svg" alt="Groove Work" class="join-logo" />

      <template v-if="loading">
        <div class="join-spin">
          <span class="material-symbols-outlined spin">progress_activity</span>
        </div>
        <p class="join-sub">Проверяем приглашение…</p>
      </template>

      <template v-else-if="ended">
        <div class="join-icon done">
          <span class="material-symbols-outlined">call_end</span>
        </div>
        <h1 class="join-title">Звонок завершён</h1>
        <p class="join-sub">Спасибо за участие! Окно можно закрыть.</p>
      </template>

      <template v-else-if="fatalError">
        <div class="join-icon error">
          <span class="material-symbols-outlined">link_off</span>
        </div>
        <h1 class="join-title">{{ fatalError }}</h1>
        <p class="join-sub">Проверьте ссылку или попросите организатора прислать новую.</p>
      </template>

      <template v-else-if="info">
        <div class="join-icon">
          <span class="material-symbols-outlined">{{ info.media === 'audio' ? 'call' : 'videocam' }}</span>
        </div>
        <h1 class="join-title">
          {{ info.initiator_fio || 'Коллега' }} приглашает в {{ info.media === 'audio' ? 'аудиозвонок' : 'видеозвонок' }}
        </h1>
        <p class="join-sub">
          Сейчас в звонке: {{ info.occupants }}
          <template v-if="info.occupants >= info.max_participants"> — свободных мест нет</template>
        </p>

        <!-- Авторизованный пользователь входит под собой -->
        <template v-if="authStore.user">
          <button class="join-btn" :disabled="joining || isFull" @click="join()">
            <span class="material-symbols-outlined">{{ joining ? 'progress_activity' : 'login' }}</span>
            {{ joining ? 'Подключаемся…' : `Войти как ${authStore.user.fio}` }}
          </button>
        </template>

        <!-- Гость представляется -->
        <template v-else>
          <input
            v-model="guestName"
            class="join-input"
            type="text"
            placeholder="Ваше имя"
            maxlength="64"
            @keydown.enter="join()"
          />
          <button class="join-btn" :disabled="joining || isFull || !guestName.trim()" @click="join()">
            <span class="material-symbols-outlined">{{ joining ? 'progress_activity' : 'login' }}</span>
            {{ joining ? 'Подключаемся…' : 'Войти в звонок' }}
          </button>
          <p class="join-hint">Браузер попросит доступ к микрофону<template v-if="info.media !== 'audio'"> и камере</template>.</p>
        </template>

        <p v-if="joinError" class="join-error">{{ joinError }}</p>
      </template>
    </div>

    <!-- Сам звонок (CallView — fullscreen поверх) -->
    <CallView />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useCallStore } from '@/stores/call.js'
import { useAuthStore } from '@/stores/auth.js'
import { getJoinInfo } from '@/api/calls.js'
import CallView from '@/components/call/CallView.vue'

const route = useRoute()
const callStore = useCallStore()
const authStore = useAuthStore()

const code = String(route.params.code || '')
const loading = ref(true)
const info = ref(null)
const fatalError = ref(null)
const joinError = ref(null)
const joining = ref(false)
const ended = ref(false)
const guestName = ref(localStorage.getItem('gw2_guest_name') || '')
// Мест нет — не пускаем до клика, чтобы гость не ловил CALL_FULL постфактум.
const isFull = computed(() => !!info.value && info.value.occupants >= info.value.max_participants)

onMounted(async () => {
  try {
    const data = await getJoinInfo(code)
    if (!data.live) {
      ended.value = true
    } else {
      info.value = data
    }
  } catch (e) {
    fatalError.value = e?.status === 404
      ? 'Звонок не найден или уже завершён'
      : (e?.message || 'Не удалось загрузить приглашение')
  } finally {
    loading.value = false
  }
})

async function join() {
  if (joining.value) return
  const name = guestName.value.trim()
  if (!authStore.user && !name) return
  joinError.value = null
  joining.value = true
  try {
    if (name) {
      try { localStorage.setItem('gw2_guest_name', name) } catch {}
    }
    await callStore.joinAsGuest({ code, name: authStore.user ? null : name })
    wasInCall.value = true
  } catch (e) {
    joinError.value = e?.message || 'Не удалось войти в звонок'
    if (e?.code === 'CALL_NOT_FOUND') {
      ended.value = true
      info.value = null
    }
  } finally {
    joining.value = false
  }
}

/* Когда звонок завершился (комнату закрыли или мы вышли) — показываем
   прощальный экран вместо формы входа. */
const wasInCall = ref(false)
watch(() => callStore.phase, (phase, prev) => {
  if (prev === 'active' && phase === 'idle' && wasInCall.value) {
    ended.value = true
    info.value = null
  }
})
</script>

<style scoped>
.join-page {
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg);
  padding: 24px;
}

.join-card {
  width: min(440px, 100%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  padding: 36px 32px;
  border-radius: 28px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  box-shadow: var(--shadow-lg, 0 12px 36px color-mix(in oklch, var(--color-scrim) 20%, transparent));
  text-align: center;
}

.join-logo { height: 36px; margin-bottom: 6px; }

.join-icon {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: grid;
  place-items: center;
}

.join-icon .material-symbols-outlined { font-size: 36px; }

.join-icon.error {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

.join-icon.done {
  background: var(--color-surface-high);
  color: var(--color-text-dim);
}

.join-title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text);
  line-height: 1.3;
}

.join-sub {
  margin: 0;
  font-size: 14px;
  color: var(--color-text-dim);
}

.join-spin { color: var(--color-primary); }

.spin {
  display: inline-block;
  animation: joinSpin 1.2s linear infinite;
  font-size: 32px;
}

@keyframes joinSpin {
  from { transform: rotate(0); }
  to { transform: rotate(360deg); }
}

.join-input {
  width: 100%;
  padding: 13px 18px;
  border: 1px solid var(--color-outline-dim);
  border-radius: 999px;
  background: var(--color-surface-high);
  color: var(--color-text);
  font-size: 15px;
  font-family: inherit;
  text-align: center;
  outline: none;
}

.join-input:focus { border-color: var(--color-primary); }

.join-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 13px 20px;
  border: 0;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 15px;
  font-weight: 600;
  font-family: inherit;
  cursor: pointer;
  transition: filter 0.15s;
}

.join-btn:hover:not(:disabled) { filter: brightness(1.07); }
.join-btn:disabled { opacity: 0.6; cursor: default; }
.join-btn .material-symbols-outlined { font-size: 20px; }

.join-hint {
  margin: 0;
  font-size: 12px;
  color: var(--color-text-dim);
}

.join-error {
  margin: 0;
  padding: 8px 14px;
  border-radius: 999px;
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  font-size: 13px;
  font-weight: 600;
}
</style>
