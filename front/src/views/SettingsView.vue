<template>
  <div class="settings-shell" :class="{ 'is-mobile-section': isMobile && activeSection }">
    <!-- Левая колонка: список секций. На мобильном — отдельный экран. -->
    <aside class="settings-nav" :class="{ 'mobile-hidden': isMobile && activeSection }" data-tutorial="settings-nav">
      <header class="settings-nav-header">
        <h1>Настройки</h1>
        <p class="settings-nav-sub">Настройте платформу под себя</p>
      </header>

      <div class="settings-search">
        <span class="material-symbols-outlined">search</span>
        <input
          v-model="searchQuery"
          type="search"
          placeholder="Найти настройку…"
        />
        <button
          v-if="searchQuery"
          class="settings-search-clear"
          title="Очистить"
          @click="searchQuery = ''"
        >
          <span class="material-symbols-outlined">close</span>
        </button>
      </div>

      <nav class="settings-sections">
        <template v-for="group in visibleGroups" :key="group.key">
          <div class="settings-group-label">{{ group.label }}</div>
          <button
            v-for="section in group.sections"
            :key="section.key"
            class="settings-nav-item"
            :class="{ active: !isMobile && activeSection === section.key }"
            :data-tutorial="`settings-section-${section.key}`"
            @click="openSection(section.key)"
          >
            <span class="nav-icon" :data-tone="section.tone || 'primary'">
              <span class="material-symbols-outlined">{{ section.icon }}</span>
            </span>
            <span class="nav-text">
              <span class="nav-title">{{ section.title }}</span>
              <span class="nav-desc">{{ section.desc }}</span>
            </span>
            <span class="material-symbols-outlined nav-chevron">chevron_right</span>
          </button>
        </template>
        <div v-if="!visibleGroups.length && searchQuery" class="settings-nav-empty">
          <span class="material-symbols-outlined">search_off</span>
          <p>Ничего не нашли. Попробуйте другие слова.</p>
        </div>
      </nav>

      <button
        class="settings-nav-footer"
        @click="changelog.open()"
        title="Открыть историю версий"
      >
        <span class="material-symbols-outlined">auto_awesome</span>
        <div>
          <div class="footer-name">Groove Work</div>
          <div class="footer-version" v-if="appVersion">v{{ appVersion }} · что нового</div>
        </div>
        <span class="material-symbols-outlined footer-chev">chevron_right</span>
      </button>
    </aside>

    <!-- Правая колонка: контент активной секции. -->
    <Transition name="pane-swap" mode="out-in">
      <section
        v-if="activeSection || !isMobile"
        :key="activeSection || 'empty'"
        class="settings-pane"
        :class="{ 'mobile-full': isMobile && activeSection }"
      >
        <header class="settings-pane-header">
          <button
            v-if="isMobile"
            class="settings-back"
            @click="activeSection = null"
            title="Назад к списку"
            aria-label="Назад"
          >
            <span class="material-symbols-outlined">arrow_back</span>
          </button>
          <div class="pane-title-icon" v-if="activeSectionMeta" :data-tone="activeSectionMeta.tone || 'primary'">
            <span class="material-symbols-outlined">{{ activeSectionMeta.icon }}</span>
          </div>
          <div class="pane-title-wrap">
            <h2 class="pane-title">{{ activeSectionMeta?.title || 'Настройки' }}</h2>
            <p v-if="activeSectionMeta?.desc" class="pane-sub">{{ activeSectionMeta.desc }}</p>
          </div>
        </header>

        <div class="settings-pane-body">
        <!-- Внешний вид -->
        <div v-show="activeSection === 'theme'" class="pane-block">
          <ThemeBuilder />
        </div>

        <!-- Пользователи -->
        <div v-show="activeSection === 'users'" class="pane-block">
          <div class="pane-toolbar">
            <div class="search-wrapper">
              <span class="material-symbols-outlined search-icon">search</span>
              <input
                v-model="userSearch"
                class="search-input"
                placeholder="Поиск по ФИО или логину…"
                @input="onUserSearch"
              />
            </div>
            <button class="btn-filled" @click="openUserDialog(null)">
              <span class="material-symbols-outlined">person_add</span>
              Создать
            </button>
          </div>

          <div class="users-grid">
            <div
              v-for="u in filteredUsers"
              :key="u.id"
              class="user-card"
              :class="{ 'is-me': u.id === authStore.user?.id }"
            >
              <img class="user-card-avatar" :src="getUserAvatar(u)" :alt="u.fio" />
              <div class="user-card-text">
                <div class="user-card-name">{{ u.fio }}</div>
                <div class="user-card-login">@{{ u.login }}</div>
                <div v-if="u.post" class="user-card-post">{{ u.post }}</div>
                <span class="user-card-role" :data-level="u.role?.level || 1">
                  <span class="material-symbols-outlined">{{ roleIcon(u.role?.level) }}</span>
                  {{ u.role?.name || '—' }}
                </span>
              </div>
              <div class="user-card-actions">
                <button class="icon-btn" title="Редактировать" @click="openUserDialog(u)">
                  <span class="material-symbols-outlined">edit</span>
                </button>
                <button class="icon-btn danger" title="Удалить" @click="confirmDeleteUser(u)">
                  <span class="material-symbols-outlined">delete</span>
                </button>
              </div>
            </div>
            <div v-if="!usersLoading && !filteredUsers.length" class="settings-empty">
              <div class="empty-icon" data-tone="primary">
                <span class="material-symbols-outlined">person_search</span>
              </div>
              <h4>Никого не нашли</h4>
              <p>{{ userSearch ? 'Попробуйте другой запрос.' : 'Создайте первого сотрудника — кнопка справа сверху.' }}</p>
            </div>
          </div>
        </div>

        <!-- Отделы -->
        <div v-show="activeSection === 'departments'" class="pane-block">
          <div class="pane-toolbar">
            <p class="pane-toolbar-hint">Используются для группировки сотрудников и в статистике.</p>
            <button
              v-if="isAtLeast(ROLES.MANAGER)"
              class="btn-filled"
              @click="startAddDept"
            >
              <span class="material-symbols-outlined">add</span>
              Добавить отдел
            </button>
          </div>

          <div class="chip-list">
            <div v-if="addingDept" class="chip-row editing">
              <span class="material-symbols-outlined chip-icon">apartment</span>
              <input
                v-model="newDeptName"
                class="chip-input"
                placeholder="Название отдела"
                autofocus
                @keyup.enter="saveDept"
                @keyup.escape="addingDept = false"
              />
              <button class="icon-btn success" @click="saveDept" title="Сохранить">
                <span class="material-symbols-outlined">check</span>
              </button>
              <button class="icon-btn" @click="addingDept = false" title="Отмена">
                <span class="material-symbols-outlined">close</span>
              </button>
            </div>

            <div v-for="dept in departments" :key="dept.id" class="chip-row">
              <template v-if="editingDeptId === dept.id">
                <span class="material-symbols-outlined chip-icon">apartment</span>
                <input
                  v-model="editingDeptName"
                  class="chip-input"
                  @keyup.enter="updateDept(dept)"
                  @keyup.escape="editingDeptId = null"
                />
                <button class="icon-btn success" @click="updateDept(dept)" title="Сохранить">
                  <span class="material-symbols-outlined">check</span>
                </button>
                <button class="icon-btn" @click="editingDeptId = null" title="Отмена">
                  <span class="material-symbols-outlined">close</span>
                </button>
              </template>
              <template v-else>
                <span class="material-symbols-outlined chip-icon">apartment</span>
                <span class="chip-name">{{ dept.name }}</span>
                <div v-if="isAtLeast(ROLES.MANAGER)" class="row-actions">
                  <button class="icon-btn" title="Редактировать" @click="startEditDept(dept)">
                    <span class="material-symbols-outlined">edit</span>
                  </button>
                  <button class="icon-btn danger" title="Удалить" @click="confirmDeleteDept(dept)">
                    <span class="material-symbols-outlined">delete</span>
                  </button>
                </div>
              </template>
            </div>
            <div v-if="!departments.length && !addingDept" class="settings-empty">
              <div class="empty-icon" data-tone="secondary">
                <span class="material-symbols-outlined">apartment</span>
              </div>
              <h4>Отделов пока нет</h4>
              <p>Создайте первый — он появится в фильтрах и статистике.</p>
            </div>
          </div>
        </div>

        <!-- Типы юнитов -->
        <div v-show="activeSection === 'unit-types'" class="pane-block">
          <div class="pane-toolbar">
            <p class="pane-toolbar-hint">Категории работы — встреча, дизайн, написание кода и т. п.</p>
            <button
              v-if="isAtLeast(ROLES.MANAGER)"
              class="btn-filled"
              @click="startAddUnitType"
            >
              <span class="material-symbols-outlined">add</span>
              Добавить тип
            </button>
          </div>

          <div class="chip-list">
            <div v-if="addingUnitType" class="chip-row editing">
              <span class="material-symbols-outlined chip-icon">category</span>
              <input
                v-model="newUnitTypeName"
                class="chip-input"
                placeholder="Название типа"
                autofocus
                @keyup.enter="saveUnitType"
                @keyup.escape="addingUnitType = false"
              />
              <button class="icon-btn success" @click="saveUnitType" title="Сохранить">
                <span class="material-symbols-outlined">check</span>
              </button>
              <button class="icon-btn" @click="addingUnitType = false" title="Отмена">
                <span class="material-symbols-outlined">close</span>
              </button>
            </div>

            <div v-for="ut in unitTypes" :key="ut.id" class="chip-row">
              <template v-if="editingUnitTypeId === ut.id">
                <span class="material-symbols-outlined chip-icon">category</span>
                <input
                  v-model="editingUnitTypeName"
                  class="chip-input"
                  @keyup.enter="updateUnitType(ut)"
                  @keyup.escape="editingUnitTypeId = null"
                />
                <button class="icon-btn success" @click="updateUnitType(ut)" title="Сохранить">
                  <span class="material-symbols-outlined">check</span>
                </button>
                <button class="icon-btn" @click="editingUnitTypeId = null" title="Отмена">
                  <span class="material-symbols-outlined">close</span>
                </button>
              </template>
              <template v-else>
                <span class="material-symbols-outlined chip-icon">category</span>
                <span class="chip-name">{{ ut.name }}</span>
                <div v-if="isAtLeast(ROLES.MANAGER)" class="row-actions">
                  <button class="icon-btn" title="Редактировать" @click="startEditUnitType(ut)">
                    <span class="material-symbols-outlined">edit</span>
                  </button>
                  <button class="icon-btn danger" title="Удалить" @click="confirmDeleteUnitType(ut)">
                    <span class="material-symbols-outlined">delete</span>
                  </button>
                </div>
              </template>
            </div>
            <div v-if="!unitTypes.length && !addingUnitType" class="settings-empty">
              <div class="empty-icon" data-tone="tertiary">
                <span class="material-symbols-outlined">category</span>
              </div>
              <h4>Типов юнитов пока нет</h4>
              <p>Без них юниты создавать нельзя — добавьте хотя бы один.</p>
            </div>
          </div>
        </div>

        <!-- Резервная копия -->
        <div v-show="activeSection === 'backup'" class="pane-block">
          <div class="settings-card">
            <div class="hero-icon" data-tone="primary">
              <span class="material-symbols-outlined">backup</span>
            </div>
            <div class="card-text">
              <h3>Создать резервную копию</h3>
              <p>Полный архив базы данных и вложений в одном файле. Сохраняйте регулярно — на всякий случай.</p>
            </div>
            <div class="card-actions">
              <button class="btn-filled" @click="doExportBackup" :disabled="backupExporting">
                <span class="material-symbols-outlined">download</span>
                {{ backupExporting ? 'Готовим архив…' : 'Скачать копию' }}
              </button>
            </div>
          </div>

          <div class="settings-card settings-card--danger">
            <div class="hero-icon" data-tone="error">
              <span class="material-symbols-outlined">restore</span>
            </div>
            <div class="card-text">
              <h3>Восстановление</h3>
              <p>Полная замена текущих данных на содержимое архива. Действие необратимо — мы дважды переспросим.</p>
            </div>
            <div class="card-actions">
              <label class="btn-outlined danger file-btn">
                <span class="material-symbols-outlined">upload</span>
                Выбрать файл
                <input type="file" accept=".zip" @change="onImportFileSelect" style="display:none" />
              </label>
            </div>
          </div>
        </div>

        <!-- Справка -->
        <div v-show="activeSection === 'help'" class="pane-block">
          <HelpCenter />
        </div>
        </div>
      </section>
    </Transition>

    <!-- Диалоги — общие для всех секций -->
    <Dialog
      v-model:visible="showUserDialog"
      :header="editingUser ? 'Редактирование пользователя' : 'Новый пользователь'"
      modal
      :draggable="false"
      style="width:480px"
    >
      <form @submit.prevent="saveUser" class="dialog-form">
        <div class="form-group">
          <label>ФИО</label>
          <InputText v-model="userForm.fio" class="w-full" placeholder="Иванов Иван Иванович" />
        </div>
        <div class="form-group">
          <label>Логин</label>
          <InputText v-model="userForm.login" class="w-full" placeholder="ivanov" />
        </div>
        <div v-if="!editingUser" class="form-group">
          <label>Пароль</label>
          <InputText v-model="userForm.password" type="password" class="w-full" placeholder="Минимум 8 символов" />
        </div>
        <div class="form-group">
          <label>Должность</label>
          <InputText v-model="userForm.post" class="w-full" placeholder="Менеджер" />
        </div>
        <div class="form-group">
          <label>Роль</label>
          <Select
            v-model="userForm.role_id"
            :options="assignableRoles"
            option-label="name"
            option-value="id"
            placeholder="Выберите роль"
            class="w-full"
          />
        </div>
        <p v-if="userFormError" class="error-msg">{{ userFormError }}</p>
        <div class="dialog-footer">
          <button type="button" class="btn-text" @click="showUserDialog = false">Отмена</button>
          <button type="submit" class="btn-filled" :disabled="userFormLoading">
            {{ userFormLoading ? 'Сохраняем…' : 'Сохранить' }}
          </button>
        </div>
      </form>
    </Dialog>

    <ConfirmDialog
      :visible="!!deletingUser"
      header="Удалить пользователя"
      :message="`Удалить пользователя «${deletingUser?.fio}»? Данные задач и юнитов сохраняются.`"
      confirm-label="Удалить"
      :danger-confirm="true"
      @confirm="doDeleteUser"
      @cancel="deletingUser = null"
    />
    <ConfirmDialog
      :visible="!!deletingDept"
      header="Удалить отдел"
      :message="`Удалить отдел «${deletingDept?.name}»?`"
      confirm-label="Удалить"
      :danger-confirm="true"
      @confirm="doDeleteDept"
      @cancel="deletingDept = null"
    />
    <ConfirmDialog
      :visible="!!deletingUnitType"
      header="Удалить тип юнита"
      :message="`Удалить тип «${deletingUnitType?.name}»? Все юниты этого типа будут удалены безвозвратно.`"
      confirm-label="Удалить"
      :danger-confirm="true"
      @confirm="doDeleteUnitType"
      @cancel="deletingUnitType = null"
    />
    <ConfirmDialog
      :visible="showImportConfirm1"
      header="Восстановление из резервной копии"
      message="Вы уверены? Текущие данные будут полностью заменены данными из файла резервной копии."
      confirm-label="Продолжить"
      :danger-confirm="true"
      @confirm="showImportConfirm1 = false; showImportConfirm2 = true"
      @cancel="showImportConfirm1 = false; importFile = null"
    />
    <ConfirmDialog
      :visible="showImportConfirm2"
      header="Подтвердите восстановление"
      message="Это последнее предупреждение. Все текущие данные будут безвозвратно заменены. Продолжить?"
      confirm-label="Да, восстановить"
      :danger-confirm="true"
      @confirm="doImportBackup"
      @cancel="showImportConfirm2 = false; importFile = null"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAuthStore } from '@/stores/auth.js'
