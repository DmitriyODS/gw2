<template>
  <div class="profile-view">
    <div class="profile-container">
      <!-- Левая колонка: карточка-идентичность (sticky на десктопе) -->
      <aside class="identity-card">
        <div class="identity-cover" aria-hidden="true"></div>

        <button
          type="button"
          class="avatar-wrapper"
          title="Открыть фото"
          @click="lightboxOpen = true"
        >
          <img :src="avatarSrc" class="profile-avatar" :alt="authStore.user?.fio" />
          <span class="avatar-zoom-overlay" aria-hidden="true">
            <span class="material-symbols-outlined">zoom_in</span>
          </span>
        </button>

        <div class="hero-info">
          <h1 class="hero-name">{{ authStore.user?.fio }}</h1>
          <p class="hero-post">{{ authStore.user?.post || 'Должность не указана' }}</p>
          <div class="hero-meta">
            <span v-if="authStore.user?.role?.name" class="role-tag">
              {{ authStore.user.role.name }}
            </span>
            <span v-if="authStore.user?.login" class="hero-login">@{{ authStore.user.login }}</span>
          </div>
        </div>

        <div class="avatar-actions">
          <button class="btn-sm" @click="showCropper = true">
            <span class="material-symbols-outlined">photo_camera</span>
            Загрузить
          </button>
          <button
            v-if="authStore.user?.avatar_path"
            class="btn-sm danger"
            @click="confirmAvatarDelete = true"
          >
            <span class="material-symbols-outlined">delete</span>
            Удалить
          </button>
        </div>

        <!-- Контакты — только на десктопе, мобильная шапка остаётся прежней -->
        <ul class="identity-contacts">
          <li class="contact-row">
            <span class="contact-ico" data-tone="primary">
              <span class="material-symbols-outlined">mail</span>
            </span>
            <span class="contact-text">
              <small>Email</small>
              <span :class="{ 'contact-empty': !authStore.user?.email }">
                {{ authStore.user?.email || 'Не указан' }}
              </span>
            </span>
          </li>
          <li class="contact-row">
            <span class="contact-ico" data-tone="secondary">
              <span class="material-symbols-outlined">call</span>
            </span>
            <span class="contact-text">
              <small>Телефон</small>
              <span :class="{ 'contact-empty': !authStore.user?.phone }">
                {{ authStore.user?.phone || 'Не указан' }}
              </span>
            </span>
          </li>
          <li v-if="authStore.companyName" class="contact-row">
            <span class="contact-ico" data-tone="tertiary">
              <span class="material-symbols-outlined">domain</span>
            </span>
            <span class="contact-text">
              <small>Компания</small>
              <span>{{ authStore.companyName }}</span>
            </span>
          </li>
        </ul>

        <button class="btn-logout" @click="authStore.logout()">
          <span class="material-symbols-outlined">logout</span>
          <span class="btn-logout-label">Выйти</span>
        </button>
      </aside>

      <!-- Правая колонка: статистика + формы -->
      <div class="profile-main">
        <!-- Личная статистика -->
        <section class="profile-card stats-card">
          <header class="card-head stats-head">
            <div class="head-icon" data-tone="primary">
              <span class="material-symbols-outlined">insights</span>
            </div>
            <div class="head-text">
              <h3>Личная статистика</h3>
              <p class="head-desc">Часы и задачи за выбранный период</p>
            </div>
            <DateRangePicker
              v-model="statsPeriod"
              class="head-period"
              @update:model-value="loadStats"
            />
          </header>

          <div v-if="statsLoading" class="loading-inline">
            <BrandLoader :size="64" />
          </div>

          <template v-else-if="profileStats">
            <div class="stat-tiles">
              <div class="stat-tile" data-tone="primary">
                <span class="tile-ico"><span class="material-symbols-outlined">schedule</span></span>
                <span class="tile-text">
                  <span class="tile-num">{{ roundHours(profileStats.total_hours) }}</span>
                  <span class="tile-label">Время</span>
                </span>
              </div>
              <div class="stat-tile" data-tone="secondary">
                <span class="tile-ico"><span class="material-symbols-outlined">task_alt</span></span>
                <span class="tile-text">
                  <span class="tile-num">{{ profileStats.tasks_count ?? 0 }}</span>
                  <span class="tile-label">Задач</span>
                </span>
              </div>
              <div class="stat-tile stat-tile--avg" data-tone="tertiary">
                <span class="tile-ico"><span class="material-symbols-outlined">avg_pace</span></span>
                <span class="tile-text">
                  <span class="tile-num">{{ roundHours(avgHoursPerDay) }}</span>
                  <span class="tile-label">В среднем за день</span>
                </span>
              </div>
            </div>

            <template v-if="profileStats.by_unit_types?.length">
              <!-- Десктоп: наглядные бары по типам; мобилка — прежняя таблица -->
              <div v-if="!isMobile" class="type-list">
                <div v-for="t in profileStats.by_unit_types" :key="t.name" class="type-row">
                  <div class="type-top">
                    <span class="type-name">{{ t.name }}</span>
                    <span class="type-meta">
                      {{ roundHours(t.hours) }} · {{ t.tasks_count }}
                      {{ plural(t.tasks_count, 'задача', 'задачи', 'задач') }}
                    </span>
                  </div>
                  <div class="type-bar">
                    <span :style="{ width: typeBarWidth(t) }"></span>
                  </div>
                </div>
              </div>

              <DataTable
                v-else
                :value="profileStats.by_unit_types"
                size="small"
                class="stats-table"
              >
                <Column field="name" header="Тип юнита" />
                <Column header="Время" style="width:120px">
                  <template #body="{ data }">{{ roundHours(data.hours) }}</template>
                </Column>
                <Column field="tasks_count" header="Задачи" style="width:100px" />
              </DataTable>
            </template>
          </template>

          <div v-else class="empty-stats">
            Нет данных за выбранный период
          </div>
        </section>

        <!-- Вход на другом устройстве: подтверждение QR-входа и ТВ-киоска -->
        <section class="profile-card">
          <header class="card-head">
            <div class="head-icon" data-tone="primary">
              <span class="material-symbols-outlined">devices</span>
            </div>
            <div class="head-text">
              <h3>Вход на другом устройстве</h3>
              <p class="head-desc">
                Подтвердите вход по QR на новом устройстве или авторизуйте ТВ-киоск
                под выбранной компанией.
              </p>
            </div>
          </header>
          <button type="button" class="btn-grad" @click="showAuthorizeDevice = true">
            <span class="material-symbols-outlined">qr_code_scanner</span>
            Сканировать или ввести код
          </button>
        </section>

        <!-- Связанные аккаунты (Яндекс ID) — только если вход через Яндекс настроен -->
        <section v-if="yandexAuth.enabled" class="profile-card">
          <header class="card-head">
            <div class="head-icon" data-tone="primary">
              <span class="material-symbols-outlined">link</span>
            </div>
            <div class="head-text">
              <h3>Связанные аккаунты</h3>
              <p class="head-desc">
                Привяжите Яндекс — и входите в Groove Work одной кнопкой, без пароля.
              </p>
            </div>
          </header>
          <div class="yandex-link-row">
            <template v-if="yandexLinked">
              <span class="yandex-linked-chip">
                <span class="material-symbols-outlined">check_circle</span>
                Яндекс привязан
              </span>
              <button type="button" class="btn-glass" :disabled="yandexBusy" @click="unlinkYandex">
                Отвязать
              </button>
            </template>
            <button v-else type="button" class="btn-grad" :disabled="yandexBusy" @click="linkYandex">
              Привязать Яндекс-аккаунт
            </button>
          </div>
        </section>

        <div class="forms-row">
          <!-- Редактирование профиля -->
          <section class="profile-card">
            <header class="card-head">
              <div class="head-icon" data-tone="secondary">
                <span class="material-symbols-outlined">badge</span>
              </div>
              <div class="head-text">
                <h3>Редактирование профиля</h3>
                <p class="head-desc">Данные и контакты, видимые коллегам</p>
              </div>
            </header>
            <form @submit.prevent="saveProfile" class="profile-form">
              <div class="form-group">
                <label>ФИО</label>
                <InputText v-model="profileForm.fio" class="w-full" placeholder="Иванов Иван Иванович" />
              </div>
              <div class="form-group">
                <label>Логин</label>
                <InputText v-model="profileForm.login" class="w-full" placeholder="ivanov" />
              </div>
              <div class="form-group">
                <label>Должность</label>
                <InputText v-model="profileForm.post" class="w-full" placeholder="Менеджер" />
              </div>
              <div class="form-group">
                <label>Телефон</label>
                <PhoneInput v-model="profileForm.phone" />
              </div>
              <div class="form-group">
                <label>Email</label>
                <InputText
                  v-model="profileForm.email"
                  class="w-full"
                  type="email"
                  inputmode="email"
                  placeholder="you@example.com"
                />
              </div>
              <p v-if="profileError" class="error-msg">{{ profileError }}</p>
              <button type="submit" class="btn-primary" :disabled="profileLoading">
                {{ profileLoading ? 'Сохраняем...' : 'Сохранить' }}
              </button>
            </form>
          </section>

          <!-- Смена пароля -->
          <section class="profile-card">
            <header class="card-head">
              <div class="head-icon" data-tone="tertiary">
                <span class="material-symbols-outlined">lock_reset</span>
              </div>
              <div class="head-text">
                <h3>Смена пароля</h3>
                <p class="head-desc">Не короче 8 символов</p>
              </div>
            </header>
            <form @submit.prevent="changePassword" class="profile-form">
              <div class="form-group">
                <label>Текущий пароль</label>
                <InputText
                  v-model="passwordForm.current"
                  type="password"
                  class="w-full"
                  placeholder="Введите текущий пароль"
                  autocomplete="current-password"
                />
              </div>
              <div class="form-group">
                <label>Новый пароль</label>
                <InputText
                  v-model="passwordForm.password"
                  type="password"
                  class="w-full"
                  placeholder="Минимум 8 символов"
                  autocomplete="new-password"
                />
              </div>
              <div class="form-group">
                <label>Подтвердите пароль</label>
                <InputText
                  v-model="passwordForm.confirm"
                  type="password"
                  class="w-full"
                  placeholder="Повторите пароль"
                  autocomplete="new-password"
                />
              </div>
              <p v-if="passwordError" class="error-msg">{{ passwordError }}</p>
              <button type="submit" class="btn-primary" :disabled="passwordLoading">
                {{ passwordLoading ? 'Изменяем...' : 'Изменить пароль' }}
              </button>
            </form>
          </section>
        </div>
      </div>
    </div>

    <!-- Авторизация устройства (QR-вход / ТВ-киоск) -->
    <AuthorizeDeviceDialog v-model="showAuthorizeDevice" />

    <!-- Диалог кроппера аватарки -->
    <AppDialog
      v-if="showCropper"
      model-value
      tone="primary"
      icon="account_circle"
      size="md"
      title="Загрузка аватарки"
      @update:model-value="showCropper = false"
    >
      <AvatarCropper @cropped="onCropped" @cancel="showCropper = false" />
    </AppDialog>

    <AvatarLightbox
      v-model="lightboxOpen"
      :src="avatarSrc"
      :alt="authStore.user?.fio"
      :caption="authStore.user?.fio"
    />

    <AppDialog
      v-model="confirmAvatarDelete"
      tone="danger"
      icon="warning"
      size="sm"
      title="Удалить аватарку?"
      subtitle="Вместо неё коллеги будут видеть автоматический аватар."
      :busy="avatarDeleting"
      :closable="!avatarDeleting"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: avatarDeleting },
        { kind: 'confirm', label: 'Удалить', icon: 'delete', disabled: avatarDeleting },
      ]"
      @confirm="handleDeleteAvatar"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { updateMe, uploadAvatar, deleteAvatar } from '@/api/users.js'
