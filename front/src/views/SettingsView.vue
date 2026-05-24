<template>
  <div class="settings-view">
    <h1>Настройки</h1>

    <Tabs v-model:value="activeTab">
      <TabList>
        <Tab value="theme" data-tutorial="settings-tab-theme">Персонализация</Tab>
        <Tab v-if="isAtLeast(ROLES.ADMIN)" value="users">Пользователи</Tab>
        <Tab
          v-if="isAtLeast(ROLES.EMPLOYEE)"
          value="lists"
        >Списки</Tab>
        <Tab v-if="isAtLeast(ROLES.SUPERADMIN)" value="backup">Копирование</Tab>
      </TabList>

      <TabPanels>
        <!-- Персонализация -->
        <TabPanel value="theme">
          <ThemeBuilder />

          <div class="tutorial-card">
            <div class="tutorial-card-info">
              <span class="material-symbols-outlined tutorial-icon">school</span>
              <div>
                <h4>Обучение</h4>
                <p>Повторно пройдите интерактивное знакомство с платформой.</p>
              </div>
            </div>
            <button class="btn-primary" @click="tutorial.open()">
              <span class="material-symbols-outlined">play_circle</span>
              Пройти обучение
            </button>
          </div>

          <div class="tutorial-card">
            <div class="tutorial-card-info">
              <span class="material-symbols-outlined tutorial-icon">newsmode</span>
              <div>
                <h4>Что нового</h4>
                <p>История обновлений платформы.<template v-if="appVersion"> Текущая версия: {{ appVersion }}.</template></p>
              </div>
            </div>
            <button class="btn-primary" @click="changelog.open()">
              <span class="material-symbols-outlined">newsmode</span>
              История версий
            </button>
          </div>
        </TabPanel>

        <!-- Пользователи -->
        <TabPanel value="users">
          <div class="panel-toolbar">
            <div class="search-wrapper">
              <span class="material-symbols-outlined search-icon">search</span>
              <input
                v-model="userSearch"
                class="search-input"
                placeholder="Поиск по ФИО..."
                @input="onUserSearch"
              />
            </div>
            <button
              class="btn-primary"
              @click="openUserDialog(null)"
            >
              <span class="material-symbols-outlined">person_add</span>
              Создать пользователя
            </button>
          </div>

          <div class="table-scroll">
          <DataTable :value="users" :loading="usersLoading" size="small" class="settings-table">
            <Column header="Аватар" style="width:56px">
              <template #body="{ data }">
                <img :src="getUserAvatar(data)" class="user-avatar-sm" :alt="data.fio" />
              </template>
            </Column>
            <Column field="fio" header="ФИО" />
            <Column field="login" header="Логин" />
            <Column field="post" header="Должность" />
            <Column header="Роль">
              <template #body="{ data }">
                {{ data.role?.name || '—' }}
              </template>
            </Column>
            <Column header="" style="width:100px">
              <template #body="{ data }">
                <div class="row-actions">
                  <button
                    class="icon-btn"
                    title="Редактировать"
                    @click="openUserDialog(data)"
                  >
                    <span class="material-symbols-outlined">edit</span>
                  </button>
                  <button
                    class="icon-btn danger"
                    title="Удалить"
                    @click="confirmDeleteUser(data)"
                  >
                    <span class="material-symbols-outlined">delete</span>
                  </button>
                </div>
              </template>
            </Column>
          </DataTable>
          </div>

          <!-- Диалог создания/редактирования пользователя -->
          <Dialog
            v-model:visible="showUserDialog"
            :header="editingUser ? 'Редактирование пользователя' : 'Создать пользователя'"
            modal
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
                <button type="button" class="btn-secondary" @click="showUserDialog = false">Отмена</button>
                <button type="submit" class="btn-primary" :disabled="userFormLoading">
                  {{ userFormLoading ? 'Сохраняем...' : 'Сохранить' }}
                </button>
              </div>
            </form>
          </Dialog>

          <!-- Подтверждение удаления пользователя -->
          <ConfirmDialog
            :visible="!!deletingUser"
            header="Удалить пользователя"
            :message="`Удалить пользователя «${deletingUser?.fio}»? Данные задач и юнитов сохраняются.`"
            confirm-label="Удалить"
            :danger-confirm="true"
            @confirm="doDeleteUser"
            @cancel="deletingUser = null"
          />
        </TabPanel>

        <!-- Списки (отделы и типы юнитов) -->
        <TabPanel value="lists">
          <Tabs v-model:value="listsTab">
            <TabList>
              <Tab value="departments">Отделы</Tab>
              <Tab value="unit-types">Типы юнитов</Tab>
            </TabList>
            <TabPanels>
              <!-- Отделы -->
              <TabPanel value="departments">
                <div class="panel-toolbar">
                  <h4 class="panel-title">Отделы</h4>
                  <button
                    v-if="isAtLeast(ROLES.MANAGER)"
                    class="btn-primary"
                    @click="startAddDept"
                  >
                    <span class="material-symbols-outlined">add</span>
                    Добавить
                  </button>
                </div>

                <div class="inline-list">
                  <!-- Новая строка -->
                  <div v-if="addingDept" class="inline-list-row editing">
                    <InputText v-model="newDeptName" placeholder="Название отдела" class="flex-input" @keyup.enter="saveDept" @keyup.escape="addingDept = false" />
                    <button class="icon-btn success" @click="saveDept" title="Сохранить">
                      <span class="material-symbols-outlined">check</span>
                    </button>
                    <button class="icon-btn" @click="addingDept = false" title="Отмена">
                      <span class="material-symbols-outlined">close</span>
                    </button>
                  </div>

                  <div v-for="dept in departments" :key="dept.id" class="inline-list-row">
                    <template v-if="editingDeptId === dept.id">
                      <InputText v-model="editingDeptName" class="flex-input" @keyup.enter="updateDept(dept)" @keyup.escape="editingDeptId = null" />
                      <button class="icon-btn success" @click="updateDept(dept)" title="Сохранить">
                        <span class="material-symbols-outlined">check</span>
                      </button>
                      <button class="icon-btn" @click="editingDeptId = null" title="Отмена">
                        <span class="material-symbols-outlined">close</span>
                      </button>
                    </template>
                    <template v-else>
                      <span class="list-item-name">{{ dept.name }}</span>
                      <div v-if="isAtLeast(ROLES.MANAGER)" class="row-actions">
                        <button
                          class="icon-btn"
                          title="Редактировать"
                          @click="startEditDept(dept)"
                        >
                          <span class="material-symbols-outlined">edit</span>
                        </button>
                        <button
                          class="icon-btn danger"
                          title="Удалить"
                          @click="confirmDeleteDept(dept)"
                        >
                          <span class="material-symbols-outlined">delete</span>
                        </button>
                      </div>
                    </template>
                  </div>
                  <div v-if="departments.length === 0 && !addingDept" class="empty-inline">Отделы не добавлены</div>
                </div>

                <ConfirmDialog
                  :visible="!!deletingDept"
                  header="Удалить отдел"
                  :message="`Удалить отдел «${deletingDept?.name}»?`"
                  confirm-label="Удалить"
                  :danger-confirm="true"
                  @confirm="doDeleteDept"
                  @cancel="deletingDept = null"
                />
              </TabPanel>

              <!-- Типы юнитов -->
              <TabPanel value="unit-types">
                <div class="panel-toolbar">
                  <h4 class="panel-title">Типы юнитов</h4>
                  <button
                    v-if="isAtLeast(ROLES.MANAGER)"
                    class="btn-primary"
                    @click="startAddUnitType"
                  >
                    <span class="material-symbols-outlined">add</span>
                    Добавить
                  </button>
                </div>

                <div class="inline-list">
                  <!-- Новая строка -->
                  <div v-if="addingUnitType" class="inline-list-row editing">
                    <InputText v-model="newUnitTypeName" placeholder="Название типа" class="flex-input" @keyup.enter="saveUnitType" @keyup.escape="addingUnitType = false" />
                    <button class="icon-btn success" @click="saveUnitType" title="Сохранить">
                      <span class="material-symbols-outlined">check</span>
                    </button>
                    <button class="icon-btn" @click="addingUnitType = false" title="Отмена">
                      <span class="material-symbols-outlined">close</span>
                    </button>
                  </div>

                  <div v-for="ut in unitTypes" :key="ut.id" class="inline-list-row">
                    <template v-if="editingUnitTypeId === ut.id">
                      <InputText v-model="editingUnitTypeName" class="flex-input" @keyup.enter="updateUnitType(ut)" @keyup.escape="editingUnitTypeId = null" />
                      <button class="icon-btn success" @click="updateUnitType(ut)" title="Сохранить">
                        <span class="material-symbols-outlined">check</span>
                      </button>
                      <button class="icon-btn" @click="editingUnitTypeId = null" title="Отмена">
                        <span class="material-symbols-outlined">close</span>
                      </button>
                    </template>
                    <template v-else>
                      <span class="list-item-name">{{ ut.name }}</span>
                      <div v-if="isAtLeast(ROLES.MANAGER)" class="row-actions">
                        <button
                          class="icon-btn"
                          title="Редактировать"
                          @click="startEditUnitType(ut)"
                        >
                          <span class="material-symbols-outlined">edit</span>
                        </button>
                        <button
                          class="icon-btn danger"
                          title="Удалить"
                          @click="confirmDeleteUnitType(ut)"
                        >
                          <span class="material-symbols-outlined">delete</span>
                        </button>
                      </div>
                    </template>
                  </div>
                  <div v-if="unitTypes.length === 0 && !addingUnitType" class="empty-inline">Типы юнитов не добавлены</div>
                </div>

                <ConfirmDialog
                  :visible="!!deletingUnitType"
                  header="Удалить тип юнита"
                  :message="`Удалить тип «${deletingUnitType?.name}»? Все юниты этого типа будут удалены безвозвратно.`"
                  confirm-label="Удалить"
                  :danger-confirm="true"
                  @confirm="doDeleteUnitType"
                  @cancel="deletingUnitType = null"
                />
              </TabPanel>
            </TabPanels>
          </Tabs>
        </TabPanel>

        <!-- Копирование -->
        <TabPanel value="backup">
          <div class="backup-panel">
            <div class="backup-card">
              <div class="backup-card-info">
                <span class="material-symbols-outlined backup-icon">backup</span>
                <div>
                  <h4>Резервная копия</h4>
                  <p>Скачать полную резервную копию базы данных в формате JSON.</p>
                </div>
              </div>
              <button class="btn-primary" @click="doExportBackup" :disabled="backupExporting">
                <span class="material-symbols-outlined">download</span>
                {{ backupExporting ? 'Создание...' : 'Создать резервную копию' }}
              </button>
            </div>

            <div class="backup-card">
              <div class="backup-card-info">
                <span class="material-symbols-outlined backup-icon">restore</span>
                <div>
                  <h4>Восстановление</h4>
                  <p>Восстановить систему из резервной копии. Текущие данные будут заменены.</p>
                </div>
              </div>
              <label class="btn-secondary file-btn">
                <span class="material-symbols-outlined">upload</span>
                Выбрать файл
                <input type="file" accept=".zip" @change="onImportFileSelect" style="display:none" />
              </label>
            </div>
          </div>

          <!-- Двойное подтверждение импорта -->
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
        </TabPanel>
      </TabPanels>
    </Tabs>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useTutorial } from '@/composables/useTutorial.js'
