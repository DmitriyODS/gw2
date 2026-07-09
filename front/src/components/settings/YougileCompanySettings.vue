<template>
  <div class="yg-settings">

    <!-- Шапка: статус интеграции -->
    <div class="settings-card head-card">
      <div class="hero-icon" :data-tone="settings?.enabled ? 'primary' : 'secondary'">
        <span class="material-symbols-outlined">{{ settings?.enabled ? 'integration_instructions' : 'extension' }}</span>
      </div>
      <div class="card-text">
        <h3>Интеграция с YouGile</h3>
        <p v-if="settings?.enabled">
          Включена. Карточки летят в доску
          <b>{{ settings.yg_board_title || '—' }}</b>
          проекта <b>{{ settings.yg_project_title || '—' }}</b>
          компании <b>{{ settings.yg_company_name || '—' }}</b>.
        </p>
        <p v-else>Когда включите — у пользователей появятся кнопки «Создать из YouGile» и «Создать в YouGile».</p>
      </div>
      <div class="card-actions">
        <label class="toggle">
          <input type="checkbox" :checked="settings?.enabled" @change="onToggleEnabled($event)" :disabled="busy || !canEnable" />
          <span>{{ settings?.enabled ? 'Включено' : 'Выключено' }}</span>
        </label>
      </div>
    </div>

    <!-- Шаг 1: подключение админа к YouGile -->
    <div v-if="!yg.status.connected" class="settings-card form-card">
      <div class="row-head">
        <div class="hero-icon" data-tone="tertiary">
          <span class="material-symbols-outlined">key</span>
        </div>
        <div class="card-text">
          <h3>Шаг 1. Войдите в свой YouGile</h3>
          <p>Нужно, чтобы Groove Work мог читать ваши проекты и доски. Пароль не сохраняется.</p>
        </div>
      </div>
      <form class="yg-form" @submit.prevent="onAdminLookup">
        <div class="field">
          <label class="lbl">Логин YouGile (email)</label>
          <input class="ctl" type="email" autocomplete="email"
                 v-model="adminForm.login" :disabled="busy" required />
        </div>
        <div class="field">
          <label class="lbl">Пароль YouGile</label>
          <input class="ctl" type="password" autocomplete="current-password"
                 v-model="adminForm.password" :disabled="busy" required />
        </div>
        <div class="actions">
          <button class="btn-filled" type="submit"
                  :disabled="busy || !adminForm.login || !adminForm.password">
            Получить список компаний
          </button>
        </div>
      </form>

      <!-- Список компаний после успешного lookup -->
      <div v-if="ygCompanies.length" class="picker">
        <div class="picker-lbl">Выберите компанию YouGile:</div>
        <div class="picker-list">
          <button v-for="c in ygCompanies" :key="c.id" class="picker-item"
                  :class="{ active: pickedCompanyId === c.id }"
                  :disabled="busy" @click="pickCompany(c)">
            <span class="material-symbols-outlined">domain</span>
            {{ c.name }}
          </button>
        </div>
      </div>
    </div>

    <!-- Шаг 2: выбор проекта/доски/колонки выполнено -->
    <div v-if="yg.status.connected" class="settings-card form-card">
      <div class="row-head">
        <div class="hero-icon" data-tone="primary">
          <span class="material-symbols-outlined">view_kanban</span>
        </div>
        <div class="card-text">
          <h3>Шаг 2. Где жить карточкам</h3>
          <p>Все новые задачи из Groove Work будут попадать в первую колонку выбранной доски.</p>
        </div>
      </div>

      <div class="yg-form">
        <div class="field">
          <label class="lbl">Проект</label>
          <Select
            :model-value="settings?.yg_project_id || null"
            :options="yg.ygProjects"
            option-label="title"
            option-value="id"
            placeholder="Выберите проект"
            class="w-full"
            :disabled="busy || projectsLoading"
            :loading="projectsLoading"
            filter
            filterPlaceholder="Поиск..."
            show-clear
            @update:model-value="onPickProject"
          />
        </div>
        <div class="field">
          <label class="lbl">Доска</label>
          <Select
            :model-value="settings?.yg_board_id || null"
            :options="yg.ygBoards"
            option-label="title"
            option-value="id"
            placeholder="Выберите доску"
            class="w-full"
            :disabled="busy || boardsLoading || !settings?.yg_project_id"
            :loading="boardsLoading"
            filter
            filterPlaceholder="Поиск..."
            show-clear
            @update:model-value="onPickBoard"
          />
          <div v-if="settings?.yg_first_column_id" class="hint">
            Новые задачи → первая колонка доски.
          </div>
        </div>
        <div class="field">
          <label class="lbl">Колонка для «выполнено» (необязательно)</label>
          <Select
            :model-value="settings?.yg_completed_column_id || null"
            :options="yg.ygColumns"
            option-label="title"
            option-value="id"
            placeholder="Не использовать"
            class="w-full"
            :disabled="busy || columnsLoading || !settings?.yg_board_id"
            :loading="columnsLoading"
            filter
            filterPlaceholder="Поиск..."
            show-clear
            @update:model-value="onPickCompleted"
          />
          <div class="hint">
            Если задана, при архивации задачи в Groove Work карточка в YouGile
            переезжает сюда.
          </div>
        </div>
      </div>
    </div>

    <p v-if="!canEnable && yg.status.connected" class="warn">
      Чтобы включить интеграцию, выберите компанию, проект и доску.
    </p>

    <!-- Сброс интеграции «начать заново» -->
    <div v-if="yg.status.connected || settings?.yg_company_id" class="settings-card danger-card">
      <div class="hero-icon" data-tone="error">
        <span class="material-symbols-outlined">logout</span>
      </div>
      <div class="card-text">
        <h3>Выйти из аккаунта и сбросить</h3>
        <p>
          Отключит webhook, очистит выбор компании, проекта и доски и отвяжет ваш
          личный YouGile-аккаунт. Карточки в YouGile не удаляются. После сброса
          настройку можно начать заново.
        </p>
      </div>
      <div class="card-actions">
        <button class="btn-outlined danger" :disabled="busy" @click="showReset = true">
          <span class="material-symbols-outlined">logout</span>
          Выйти и сбросить
        </button>
      </div>
    </div>

    <ConfirmDialog
      :visible="showReset"
      header="Сбросить интеграцию YouGile"
      message="Будут сброшены настройки интеграции компании и отвязан ваш личный YouGile-аккаунт. Связи существующих задач с карточками сохранятся, но создавать новые карточки будет нельзя, пока интеграцию не настроят заново. Это действие необратимо."
      confirm-label="Выйти и сбросить"
      danger-confirm
      @confirm="onReset"
      @cancel="showReset = false"
    />
  </div>