import { useTutorial } from '@/composables/useTutorial.js'
import { useChangelog } from '@/composables/useChangelog.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { version as appVersion } from '../../package.json'
import {
  getUsers, createUser, updateUser, deleteUser, assignRole,
} from '@/api/users.js'
import { getRoles } from '@/api/roles.js'
import {
  getDepartments, createDepartment, updateDepartment, deleteDepartment,
} from '@/api/departments.js'
import {
  getUnitTypes, createUnitType, updateUnitType as apiUpdateUnitType, deleteUnitType,
} from '@/api/unitTypes.js'
import { exportBackup, importBackup } from '@/api/backup.js'
import ThemeBuilder from '@/components/settings/ThemeBuilder.vue'
import HelpCenter from '@/components/settings/HelpCenter.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'

const { isAtLeast, myLevel } = usePermission()
const notif = useNotificationsStore()
const authStore = useAuthStore()
const tutorial = useTutorial()
const changelog = useChangelog()
const { isMobile } = useBreakpoint()
const route = useRoute()
const router = useRouter()

const searchQuery = ref('')
const activeSection = ref(null) // null = показывать список секций (на мобильном)

/* ── Конфигурация разделов ─────────────────────────────────────── */
const allGroups = computed(() => [
  {
    key: 'personal',
    label: 'Персонализация',
    sections: [
      { key: 'theme', title: 'Внешний вид', desc: 'Цвета, тёмная тема и стиль интерфейса', icon: 'palette', tone: 'primary' },
      { key: 'help', title: 'Справка', desc: 'Как пользоваться разделами платформы', icon: 'help_center', tone: 'secondary' },
    ],
  },
  {
    key: 'admin',
    label: 'Администрирование',
    visible: () => isAtLeast(ROLES.EMPLOYEE),
    sections: [
      ...(isAtLeast(ROLES.ADMIN) ? [
        { key: 'users', title: 'Пользователи', desc: 'Создание сотрудников и управление ролями', icon: 'group', tone: 'primary' },
      ] : []),
      { key: 'departments', title: 'Отделы', desc: 'Группировка сотрудников для статистики', icon: 'apartment', tone: 'secondary' },
      { key: 'unit-types', title: 'Типы юнитов', desc: 'Категории работы для учёта времени', icon: 'category', tone: 'tertiary' },
    ],
  },
  ...(isAtLeast(ROLES.SUPERADMIN) ? [{
    key: 'system',
    label: 'Система',
    sections: [
      { key: 'backup', title: 'Резервная копия', desc: 'Экспорт и восстановление базы данных', icon: 'backup', tone: 'error' },
    ],
  }] : []),
])

