<template>
  <!-- Выбрана компания — показываем её карточку прямо в панели настроек. -->
  <CompanyManagePanel
    v-if="manageId"
    :id="manageId"
    @back="closeManage"
    @deleted="onDeleted"
    @imported="onImported"
  />

  <AppStack v-else>
    <AppCard
      title="Компании"
      :hint="isSuper
        ? 'Все компании платформы: состав, статистика и доступность.'
        : 'Компании, где вы администратор или создатель. Новую можно завести в любой момент — вы станете её администратором.'"
    >
      <!-- Счётчики и действие — одной строкой: кнопка в шапке карточки спорила
           с длинным пояснением и жалась к самому краю. -->
      <AppStack :gap="10" row>
        <AppChip tone="primary" icon="domain" :count="rows.length" :label="isSuper ? 'всего' : 'под управлением'" />
        <template v-if="isSuper">
          <AppChip tone="success" :count="activeCount" label="активных" />
          <AppChip v-if="disabledCount" tone="error" :count="disabledCount" label="отключённых" />
        </template>
        <AppButton
          class="cs-create"
          variant="filled"
          size="sm"
          icon="add"
          label="Компания"
          @click="openCreate"
        />
      </AppStack>

      <SearchField
        v-if="rows.length > 5"
        v-model="search"
        placeholder="Поиск по названию"
        :collapsible="false"
      />
    </AppCard>

    <BrandLoader v-if="loading" block :size="64" />

    <EmptyState
      v-else-if="!visible.length"
      size="sm"
      :icon="search ? 'search_off' : 'domain'"
      :title="search ? 'Ничего не нашли' : 'Компаний пока нет'"
      :subtitle="search ? 'Попробуйте уточнить запрос.' : 'Создайте компанию — вы станете её администратором.'"
    >
      <AppButton
        v-if="!search"
        variant="filled"
        icon="add"
        label="Создать компанию"
        @click="openCreate"
      />
    </EmptyState>

    <AppStack v-else :gap="8">
      <AppRow
        v-for="c in visible"
        :key="c.id"
        :title="c.name"
        clickable
        inline
        :disabled="!c.is_active && !isSuper"
        @click="openManage(c)"
      >
        <template #lead>
          <span class="cs-avatar" :class="['tone-' + toneOf(c)]">{{ initials(c.name) }}</span>
        </template>

        <template #hint>
          <span class="cs-meta">
            <span v-if="!isSuper">{{ roleBadge(c) }}</span>
            <span v-else-if="creatorName(c)">{{ creatorName(c) }}</span>
            <span>· {{ c.employees_count }} чел.</span>
            <span>· {{ c.tasks_count }} задач</span>
            <span v-if="!c.is_active" class="cs-off">· отключена</span>
          </span>
        </template>

        <!-- Супер-админ переключает доступность компании прямо в списке. -->
        <AppSwitch
          v-if="isSuper"
          :model-value="c.is_active"
          :disabled="togglingId === c.id"
          @update:model-value="onToggle(c)"
        />
        <AppButton
          v-if="isSuper"
          variant="icon"
          size="sm"
          tone="danger"
          icon="delete"
          title="Удалить компанию"
          aria-label="Удалить компанию"
          @click.stop="askDelete(c)"
        />
        <span class="material-symbols-outlined cs-chev">chevron_right</span>
      </AppRow>
    </AppStack>

    <CreateCompanyDialog v-model="createOpen" @created="loadData" />

    <AppDialog
      v-model="confirmOpen"
      tone="danger"
      size="sm"
      :title="`Удалить «${deleteTarget?.name || ''}»?`"
      subtitle="Вместе с компанией удалятся её задачи, юниты и статистика. Действие необратимо."
      :busy="deleting"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: deleting },
        { kind: 'confirm', label: 'Удалить', icon: 'delete', disabled: deleting },
      ]"
      @confirm="doDelete"
    />
  </AppStack>
</template>

<script setup>
/* Компании — панель настроек, а не самостоятельный раздел: управление своей
   организацией стоит рядом с аккаунтом и прочими личными настройками. Карточка
   конкретной компании (участники, роли, опасная зона) открывается здесь же,
   вторым уровнем панели; прежние адреса `/companies` и `/companies/:id` ведут
   сюда редиректом. */
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitch from '@/components/ui/AppSwitch.vue'
import CompanyManagePanel from './CompanyManagePanel.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import CreateCompanyDialog from '@/components/common/CreateCompanyDialog.vue'
import { useCompaniesStore } from '@/stores/companies.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAuthStore } from '@/stores/auth.js'
import { usePermission } from '@/composables/usePermission.js'
import { listMyCompanies } from '@/api/companies.js'

