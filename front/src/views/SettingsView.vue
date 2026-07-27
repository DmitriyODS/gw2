<template>
  <!-- Ширину меряем у САМОЙ шеллы, а не у окна браузера: настройки живут в окне
       рабочего стола, которое пользователь волен ужать до половины экрана —
       media-запросы про это ничего не знают. Узко → drill-down: список
       разделов ⇄ раздел с кнопкой «назад». -->
  <div ref="shellEl" class="settings-shell" :class="{ narrow, 'pane-open': paneOpen }">
    <aside class="settings-nav" data-tutorial="settings-nav">
      <div class="settings-search">
        <span class="material-symbols-outlined">search</span>
        <input v-model="searchQuery" type="search" placeholder="поиск по настройкам" />
        <button v-if="searchQuery" class="search-clear" title="Очистить" @click="searchQuery = ''">
          <span class="material-symbols-outlined">close</span>
        </button>
      </div>

      <nav class="settings-list">
        <template v-for="group in visibleGroups" :key="group.key">
          <div v-if="showGroupLabels" class="settings-group-label">{{ group.label }}</div>
          <button
            v-for="section in group.sections"
            :key="section.key"
            class="settings-item"
            :class="{ active: activeSection === section.key && (!narrow || paneOpen) }"
            :data-tutorial="`settings-section-${section.key}`"
            type="button"
            @click="openSection(section.key)"
          >
            <span class="material-symbols-outlined item-icon">{{ section.icon }}</span>
            <span class="item-title">{{ section.title }}</span>
            <span v-if="narrow" class="material-symbols-outlined item-chev">chevron_right</span>
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
    </aside>

    <section class="settings-pane">
      <header class="pane-head">
        <button v-if="narrow" class="pane-back" title="К списку" @click="paneOpen = false">
          <span class="material-symbols-outlined">arrow_back</span>
          Назад
        </button>
        <h2 class="pane-title">{{ activeSectionMeta?.title || 'Настройки' }}</h2>
      </header>

      <div class="pane-body">
        <GeneralSection v-if="activeSection === 'general'" :show-yougile="showYougile" />

        <ThemesSection v-else-if="activeSection === 'theme'" />

        <div v-else-if="activeSection === 'desktop'" class="pane-stack">
          <DesktopWallpaperCard />
          <AppGradientCard />
          <DesktopTilesCard />
          <DesktopAppCard />
        </div>

        <ChatsPortalSection v-else-if="activeSection === 'chats'" :has-company="hasCompany" />

        <div v-else-if="activeSection === 'help'" class="pane-stack">
          <HelpCenter />
          <SupportCard />
        </div>

        <AboutSection v-else-if="activeSection === 'about'" />

        <!-- Резервная копия — только супер-админ платформы. -->
        <div v-else-if="activeSection === 'backup'" class="pane-stack">
          <div class="backup-card">
            <span class="backup-icon">
              <span class="material-symbols-outlined">backup</span>
            </span>
            <div class="backup-text">
              <strong>Создать резервную копию</strong>
              <small>
                Полный архив базы данных и всех файлов (вложения чатов, аватары,
                картинки и документы разделов) в одном файле.
              </small>
            </div>
            <button class="backup-btn" :disabled="backupExporting" @click="exportDialogOpen = true">
              <span class="material-symbols-outlined">download</span>
              {{ backupExporting ? 'Готовим архив…' : 'Скачать копию' }}
            </button>
          </div>

          <div class="backup-card danger">
            <span class="backup-icon danger">
              <span class="material-symbols-outlined">restore</span>
            </span>
            <div class="backup-text">
              <strong>Восстановление</strong>
              <small>
                Полная замена текущих данных содержимым архива. Действие
                необратимо — мы дважды переспросим.
              </small>
            </div>
            <label class="backup-btn danger">
              <span class="material-symbols-outlined">upload</span>
              Выбрать файл
              <input type="file" accept=".zip" @change="onImportFileSelect" />
            </label>
          </div>
        </div>
      </div>
    </section>

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
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAuthStore } from '@/stores/auth.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { exportBackup, importBackup } from '@/api/backup.js'
import { settingsGroups, resolveSectionKey } from '@/utils/settingsSections.js'
import BackupSectionsDialog from '@/components/settings/BackupSectionsDialog.vue'
import GeneralSection from '@/components/settings/GeneralSection.vue'
import ThemesSection from '@/components/settings/ThemesSection.vue'
import ChatsPortalSection from '@/components/settings/ChatsPortalSection.vue'
import AboutSection from '@/components/settings/AboutSection.vue'
import SupportCard from '@/components/settings/SupportCard.vue'
import HelpCenter from '@/components/settings/HelpCenter.vue'
import DesktopAppCard from '@/components/settings/DesktopAppCard.vue'
import DesktopWallpaperCard from '@/components/settings/DesktopWallpaperCard.vue'
import DesktopTilesCard from '@/components/settings/DesktopTilesCard.vue'
import AppGradientCard from '@/components/settings/AppGradientCard.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const { isAtLeast } = usePermission()
const notif = useNotificationsStore()
const authStore = useAuthStore()
const { isMobile } = useBreakpoint()
const route = useRoute()
const router = useRouter()

