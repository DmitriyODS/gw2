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
        <EmptyState
          v-if="!visibleGroups.length && searchQuery"
          size="sm"
          icon="search_off"
          title="Ничего не нашли"
          subtitle="Попробуйте другие слова."
        />
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
            @click="closeSection"
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
          <ThemeBuilder ref="themeBuilder" />
          <!-- Настройки десктоп-обёртки — рендерится только внутри Electron. -->
          <DesktopAppCard />
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
              <button class="btn-filled" @click="exportDialogOpen = true" :disabled="backupExporting">
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

        <!-- YouGile — личный коннект (любой авторизованный с компанией).
             Настройки компании (ИИ, выходные, «Мой Groove», ссылка-приглашение,
             YouGile-компания) переехали в раздел «Компании» → карточка компании. -->
        <div v-show="activeSection === 'yougile'" class="pane-block">
          <YougileUserSettings v-if="hasCompany && activeSection === 'yougile'" />
        </div>

        <!-- Справка -->
        <div v-show="activeSection === 'help'" class="pane-block">
          <HelpCenter />
        </div>

        <!-- О приложении -->
        <div v-show="activeSection === 'about'" class="pane-block">
          <AboutApp />
        </div>
        </div>
      </section>
    </Transition>

    <BackupSectionsDialog
      v-model="exportDialogOpen"
      mode="export"
      :busy="backupExporting"
      @confirm="onExportConfirm"
    />
    <BackupSectionsDialog
      v-model="importSectionsOpen"
      mode="import"
      @confirm="onImportSectionsConfirm"
    />

    <ConfirmDialog
      :visible="showImportConfirm1"
      header="Восстановление из резервной копии"
      message="Вы уверены? Выбранные разделы будут полностью заменены данными из файла резервной копии."
      confirm-label="Продолжить"
      :danger-confirm="true"
      @confirm="showImportConfirm1 = false; showImportConfirm2 = true"
      @cancel="cancelImport"
    />
    <ConfirmDialog
      :visible="showImportConfirm2"
      header="Подтвердите восстановление"
      message="Это последнее предупреждение. Все текущие данные будут безвозвратно заменены. Продолжить?"
      confirm-label="Да, восстановить"
      :danger-confirm="true"
      @confirm="doImportBackup"
      @cancel="cancelImport"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAuthStore } from '@/stores/auth.js'
import { useTutorial } from '@/composables/useTutorial.js'
import { useChangelog } from '@/composables/useChangelog.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { exportBackup, importBackup } from '@/api/backup.js'
import BackupSectionsDialog from '@/components/settings/BackupSectionsDialog.vue'
import ThemeBuilder from '@/components/settings/ThemeBuilder.vue'
import HelpCenter from '@/components/settings/HelpCenter.vue'
import AboutApp from '@/components/settings/AboutApp.vue'
import DesktopAppCard from '@/components/settings/DesktopAppCard.vue'
import YougileUserSettings from '@/components/settings/YougileUserSettings.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const { isAtLeast } = usePermission()
const notif = useNotificationsStore()
const authStore = useAuthStore()
const tutorial = useTutorial()
const changelog = useChangelog()
// Версия продукта — только с сервера (первая запись changelog), не из бандла.
const appVersion = changelog.latestVersion
const { isMobile } = useBreakpoint()
const route = useRoute()
const router = useRouter()

const searchQuery = ref('')
const activeSection = ref(null) // null = показывать список секций (на мобильном)

// Root-админ системы — без company_id. YouGile-разделы (личный и
// «Интеграция с YouGile») в этом случае не показываем: они привязаны к
// конкретной компании.
const hasCompany = computed(() => authStore.companyId != null)