</template>

<script setup>
import { reactive, ref, computed, onMounted } from 'vue'
import Select from 'primevue/select'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { useYougileStore } from '@/stores/yougile.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const yg = useYougileStore()
const notif = useNotificationsStore()

const adminForm = reactive({ login: '', password: '' })
const ygCompanies = computed(() => yg.ygCompanies)
const pickedCompanyId = ref(null)

const busy = ref(false)
const showReset = ref(false)
const projectsLoading = ref(false)
const boardsLoading = ref(false)
const columnsLoading = ref(false)

const settings = computed(() => yg.companySettings)

const canEnable = computed(() => !!(
  settings.value?.yg_company_id && settings.value?.yg_project_id && settings.value?.yg_board_id
))

async function onAdminLookup() {
  busy.value = true
  try {
    await yg.lookupCompanies({ login: adminForm.login.trim(), password: adminForm.password })
  } catch (e) {
    notif.error(e?.data?.message || 'Не удалось получить список')
  } finally {
    busy.value = false
  }
}

async function pickCompany(c) {
  busy.value = true
  try {
    // Подключаем админа — он же подключает и свой личный YG-аккаунт.
    await yg.connect({
      login: adminForm.login.trim(),
      password: adminForm.password,
      yg_company_id: c.id,
    })
    // Сразу пишем компанию в настройки, чтобы UI шага 2 видел id.
    await yg.updateCompanySettings({
      yg_company_id: c.id,
      yg_company_name: c.name,
    })
    pickedCompanyId.value = c.id
    adminForm.password = ''
    await loadProjects()
    notif.success(`YouGile-компания «${c.name}» выбрана`)
  } catch (e) {
    notif.error(e?.data?.message || 'Не удалось подключить компанию')
  } finally {
    busy.value = false
  }
}

