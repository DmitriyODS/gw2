<template>
  <AuthShell title="создание аккаунта" size="lg" back="/welcome">
    <form class="rg" @submit.prevent="handleRegister">
      <div class="rg-main">
        <!-- Фото профиля: обрезается сразу, а уходит на сервер после
             подтверждения почты (до неё сессии нет). -->
        <div class="rg-photo">
          <button type="button" class="rg-photo-tile" @click="cropping = true">
            <img v-if="avatarPreview" :src="avatarPreview" alt="" class="rg-photo-img" />
            <template v-else>
              <span class="material-symbols-outlined">person</span>
              <span class="rg-photo-hint">нажмите, чтобы выбрать фото</span>
            </template>
          </button>
          <button v-if="avatarPreview" type="button" class="rg-photo-btn ghost" @click="dropAvatar">
            убрать фото
          </button>
        </div>

        <div class="rg-fields">
          <AuthField
            v-model="form.fio"
            label="фио"
            placeholder="Фамилия Имя Отчество"
            autocomplete="name"
            :disabled="loading"
            @update:modelValue="onFioInput"
          />
          <AuthField
            v-model="form.login"
            label="логин"
            placeholder="подставим из ФИО"
            autocomplete="username"
            :disabled="loading"
            @update:modelValue="loginTouched = true"
          />
          <AuthField
            v-model="form.email"
            label="email"
            type="email"
            placeholder="name@example.com"
            autocomplete="email"
            :disabled="loading"
          />
          <AuthField
            v-model="form.password"
            label="пароль"
            type="password"
            placeholder="не короче 8 символов"
            autocomplete="new-password"
            :disabled="loading"
            hint="Сохраните пароль — он понадобится для входа."
          >
            <template #tools>
              <button type="button" class="af-tool" title="Сгенерировать новый" tabindex="-1" @click="regeneratePassword">
                <span class="material-symbols-outlined">autorenew</span>
              </button>
              <button
                type="button"
                class="af-tool"
                :title="copied ? 'Скопировано' : 'Скопировать'"
                tabindex="-1"
                @click="copyPassword"
              >
                <span class="material-symbols-outlined">{{ copied ? 'check' : 'content_copy' }}</span>
              </button>
            </template>
          </AuthField>
        </div>
      </div>

      <!-- Оформление выбирается сразу, как в первоначальной настройке системы. -->
      <AuthThemeTiles class="rg-themes" />

      <p v-if="error" class="auth-error">{{ error }}</p>

      <div class="rg-alt">
        <button type="button" class="auth-alt" @click="goYandex">
          <YandexLogo :size="16" />
          Войти через Яндекс
        </button>
        <RouterLink to="/login" class="rg-switch">уже есть аккаунт</RouterLink>
      </div>
    </form>

    <template #actions>
      <button type="button" class="rg-submit" :disabled="loading" @click="handleRegister">
        {{ loading ? 'создаём…' : 'создать' }}
      </button>
    </template>

    <template #overlays>
      <AppDialog
        v-if="cropping"
        model-value
        size="md"
        title="Фото профиля"
        subtitle="Выберите снимок и обрежьте его под аватар."
        @update:modelValue="cropping = false"
      >
        <AvatarCropper @cropped="onCropped" @cancel="cropping = false" />
      </AppDialog>
    </template>
  </AuthShell>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { suggestLogin, yandexConfig, yandexAuthURL } from '@/api/auth.js'
import { inAppShell } from '@/utils/appShell.js'
import { savePendingAvatar, clearPendingAvatar } from '@/utils/pendingAvatar.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import AuthField from '@/components/auth/AuthField.vue'
import AuthThemeTiles from '@/components/auth/AuthThemeTiles.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AvatarCropper from '@/components/settings/AvatarCropper.vue'
import YandexLogo from '@/components/common/YandexLogo.vue'

const router = useRouter()
const authStore = useAuthStore()

const form = reactive({ fio: '', email: '', login: '', password: '' })
const error = ref('')
const loading = ref(false)
const copied = ref(false)
const loginTouched = ref(false)
const cropping = ref(false)
const avatarPreview = ref('')
let suggestTimer = null

// Регистрация через Яндекс ID. Кнопка на экране есть всегда; о ненастроенном
// на сервере входе честно сообщаем по клику, а не прячем способ регистрации.
const yandexAuth = ref({ enabled: false, client_id: '' })
function goYandex() {
  if (!yandexAuth.value.enabled) {
    error.value = 'Вход через Яндекс на этом сервере не настроен'
    return
  }
  window.location.href = yandexAuthURL(yandexAuth.value.client_id, inAppShell() ? 'app' : '')
}

onMounted(() => {
  regeneratePassword()
  clearPendingAvatar()
  yandexConfig().then((cfg) => { yandexAuth.value = cfg }).catch(() => {})
})

