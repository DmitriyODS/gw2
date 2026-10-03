<template>
  <AuthShell
    title="Создание аккаунта"
    subtitle="Пароль мы уже придумали — его можно сменить или скопировать."
    size="lg"
    back="/welcome"
  >
    <form class="rg" @submit.prevent="handleRegister">
      <div class="rg-main">
        <!-- Фото профиля: обрезается сразу, а уходит на сервер после
             подтверждения почты (до неё сессии нет). -->
        <div class="rg-photo">
          <button type="button" class="rg-photo-tile" aria-label="Выбрать фото профиля" @click="cropping = true">
            <img v-if="avatarPreview" loading="lazy" decoding="async" :src="avatarPreview" alt="" class="rg-photo-img" />
            <template v-else>
              <span class="material-symbols-outlined">add_a_photo</span>
              <span class="rg-photo-hint">Фото профиля</span>
            </template>
          </button>
          <AppButton v-if="avatarPreview" variant="text" size="sm" label="Убрать фото" @click="dropAvatar" />
        </div>

        <div class="rg-fields">
          <AuthField
            v-model="form.fio"
            label="ФИО"
            placeholder="Фамилия Имя Отчество"
            autocomplete="name"
            :disabled="loading"
            @update:model-value="onFioInput"
          />
          <AuthField
            v-model="form.login"
            label="Логин"
            placeholder="Подставим из ФИО"
            autocomplete="username"
            :disabled="loading"
            @update:model-value="loginTouched = true"
          />
          <AuthField
            v-model="form.email"
            label="Email"
            type="email"
            placeholder="name@example.com"
            autocomplete="email"
            :disabled="loading"
          />
          <AuthField
            v-model="form.password"
            label="Пароль"
            type="password"
            placeholder="Не короче 8 символов"
            autocomplete="new-password"
            :disabled="loading"
            hint="Сохраните пароль — он понадобится для входа."
          >
            <template #tools>
              <AppButton
                variant="text"
                size="sm"
                icon="autorenew"
                tabindex="-1"
                aria-label="Сгенерировать новый"
                title="Сгенерировать новый"
                @click="regeneratePassword"
              />
              <AppButton
                variant="text"
                size="sm"
                tabindex="-1"
                :icon="copied ? 'check' : 'content_copy'"
                :aria-label="copied ? 'Скопировано' : 'Скопировать'"
                :title="copied ? 'Скопировано' : 'Скопировать'"
                @click="copyPassword"
              />
            </template>
          </AuthField>
        </div>
      </div>

      <!-- Оформление выбирается сразу, как в первоначальной настройке системы. -->
      <AuthThemeTiles class="rg-themes" />

      <AppInfoBar v-if="error" tone="error" inline :message="error" />

      <!-- Согласие берётся плашкой после подтверждения почты (там его нужно
           прочитать и подтвердить галочками), здесь — только доступ к текстам:
           закон требует, чтобы документы можно было прочитать ДО регистрации. -->
      <p v-if="LEGAL_CONSENT_VISIBLE" class="rg-legal">
        Продолжая, вы соглашаетесь принять
        <RouterLink to="/legal">правовые документы</RouterLink>
        при первом входе.
      </p>
    </form>

    <template #actions>
      <AppButton class="rg-yandex" @click="goYandex">
        <YandexLogo :size="16" />
        Через Яндекс ID
      </AppButton>
      <span class="rg-gap" />
      <AppButton tag="router-link" to="/login" variant="text" label="Уже есть аккаунт" />
      <AppButton variant="filled" :loading="loading" label="Создать аккаунт" @click="handleRegister" />
    </template>

    <template #overlays>
      <AppDialog
        v-if="cropping"
        model-value
        size="md"
        title="Фото профиля"
        subtitle="Выберите снимок и обрежьте его под аватар."
        @update:model-value="cropping = false"
      >
        <AvatarCropper @cropped="onCropped" @cancel="cropping = false" />
      </AppDialog>
    </template>
  </AuthShell>
</template>

<script setup>
import { reactive, ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useThemeStore } from '@/stores/theme.js'
import { suggestLogin, yandexConfig, yandexAuthURL } from '@/api/auth.js'
import { inAppShell } from '@/utils/appShell.js'
import { LEGAL_CONSENT_VISIBLE } from '@/utils/release.js'
import { savePendingAvatar, clearPendingAvatar } from '@/utils/pendingAvatar.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import AuthField from '@/components/auth/AuthField.vue'
import AuthThemeTiles from '@/components/auth/AuthThemeTiles.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AvatarCropper from '@/components/settings/AvatarCropper.vue'
import YandexLogo from '@/components/common/YandexLogo.vue'

const router = useRouter()
const authStore = useAuthStore()
const theme = useThemeStore()

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
  theme.startThemeTrial()
  yandexConfig().then((cfg) => { yandexAuth.value = cfg }).catch(() => {})
})

// Выбранное здесь оформление закрепляет только созданный аккаунт: уход с
// экрана любым способом (назад, «уже есть аккаунт», перезагрузка) откатывает
// примерку — на устройстве может сидеть другой человек со своей темой.
onUnmounted(() => { theme.cancelThemeTrial() })

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
    theme.commitThemeTrial()
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
  gap: 22px;
}

.rg-main {
  display: grid;
  grid-template-columns: 160px minmax(0, 1fr);
  gap: 24px;
  align-items: start;
}

/* ── Фото: углубление в листе, куда «кладут» снимок ───────────── */
.rg-photo {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
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
  border: 1px solid var(--sk-edge);
  border-radius: var(--radius-lg);
  background: var(--sk-well-bg);
  box-shadow: var(--sk-well-shadow);
  color: var(--color-text-dim);
  font: inherit;
  cursor: pointer;
  overflow: hidden;
  transition: color 0.15s, border-color 0.15s;
}

.rg-photo-tile:hover {
  border-color: color-mix(in oklch, var(--color-primary) 45%, var(--sk-edge));
  color: var(--color-primary);
}

.rg-photo-tile .material-symbols-outlined { font-size: 38px; }

.rg-photo-hint {
  font-size: 12.5px;
  font-weight: 600;
  text-align: center;
}

.rg-photo-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: var(--radius-md);
}

/* ── Поля ─────────────────────────────────────────────────────── */
.rg-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px 18px;
  align-items: start;
}

.rg-themes {
  padding-top: 20px;
  border-top: 1px solid var(--color-outline-dim);
}

.rg-legal {
  margin: 0;
  font-size: 12.5px;
  line-height: 1.45;
  color: var(--color-text-dim);
  text-align: center;
}

.rg-legal a {
  color: var(--color-primary);
  text-decoration: none;
}

.rg-gap { flex: 1 1 auto; }

.rg-yandex :deep(.btn-label) {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}

@media (max-width: 760px) {
  .rg-main { grid-template-columns: minmax(0, 1fr); }
  .rg-photo { flex-direction: row; justify-content: flex-start; gap: 12px; }
  .rg-photo-tile { width: 88px; flex-shrink: 0; }
  .rg-photo-tile .material-symbols-outlined { font-size: 28px; }
  .rg-photo-hint { display: none; }
  .rg-fields { grid-template-columns: minmax(0, 1fr); }
}

/* Узкий подвал: главное действие — первым и во всю ширину. */
@media (max-width: 560px) {
  .rg-gap { display: none; }
  :deep(.auth-foot) { flex-direction: column-reverse; align-items: stretch; }
}
</style>