/* ── Конфигурация разделов ─────────────────────────────────────── */
const allGroups = computed(() => [
  {
    key: 'personal',
    label: 'Персонализация',
    sections: [
      { key: 'theme', title: 'Внешний вид', desc: 'Цвета, тёмная тема и стиль интерфейса', icon: 'palette', tone: 'primary' },
      // YouGile (личный) — только для НЕ-директоров и только для тех, кто
      // привязан к компании. Root-администратор системы (без company_id)
      // эту настройку видеть не должен: она бессмысленна без компании.
      ...((!isAtLeast(ROLES.ADMIN) && hasCompany.value) ? [
        { key: 'yougile', title: 'YouGile', desc: 'Подключение личного аккаунта для импорта и создания карточек', icon: 'sync_alt', tone: 'secondary' },
      ] : []),
      { key: 'help', title: 'Справка', desc: 'Как пользоваться разделами платформы', icon: 'help_center', tone: 'secondary' },
      { key: 'about', title: 'О приложении', desc: 'Версия, тур, написать в техподдержку', icon: 'info', tone: 'tertiary' },
    ],
  },
  // Настройки компании (ИИ, выходные, «Мой Groove», ссылка-приглашение,
  // интеграция YouGile) переехали в раздел «Компании» → карточка компании:
  // один пользователь может администрировать несколько компаний, и настройки
  // привязаны к конкретной компании, а не к активной сессии.
  ...(authStore.isSuperAdmin ? [{
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

const themeBuilder = ref(null)

/* Уход из «Внешнего вида» с несохранёнными цветами — ThemeBuilder сам
   показывает предупреждение и откатывает превью; false = пользователь
   решил остаться. Спросить нужно ДО смены секции: pane пересоздаётся
   по :key, и компонент размонтируется. */
async function canLeaveTheme() {
  if (activeSection.value !== 'theme') return true
  return await (themeBuilder.value?.confirmLeave?.() ?? true)
}

async function openSection(key) {
  if (key === activeSection.value) return
  if (!(await canLeaveTheme())) return
  activeSection.value = key
  router.replace({ query: { ...route.query, section: key } }).catch(() => {})
}

async function closeSection() {
  if (!(await canLeaveTheme())) return
  activeSection.value = null
}

/* ── Backup ────────────────────────────────────────────────────── */
const backupExporting = ref(false)
const showImportConfirm1 = ref(false)
const showImportConfirm2 = ref(false)
const importFile = ref(null)
const exportDialogOpen = ref(false)
const importSectionsOpen = ref(false)
const importSections = ref([])

function onExportConfirm(sections) {
  exportDialogOpen.value = false
  doExportBackup(sections)
}

async function doExportBackup(sections) {
  backupExporting.value = true
  try {
    const response = await exportBackup(sections)
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
  importSectionsOpen.value = true
  event.target.value = ''
}

function onImportSectionsConfirm(sections) {
  importSections.value = sections
  importSectionsOpen.value = false
  showImportConfirm1.value = true
}

function cancelImport() {
  showImportConfirm1.value = false
  showImportConfirm2.value = false
  importFile.value = null
  importSections.value = []
}

async function doImportBackup() {
  showImportConfirm2.value = false
  if (!importFile.value) return
  try {
    await importBackup(importFile.value, importSections.value)
    notif.success('База данных восстановлена. Страница перезагрузится.')
    setTimeout(() => window.location.reload(), 2000)
  } catch (e) { notif.error(e.message || 'Ошибка восстановления') }
  finally { importFile.value = null; importSections.value = [] }
}

onMounted(() => {
  changelog.loadLatest()
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
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
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
  background: var(--acrylic-card-bg);
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
  height: 28px; min-height: 0;
  border-radius: 50%;
  border: 0;
  background: transparent;
  display: grid;
  place-items: center;
  color: var(--color-text-dim);
  cursor: pointer;
}

.settings-search-clear:hover { background: var(--acrylic-card-bg); color: var(--color-text); }
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
  background: var(--acrylic-card-bg);
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
  background: var(--acrylic-card-bg);
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
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
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

.settings-back:hover { background: var(--acrylic-card-bg); }

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
  background: var(--acrylic-card-bg);
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
  background: var(--acrylic-card-bg);
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

/* ── Users: toolbar ─────────────────────────────────────────── */
.users-block {
  gap: 14px;
  max-width: 100%;
}

.users-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
}

.users-search {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 14px;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--color-outline-dim);
  border-radius: 999px;
  transition: border-color 0.15s, background 0.15s;
  min-width: 0;
}

.users-search:focus-within {
  border-color: var(--color-primary);
  background: var(--color-surface-low);
}

.users-search > .material-symbols-outlined {
  font-size: 20px;
  color: var(--color-text-dim);
  flex-shrink: 0;
}

.users-search input {
  flex: 1;
  min-width: 0;
  background: transparent;
  border: 0;
  outline: 0;
  padding: 11px 0;
  font-size: 14px;
  color: var(--color-text);
}

.users-search input::placeholder { color: var(--color-text-dim); }
.users-search input::-webkit-search-cancel-button { display: none; }

.users-search-clear {
  width: 28px;
  height: 28px; min-height: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}

.users-search-clear:hover { background: var(--color-surface-high); color: var(--color-text); }
.users-search-clear .material-symbols-outlined { font-size: 16px; }

.users-add {
  flex-shrink: 0;
}

.users-count {
  font-size: 12px;
  color: var(--color-text-dim);
  padding: 0 4px;
}

.users-count b { color: var(--color-text); }

/* ── Users: list rows ───────────────────────────────────────── */
.users-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.urow {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 14px;
  padding: 12px 16px;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--color-outline-dim);
  border-radius: 16px;
  transition: border-color 0.15s, background 0.15s, transform 0.15s;
}

.urow:hover {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--color-outline-dim));
  background: var(--color-surface-low);
}

.urow.is-me {
  background: color-mix(in oklch, var(--color-primary-container) 60%, var(--color-surface));
  border-color: color-mix(in oklch, var(--color-primary) 35%, var(--color-outline-dim));
}

.urow-avatar-wrap {
  position: relative;
  flex-shrink: 0;
}

.urow-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  object-fit: cover;
  display: block;
  border: 2px solid var(--color-surface-low);
}

.urow.is-me .urow-avatar { border-color: var(--color-primary); }

.urow-me-dot {
  position: absolute;
  right: -2px;
  bottom: -2px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--color-primary);
  border: 2px solid var(--color-surface);
}