const visibleGroups = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return allGroups.value
    .filter(g => !g.visible || g.visible())
    .map(g => ({
      ...g,
      sections: g.sections.filter(s => !q || [s.title, s.desc].some(x => x.toLowerCase().includes(q))),
    }))
    .filter(g => g.sections.length)
})

const sectionByKey = computed(() => {
  const map = {}
  allGroups.value.forEach(g => g.sections.forEach(s => { map[s.key] = s }))
  return map
})

const activeSectionMeta = computed(() => activeSection.value ? sectionByKey.value[activeSection.value] : null)

function openSection(key) {
  activeSection.value = key
  router.replace({ query: { ...route.query, section: key } }).catch(() => {})
}

function roleIcon(level) {
  if (level >= 4) return 'workspace_premium'
  if (level >= 3) return 'shield_person'
  if (level >= 2) return 'badge'
  return 'person'
}

/* ── Пользователи ──────────────────────────────────────────────── */
const users = ref([])
const usersLoading = ref(false)
const userSearch = ref('')
const showUserDialog = ref(false)
const editingUser = ref(null)
const deletingUser = ref(null)
const userFormLoading = ref(false)
const userFormError = ref('')
const userForm = reactive({ fio: '', login: '', password: '', post: '', role_id: null })

const roles = ref([])
const assignableRoles = computed(() => {
  const level = myLevel()
  return roles.value.filter(r => r.level < level)
})

