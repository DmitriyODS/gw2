<template>
  <!-- Запертый экран поверх всего приложения. Сессия при этом жива: окна,
       черновики и позиция в разделах остаются на месте — блокировка закрывает
       вид, а не выходит из аккаунта. -->
  <Teleport to="body">
    <div class="lock" role="dialog" aria-modal="true" aria-label="Экран заблокирован">
      <!-- Обои запертого экрана — тот же рецепт фона, что у рабочего стола и
           чатов; не выбраны — остаётся фирменная волна экранов входа. -->
      <ChatBackgroundLayer v-if="wallpaper" :recipe="wallpaper" class="lock-bg" />
      <AuthWave v-else class="lock-wave" />

      <div class="lock-card">
        <BrandWordmark class="lock-brand" />

        <img :src="avatar" class="lock-avatar" :alt="user?.fio">
        <h1 class="lock-name">{{ user?.fio || 'Заблокировано' }}</h1>
        <p class="lock-hint">Введите пин-код, чтобы продолжить</p>

        <form class="lock-form" @submit.prevent="submit">
          <input
            ref="inputEl"
            v-model="secret"
            class="lock-input"
            :type="showSecret ? 'text' : 'password'"
            inputmode="numeric"
            autocomplete="off"
            :placeholder="usePassword ? 'Пароль от аккаунта' : 'Пин-код'"
            :disabled="busy"
            @input="error = ''"
          >
          <AppButton
            variant="icon"
            :icon="showSecret ? 'visibility_off' : 'visibility'"
            :aria-label="showSecret ? 'Скрыть' : 'Показать'"
            @click="showSecret = !showSecret"
          />
          <AppButton
            type="submit"
            variant="filled"
            icon="lock_open"
            label="Войти"
            :loading="busy"
            :disabled="!secret"
          />
        </form>

        <p v-if="error" class="lock-error">{{ error }}</p>

        <div class="lock-actions">
          <AppButton
            variant="text"
            :label="usePassword ? 'Ввести пин-код' : 'Забыли пин-код?'"
            @click="usePassword = !usePassword"
          />
          <AppButton variant="text" tone="danger" label="Выйти из аккаунта" @click="logout" />
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'
import AuthWave from '@/components/auth/AuthWave.vue'
import BrandWordmark from '@/components/common/BrandWordmark.vue'
import AppButton from '@/components/ui/AppButton.vue'
import { useAuthStore } from '@/stores/auth.js'
import ChatBackgroundLayer from '@/components/common/ChatBackgroundLayer.vue'
import { useScreenLock } from '@/composables/useScreenLock.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { avatarUrl } from '@/utils/pets.js'

const auth = useAuthStore()
const lock = useScreenLock()

const secret = ref('')
const showSecret = ref(false)
// Пин забывается чаще пароля, поэтому пароль от аккаунта работает всегда —
// иначе единственным выходом был бы выход из системы.
const usePassword = ref(false)
const busy = ref(false)
const error = ref('')
const inputEl = ref(null)

const prefs = useDesktopPrefsStore()
const wallpaper = computed(() => prefs.lockWallpaper)
const user = computed(() => auth.user)
const avatar = computed(() => avatarUrl(auth.user) || '')

async function submit() {
  if (!secret.value || busy.value) return
  busy.value = true
  error.value = ''
  try {
    await lock.unlock(secret.value)
    secret.value = ''
  } catch (e) {
    error.value = e?.message || 'Не подошло — попробуйте ещё раз'
    secret.value = ''
    await nextTick()
    inputEl.value?.focus()
  } finally {
    busy.value = false
  }
}

async function logout() {
  lock.reset()
  await auth.logout()
}

onMounted(() => inputEl.value?.focus())
</script>

<style scoped>
.lock {
  position: fixed;
  inset: 0;
  /* Выше всего, включая тосты и плавающего питомца: за запертым экраном
     приложения быть не должно. */
  z-index: 20000;
  display: grid;
  place-items: center;
  background: var(--color-surface);
  overflow: hidden;
}

.lock-wave,
.lock-bg {
  position: absolute;
  inset: 0;
}

.lock-card {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  width: min(420px, calc(100vw - 32px));
  padding: 28px 24px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  text-align: center;
}

.lock-brand { margin-bottom: 4px; }

.lock-avatar {
  width: 88px;
  height: 88px;
  border-radius: 50%;
  object-fit: cover;
}

.lock-name {
  margin: 0;
  font-size: 1.1rem;
}

.lock-hint {
  margin: 0;
  color: var(--color-text-dim);
  font-size: 0.9rem;
}

.lock-form {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  margin-top: 4px;
}

.lock-input {
  flex: 1;
  min-width: 0;
  padding: 12px 14px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--glass-bg), var(--color-surface-variant);
  color: var(--color-text);
  font-size: 1.05rem;
  letter-spacing: 0.15em;
  text-align: center;
}

.lock-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.lock-error {
  margin: 0;
  color: var(--color-error);
  font-size: 0.88rem;
}

.lock-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 4px;
}
</style>