.urow-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.urow-name-line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.urow-name {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.urow-me-chip {
  flex-shrink: 0;
  font-size: 10px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.urow-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-dim);
  min-width: 0;
}

.urow-login {
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', monospace;
  flex-shrink: 0;
}

.urow-dot { opacity: 0.5; flex-shrink: 0; }

.urow-post {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.urow-role {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px 5px 9px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  white-space: nowrap;
  line-height: 1.4;
  flex-shrink: 0;
}

.urow-role .material-symbols-outlined { font-size: 15px; }

.urow-role[data-level="2"] {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.urow-role[data-level="3"] {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.urow-role[data-level="4"] {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.urow-actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
}

.urow-action {
  width: 36px;
  height: 36px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: grid;
  place-items: center;
  transition: background 0.15s, color 0.15s;
}

.urow-action:hover {
  background: var(--color-surface-high);
  color: var(--color-text);
}

.urow-action.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

.urow-action .material-symbols-outlined { font-size: 18px; }

.urow.is-me .urow-action:hover {
  background: color-mix(in oklch, var(--color-primary) 18%, transparent);
}

/* Loading state */
.users-loading {
  display: grid;
  place-items: center;
  padding: 60px 0;
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
  background: var(--acrylic-card-bg);
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
    /* Резерв под нижнюю навигацию (64px) + 12px воздуха: шелл скроллится
       через .main-content, список секций уходит под стекло навигации. */
    padding-bottom: calc(76px + env(safe-area-inset-bottom, 0px));
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
    background: var(--acrylic-card-bg);
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
    -webkit-backdrop-filter: none;
    backdrop-filter: none;
    border-radius: 0;
    border: 0;
    display: flex;
    flex-direction: column;
    /* Резерв под нижнюю навигацию — НЕ на этой fixed-обёртке, а внутри
       скроллера (.settings-pane-body ниже): контент уходит под стекло. */
  }

  /* При включённом градиенте фона фуллскрин-панель не глушит его:
     позади лежит статичный фон .app-layout — пусть просвечивает. */
  [data-bg-gradient="true"] .settings-pane.mobile-full {
    background: transparent;
  }

  .settings-pane.mobile-full .settings-pane-header {
    position: sticky;
    top: 0;
    z-index: 2;
    padding: 12px 12px 12px;
    background: var(--acrylic-bg-strong);
    -webkit-backdrop-filter: var(--acrylic-blur);
    backdrop-filter: var(--acrylic-blur);
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
    /* 64px навигации + 12px воздуха — последние настройки не прячутся. */
    padding: 16px 12px calc(76px + env(safe-area-inset-bottom, 0px));
    -webkit-overflow-scrolling: touch;
  }

  /* Users — на мобильном раскладываем строку в 2 линии:
       row1: avatar | main | actions
       row2:           | role-pill (растянута), без actions
     Это даёт компактные карточки, видна вся роль, кнопки рядом с аватаром. */
  .users-toolbar { flex-direction: column; align-items: stretch; gap: 8px; }
  .users-add { width: 100%; justify-content: center; }

  .urow {
    grid-template-columns: auto minmax(0, 1fr) auto;
    grid-template-areas:
      "avatar main actions"
      "role   role role";
    gap: 8px 12px;
    padding: 12px;
    border-radius: 18px;
  }

  .urow-avatar-wrap { grid-area: avatar; }
  .urow-main { grid-area: main; }
  .urow-actions { grid-area: actions; }
  .urow-role {
    grid-area: role;
    justify-self: start;
    margin-top: 2px;
  }

  .urow-avatar { width: 40px; height: 40px; }
  .urow-name { font-size: 14px; }
  .urow-meta { font-size: 11px; flex-wrap: wrap; }
  .urow-role { font-size: 11px; padding: 4px 11px 4px 7px; }
  .urow-role .material-symbols-outlined { font-size: 13px; }
  .urow-action { width: 34px; height: 34px; }
  .urow-action .material-symbols-outlined { font-size: 16px; }

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