// getUsers() не принимает поисковую строку (бэк отдаёт всех админу) —
// фильтруем на клиенте по ФИО/логину/должности. Это и быстрее: нет
// сетевого round-trip на каждый ввод.
function onUserSearch() { /* фильтрация делает computed filteredUsers */ }

const filteredUsers = computed(() => {
  const q = userSearch.value.trim().toLowerCase()
  if (!q) return users.value
  return users.value.filter(u =>
    (u.fio || '').toLowerCase().includes(q)
    || (u.login || '').toLowerCase().includes(q)
    || (u.post || '').toLowerCase().includes(q)
    || (u.role?.name || '').toLowerCase().includes(q)
  )
})

async function loadUsers() {
  usersLoading.value = true
  try { users.value = await getUsers() }
  catch (e) { notif.error(e.message || 'Ошибка загрузки пользователей') }
  finally { usersLoading.value = false }
}

async function loadRoles() {
  try { roles.value = await getRoles() }
  catch (e) { notif.error(e.message || 'Ошибка загрузки ролей') }
}

function getUserAvatar(user) {
  if (user.avatar_path) return `/uploads/${user.avatar_path}`
  return `/api/users/${user.id}/identicon`
}

function openUserDialog(user) {
  editingUser.value = user
  userFormError.value = ''
  if (user) {
    Object.assign(userForm, {
      fio: user.fio || '',
      login: user.login || '',
      password: '',
      post: user.post || '',
      role_id: user.role?.id || null,
    })
  } else {
    Object.assign(userForm, { fio: '', login: '', password: '', post: '', role_id: null })
  }
  showUserDialog.value = true
}

async function saveUser() {
  userFormError.value = ''
  if (!userForm.fio.trim() || !userForm.login.trim()) {
    userFormError.value = 'ФИО и логин обязательны'
    return
  }
  if (!editingUser.value && userForm.password.length < 8) {
    userFormError.value = 'Пароль должен содержать не менее 8 символов'
    return
  }
  userFormLoading.value = true
  try {
    if (editingUser.value) {
      const payload = { fio: userForm.fio.trim(), login: userForm.login.trim(), post: userForm.post.trim() }
      await updateUser(editingUser.value.id, payload)
      if (userForm.role_id && userForm.role_id !== editingUser.value.role?.id) {
        await assignRole(editingUser.value.id, { role_id: userForm.role_id })
      }
      notif.success('Пользователь обновлён')
    } else {
      const payload = {
        fio: userForm.fio.trim(), login: userForm.login.trim(), post: userForm.post.trim(),
        password: userForm.password, role_id: userForm.role_id,
      }
      await createUser(payload)
      notif.success('Пользователь создан')
    }
    showUserDialog.value = false
    loadUsers()
  } catch (e) { userFormError.value = e.message || 'Ошибка сохранения' }
  finally { userFormLoading.value = false }
}

function confirmDeleteUser(user) { deletingUser.value = user }
async function doDeleteUser() {
  if (!deletingUser.value) return
  try {
    await deleteUser(deletingUser.value.id)
    notif.success('Пользователь удалён')
    users.value = users.value.filter(u => u.id !== deletingUser.value.id)
  } catch (e) { notif.error(e.message || 'Ошибка удаления') }
  finally { deletingUser.value = null }
}

/* ── Отделы ────────────────────────────────────────────────────── */
const departments = ref([])
const addingDept = ref(false)
const newDeptName = ref('')
const editingDeptId = ref(null)
const editingDeptName = ref('')
const deletingDept = ref(null)

async function loadDepartments() {
  try { departments.value = await getDepartments() }
  catch (e) { notif.error(e.message || 'Ошибка загрузки отделов') }
}

function startAddDept() { addingDept.value = true; newDeptName.value = '' }

async function saveDept() {
  if (!newDeptName.value.trim()) return
  try {
    await createDepartment({ name: newDeptName.value.trim() })
    notif.success('Отдел создан'); addingDept.value = false; loadDepartments()
  } catch (e) { notif.error(e.message || 'Ошибка создания отдела') }
}