import { yandexConfig, yandexAuthURL, yandexLinkStatus, yandexUnlink } from '@/api/auth.js'
import { inAppShell } from '@/utils/appShell.js'
import { getStatsProfile } from '@/api/stats.js'
import { formatHours } from '@/utils/time.js'
import AvatarCropper from '@/components/settings/AvatarCropper.vue'
import AvatarLightbox from '@/components/common/AvatarLightbox.vue'
import PhoneInput from '@/components/common/PhoneInput.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import InputText from 'primevue/inputtext'
import AppDialog from '@/components/common/AppDialog.vue'
import AuthorizeDeviceDialog from '@/components/devicelink/AuthorizeDeviceDialog.vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import BrandLoader from '@/components/common/BrandLoader.vue'

const authStore = useAuthStore()
const notif = useNotificationsStore()
const { isMobile } = useBreakpoint()

// ---- Avatar ----
const showCropper = ref(false)
const showAuthorizeDevice = ref(false)
const lightboxOpen = ref(false)

const avatarSrc = computed(() => {
  const user = authStore.user
  if (!user) return ''
  if (user.avatar_path) return `/uploads/${user.avatar_path}`
  return `/api/users/${user.id}/identicon`
})

async function onCropped(blob) {
  showCropper.value = false
  try {
    await uploadAvatar(blob)
    await authStore.loadMe()
    notif.success('Аватарка обновлена')
  } catch (e) {
    notif.error(e.message || 'Ошибка загрузки аватарки')
  }
}