async function loadProjects() {
  projectsLoading.value = true
  try { await yg.loadProjects() }
  catch (e) { notif.error(e?.data?.message || 'Не удалось загрузить проекты') }
  finally { projectsLoading.value = false }
}

async function onPickProject(id) {
  const title = id ? (yg.ygProjects.find(p => p.id === id)?.title || null) : null
  busy.value = true
  try {
    await yg.updateCompanySettings({
      yg_project_id: id, yg_project_title: title,
      yg_board_id: null, yg_board_title: null,
      yg_completed_column_id: null,
    })
    yg.ygBoards = []; yg.ygColumns = []
    if (id) {
      boardsLoading.value = true
      try { await yg.loadBoards(id) } finally { boardsLoading.value = false }
    }
  } catch (e) {
    notif.error(e?.data?.message || 'Не удалось сохранить выбор проекта')
  } finally {
    busy.value = false
  }
}

async function onPickBoard(id) {
  const title = id ? (yg.ygBoards.find(b => b.id === id)?.title || null) : null
  busy.value = true
  try {
    await yg.updateCompanySettings({
      yg_board_id: id, yg_board_title: title,
      yg_completed_column_id: null,
    })
    yg.ygColumns = []
    if (id) {
      columnsLoading.value = true
      try { await yg.loadColumns(id) } finally { columnsLoading.value = false }
    }
  } catch (e) {
    notif.error(e?.data?.message || 'Не удалось сохранить выбор доски')
  } finally {
    busy.value = false
  }
}

async function onPickCompleted(id) {
  busy.value = true
  try {
    await yg.updateCompanySettings({ yg_completed_column_id: id || null })
  } catch (e) {
    notif.error(e?.data?.message || 'Не удалось сохранить выбор колонки')
  } finally {
    busy.value = false
  }
}

async function onToggleEnabled(ev) {
  const v = ev.target.checked
  if (v && !canEnable.value) {
    ev.target.checked = false
    notif.warn('Сначала выберите компанию, проект и доску')
    return
  }
  busy.value = true
  try {
    await yg.updateCompanySettings({ enabled: v })
    notif.success(v ? 'Интеграция включена' : 'Интеграция выключена')
  } catch (e) {
    ev.target.checked = !v
    notif.error(e?.data?.message || 'Не удалось переключить')
  } finally {
    busy.value = false
  }
}

async function onReset() {
  if (busy.value) return
  busy.value = true
  try {
    await yg.resetIntegration()
    pickedCompanyId.value = null
    adminForm.login = ''
    adminForm.password = ''
    showReset.value = false
    notif.success('Интеграция сброшена — можно настроить заново')
  } catch (e) {
    notif.error(e?.data?.message || 'Не удалось сбросить интеграцию')
  } finally {
    busy.value = false
  }
}

onMounted(async () => {
  try {
    await Promise.all([yg.refreshStatus(), yg.loadCompanySettings()])
    if (yg.status.connected && settings.value?.yg_project_id) {
      await loadProjects()
      if (settings.value?.yg_board_id) {
        boardsLoading.value = true
        try { await yg.loadBoards(settings.value.yg_project_id) }
        finally { boardsLoading.value = false }
        columnsLoading.value = true
        try { await yg.loadColumns(settings.value.yg_board_id) }
        finally { columnsLoading.value = false }
      }
    } else if (yg.status.connected) {
      await loadProjects()
    }
  } catch (e) {
    notif.error(e?.data?.message || 'Не удалось загрузить настройки YouGile')
  }
})
</script>

<style scoped>
.yg-settings { display: flex; flex-direction: column; gap: 16px; }