function startEditDept(dept) { editingDeptId.value = dept.id; editingDeptName.value = dept.name }

async function updateDept(dept) {
  if (!editingDeptName.value.trim()) return
  try {
    await updateDepartment(dept.id, { name: editingDeptName.value.trim() })
    notif.success('Отдел обновлён'); editingDeptId.value = null; loadDepartments()
  } catch (e) { notif.error(e.message || 'Ошибка обновления') }
}

function confirmDeleteDept(dept) { deletingDept.value = dept }
async function doDeleteDept() {
  if (!deletingDept.value) return
  try {
    await deleteDepartment(deletingDept.value.id)
    notif.success('Отдел удалён')
    departments.value = departments.value.filter(d => d.id !== deletingDept.value.id)
  } catch (e) { notif.error(e.message || 'Ошибка удаления') }
  finally { deletingDept.value = null }
}

/* ── Типы юнитов ───────────────────────────────────────────────── */
const unitTypes = ref([])
const addingUnitType = ref(false)
const newUnitTypeName = ref('')
const editingUnitTypeId = ref(null)
const editingUnitTypeName = ref('')
const deletingUnitType = ref(null)

async function loadUnitTypes() {
  try { unitTypes.value = await getUnitTypes() }
  catch (e) { notif.error(e.message || 'Ошибка загрузки типов юнитов') }
}

function startAddUnitType() { addingUnitType.value = true; newUnitTypeName.value = '' }

async function saveUnitType() {
  if (!newUnitTypeName.value.trim()) return
  try {
    await createUnitType({ name: newUnitTypeName.value.trim() })
    notif.success('Тип юнита создан'); addingUnitType.value = false; loadUnitTypes()
  } catch (e) { notif.error(e.message || 'Ошибка создания') }
}

function startEditUnitType(ut) { editingUnitTypeId.value = ut.id; editingUnitTypeName.value = ut.name }

async function updateUnitType(ut) {
  if (!editingUnitTypeName.value.trim()) return
  try {
    await apiUpdateUnitType(ut.id, { name: editingUnitTypeName.value.trim() })
    notif.success('Тип юнита обновлён'); editingUnitTypeId.value = null; loadUnitTypes()
  } catch (e) { notif.error(e.message || 'Ошибка обновления') }
}

function confirmDeleteUnitType(ut) { deletingUnitType.value = ut }
async function doDeleteUnitType() {
  if (!deletingUnitType.value) return
  try {
    await deleteUnitType(deletingUnitType.value.id)
    notif.success('Тип юнита удалён')
    unitTypes.value = unitTypes.value.filter(u => u.id !== deletingUnitType.value.id)
  } catch (e) { notif.error(e.message || 'Ошибка удаления') }
  finally { deletingUnitType.value = null }
}

/* ── Backup ────────────────────────────────────────────────────── */
const backupExporting = ref(false)
const showImportConfirm1 = ref(false)
const showImportConfirm2 = ref(false)
const importFile = ref(null)