const confirmAvatarDelete = ref(false)
const avatarDeleting = ref(false)

async function handleDeleteAvatar() {
  avatarDeleting.value = true
  try {
    await deleteAvatar()
    await authStore.loadMe()
    notif.success('Аватарка удалена')
    confirmAvatarDelete.value = false
  } catch (e) {
    notif.error(e.message || 'Ошибка удаления аватарки')
  } finally {
    avatarDeleting.value = false
  }
}

// ---- Profile form ----
const profileForm = reactive({ fio: '', login: '', post: '', phone: '', email: '' })
const profileError = ref('')
const profileLoading = ref(false)

function syncProfileForm() {
  const user = authStore.user
  if (user) {
    profileForm.fio = user.fio || ''
    profileForm.login = user.login || ''
    profileForm.post = user.post || ''
    profileForm.phone = user.phone || ''
    profileForm.email = user.email || ''
  }
}

async function saveProfile() {
  profileError.value = ''
  if (!profileForm.fio.trim() || !profileForm.login.trim()) {
    profileError.value = 'ФИО и логин обязательны'
    return
  }
  profileLoading.value = true
  try {
    await updateMe({
      fio: profileForm.fio.trim(),
      login: profileForm.login.trim(),
      post: profileForm.post.trim(),
      phone: profileForm.phone.trim() || null,
      email: profileForm.email.trim() || null,
    })
    await authStore.loadMe()
    notif.success('Профиль обновлён')
  } catch (e) {
    profileError.value = e.message || 'Ошибка сохранения'
  } finally {
    profileLoading.value = false
  }
}