const searchQuery = ref('')
const activeSection = ref('general')
/** Узко: показан раздел, а не список. Широко — обе колонки видны всегда. */
const paneOpen = ref(true)

const hasCompany = computed(() => authStore.companyId != null)
const isAdmin = computed(() => isAtLeast(ROLES.ADMIN))
// Личный ключ YouGile настраивает рядовой участник; администратор подключает
// компанию целиком в её карточке.
const showYougile = computed(() => hasCompany.value && !isAdmin.value)

/* ── Ширина шеллы: настройки открыты окном, ширина окна ≠ ширина экрана ── */
const shellEl = ref(null)
const narrow = ref(false)
const NARROW_AT = 720
let ro = null

onMounted(() => {
  if (typeof ResizeObserver === 'undefined') {
    narrow.value = isMobile.value
    return
  }
  ro = new ResizeObserver(([entry]) => {
    const wasNarrow = narrow.value
    narrow.value = entry.contentRect.width < NARROW_AT
    // Разъехались до двух колонок — раздел снова виден всегда.
    if (wasNarrow && !narrow.value) paneOpen.value = true
  })
  ro.observe(shellEl.value)
})

onBeforeUnmount(() => ro?.disconnect())

/* ── Каталог разделов (общий: его же ищет Spotlight) ── */
const allGroups = computed(() => settingsGroups({
  isMobile: isMobile.value,
  hasCompany: hasCompany.value,
  isAdmin: isAdmin.value,
  isSuperAdmin: authStore.isSuperAdmin,
}))

const visibleGroups = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return allGroups.value
    .map((g) => ({
      ...g,
      sections: g.sections.filter((s) => !q || [s.title, s.desc].some((x) => x.toLowerCase().includes(q))),
    }))
    .filter((g) => g.sections.length)
})

// Заголовки групп нужны, только когда групп больше одной (супер-админ со
// «Системой») — иначе над единственным списком висела бы лишняя строка.
const showGroupLabels = computed(() => visibleGroups.value.length > 1)

const sectionByKey = computed(() => {
  const map = {}
  allGroups.value.forEach((g) => g.sections.forEach((s) => { map[s.key] = s }))
  return map
})

const activeSectionMeta = computed(() => sectionByKey.value[activeSection.value] || null)

function openSection(key) {
  activeSection.value = key
  paneOpen.value = true
  if (route.query.section !== key) {
    router.replace({ query: { ...route.query, section: key } }).catch(() => {})
  }
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
  const requested = resolveSectionKey(route.query.section)
  if (requested && sectionByKey.value[requested]) {
    activeSection.value = requested
  } else {
    // Пришли без адреса раздела: на узком экране встречаем списком.
    activeSection.value = 'general'
    paneOpen.value = !isMobile.value
  }
})

// Раздел может смениться в уже открытом окне: «Персонализация» с рабочего
// стола ведёт на /settings?section=desktop, когда настройки уже открыты.
watch(() => route.query.section, (key) => {
  const resolved = resolveSectionKey(key)
  if (resolved && sectionByKey.value[resolved] && resolved !== activeSection.value) {
    activeSection.value = resolved
    paneOpen.value = true
  }
})
</script>

<style scoped>
/* ──────────────────────────────────────────────────────────────────
   Настройки: две стеклянные панели — список разделов и раздел.
   Узкая шелла (<720px по СВОЕЙ ширине, а не по экрану) превращает их в
   один экран с переходом «список → раздел → назад».
────────────────────────────────────────────────────────────────── */
.settings-shell {
  display: grid;
  grid-template-columns: 300px 1fr;
  gap: 16px;
  padding: 16px;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

/* ── Список разделов ────────────────────────────────────────── */
.settings-nav {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-height: 0;
  padding: 14px 12px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  overflow-y: auto;
}

.settings-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border: 1px solid var(--acrylic-border);
  border-radius: 999px;
  /* Стекло — как у поиска в справке и других разделах. */
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  transition: border-color 0.15s, background 0.15s;
}