async function doExportBackup() {
  backupExporting.value = true
  try {
    const response = await exportBackup()
    let blob
    if (response instanceof Blob) blob = response
    else if (response && typeof response.blob === 'function') blob = await response.blob()
    else blob = new Blob([JSON.stringify(response)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `backup_${new Date().toISOString().split('T')[0]}.zip`
    document.body.appendChild(a); a.click(); document.body.removeChild(a)
    URL.revokeObjectURL(url)
    notif.success('Резервная копия создана')
  } catch (e) { notif.error(e.message || 'Ошибка создания резервной копии') }
  finally { backupExporting.value = false }
}

function onImportFileSelect(event) {
  const file = event.target.files[0]
  if (!file) return
  importFile.value = file
  showImportConfirm1.value = true
  event.target.value = ''
}

async function doImportBackup() {
  showImportConfirm2.value = false
  if (!importFile.value) return
  try {
    await importBackup(importFile.value)
    notif.success('База данных восстановлена. Страница перезагрузится.')
    setTimeout(() => window.location.reload(), 2000)
  } catch (e) { notif.error(e.message || 'Ошибка восстановления') }
  finally { importFile.value = null }
}

/* ── Загрузка при смене секции ─────────────────────────────────── */
watch(activeSection, (key) => {
  if (key === 'users') { loadUsers(); loadRoles() }
  if (key === 'departments') loadDepartments()
  if (key === 'unit-types') loadUnitTypes()
})

onMounted(() => {
  loadRoles()
  // Стартовая секция: ?section=… или дефолт
  const requested = route.query.section
  const initial = (requested && sectionByKey.value[requested]) ? requested : (isMobile.value ? null : 'theme')
  if (initial) {
    activeSection.value = initial
  } else if (!isMobile.value) {
    activeSection.value = 'theme'
  }
})

// Если стартовали на десктопе и потом перешли на мобильный/обратно — никаких
// особенных действий не нужно, layout сам реагирует.
</script>

<style scoped>
/* ──────────────────────────────────────────────────────────────────
   M3 Expressive Settings Layout
   Десктоп: фиксированный двухколоночный layout, каждая колонка имеет
            собственный scroll, общий main-content не скроллит.
   Планшет (≤1024): sidebar сужается, описания скрываются.
   Мобильный (≤768): drill-down. Список секций — обычный flow.
            При выборе секции — fixed fullscreen pane со sticky header.
────────────────────────────────────────────────────────────────── */
.settings-shell {
  display: grid;
  grid-template-columns: 340px 1fr;
  gap: 24px;
  padding: 24px;
  max-width: 1400px;
  margin: 0 auto;
  /* Берём всю доступную высоту main-content (он flex:1; min-height:0; overflow:auto).
     overflow:hidden на самой шелле — чтобы общий main-content scroll не активировался;
     внутренний scroll живёт на settings-nav и settings-pane-body. */
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

/* ── Левая колонка ──────────────────────────────────────────── */
.settings-nav {
  display: flex;
  flex-direction: column;
  gap: 16px;
  background: var(--color-surface-low);
  border: 1px solid var(--color-outline-dim);
  border-radius: 24px;
  padding: 20px 14px;
  overflow-y: auto;
  min-height: 0;
}

.settings-nav-header h1 {
  margin: 0 0 4px;
  padding: 0 10px;
  font-size: 22px;
  font-weight: 800;
  color: var(--color-text);
  letter-spacing: -0.01em;
}

.settings-nav-sub {
  margin: 0;
  padding: 0 10px;
  font-size: 13px;
  color: var(--color-text-dim);
}

.settings-search {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 4px;
  padding: 0 14px;
  background: var(--color-surface-high);
  border: 1px solid transparent;
  border-radius: 999px;
  transition: border-color 0.15s, background 0.15s;
}

.settings-search:focus-within {
  background: var(--color-surface);
  border-color: var(--color-primary);
}

.settings-search > .material-symbols-outlined {
  font-size: 20px;
  color: var(--color-text-dim);
}

.settings-search input {
  flex: 1;
  min-width: 0;
  background: transparent;
  border: 0;
  outline: 0;
  padding: 11px 0;
  font-size: 14px;
  color: var(--color-text);
}

.settings-search input::placeholder { color: var(--color-text-dim); }

.settings-search-clear {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 0;
  background: transparent;
  display: grid;
  place-items: center;
  color: var(--color-text-dim);
  cursor: pointer;
}

.settings-search-clear:hover { background: var(--color-surface); color: var(--color-text); }
.settings-search-clear .material-symbols-outlined { font-size: 16px; }

.settings-sections {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
}

.settings-group-label {
  margin: 12px 14px 4px;
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-text-dim);
  font-weight: 700;
}

.settings-nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background: transparent;
  border: 0;
  border-radius: 16px;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s;
  position: relative;
}

.settings-nav-item:hover {
  background: var(--color-surface);
}

.settings-nav-item.active {
  background: var(--color-primary-container);
}

.settings-nav-item.active .nav-title { color: var(--color-on-primary-container); }
.settings-nav-item.active .nav-desc { color: color-mix(in oklch, var(--color-on-primary-container) 70%, transparent); }

.nav-icon {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}

.nav-icon[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.nav-icon[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.nav-icon[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }
.nav-icon[data-tone="error"]     { --tone-bg: var(--color-error-container);     --tone-fg: var(--color-on-error-container); }

.nav-icon .material-symbols-outlined { font-size: 22px; }

.nav-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  line-height: 1.2;
}

.nav-desc {
  font-size: 12px;
  color: var(--color-text-dim);
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nav-chevron {
  font-size: 18px;
  color: var(--color-text-dim);
  opacity: 0.7;
}

.settings-nav-footer {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  background: var(--color-surface);
  border: 0;
  border-radius: 16px;
  font-size: 12px;
  color: var(--color-text-dim);
  cursor: pointer;
  width: 100%;
  text-align: left;
  transition: background 0.15s, transform 0.1s;
}

.settings-nav-footer:hover {
  background: var(--color-surface-high);
}

.settings-nav-footer:active { transform: scale(0.99); }

.settings-nav-footer > div { flex: 1; min-width: 0; }

.settings-nav-footer .material-symbols-outlined {
  color: var(--color-primary);
  font-size: 22px;
}

.settings-nav-footer .footer-chev {
  color: var(--color-text-dim);
  opacity: 0.6;
  font-size: 18px;
}

.footer-name { font-weight: 700; color: var(--color-text); }
.footer-version { color: var(--color-text-dim); margin-top: 1px; }

/* ── Правая колонка ───────────────────────────────────────────── */
.settings-pane {
  display: flex;
  flex-direction: column;
  background: var(--color-surface-low);
  border: 1px solid var(--color-outline-dim);
  border-radius: 24px;
  min-height: 0;
  overflow: hidden;
}

.settings-pane-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 24px;
  border-bottom: 1px solid var(--color-outline-dim);
}

.settings-back {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: 0;
  background: transparent;
  color: var(--color-text);
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: background 0.15s;
}

.settings-back:hover { background: var(--color-surface); }

.pane-title-wrap { min-width: 0; }

.pane-title {
  margin: 0;
  font-size: 22px;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--color-text);
  line-height: 1.2;
}

.pane-sub {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--color-text-dim);
}

.settings-pane-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 24px;
}

.pane-block {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 880px;
}

/* ── Toolbar inside section ───────────────────────────────────── */
.pane-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.pane-toolbar-hint {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-dim);
  flex: 1;
  min-width: 200px;
}

.search-wrapper {
  flex: 1;
  min-width: 200px;
  max-width: 360px;
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 14px;
  font-size: 18px;
  color: var(--color-text-dim);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 11px 14px 11px 40px;
  border: 1px solid transparent;
  border-radius: 999px;
  font-size: 14px;
  background: var(--color-surface-high);
  color: var(--color-text);
  outline: none;
  transition: border-color 0.15s, background 0.15s;
}

.search-input:focus {
  border-color: var(--color-primary);
  background: var(--color-surface);
}

/* ── Кнопки M3 ─────────────────────────────────────────────────── */
.btn-filled, .btn-filled-tonal, .btn-outlined, .btn-text {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 22px;
  border-radius: 999px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.15s, color 0.15s, border-color 0.15s, box-shadow 0.15s;
  border: 1px solid transparent;
}

.btn-filled {
  background: var(--color-primary);
  color: var(--color-on-primary);
}
.btn-filled:hover:not(:disabled) {
  background: color-mix(in oklch, var(--color-primary) 88%, var(--color-on-primary) 12%);
}
.btn-filled:disabled { opacity: 0.55; cursor: not-allowed; }
.btn-filled .material-symbols-outlined { font-size: 18px; }