import { useChangelog } from '@/composables/useChangelog.js'
import { version as appVersion } from '../../package.json'
import {
  getUsers, createUser, updateUser, deleteUser, assignRole
} from '@/api/users.js'
import { getRoles } from '@/api/roles.js'
import {
  getDepartments, createDepartment, updateDepartment, deleteDepartment
} from '@/api/departments.js'
import {
  getUnitTypes, createUnitType, updateUnitType as apiUpdateUnitType, deleteUnitType
} from '@/api/unitTypes.js'
import { exportBackup, importBackup } from '@/api/backup.js'
import ThemeBuilder from '@/components/settings/ThemeBuilder.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Tabs from 'primevue/tabs'
import TabList from 'primevue/tablist'
import Tab from 'primevue/tab'
import TabPanels from 'primevue/tabpanels'
import TabPanel from 'primevue/tabpanel'
import Dialog from 'primevue/dialog'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import ProgressSpinner from 'primevue/progressspinner'

const { isAtLeast, myLevel } = usePermission()
const notif = useNotificationsStore()
const tutorial = useTutorial()
const changelog = useChangelog()

const activeTab = ref('theme')
const listsTab = ref('departments')

// ---- Пользователи ----
const users = ref([])
const usersLoading = ref(false)
const userSearch = ref('')
const showUserDialog = ref(false)
const editingUser = ref(null)
const deletingUser = ref(null)
const userFormLoading = ref(false)
const userFormError = ref('')
const userForm = reactive({ fio: '', login: '', password: '', post: '', role_id: null })

