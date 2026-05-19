<template>
  <div class="profile-view">
    <div class="profile-layout">
      <!-- Sidebar -->
      <aside class="profile-sidebar">
        <div class="avatar-section">
          <div class="avatar-wrapper">
            <img :src="avatarSrc" class="profile-avatar" :alt="authStore.user?.fio" />
          </div>
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

        <div class="user-info">
          <h2>{{ authStore.user?.fio }}</h2>
          <p class="user-post">{{ authStore.user?.post || 'Должность не указана' }}</p>
          <span v-if="authStore.user?.role?.name" class="role-tag">
            {{ authStore.user.role.name }}
          </span>
        </div>

        <button class="btn-logout" @click="authStore.logout()">
          <span class="material-symbols-outlined">logout</span>
          Выйти
        </button>
      </aside>

      <!-- Main -->
      <main class="profile-main">
        <!-- Редактирование профиля -->
        <section class="profile-section">
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
            <p v-if="profileError" class="error-msg">{{ profileError }}</p>
            <button type="submit" class="btn-primary" :disabled="profileLoading">
              {{ profileLoading ? 'Сохраняем...' : 'Сохранить' }}
            </button>
          </form>
        </section>

        <!-- Смена пароля -->
        <section class="profile-section">
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
        <section class="profile-section">
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
      </main>
    </div>

    <!-- Диалог кроппера аватарки -->
    <Dialog
      v-if="showCropper"
      :visible="true"
      @update:visible="showCropper = false"
      modal
      header="Загрузка аватарки"
      style="width:520px"
    >
      <AvatarCropper @cropped="onCropped" @cancel="showCropper = false" />
    </Dialog>
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
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import InputText from 'primevue/inputtext'
import Dialog from 'primevue/dialog'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import ProgressSpinner from 'primevue/progressspinner'

const authStore = useAuthStore()
const notif = useNotificationsStore()

// ---- Avatar ----
const showCropper = ref(false)

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
const profileForm = reactive({ fio: '', login: '', post: '' })
const profileError = ref('')
const profileLoading = ref(false)

function syncProfileForm() {
  const user = authStore.user
  if (user) {
    profileForm.fio = user.fio || ''
    profileForm.login = user.login || ''
    profileForm.post = user.post || ''
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
      post: profileForm.post.trim()
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

.profile-layout {
  display: flex;
  gap: 28px;
  max-width: 1000px;
  margin: 0 auto;
  align-items: flex-start;
}

/* Sidebar */
.profile-sidebar {
  width: 240px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
  position: sticky;
  top: 0;
}

.avatar-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.avatar-wrapper {
  width: 120px;
  height: 120px;
  border-radius: 50%;
  overflow: hidden;
  border: 3px solid var(--gw-primary);
  box-shadow: 0 0 0 4px var(--gw-bg);
  flex-shrink: 0;
}

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
  border: 1px solid var(--gw-border);
  background: var(--gw-surface);
  color: var(--gw-text);
  transition: background 0.15s, color 0.15s;
}

.btn-sm:hover {
  background: var(--gw-bg);
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

.user-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  text-align: center;
}

.user-info h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: var(--gw-text);
  line-height: 1.3;
}

.user-post {
  margin: 0;
  font-size: 13px;
  color: var(--gw-text-secondary);
}

.role-tag {
  display: inline-block;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  border-radius: 20px;
  padding: 4px 14px;
  font-size: 12px;
  font-weight: 600;
}

.btn-logout {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px;
  border: 1px solid color-mix(in oklch, var(--color-error) 30%, var(--color-outline-dim));
  border-radius: 10px;
  background: transparent;
  color: var(--color-error);
  font-size: 14px;
  cursor: pointer;
  transition: background 0.15s;
  width: 100%;
}

.btn-logout:hover {
  background: var(--color-error-container);
}

.btn-logout .material-symbols-outlined {
  font-size: 18px;
}

/* Main */
.profile-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 24px;
  min-width: 0;
}

.profile-section {
  background: var(--gw-surface);
  border: 1px solid var(--gw-border);
  border-radius: var(--gw-radius);
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.profile-section h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--gw-text);
  padding-bottom: 12px;
  border-bottom: 1px solid var(--gw-border);
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
  color: var(--gw-text-secondary);
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
  background: var(--gw-primary);
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
  background: var(--gw-primary-hover);
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
  background: var(--gw-bg);
  border: 1px solid var(--gw-border);
  border-radius: 10px;
}

.stat-value {
  font-size: 28px;
  font-weight: 800;
  color: var(--gw-primary);
  line-height: 1;
}

.stat-label {
  font-size: 12px;
  color: var(--gw-text-secondary);
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
  color: var(--gw-text-secondary);
  font-size: 14px;
}

/* Responsive */
@media (max-width: 768px) {
  .profile-view {
    padding: 12px;
    padding-bottom: calc(60px + 12px + env(safe-area-inset-bottom, 0px));
  }

  .profile-layout {
    flex-direction: column;
    gap: 16px;
  }

  .profile-sidebar {
    width: 100%;
    position: static;
  }

  /* Горизонтальный скролл для таблицы статистики */
  .stats-table {
    overflow-x: auto;
    display: block;
  }
}
</style>