.btn-filled-tonal {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.btn-filled-tonal:hover {
  background: color-mix(in oklch, var(--color-secondary-container) 80%, var(--color-on-secondary-container) 20%);
}
.btn-filled-tonal .material-symbols-outlined { font-size: 18px; }

.btn-outlined {
  background: transparent;
  color: var(--color-primary);
  border-color: var(--color-outline);
}
.btn-outlined:hover {
  background: color-mix(in oklch, var(--color-primary) 8%, transparent);
}
.btn-outlined.danger {
  color: var(--color-error);
  border-color: color-mix(in oklch, var(--color-error) 40%, var(--color-outline-dim));
}
.btn-outlined.danger:hover {
  background: color-mix(in oklch, var(--color-error) 8%, transparent);
}
.btn-outlined .material-symbols-outlined { font-size: 18px; }

.btn-text {
  background: transparent;
  color: var(--color-primary);
  padding: 10px 18px;
}
.btn-text:hover {
  background: color-mix(in oklch, var(--color-primary) 8%, transparent);
}

/* ── Карточки настроек ──────────────────────────────────────── */
.settings-card {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 20px 22px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: 20px;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.settings-card:hover {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--color-outline-dim));
}

.settings-card--hero {
  padding: 24px;
  border-radius: 24px;
}

.settings-card--danger {
  border-color: color-mix(in oklch, var(--color-error) 28%, var(--color-outline-dim));
}

.hero-icon {
  flex-shrink: 0;
  width: 56px;
  height: 56px;
  border-radius: 16px;
  display: grid;
  place-items: center;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}

.hero-icon[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.hero-icon[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.hero-icon[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }
.hero-icon[data-tone="error"]     { --tone-bg: var(--color-error-container);     --tone-fg: var(--color-on-error-container); }

.hero-icon .material-symbols-outlined { font-size: 28px; }

.card-text { flex: 1; min-width: 0; }

.card-text h3 {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text);
}

.card-text p {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-text-dim);
}

.card-text b { color: var(--color-text); }

.card-actions {
  flex-shrink: 0;
}

/* ── Users grid ─────────────────────────────────────────────── */
.users-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}

.user-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: 18px;
  transition: border-color 0.15s, transform 0.15s;
  position: relative;
}

.user-card:hover {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--color-outline-dim));
}

.user-card.is-me {
  background: var(--color-primary-container);
  border-color: color-mix(in oklch, var(--color-primary) 50%, transparent);
}
.user-card.is-me .user-card-name { color: var(--color-on-primary-container); }

.user-card-avatar {
  width: 52px;
  height: 52px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  border: 2px solid var(--color-surface-low);
}