// ---- Роли (только для dropdown) ----
const roles = ref([])

const assignableRoles = computed(() => {
  const level = myLevel()
  // Суперадмин может назначить любую роль ниже своей
  // Админ может назначить только сотрудника и менеджера
  return roles.value.filter(r => r.level < level)
})

let userSearchTimer = null

function onUserSearch() {
  clearTimeout(userSearchTimer)
  userSearchTimer = setTimeout(() => loadUsers(), 400)
}

async function loadUsers() {
  usersLoading.value = true
  try {
    users.value = await getUsers(userSearch.value)
  } catch (e) {
    notif.error(e.message || 'Ошибка загрузки пользователей')
  } finally {
    usersLoading.value = false
  }
}

async function loadRoles() {
  try {
    roles.value = await getRoles()
  } catch (e) {
    notif.error(e.message || 'Ошибка загрузки ролей')
  }
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
      role_id: user.role?.id || null
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
      const payload = {
        fio: userForm.fio.trim(),
        login: userForm.login.trim(),
        post: userForm.post.trim()
      }
      await updateUser(editingUser.value.id, payload)
      if (userForm.role_id && userForm.role_id !== editingUser.value.role?.id) {
        await assignRole(editingUser.value.id, { role_id: userForm.role_id })
      }
      notif.success('Пользователь обновлён')
    } else {
      const payload = {
        fio: userForm.fio.trim(),
        login: userForm.login.trim(),
        post: userForm.post.trim(),
        password: userForm.password,
        role_id: userForm.role_id
      }
      await createUser(payload)
      notif.success('Пользователь создан')
    }
    showUserDialog.value = false
    loadUsers()
  } catch (e) {
    userFormError.value = e.message || 'Ошибка сохранения'
  } finally {
    userFormLoading.value = false
  }
}