.settings-search:focus-within { border-color: var(--color-primary); }

.settings-search .material-symbols-outlined {
  font-size: 19px;
  color: var(--color-text-dim);
}

.settings-search input {
  flex: 1;
  min-width: 0;
  border: none;
  background: none;
  color: var(--color-text);
  font-size: 0.88rem;
  outline: none;
}

.settings-search input::placeholder { color: var(--color-text-dim); }

.settings-search input::-webkit-search-cancel-button { display: none; }

.search-clear {
  display: grid;
  place-items: center;
  width: 22px;
  min-width: 22px;
  max-width: 22px;
  height: 22px;
  min-height: 22px;
  max-height: 22px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  cursor: pointer;
}

.search-clear .material-symbols-outlined { font-size: 15px; }

.settings-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.settings-group-label {
  padding: 10px 12px 2px;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

.settings-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font-size: 0.92rem;
  font-weight: 500;
  text-align: left;
  cursor: pointer;
  transition: background 0.18s ease, color 0.18s ease;
}

.settings-item:hover {
  background: var(--glass-hover-bg, var(--glass-bg)), var(--acrylic-card-bg);
}

.settings-item.active {
  background: var(--glass-bg), var(--color-primary-container);
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
  color: var(--color-on-primary-container);
  font-weight: 600;
}

.item-icon { font-size: 21px; }

.item-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-chev { font-size: 19px; color: var(--color-text-dim); }

/* ── Раздел ─────────────────────────────────────────────────── */
.settings-pane {
  display: flex;
  flex-direction: column;
  min-height: 0;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  /* Вложенные разделы меряют себя по панели, а не по экрану. */
  container-type: inline-size;
}

.pane-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 20px 24px 12px;
}

/* Пилюля-«капелька стекла» — как кнопки возврата в других разделах. */
.pane-back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  align-self: center;
  padding: 8px 14px;
  border: 1px solid var(--acrylic-border);
  border-radius: 999px;
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font-size: 0.82rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.pane-back .material-symbols-outlined { font-size: 19px; }

.pane-back:hover {
  background: var(--glass-hover-bg, var(--glass-bg)), var(--acrylic-card-bg);
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
}

.pane-title {
  margin: 0;
  font-size: 1.65rem;
  font-weight: 700;
  letter-spacing: -0.01em;
}

.pane-body {
  flex: 1;
  min-height: 0;
  padding: 4px 24px 24px;
  overflow-y: auto;
}

.pane-stack {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* ── Резервная копия ────────────────────────────────────────── */
.backup-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 18px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
}

.backup-card.danger { border-color: var(--color-error-container); }

.backup-icon {
  display: grid;
  place-items: center;
  width: 46px;
  min-width: 46px;
  max-width: 46px;
  height: 46px;
  min-height: 46px;
  max-height: 46px;
  border-radius: var(--radius-md);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.backup-icon.danger {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

.backup-icon .material-symbols-outlined { font-size: 24px; }

.backup-text {
  display: flex;
  flex-direction: column;
  gap: 3px;
  flex: 1;
  min-width: 0;
}

.backup-text strong { font-size: 0.95rem; font-weight: 600; }

.backup-text small {
  font-size: 0.82rem;
  line-height: 1.4;
  color: var(--color-text-dim);
}

.backup-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border: none;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 0.86rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
}

.backup-btn:disabled { opacity: 0.65; cursor: progress; }

.backup-btn.danger {
  background: var(--color-error);
  color: var(--color-on-error);
}

.backup-btn input[type="file"] { display: none; }

.backup-btn .material-symbols-outlined { font-size: 19px; }

/* ── Узкая шелла: один экран ────────────────────────────────── */
.settings-shell.narrow {
  grid-template-columns: 1fr;
  padding: 12px;
  gap: 0;
}

.settings-shell.narrow .settings-pane { display: none; }
.settings-shell.narrow.pane-open .settings-nav { display: none; }
.settings-shell.narrow.pane-open .settings-pane { display: flex; }

.settings-shell.narrow .pane-head { padding: 16px 16px 10px; }
.settings-shell.narrow .pane-body { padding: 4px 16px 20px; }
.settings-shell.narrow .pane-title { font-size: 1.35rem; }

@container (max-width: 620px) {
  .backup-card { flex-wrap: wrap; }
  .backup-btn { width: 100%; justify-content: center; }
}

@media (max-width: 620px) {
  .backup-card { flex-wrap: wrap; }
  .backup-btn { width: 100%; justify-content: center; }
}
</style>