// ---- Password form ----
const passwordForm = reactive({ current: '', password: '', confirm: '' })
const passwordError = ref('')
const passwordLoading = ref(false)

async function changePassword() {
  passwordError.value = ''
  if (!passwordForm.current) {
    passwordError.value = 'Введите текущий пароль'
    return
  }
  if (passwordForm.password.length < 8) {
    passwordError.value = 'Пароль должен содержать не менее 8 символов'
    return
  }
  if (passwordForm.password !== passwordForm.confirm) {
    passwordError.value = 'Пароли не совпадают'
    return
  }
  passwordLoading.value = true
  try {
    await updateMe({
      current_password: passwordForm.current,
      new_password: passwordForm.password,
      confirm_password: passwordForm.confirm,
    })
    notif.success('Пароль изменён')
    passwordForm.current = ''
    passwordForm.password = ''
    passwordForm.confirm = ''
  } catch (e) {
    passwordError.value = e.message || 'Ошибка смены пароля'
  } finally {
    passwordLoading.value = false
  }
}

// ---- Stats ----
const statsPeriod = ref(null)
const profileStats = ref(null)
const statsLoading = ref(false)

function getDefaultPeriod() {
  const today = new Date()
  const day = today.getDay()
  const monday = new Date(today)
  monday.setDate(today.getDate() - (day === 0 ? 6 : day - 1))
  monday.setHours(0, 0, 0, 0)
  return [monday, today]
}

function formatDate(d) {
  if (!d) return ''
  return d.toISOString().split('T')[0]
}

function roundHours(val) {
  return formatHours(val)
}

function plural(n, one, few, many) {
  const mod10 = n % 10, mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return one
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return few
  return many
}

const periodDays = computed(() => {
  const r = statsPeriod.value
  if (!r || !r[0] || !r[1]) return 0
  return Math.max(1, Math.round((r[1] - r[0]) / 86400000) + 1)
})

const avgHoursPerDay = computed(() => {
  const total = profileStats.value?.total_hours || 0
  return periodDays.value ? total / periodDays.value : 0
})