function confirmDeleteUser(user) {
  deletingUser.value = user
}

async function doDeleteUser() {
  if (!deletingUser.value) return
  try {
    await deleteUser(deletingUser.value.id)
    notif.success('Пользователь удалён')
    users.value = users.value.filter(u => u.id !== deletingUser.value.id)
  } catch (e) {
    notif.error(e.message || 'Ошибка удаления')
  } finally {
    deletingUser.value = null
  }
}

// ---- Отделы ----
const departments = ref([])
const addingDept = ref(false)
const newDeptName = ref('')
const editingDeptId = ref(null)
const editingDeptName = ref('')
const deletingDept = ref(null)

async function loadDepartments() {
  try {
    departments.value = await getDepartments()
  } catch (e) {
    notif.error(e.message || 'Ошибка загрузки отделов')
  }
}

function startAddDept() {
  addingDept.value = true
  newDeptName.value = ''
}

async function saveDept() {
  if (!newDeptName.value.trim()) return
  try {
    await createDepartment({ name: newDeptName.value.trim() })
    notif.success('Отдел создан')
    addingDept.value = false
    loadDepartments()
  } catch (e) {
    notif.error(e.message || 'Ошибка создания отдела')
  }
}

function startEditDept(dept) {
  editingDeptId.value = dept.id
  editingDeptName.value = dept.name
}

async function updateDept(dept) {
  if (!editingDeptName.value.trim()) return
  try {
    await updateDepartment(dept.id, { name: editingDeptName.value.trim() })
    notif.success('Отдел обновлён')
    editingDeptId.value = null
    loadDepartments()
  } catch (e) {
    notif.error(e.message || 'Ошибка обновления')
  }
}

function confirmDeleteDept(dept) {
  deletingDept.value = dept
}

async function doDeleteDept() {
  if (!deletingDept.value) return
  try {
    await deleteDepartment(deletingDept.value.id)
    notif.success('Отдел удалён')
    departments.value = departments.value.filter(d => d.id !== deletingDept.value.id)
  } catch (e) {
    notif.error(e.message || 'Ошибка удаления')
  } finally {
    deletingDept.value = null
  }
}

