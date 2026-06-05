<template>
  <div class="profile-view">
    <div class="profile-container">
      <!-- Hero-шапка профиля -->
      <section class="profile-hero">
        <div class="hero-avatar-block">
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
          <div class="avatar-actions">
            <button class="btn-sm" @click="showCropper = true">
              <span class="material-symbols-outlined">photo_camera</span>
              Загрузить
            </button>
            <button
              v-if="authStore.user?.avatar_path"
              class="btn-sm danger"
              @click="handleDeleteAvatar"
            >
              <span class="material-symbols-outlined">delete</span>
              Удалить
            </button>
          </div>
        </div>

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

        <button class="btn-logout" @click="authStore.logout()">
          <span class="material-symbols-outlined">logout</span>
          <span class="btn-logout-label">Выйти</span>
        </button>
      </section>

      <!-- Адаптивная сетка карточек -->
      <div class="profile-grid">
        <!-- Редактирование профиля -->
        <section class="profile-card">
          <h3>Редактирование профиля</h3>
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
          <h3>Смена пароля</h3>
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

        <!-- Личная статистика -->
        <section class="profile-card profile-card--wide">
          <h3>Личная статистика</h3>
          <DateRangePicker v-model="statsPeriod" @update:model-value="loadStats" />

          <div v-if="statsLoading" class="loading-inline">
            <ProgressSpinner style="width:32px;height:32px" />
          </div>

          <template v-else-if="profileStats">
            <div class="profile-stats-cards">
              <div class="stat-card">
                <span class="stat-value">{{ roundHours(profileStats.total_hours) }}</span>
                <span class="stat-label">Время</span>
              </div>
              <div class="stat-card">
                <span class="stat-value">{{ profileStats.tasks_count ?? 0 }}</span>
                <span class="stat-label">Задач</span>
              </div>
            </div>

            <DataTable
              v-if="profileStats.by_unit_types?.length"
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

          <div v-else class="empty-stats">
            Нет данных за выбранный период
          </div>
        </section>
      </div>
    </div>

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
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { updateMe, uploadAvatar, deleteAvatar } from '@/api/users.js'
import { getStatsProfile } from '@/api/stats.js'
import { formatHours } from '@/utils/time.js'
import AvatarCropper from '@/components/settings/AvatarCropper.vue'
import AvatarLightbox from '@/components/common/AvatarLightbox.vue'
import PhoneInput from '@/components/common/PhoneInput.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import InputText from 'primevue/inputtext'
import AppDialog from '@/components/common/AppDialog.vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import ProgressSpinner from 'primevue/progressspinner'

const authStore = useAuthStore()
const notif = useNotificationsStore()

// ---- Avatar ----
const showCropper = ref(false)
const lightboxOpen = ref(false)

const avatarSrc = computed(() => {
  const user = authStore.user
  if (!user) return ''
  if (user.avatar_path) return `/uploads/${user.avatar_path}`
  return `/api/users/${user.id}/identicon`
})

const hasAvatar = computed(() => !!authStore.user?.avatar_path)

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

async function handleDeleteAvatar() {
  try {
    await deleteAvatar()
    await authStore.loadMe()
    notif.success('Аватарка удалена')
  } catch (e) {
    notif.error(e.message || 'Ошибка удаления аватарки')
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

onMounted(() => {
  syncProfileForm()
  statsPeriod.value = getDefaultPeriod()
  loadStats(statsPeriod.value)
})
</script>

<style scoped>
.profile-view {
  padding: 24px;
  height: 100%;
  overflow-y: auto;
}

.profile-container {
  max-width: 1100px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

/* ── Hero-шапка ──────────────────────────────────────────────── */
.profile-hero {
  display: flex;
  align-items: center;
  gap: 28px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-xl);
  padding: 28px;
  box-shadow: var(--shadow-sm);
}

.hero-avatar-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.avatar-wrapper {
  position: relative;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  overflow: hidden;
  border: 3px solid var(--color-primary);
  box-shadow: 0 0 0 4px var(--color-surface);
  flex-shrink: 0;
  padding: 0;
  background: transparent;
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
  border-radius: 8px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
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

.hero-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.hero-name {
  margin: 0;
  font-size: 26px;
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

.btn-logout {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 18px;
  border: 1px solid color-mix(in oklch, var(--color-error) 30%, var(--color-outline-dim));
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-error);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
  flex-shrink: 0;
  align-self: flex-start;
}

.btn-logout:hover {
  background: var(--color-error-container);
}

.btn-logout .material-symbols-outlined {
  font-size: 18px;
}

/* ── Адаптивная сетка карточек ───────────────────────────────── */
.profile-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 24px;
  align-items: start;
}

.profile-card {
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  padding: 22px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Карточка во всю ширину сетки (статистика) */
.profile-card--wide {
  grid-column: 1 / -1;
}

.profile-card h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text);
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-outline-dim);
}

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
  border-radius: 10px;
  padding: 9px 20px;
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

/* Stats */
.profile-stats-cards {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.stat-card {
  flex: 1;
  min-width: 100px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 16px;
  background: var(--color-surface-low);
  border: 1px solid var(--color-outline-dim);
  border-radius: 10px;
}

.stat-value {
  font-size: 28px;
  font-weight: 800;
  color: var(--color-primary);
  line-height: 1;
}

.stat-label {
  font-size: 12px;
  color: var(--color-text-dim);
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

/* ── Адаптив ─────────────────────────────────────────────────── */
@media (max-width: 768px) {
  .profile-view {
    padding: 12px;
    padding-bottom: calc(60px + 12px + env(safe-area-inset-bottom, 0px));
  }

  .profile-hero {
    flex-direction: column;
    text-align: center;
    padding: 24px 16px;
    gap: 16px;
  }

  .hero-info {
    align-items: center;
  }

  .hero-meta {
    justify-content: center;
  }

  .btn-logout {
    align-self: stretch;
  }

  /* На узком экране карточки всегда в одну колонку на всю ширину */
  .profile-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  /* Горизонтальный скролл для таблицы статистики */
  .stats-table {
    overflow-x: auto;
    display: block;
  }
}
</style>