const maxTypeHours = computed(() => {
  const types = profileStats.value?.by_unit_types || []
  return Math.max(...types.map(t => t.hours || 0), 0.001)
})

function typeBarWidth(t) {
  return `${Math.max(4, ((t.hours || 0) / maxTypeHours.value) * 100)}%`
}

async function loadStats(range) {
  const r = range || statsPeriod.value
  if (!r || !r[0] || !r[1]) return
  statsLoading.value = true
  try {
    profileStats.value = await getStatsProfile(formatDate(r[0]), formatDate(r[1]))
  } catch (e) {
    notif.error(e.message || 'Ошибка загрузки статистики')
  } finally {
    statsLoading.value = false
  }
}

// Привязка Яндекс-аккаунта: статус и кнопки «Привязать»/«Отвязать».
const yandexAuth = ref({ enabled: false, client_id: '' })
const yandexLinked = ref(false)
const yandexBusy = ref(false)

async function loadYandexLink() {
  try {
    yandexAuth.value = await yandexConfig()
    if (yandexAuth.value.enabled) {
      yandexLinked.value = (await yandexLinkStatus()).linked
    }
  } catch { /* карточка просто не показывается */ }
}

function linkYandex() {
  window.location.href = yandexAuthURL(yandexAuth.value.client_id, inAppShell() ? 'app-link' : 'link')
}

async function unlinkYandex() {
  yandexBusy.value = true
  try {
    await yandexUnlink()
    yandexLinked.value = false
    notif.success('Яндекс-аккаунт отвязан')
  } catch (e) {
    notif.error(e?.message || 'Не удалось отвязать аккаунт')
  } finally {
    yandexBusy.value = false
  }
}

onMounted(() => {
  syncProfileForm()
  statsPeriod.value = getDefaultPeriod()
  loadStats(statsPeriod.value)
  loadYandexLink()
})
</script>

<style scoped>
.profile-view {
  padding: 24px;
  height: 100%;
  overflow-y: auto;
}

/* Десктоп: identity-рейл слева + контент справа — экран используется
   целиком, а не узкой колонкой. */
.profile-container {
  max-width: 1280px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 340px minmax(0, 1fr);
  gap: 24px;
  align-items: start;
}

/* ── Identity-карточка ───────────────────────────────────────── */
/* Не sticky: прилипание держало карточку на 24px ниже кромки при прокрутке,
   и её верх расходился с карточками правой колонки. */
.identity-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  background: var(--acrylic-card-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  padding: 0 20px 20px;
  box-shadow: var(--shadow-sm);
  overflow: hidden;
  text-align: center;
}

/* Экспрессивная обложка — hero-момент идентичности (M3 Expressive).
   Пастельные тона (контейнеры, приглушённые поверхностью) + плавное
   растворение к низу через маску — без резкой кромки. */
.identity-cover {
  width: calc(100% + 40px);
  margin: 0 -20px -20px;
  height: 128px;
  flex-shrink: 0;
  background:
    radial-gradient(120% 140% at 85% 0%,
      color-mix(in oklch, var(--color-tertiary-container) 40%, transparent) 0%,
      transparent 60%),
    linear-gradient(120deg,
      color-mix(in oklch, var(--color-primary-container) 55%, var(--color-surface)),
      color-mix(in oklch, var(--color-secondary-container) 55%, var(--color-surface)));
  -webkit-mask-image: linear-gradient(to bottom, black 30%, transparent 100%);
  mask-image: linear-gradient(to bottom, black 30%, transparent 100%);
}

.avatar-wrapper {
  position: relative;
  width: 120px;
  height: 120px;
  margin-top: -56px;
  border-radius: 50%;
  overflow: hidden;
  border: 3px solid var(--color-primary);
  box-shadow: 0 0 0 4px var(--color-surface);
  flex-shrink: 0;
  padding: 0;
  background: var(--acrylic-card-bg);
  cursor: zoom-in;
  transition: transform .18s, box-shadow .18s;
}

.avatar-wrapper:hover {
  transform: scale(1.03);
  box-shadow: 0 0 0 4px var(--color-surface), var(--shadow-md);
}