// ---- Типы юнитов ----
const unitTypes = ref([])
const addingUnitType = ref(false)
const newUnitTypeName = ref('')
const editingUnitTypeId = ref(null)
const editingUnitTypeName = ref('')
const deletingUnitType = ref(null)

async function loadUnitTypes() {
  try {
    unitTypes.value = await getUnitTypes()
  } catch (e) {
    notif.error(e.message || 'Ошибка загрузки типов юнитов')
  }
}

function startAddUnitType() {
  addingUnitType.value = true
  newUnitTypeName.value = ''
}

async function saveUnitType() {
  if (!newUnitTypeName.value.trim()) return
  try {
    await createUnitType({ name: newUnitTypeName.value.trim() })
    notif.success('Тип юнита создан')
    addingUnitType.value = false
    loadUnitTypes()
  } catch (e) {
    notif.error(e.message || 'Ошибка создания')
  }
}

function startEditUnitType(ut) {
  editingUnitTypeId.value = ut.id
  editingUnitTypeName.value = ut.name
}

async function updateUnitType(ut) {
  if (!editingUnitTypeName.value.trim()) return
  try {
    await apiUpdateUnitType(ut.id, { name: editingUnitTypeName.value.trim() })
    notif.success('Тип юнита обновлён')
    editingUnitTypeId.value = null
    loadUnitTypes()
  } catch (e) {
    notif.error(e.message || 'Ошибка обновления')
  }
}

function confirmDeleteUnitType(ut) {
  deletingUnitType.value = ut
}

async function doDeleteUnitType() {
  if (!deletingUnitType.value) return
  try {
    await deleteUnitType(deletingUnitType.value.id)
    notif.success('Тип юнита удалён')
    unitTypes.value = unitTypes.value.filter(u => u.id !== deletingUnitType.value.id)
  } catch (e) {
    notif.error(e.message || 'Ошибка удаления')
  } finally {
    deletingUnitType.value = null
  }
}

// ---- Backup ----
const backupExporting = ref(false)
const showImportConfirm1 = ref(false)
const showImportConfirm2 = ref(false)
const importFile = ref(null)

async function doExportBackup() {
  backupExporting.value = true
  try {
    const response = await exportBackup()
    let blob
    if (response instanceof Blob) {
      blob = response
    } else if (response && typeof response.blob === 'function') {
      blob = await response.blob()
    } else {
      blob = new Blob([JSON.stringify(response)], { type: 'application/json' })
    }
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `backup_${new Date().toISOString().split('T')[0]}.zip`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    notif.success('Резервная копия создана')
  } catch (e) {
    notif.error(e.message || 'Ошибка создания резервной копии')
  } finally {
    backupExporting.value = false
  }
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
  } catch (e) {
    notif.error(e.message || 'Ошибка восстановления')
  } finally {
    importFile.value = null
  }
}

// ---- Загрузка данных при смене вкладки ----
watch(activeTab, (tab) => {
  if (tab === 'users') { loadUsers(); loadRoles() }
  if (tab === 'lists') { loadDepartments(); loadUnitTypes() }
})

onMounted(() => {
  loadRoles()
  if (isAtLeast(ROLES.ADMIN)) loadUsers()
})
</script>

<style scoped>
.settings-view {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
  height: 100%;
  overflow-y: auto;
}

.settings-view h1 {
  margin: 0 0 20px;
  font-size: 24px;
  font-weight: 800;
  color: var(--gw-text);
}

/* Panel toolbar */
.panel-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.panel-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--gw-text);
}

/* Search */
.search-wrapper {
  flex: 1;
  min-width: 200px;
  max-width: 320px;
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 10px;
  font-size: 18px;
  color: var(--gw-text-secondary);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 8px 12px 8px 36px;
  border: 1px solid var(--gw-border);
  border-radius: 10px;
  font-size: 14px;
  background: var(--gw-bg);
  color: var(--gw-text);
  outline: none;
  transition: border-color 0.15s;
}

.search-input:focus {
  border-color: var(--gw-primary);
}

/* Buttons */
.btn-primary {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--gw-primary);
  color: var(--color-on-primary);
  border: none;
  border-radius: 10px;
  padding: 9px 18px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
  white-space: nowrap;
}