.settings-card {
  display: flex; align-items: flex-start; gap: 18px;
  padding: 20px 22px;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--color-outline-dim);
  border-radius: 20px;
}
.head-card { align-items: center; flex-wrap: wrap; }
.form-card { flex-direction: column; }
.row-head { display: flex; gap: 18px; align-items: flex-start; }

.hero-icon {
  flex-shrink: 0; width: 56px; height: 56px;
  border-radius: 16px; display: grid; place-items: center;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}
.hero-icon[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.hero-icon[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.hero-icon[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }
.hero-icon[data-tone="error"]     { --tone-bg: var(--color-error-container);     --tone-fg: var(--color-on-error-container); }
.hero-icon .material-symbols-outlined { font-size: 28px; }

.card-text { flex: 1; min-width: 0; }
.card-text h3 { margin: 0 0 4px; font-size: 16px; font-weight: 700; color: var(--color-text); }
.card-text p { margin: 0; font-size: 13px; line-height: 1.5; color: var(--color-text-dim); }
.card-text p b { color: var(--color-text); }
.card-actions { display: flex; gap: 10px; flex-wrap: wrap; }

.yg-form { display: flex; flex-direction: column; gap: 14px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.lbl { font-size: 13px; font-weight: 600; color: var(--color-on-surface-variant); }
.hint { font-size: 12px; color: var(--color-text-dim); }

.ctl {
  appearance: none; width: 100%;
  border: 1px solid var(--color-outline-variant);
  background: var(--acrylic-card-bg); color: var(--color-on-surface);
  padding: 10px 12px; border-radius: var(--radius-md, 12px);
  font: inherit;
}
.ctl:focus { outline: 2px solid var(--color-primary); outline-offset: -1px; }

.actions { display: flex; justify-content: flex-end; }

.picker { margin-top: 16px; }
.picker-lbl { font-size: 13px; font-weight: 600; margin-bottom: 8px; color: var(--color-on-surface-variant); }
.picker-list { display: flex; flex-direction: column; gap: 6px; }
.picker-item {
  display: flex; align-items: center; gap: 10px;
  padding: 12px 14px; border-radius: 14px;
  background: var(--acrylic-card-bg); color: var(--color-text);
  border: 1px solid var(--color-outline-variant);
  font: inherit; text-align: left; cursor: pointer;
}
.picker-item:hover:not(:disabled) { background: var(--color-surface-high); }
.picker-item.active {
  background: var(--color-primary-container); color: var(--color-on-primary-container);
  border-color: var(--color-primary);
}
.picker-item:disabled { opacity: 0.5; cursor: not-allowed; }

.toggle { display: inline-flex; align-items: center; gap: 8px; cursor: pointer; }
.toggle input { width: 20px; height: 20px; accent-color: var(--color-primary); }

.warn { color: var(--color-warning); font-size: 13px; }

.btn-filled, .btn-outlined, .btn-text {
  display: inline-flex; align-items: center; gap: 8px;
  height: 40px; padding: 0 18px; border-radius: 20px;
  font: inherit; font-weight: 600; cursor: pointer;
  border: 1px solid transparent;
}
.btn-filled { background: var(--color-primary); color: var(--color-on-primary); }
.btn-filled:hover:not(:disabled) { background: color-mix(in oklch, var(--color-primary) 90%, black); }
.btn-outlined { background: transparent; border-color: var(--color-outline-variant); color: var(--color-text); }
.btn-outlined:hover:not(:disabled) { background: var(--color-surface-high); }
.btn-outlined.danger { color: var(--color-error); border-color: var(--color-error); }
.btn-outlined.danger:hover:not(:disabled) { background: var(--color-error-container); color: var(--color-on-error-container); }
.btn-text { background: transparent; color: var(--color-text); }
.btn-text:hover:not(:disabled) { background: var(--color-surface-high); }
button:disabled { opacity: 0.5; cursor: not-allowed; }

.danger-card { align-items: center; flex-wrap: wrap; border-color: color-mix(in oklch, var(--color-error) 35%, var(--color-outline-dim)); }
</style>