.avatar-zoom-overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: color-mix(in oklch, var(--color-scrim) 70%, transparent);
  color: var(--color-on-primary);
  opacity: 0;
  transition: opacity .15s;
}
.avatar-wrapper:hover .avatar-zoom-overlay { opacity: 1; }
.avatar-zoom-overlay .material-symbols-outlined { font-size: 32px; }

.profile-avatar {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.hero-info {
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.hero-name {
  margin: 0;
  font-size: 24px;
  font-weight: 800;
  letter-spacing: -0.3px;
  color: var(--color-text);
  line-height: 1.2;
}

.hero-post {
  margin: 0;
  font-size: 15px;
  color: var(--color-text-dim);
}

.hero-meta {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 4px;
}

.hero-login {
  font-size: 13px;
  color: var(--color-text-dim);
}

.role-tag {
  display: inline-block;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  border-radius: var(--radius-full);
  padding: 4px 14px;
  font-size: 12px;
  font-weight: 600;
}

.avatar-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: center;
}

.btn-sm {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 5px 12px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid var(--color-outline-dim);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  transition: background 0.15s, color 0.15s;
}

.btn-sm:hover {
  background: var(--color-surface-low);
}

.btn-sm.danger {
  color: var(--color-error);
  border-color: color-mix(in oklch, var(--color-error) 30%, var(--color-outline-dim));
}

.btn-sm.danger:hover {
  background: var(--color-error-container);
}

.btn-sm .material-symbols-outlined {
  font-size: 14px;
}

/* Контакты в рейле — тональные иконки в духе разделов настроек. */
.identity-contacts {
  list-style: none;
  width: 100%;
  margin: 4px 0 0;
  padding: 14px 0 0;
  border-top: 1px solid var(--color-outline-dim);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.contact-row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  text-align: left;
}

.contact-ico {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}
.contact-ico[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.contact-ico[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.contact-ico[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }
.contact-ico .material-symbols-outlined { font-size: 18px; }

.contact-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.contact-text small {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

.contact-text > span {
  font-size: 13px;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.contact-text > span.contact-empty { color: var(--color-text-dim); }

.btn-logout {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  margin-top: 4px;
  padding: 10px 18px;
  border: 1px solid color-mix(in oklch, var(--color-error) 30%, var(--color-outline-dim));
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-error);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-logout:hover {
  background: var(--color-error-container);
}

.btn-logout .material-symbols-outlined {
  font-size: 18px;
}

/* ── Правая колонка ──────────────────────────────────────────── */
.profile-main {
  display: flex;
  flex-direction: column;
  gap: 24px;
  min-width: 0;
}

.forms-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  align-items: start;
}

.yandex-link-row {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-wrap: wrap;
}
.yandex-linked-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: var(--radius-pill, 999px);
  background: var(--color-success-container, var(--color-surface-variant));
  color: var(--color-on-success-container, var(--color-text));
  font-size: 0.9rem;
  font-weight: 600;
}
.yandex-linked-chip .material-symbols-outlined { font-size: 18px; }

.profile-card {
  background: var(--acrylic-card-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  padding: 22px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  transition: border-color 0.15s;
}

.profile-card:hover {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--color-outline-dim));
}

/* Шапка карточки: тональная иконка + заголовок + описание. */
.card-head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-outline-dim);
}

.card-head h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text);
}

.head-text { min-width: 0; }

.head-desc {
  margin: 2px 0 0;
  font-size: 12.5px;
  color: var(--color-text-dim);
}

.head-icon {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}
.head-icon[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.head-icon[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.head-icon[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }
.head-icon .material-symbols-outlined { font-size: 22px; }

.stats-head { flex-wrap: wrap; }
.head-period { margin-left: auto; }

/* ── Формы ───────────────────────────────────────────────────── */
.profile-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-dim);
}

.w-full {
  width: 100%;
}

.error-msg {
  margin: 0;
  font-size: 13px;
  color: var(--color-on-error-container);
  padding: 8px 12px;
  background: var(--color-error-container);
  border-radius: 8px;
  border: 1px solid color-mix(in oklch, var(--color-error) 30%, var(--color-outline-dim));
}