.btn-primary:hover:not(:disabled) {
  background: var(--gw-primary-hover);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary .material-symbols-outlined {
  font-size: 18px;
}

.btn-secondary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 9px 18px;
  border-radius: 10px;
  font-size: 14px;
  cursor: pointer;
  border: 1px solid var(--gw-border);
  background: var(--gw-surface);
  color: var(--gw-text);
  transition: background 0.15s, border-color 0.15s;
  white-space: nowrap;
}

.btn-secondary:hover {
  background: var(--gw-bg);
  border-color: var(--gw-primary);
  color: var(--gw-primary);
}

/* Table */
.settings-table {
  width: 100%;
}

.user-avatar-sm {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  display: block;
  border: 2px solid var(--gw-border);
}

/* Row actions */
.row-actions {
  display: flex;
  gap: 4px;
  justify-content: flex-end;
}

.icon-btn {
  width: 32px;
  height: 32px;
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--gw-text-secondary);
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.icon-btn:hover {
  background: var(--gw-bg);
  color: var(--gw-text);
}

.icon-btn.danger:hover {
  background: var(--color-error-container);
  border-color: color-mix(in oklch, var(--color-error) 30%, var(--color-outline-dim));
  color: var(--color-error);
}

.icon-btn.success:hover {
  background: var(--color-success-container);
  border-color: color-mix(in oklch, var(--color-success) 30%, var(--color-outline-dim));
  color: var(--color-success);
}

.icon-btn .material-symbols-outlined {
  font-size: 16px;
}

/* Dialog */
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

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 8px;
  border-top: 1px solid var(--gw-border);
}

/* Inline list (отделы, типы юнитов) */
.inline-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: 600px;
}

.inline-list-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--gw-bg);
  border-radius: 8px;
  border: 1px solid var(--gw-border);
}

.inline-list-row.editing {
  background: var(--gw-surface);
  border-color: var(--gw-primary);
}

.list-item-name {
  flex: 1;
  font-size: 14px;
  color: var(--gw-text);
}

.flex-input {
  flex: 1;
}

.empty-inline {
  padding: 24px;
  text-align: center;
  color: var(--gw-text-secondary);
  font-size: 14px;
}

.loading-inline {
  display: flex;
  justify-content: center;
  padding: 32px;
}

/* Backup */
.backup-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 600px;
}

.backup-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 20px;
  background: var(--gw-bg);
  border: 1px solid var(--gw-border);
  border-radius: var(--gw-radius);
  flex-wrap: wrap;
}

.backup-card-info {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  flex: 1;
}

.backup-icon {
  font-size: 32px;
  color: var(--gw-primary);
  flex-shrink: 0;
}

.backup-card-info h4 {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 700;
  color: var(--gw-text);
}

.backup-card-info p {
  margin: 0;
  font-size: 13px;
  color: var(--gw-text-secondary);
  line-height: 1.4;
}

.file-btn {
  cursor: pointer;
}

/* Горизонтальный скролл для таблиц */
.table-scroll {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

/* Tutorial card */
.tutorial-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 20px;
  margin-top: 24px;
  max-width: 600px;
  background: var(--gw-bg);
  border: 1px solid var(--gw-border);
  border-radius: var(--gw-radius);
  flex-wrap: wrap;
}

.tutorial-card-info {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  flex: 1;
}

.tutorial-icon {
  font-size: 32px;
  color: var(--gw-primary);
  flex-shrink: 0;
}

.tutorial-card-info h4 {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 700;
  color: var(--gw-text);
}

.tutorial-card-info p {
  margin: 0;
  font-size: 13px;
  color: var(--gw-text-secondary);
  line-height: 1.4;
}

@media (max-width: 768px) {
  .settings-view {
    padding: 12px;
    padding-bottom: calc(60px + 12px + env(safe-area-inset-bottom, 0px));
  }

  .settings-view h1 {
    font-size: 20px;
    margin-bottom: 12px;
  }

  .backup-panel,
  .inline-list {
    max-width: 100%;
  }

  .backup-card {
    flex-direction: column;
    align-items: flex-start;
  }

  .tutorial-card {
    flex-direction: column;
    align-items: flex-start;
    max-width: 100%;
  }

  /* Диалог пользователя — полная ширина */
  :deep(.p-dialog) {
    width: 95vw !important;
    max-width: 95vw !important;
  }
}
</style>