// Безопасный пароль на клиенте (Web Crypto): без двусмысленных символов.
function generatePassword(len = 12) {
  const alphabet = 'abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789'
  const arr = new Uint32Array(len)
  crypto.getRandomValues(arr)
  let out = ''
  for (let i = 0; i < len; i++) out += alphabet[arr[i] % alphabet.length]
  return out
}

function regeneratePassword() {
  form.password = generatePassword()
}

async function copyPassword() {
  try {
    await navigator.clipboard.writeText(form.password)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1500)
  } catch { /* clipboard недоступен */ }
}

function onCropped(blob) {
  const reader = new FileReader()
  reader.onload = () => {
    avatarPreview.value = String(reader.result || '')
    savePendingAvatar(avatarPreview.value)
    cropping.value = false
  }
  reader.readAsDataURL(blob)
}

function dropAvatar() {
  avatarPreview.value = ''
  clearPendingAvatar()
}

// Live-подсказка логина по ФИО (debounce), пока пользователь не правил поле сам.
function onFioInput() {
  if (loginTouched.value) return
  clearTimeout(suggestTimer)
  const fio = form.fio
  suggestTimer = setTimeout(async () => {
    if (loginTouched.value || !fio.trim()) return
    try {
      const { login } = await suggestLogin(fio)
      if (!loginTouched.value && login) form.login = login
    } catch { /* подсказка необязательна */ }
  }, 400)
}

async function handleRegister() {
  if (loading.value) return
  error.value = ''
  // Модификатор .trim у v-model компонента не работает без modelModifiers —
  // подрезаем поля здесь, в одном месте.
  form.fio = form.fio.trim()
  form.login = form.login.trim()
  form.email = form.email.trim()
  if (!form.fio) { error.value = 'Укажите ФИО'; return }
  if (!form.email || !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(form.email)) {
    error.value = 'Укажите корректный email'
    return
  }
  if (form.login && form.login.length < 3) {
    error.value = 'Логин должен содержать не менее 3 символов'
    return
  }
  if (form.password.length < 8) {
    error.value = 'Пароль должен содержать не менее 8 символов'
    return
  }
  loading.value = true
  try {
    const { email } = await authStore.register({
      fio: form.fio, email: form.email, login: form.login, password: form.password,
    })
    router.push({ path: '/verify-email', query: { email: email || form.email } })
  } catch (e) {
    error.value = e?.message || 'Не удалось зарегистрироваться'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.rg {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.rg-main {
  display: grid;
  grid-template-columns: 176px minmax(0, 1fr);
  gap: 22px;
  align-items: start;
}

/* ── Фото ─────────────────────────────────────────────────────── */
.rg-photo {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rg-photo-tile {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  aspect-ratio: 1 / 1;
  padding: 12px;
  border: 1px dashed color-mix(in oklch, var(--color-outline) 55%, transparent);
  border-radius: var(--radius-lg);
  background: color-mix(in oklch, var(--color-surface) 55%, transparent);
  color: var(--color-text-dim);
  font: inherit;
  cursor: pointer;
  overflow: hidden;
  transition: border-color 0.15s, background 0.15s;
}

.rg-photo-tile:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.rg-photo-tile .material-symbols-outlined { font-size: 44px; }

.rg-photo-hint {
  font-size: 11.5px;
  line-height: 1.35;
  text-align: center;
}

.rg-photo-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: var(--radius-md);
}

.rg-photo-btn {
  height: 34px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg), color-mix(in oklch, var(--color-surface) 45%, transparent);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit;
  font-size: 12.5px;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s;
}

.rg-photo-btn:hover { color: var(--color-primary); border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border)); }
.rg-photo-btn.ghost { background: none; box-shadow: none; border-color: transparent; color: var(--color-text-dim); }

/* ── Поля ─────────────────────────────────────────────────────── */
.rg-fields {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px 18px;
  align-items: start;
}

.rg-themes {
  padding-top: 4px;
  border-top: 1px solid color-mix(in oklch, var(--acrylic-border) 70%, transparent);
}

.rg-alt {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14px;
  flex-wrap: wrap;
}

.rg-switch {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-dim);
  text-decoration: none;
}

.rg-switch:hover { color: var(--color-primary); }

.rg-submit {
  height: 40px;
  padding: 0 28px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--grad-primary);
  color: var(--color-on-primary);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: var(--shadow-sm);
  transition: filter 0.15s, box-shadow 0.15s;
}

.rg-submit:hover:not(:disabled) { filter: brightness(1.06); box-shadow: var(--shadow-md); }
.rg-submit:disabled { opacity: 0.55; cursor: not-allowed; }

/* Раскладка считается от ширины ОКНА карточки, а не экрана. */
@media (max-width: 760px) {
  .rg-main { grid-template-columns: 1fr; }
  .rg-photo { flex-direction: row; align-items: center; }
  .rg-photo-tile { width: 92px; aspect-ratio: 1 / 1; flex-shrink: 0; }
  .rg-photo-tile .material-symbols-outlined { font-size: 30px; }
  .rg-photo-hint { display: none; }
  .rg-fields { grid-template-columns: 1fr; }
}
</style>