.btn-primary {
  align-self: flex-start;
  background: var(--color-primary);
  color: var(--color-on-primary);
  border: none;
  border-radius: var(--radius-full);
  padding: 10px 22px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-primary:hover:not(:disabled) {
  background: var(--color-primary-hover);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* ── Статистика ──────────────────────────────────────────────── */
.stat-tiles {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14px;
}

.stat-tile {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 18px;
  border-radius: 18px;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}
.stat-tile[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.stat-tile[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.stat-tile[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }

.tile-ico {
  flex-shrink: 0;
  width: 38px;
  height: 38px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: color-mix(in oklch, currentColor 14%, transparent);
}
.tile-ico .material-symbols-outlined { font-size: 20px; }

.tile-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.tile-num {
  font-size: 26px;
  font-weight: 800;
  line-height: 1.1;
}

.tile-label {
  font-size: 12px;
  opacity: 0.8;
}

/* Бары по типам юнитов (десктоп). */
.type-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.type-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.type-top {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 12px;
  font-size: 13px;
}

.type-name {
  font-weight: 600;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.type-meta {
  color: var(--color-text-dim);
  white-space: nowrap;
}

.type-bar {
  height: 8px;
  border-radius: var(--radius-full);
  background: var(--color-surface-low);
  overflow: hidden;
}

.type-bar > span {
  display: block;
  height: 100%;
  border-radius: var(--radius-full);
  background: linear-gradient(90deg, var(--color-primary), var(--color-tertiary));
  transition: width 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.stats-table {
  margin-top: 4px;
}

.loading-inline {
  display: flex;
  justify-content: center;
  padding: 24px;
}

.empty-stats {
  text-align: center;
  padding: 24px;
  color: var(--color-text-dim);
  font-size: 14px;
}

/* ── Узкий десктоп: формы в одну колонку ─────────────────────── */
@media (max-width: 1100px) and (min-width: 769px) {
  .profile-container { grid-template-columns: 300px minmax(0, 1fr); }
  .forms-row { grid-template-columns: 1fr; }
  .stat-tiles { grid-template-columns: repeat(2, 1fr); }
  .stat-tile--avg { display: none; }
}

/* ── Мобилка: воспроизводим прежний вид без изменений ────────── */
@media (max-width: 768px) {
  .profile-view {
    padding: 12px;
    /* Резерв под нижнюю навигацию (64px) + 12px воздуха: контент профиля
       скроллится под стекло навигации. */
    padding-bottom: calc(76px + env(safe-area-inset-bottom, 0px));
  }

  .profile-container {
    display: flex;
    flex-direction: column;
    /* Десктопный align-items: start здесь дал бы flex-колонке сжатие по
       содержимому и прижатие влево — на мобилке карточки во всю ширину. */
    align-items: stretch;
    gap: 16px;
    max-width: none;
  }

  .identity-card {
    padding: 24px 16px;
    gap: 16px;
    overflow: visible;
  }

  /* Обложка и контакты — только десктопная фишка. */
  .identity-cover,
  .identity-contacts { display: none; }

  .avatar-wrapper { margin-top: 0; }

  .hero-name { font-size: 26px; }

  .btn-logout { margin-top: 0; }

  .profile-main { gap: 16px; }

  /* Прежний порядок карточек: профиль → пароль → статистика. */
  .forms-row { display: contents; }
  .stats-card { order: 1; }

  /* Шапки карточек — как раньше: просто заголовок с разделителем. */
  .head-icon,
  .head-desc { display: none; }

  .stats-head { flex-wrap: wrap; }
  .head-period { margin-left: 0; flex-basis: 100%; }

  /* Плитки статистики — прежние нейтральные карточки, без третьей. */
  .stat-tiles {
    display: flex;
    gap: 16px;
    flex-wrap: wrap;
  }

  .stat-tile {
    flex: 1;
    min-width: 100px;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 16px;
    border-radius: 10px;
    background: var(--color-surface-low);
    border: 1px solid var(--color-outline-dim);
    color: var(--color-text);
  }

  .stat-tile--avg { display: none; }

  .tile-ico { display: none; }

  .tile-text { align-items: center; gap: 4px; }

  .tile-num {
    font-size: 28px;
    color: var(--color-primary);
    line-height: 1;
  }

  .tile-label {
    font-size: 12px;
    color: var(--color-text-dim);
    opacity: 1;
  }

  .btn-primary { border-radius: 10px; padding: 9px 20px; }

  /* Горизонтальный скролл для таблицы статистики */
  .stats-table {
    overflow-x: auto;
    display: block;
  }
}
</style>