const route = useRoute()
const router = useRouter()
const companies = useCompaniesStore()
const notif = useNotificationsStore()
const auth = useAuthStore()
const { isSuperAdmin } = usePermission()
const isSuper = computed(() => isSuperAdmin())

/* Источник данных: супер-админ видит ВСЕ компании платформы (стор), обычный
   пользователь — те, где он администратор или создатель (/companies/mine). */
const myItems = ref([])
const myLoading = ref(false)
const rows = computed(() => (isSuper.value ? companies.items : myItems.value))
const loading = computed(() =>
  isSuper.value ? companies.loading && !companies.loaded : myLoading.value)

const search = ref('')
const createOpen = ref(false)
const confirmOpen = ref(false)
const deleteTarget = ref(null)
const deleting = ref(false)
const togglingId = ref(null)

onMounted(loadData)

async function loadData() {
  if (isSuper.value) {
    companies.load(true)
    return
  }
  myLoading.value = true
  try {
    const res = await listMyCompanies()
    myItems.value = res.items || []
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить компании')
  } finally {
    myLoading.value = false
  }
}

const activeCount = computed(() => rows.value.filter((c) => c.is_active).length)
const disabledCount = computed(() => rows.value.filter((c) => !c.is_active).length)

const visible = computed(() => {
  const q = search.value.trim().toLowerCase()
  return q ? rows.value.filter((c) => c.name.toLowerCase().includes(q)) : rows.value
})

function initials(name) {
  if (!name) return '?'
  const parts = name.trim().split(/\s+/).slice(0, 2)
  return parts.map((p) => p[0]).join('').toUpperCase()
}

function creatorName(c) {
  return c.creator?.fio || c.creator?.name || null
}

const TONES = ['primary', 'secondary', 'tertiary']
function toneOf(c) {
  return TONES[(c.id || 0) % TONES.length]
}

// Создатель ли текущий пользователь этой компании (полные права на участников).
function isCreator(c) {
  return c.created_by != null && c.created_by === auth.userId
}

function roleBadge(c) {
  return isCreator(c) ? 'Создатель' : 'Администратор'
}

/* Какая компания открыта — в адресе (?company=), чтобы ссылка на карточку
   работала и переживала перезагрузку. */
const manageId = computed(() => {
  const raw = Number(route.query.company)
  return Number.isFinite(raw) && raw > 0 ? raw : null
})

function openManage(c) {
  router.push({ query: { ...route.query, section: 'companies', company: String(c.id) } })
}

function closeManage() {
  const query = { ...route.query }
  delete query.company
  router.push({ query })
}

function onDeleted() {
  closeManage()
  loadData()
}

// Поднятая из архива компания — новая строка списка: возвращаемся к нему,
// иначе человек остаётся в карточке прежней компании и решает, что ничего
// не произошло.
function onImported() {
  manageId.value = null
  loadData()
}

function openCreate() {
  createOpen.value = true
}

async function onToggle(c) {
  togglingId.value = c.id
  try {
    await companies.toggleActive(c.id, !c.is_active)
    notif.success(c.is_active ? 'Компания отключена' : 'Компания включена')
  } catch (e) {
    notif.error(e?.message || 'Не удалось переключить статус')
  } finally {
    togglingId.value = null
  }
}

function askDelete(c) {
  deleteTarget.value = c
  confirmOpen.value = true
}

async function doDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    await companies.remove(deleteTarget.value.id)
    notif.success('Компания удалена')
    confirmOpen.value = false
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить')
  } finally {
    deleting.value = false
  }
}
</script>

<style scoped>
/* Плашка с инициалами — единственное «лицо» компании: логотипов у них нет. */
.cs-avatar {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: var(--radius-md);
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.cs-avatar.tone-primary { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.cs-avatar.tone-secondary { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.cs-avatar.tone-tertiary { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }

.cs-create { margin-left: auto; }
.cs-meta { display: inline-flex; flex-wrap: wrap; gap: 4px; }
.cs-off { color: var(--color-error); }
.cs-chev { color: var(--color-text-dim); }
</style>