.user-card-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.user-card-name {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-card-login {
  font-size: 12px;
  color: var(--color-text-dim);
}

.user-card-post {
  font-size: 12px;
  color: var(--color-text-dim);
  font-style: italic;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-card-role {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  margin-top: 6px;
  padding: 4px 12px 4px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  align-self: flex-start;
  /* Текст роли всегда виден целиком — не обрезаем; если родитель уже —
     плашка перенесёт всё на новую строку. */
  max-width: 100%;
  white-space: nowrap;
  line-height: 1.4;
}

.user-card-role .material-symbols-outlined { font-size: 14px; flex-shrink: 0; }

.user-card-role[data-level="2"] {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.user-card-role[data-level="3"] {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.user-card-role[data-level="4"] {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.user-card-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

/* ── Chip list (отделы/типы) ───────────────────────────────── */
.chip-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.chip-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: 14px;
  transition: border-color 0.15s, background 0.15s;
}

.chip-row:hover { border-color: color-mix(in oklch, var(--color-primary) 30%, var(--color-outline-dim)); }

.chip-row.editing {
  background: var(--color-primary-container);
  border-color: var(--color-primary);
}

.chip-row.editing .chip-icon { color: var(--color-on-primary-container); }

.chip-icon {
  font-size: 20px;
  color: var(--color-text-dim);
}

.chip-name {
  flex: 1;
  font-size: 14px;
  color: var(--color-text);
  font-weight: 500;
}

.chip-input {
  flex: 1;
  background: transparent;
  border: 0;
  outline: 0;
  padding: 4px 0;
  font-size: 14px;
  color: var(--color-text);
}

/* ── Icon buttons ───────────────────────────────────────────── */
.row-actions {
  display: flex;
  gap: 4px;
}

.icon-btn {
  width: 36px;
  height: 36px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  cursor: pointer;
  display: grid;
  place-items: center;
  color: var(--color-text-dim);
  transition: background 0.15s, color 0.15s;
}

.icon-btn:hover {
  background: var(--color-surface-high);
  color: var(--color-text);
}

.icon-btn.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

.icon-btn.success:hover {
  background: var(--color-success-container);
  color: var(--color-on-success-container);
}

.icon-btn .material-symbols-outlined { font-size: 18px; }

/* ── Empty state ────────────────────────────────────────────── */
.settings-empty {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 40px 20px;
  text-align: center;
}

.settings-empty .empty-icon {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
  margin-bottom: 4px;
}

.settings-empty .empty-icon[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.settings-empty .empty-icon[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.settings-empty .empty-icon[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }

.settings-empty .empty-icon .material-symbols-outlined { font-size: 30px; }

.settings-empty h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text);
}

.settings-empty p {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-dim);
  max-width: 320px;
  line-height: 1.5;
}

/* ── Dialog form ────────────────────────────────────────────── */
.dialog-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
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

.w-full { width: 100%; }

.error-msg {
  margin: 0;
  font-size: 13px;
  color: var(--color-on-error-container);
  padding: 10px 14px;
  background: var(--color-error-container);
  border-radius: 12px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 8px;
}

/* ── Pane title icon (в шапке секции) ───────────────────────── */
.pane-title-icon {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: 14px;
  display: grid;
  place-items: center;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}

.pane-title-icon[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.pane-title-icon[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.pane-title-icon[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }
.pane-title-icon[data-tone="error"]     { --tone-bg: var(--color-error-container);     --tone-fg: var(--color-on-error-container); }

.pane-title-icon .material-symbols-outlined { font-size: 24px; }

/* ── Empty state в навигации (пустой поиск) ─────────────────── */
.settings-nav-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 28px 16px;
  text-align: center;
  color: var(--color-text-dim);
}

.settings-nav-empty .material-symbols-outlined {
  font-size: 32px;
  opacity: 0.5;
}

.settings-nav-empty p {
  margin: 0;
  font-size: 13px;
}

/* ── Transition между секциями (десктоп) ────────────────────── */
.pane-swap-enter-active, .pane-swap-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}
.pane-swap-enter-from { opacity: 0; transform: translateX(8px); }
.pane-swap-leave-to   { opacity: 0; transform: translateX(-8px); }

/* ── Adaptive: планшет 1024px ───────────────────────────────── */
@media (max-width: 1100px) {
  .settings-shell {
    grid-template-columns: 280px 1fr;
    padding: 16px;
    gap: 16px;
  }
  .nav-desc { display: none; }
  .settings-nav-item { padding: 12px 12px; }
  .settings-pane-header { padding: 16px 20px; }
  .settings-pane-body { padding: 20px; }
}

/* ── Adaptive: 768-880px — узкий десктоп, sidebar становится rail ─ */
@media (max-width: 900px) and (min-width: 769px) {
  .settings-shell {
    grid-template-columns: 88px 1fr;
    gap: 12px;
  }
  .settings-nav { padding: 14px 8px; }
  .settings-nav-header,
  .settings-search,
  .settings-group-label,
  .settings-nav-footer,
  .nav-text,
  .nav-chevron { display: none; }
  .settings-nav-item {
    padding: 8px;
    justify-content: center;
  }
  .settings-nav-item .nav-icon {
    width: 48px;
    height: 48px;
    border-radius: 14px;
  }
  .settings-nav-item .nav-icon .material-symbols-outlined { font-size: 24px; }
  .settings-nav-item.active .nav-icon {
    box-shadow: 0 0 0 3px var(--color-primary);
  }
}

/* ── Adaptive: мобильный ≤768 ───────────────────────────────── */
@media (max-width: 768px) {
  .settings-shell {
    grid-template-columns: 1fr;
    padding: 0;
    gap: 0;
    height: auto;
    min-height: 100%;
    overflow: visible;
    max-width: 100%;
    padding-bottom: calc(60px + 12px + env(safe-area-inset-bottom, 0px));
  }

  /* Когда выбрана секция — список секций прячется, контент секции
     становится full-screen, шапка липкая. */
  .settings-nav.mobile-hidden { display: none; }

  .settings-nav {
    padding: 16px 12px;
    border-radius: 0;
    border: 0;
    background: transparent;
    overflow: visible;
  }

  .settings-nav-header h1 { font-size: 22px; }

  .settings-sections { gap: 4px; }

  /* На мобильном — карточки секций крупнее и видимее */
  .settings-nav-item {
    background: var(--color-surface);
    border: 1px solid var(--color-outline-dim);
    padding: 14px 14px;
    border-radius: 18px;
    min-height: 64px;
  }

  .settings-nav-item:active {
    background: var(--color-surface-high);
    transform: scale(0.985);
  }

  .nav-desc {
    display: block;
    white-space: normal;
    overflow: visible;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    display: -webkit-box;
    -webkit-box-orient: vertical;
  }

  .settings-pane.mobile-full {
    position: fixed;
    inset: 0;
    z-index: 90;
    background: var(--color-bg);
    border-radius: 0;
    border: 0;
    display: flex;
    flex-direction: column;
    /* Учитываем нижнюю навигацию */
    padding-bottom: calc(60px + env(safe-area-inset-bottom, 0px));
  }

  .settings-pane.mobile-full .settings-pane-header {
    position: sticky;
    top: 0;
    z-index: 2;
    padding: 12px 12px 12px;
    background: var(--color-bg);
    border-bottom: 1px solid var(--color-outline-dim);
    padding-top: calc(12px + env(safe-area-inset-top, 0px));
  }

  .settings-pane.mobile-full .pane-title-icon {
    width: 40px;
    height: 40px;
    border-radius: 12px;
  }

  .settings-pane.mobile-full .pane-title-icon .material-symbols-outlined { font-size: 22px; }

  .pane-title { font-size: 18px; }
  .pane-sub { font-size: 12px; }

  .settings-pane.mobile-full .settings-pane-body {
    padding: 16px 12px 24px;
    -webkit-overflow-scrolling: touch;
  }

  .users-grid { grid-template-columns: 1fr; }

  .settings-card,
  .settings-card--hero {
    flex-direction: column;
    align-items: flex-start;
    text-align: left;
    padding: 18px;
    gap: 12px;
    border-radius: 20px;
  }

  .settings-card .card-actions { width: 100%; }

  .settings-card .card-actions .btn-filled,
  .settings-card .card-actions .btn-filled-tonal,
  .settings-card .card-actions .btn-outlined {
    width: 100%;
    justify-content: center;
  }

  /* Toolbar на мобильном: hint выше кнопки */
  .pane-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .pane-toolbar .btn-filled {
    width: 100%;
    justify-content: center;
  }

  .search-wrapper { max-width: 100%; }

  /* User card: компактный однострочный layout на мобильном.
     actions справа в столбик икон, чтобы не уезжали и не делали карточку
     слишком высокой; имя/логин/роль занимают центр. */
  .user-card {
    padding: 12px;
    border-radius: 16px;
    gap: 10px;
    align-items: center;
  }

  .user-card-avatar { width: 44px; height: 44px; }

  .user-card-name { font-size: 13px; }
  .user-card-login,
  .user-card-post { font-size: 11px; }

  .user-card-role {
    padding: 3px 10px 3px 6px;
    font-size: 10px;
    margin-top: 4px;
  }

  .user-card-role .material-symbols-outlined { font-size: 12px; }

  .user-card-actions {
    flex-direction: column;
    gap: 2px;
  }

  .user-card-actions .icon-btn { width: 32px; height: 32px; }
  .user-card-actions .icon-btn .material-symbols-outlined { font-size: 16px; }

  /* Chip-row: тоже растягиваем на всю ширину */
  .chip-row {
    padding: 12px 14px;
  }

  :deep(.p-dialog) {
    width: 95vw !important;
    max-width: 95vw !important;
  }
}

/* ── Adaptive: очень узкий мобильный <380 ───────────────────── */
@media (max-width: 380px) {
  .pane-title { font-size: 17px; }
  .pane-sub { display: none; }
  .settings-pane.mobile-full .pane-title-icon { width: 36px; height: 36px; }
}
</style>
