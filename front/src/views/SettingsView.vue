<template>
  <AppListDetail v-model:open="paneOpen" @narrow-change="narrow = $event">
    <!-- Список разделов настроек -->
    <template #list="{ toggle }">
      <!-- Список подписан всегда: без заголовка непонятно, перечень чего это. -->
      <AppPage
        embedded
        title="Настройки"
        show-title
        :menu="!narrow"
        menu-icon="left_panel_close"
        menu-label="Свернуть список"
        @menu="toggle"
      >
        <template #subhead>
          <SearchField v-model="searchQuery" placeholder="поиск по настройкам" :collapsible="false" />
        </template>

        <AppStack :gap="2">
          <template v-for="group in visibleGroups" :key="group.key">
            <div v-if="showGroupLabels" class="group-label">{{ group.label }}</div>
            <!-- Список навигации, а не набор карточек: рамка у каждого пункта
                 забивала перечень, поэтому строки plain, а выделен только
                 открытый раздел. Шеврон остаётся лишь в узкой раскладке —
                 там он и правда ведёт на отдельный экран. -->
            <AppRow
              v-for="section in group.sections"
              :key="section.key"
              :title="section.title"
              :icon="section.icon"
              dense
              plain
              clickable
              :selected="activeSection === section.key && (!narrow || paneOpen)"
              :show-chevron="narrow"
              @click="openSection(section.key)"
            />
          </template>

          <EmptyState
            v-if="!visibleGroups.length && searchQuery"
            size="sm"
            icon="search_off"
            title="Ничего не нашли"
            subtitle="Попробуйте другие слова."
          />
        </AppStack>
      </AppPage>
    </template>

    <!-- Выбранный раздел -->
    <template #detail="{ collapsed, toggle }">
      <AppPage
        embedded
        :title="activeSectionMeta?.title || 'Настройки'"
        :back="narrow"
        back-label="К списку"
        :menu="!narrow && collapsed"
        menu-icon="left_panel_open"
        menu-label="Показать список"
        @back="paneOpen = false"
        @menu="toggle"
      >
        <GeneralSection v-if="activeSection === 'general'" />

        <NotificationsSection v-else-if="activeSection === 'notifications'" />

        <ThemesSection v-else-if="activeSection === 'theme'" />

        <DesktopSection v-else-if="activeSection === 'desktop'" />

        <ChatsPortalSection v-else-if="activeSection === 'chats'" :has-company="hasCompany" />

        <AccountSection v-else-if="activeSection === 'account'" :show-yougile="showYougile" />

        <CompaniesSection v-else-if="activeSection === 'companies'" />

        <StorageSection v-else-if="activeSection === 'storage'" />

        <AiSection v-else-if="activeSection === 'ai'" />

        <LegalSection v-else-if="activeSection === 'legal'" />

        <AppStack v-else-if="activeSection === 'help'">
          <HelpCenter />
          <SupportCard />
        </AppStack>

        <AboutSection v-else-if="activeSection === 'about'" />

        <!-- Скрытый раздел разработчика: открывается пятью нажатиями по номеру
             сборки в «О приложении». -->
        <DevToolsSection v-else-if="activeSection === 'devtools'" @close="openSection('about')" />

        <!-- Аудит платформы — единая точка управления для супер-админа. -->
        <AuditSection v-else-if="activeSection === 'audit'" />

        <!-- Резервная копия — только супер-админ платформы. -->
        <AppStack v-else-if="activeSection === 'backup'">
          <AppRow
            title="Создать резервную копию"
            hint="Полный архив базы данных и всех файлов (вложения чатов, аватары, картинки и документы разделов) в одном файле."
          >
            <AppButton
              variant="filled"
              icon="download"
              :loading="backupExporting"
              :label="backupExporting ? 'Готовим архив…' : 'Скачать копию'"
              @click="exportDialogOpen = true"
            />
          </AppRow>

          <AppRow
            tone="danger"
            title="Восстановление"
            hint="Полная замена текущих данных содержимым архива. Действие необратимо — мы дважды переспросим."
          >
            <!-- Выбор файла — нативный input, спрятанный под кнопкой-меткой. -->
            <label class="backup-pick">
              <AppButton variant="filled" tone="danger" icon="upload" label="Выбрать файл" tag="span" />
              <input type="file" accept=".zip" @change="onImportFileSelect" />
            </label>
          </AppRow>
        </AppStack>
      </AppPage>
    </template>

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
  </AppListDetail>
</template>

<script setup>
import { ref, computed, defineAsyncComponent, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAuthStore } from '@/stores/auth.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { exportBackup, importBackup } from '@/api/backup.js'
import { settingsGroups, resolveSectionKey } from '@/utils/settingsSections.js'
import AppButton from '@/components/ui/AppButton.vue'
import AppListDetail from '@/components/ui/AppListDetail.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import SearchField from '@/components/common/SearchField.vue'
import BackupSectionsDialog from '@/components/settings/BackupSectionsDialog.vue'
import GeneralSection from '@/components/settings/GeneralSection.vue'

/* Панель показывается ровно одна, поэтому все, кроме открытой по умолчанию
   «Общих», — ленивые чанки: раздел «Настройки» вместе с ними весил как
   четверть приложения (управление компанией, справка, реестры, аудит). */
const AccountSection = defineAsyncComponent(() => import('@/components/settings/AccountSection.vue'))
const CompaniesSection = defineAsyncComponent(() => import('@/components/settings/CompaniesSection.vue'))
const AiSection = defineAsyncComponent(() => import('@/components/settings/AiSection.vue'))
const StorageSection = defineAsyncComponent(() => import('@/components/settings/StorageSection.vue'))
const ThemesSection = defineAsyncComponent(() => import('@/components/settings/ThemesSection.vue'))
const ChatsPortalSection = defineAsyncComponent(() => import('@/components/settings/ChatsPortalSection.vue'))
const AboutSection = defineAsyncComponent(() => import('@/components/settings/AboutSection.vue'))
const LegalSection = defineAsyncComponent(() => import('@/components/settings/LegalSection.vue'))
const AuditSection = defineAsyncComponent(() => import('@/components/settings/AuditSection.vue'))
const SupportCard = defineAsyncComponent(() => import('@/components/settings/SupportCard.vue'))
const HelpCenter = defineAsyncComponent(() => import('@/components/settings/HelpCenter.vue'))
const DesktopSection = defineAsyncComponent(() => import('@/components/settings/DesktopSection.vue'))
const NotificationsSection = defineAsyncComponent(() => import('@/components/settings/NotificationsSection.vue'))
const DevToolsSection = defineAsyncComponent(() => import('@/components/settings/DevToolsSection.vue'))
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { saveBlob } from '@/utils/download.js'

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

/* Узкая раскладка — свойство самого раздела (он живёт окном рабочего стола);
   её меряет и сообщает AppListDetail. */
const narrow = ref(false)

/* ── Каталог разделов (общий: его же ищет Hola) ── */
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
    await saveBlob(blob, `backup_${new Date().toISOString().split('T')[0]}.zip`)
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
/* Каркас (две стеклянные панели, drill-down в узкой раскладке), шапка, поиск и
   строки списка — общие компоненты (AppListDetail / AppPage / AppRow). Здесь
   осталось только то, чего нет в ядре. */

.group-label {
  padding: 10px 12px 2px;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

/* Выбор файла архива: нативный input прячется под кнопкой-меткой. */
.backup-pick { display: inline-flex; cursor: pointer; }
.backup-pick input[type="file"] { display: none; }
</style>
