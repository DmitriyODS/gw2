<template>
  <AppDialog
    :model-value="modelValue"
    :title="company ? 'Поделиться с компанией' : 'Поделиться с людьми'"
    :subtitle="company
      ? 'Реестр увидят все сотрудники выбранной компании'
      : 'Доступ настраивается для каждого отдельно'"
    size="md"
    :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
  >
    <AppStack :gap="14">
      <!-- Компанийный режим: выбирать не из каталога людей, а из своих компаний. -->
      <div v-if="company" class="su-add">
        <Select
          v-model="pickCompany"
          :options="companies"
          option-label="name"
          option-value="id"
          placeholder="Компания"
          class="su-pick"
        />
        <RegistryAccessSelect v-model="pickAccess" />
        <AppButton
          variant="filled" icon="add" label="Открыть доступ"
          :disabled="!pickCompany || busy" :loading="busy"
          @click="addCompany"
        />
      </div>

      <!-- Людской режим: поиск по коллегам, как при создании диалога в мессенджере. -->
      <div v-else class="su-add">
        <SearchField
          v-model="query"
          class="su-search"
          placeholder="Имя или логин коллеги…"
          @update:model-value="onSearch"
        />
        <RegistryAccessSelect v-model="pickAccess" />
      </div>

      <ul v-if="!company && candidates.length" class="su-found">
        <li v-for="u in candidates" :key="u.id" class="su-found-item">
          <img class="su-avatar" :src="avatarUrl(u)" :alt="u.fio" />
          <span class="su-found-name">{{ u.fio }}</span>
          <span class="su-spacer" />
          <AppButton size="sm" variant="glass" icon="add" label="Добавить" @click="addUser(u)" />
        </li>
      </ul>

      <div class="su-current">
        <span class="su-title">Уже есть доступ</span>
        <div v-if="loading" class="su-empty">Загрузка…</div>
        <EmptyState
          v-else-if="!list.length"
          size="sm"
          icon="person_off"
          title="Пока никого"
          subtitle="Добавьте первого — он увидит реестр сразу."
        />
        <ul v-else class="su-list">
          <li v-for="s in list" :key="s.id" class="su-item">
            <span class="material-symbols-outlined su-icon">
              {{ s.company_id ? 'domain' : 'person' }}
            </span>
            <span class="su-name">{{ s.name }}</span>
            <span class="su-spacer" />
            <RegistryAccessSelect
              :model-value="s.access"
              @update:model-value="changeAccess(s, $event)"
            />
            <AppButton
              variant="icon" size="sm" tone="danger" icon="close"
              title="Забрать доступ" aria-label="Забрать доступ"
              @click="askRemove(s)"
            />
          </li>
        </ul>
      </div>
    </AppStack>

    <ConfirmDialog
      :visible="!!removing"
      header="Забрать доступ?"
      :message="`${removing?.name || ''} потеряет доступ к реестру.`"
      confirm-label="Забрать" danger-confirm
      @confirm="remove" @cancel="removing = null"
    />
  </AppDialog>
</template>

<script setup>
/* Адресный доступ: людям либо компании целиком. Оба режима — одна форма,
   различаются источником адресатов; уровень доступа настраивается для каждого
   адресата отдельно и меняется прямо в списке. */
import { ref, watch } from 'vue'
import Select from 'primevue/select'
import AppButton from '@/components/ui/AppButton.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppStack from '@/components/ui/AppStack.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import RegistryAccessSelect from './RegistryAccessSelect.vue'
import { getAccess, getCompanies, getDirectory, shareWith, unshare } from '@/api/registries.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  registryId: { type: [Number, null], default: null },
  /** true — режим «поделиться с компанией». */
  company: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'error', 'changed'])

const list = ref([])
const loading = ref(false)
const busy = ref(false)
const removing = ref(null)

const query = ref('')
const candidates = ref([])
const pickAccess = ref('view')
const pickCompany = ref(null)
const companies = ref([])

watch(() => props.modelValue, (open) => {
  if (!open) return
  query.value = ''
  candidates.value = []
  pickAccess.value = 'view'
  load()
  if (props.company) loadCompanies()
  else search('')
})

async function load() {
  loading.value = true
  try {
    const d = await getAccess(props.registryId)
    // В своей модалке показываем только «своих» адресатов: людей или компании.
    list.value = (d.access ?? []).filter((s) => (props.company ? s.company_id : s.user_id))
  } catch (e) {
    emit('error', e?.message || 'Не удалось загрузить список доступа')
  } finally {
    loading.value = false
  }
}

async function loadCompanies() {
  try {
    const d = await getCompanies()
    companies.value = d.items ?? []
    if (companies.value.length === 1) pickCompany.value = companies.value[0].id
  } catch (e) {
    emit('error', e?.message || 'Не удалось загрузить компании')
  }
}

// Поиск с задержкой: каталог тянется на каждый ввод, а не на каждый символ.
let searchTimer = null
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => search(query.value.trim()), 300)
}

async function search(q) {
  try {
    const d = await getDirectory(q)
    const already = new Set(list.value.map((s) => s.user_id))
    candidates.value = (d.items ?? []).filter((u) => !already.has(u.id))
  } catch {
    candidates.value = []
  }
}

async function addUser(u) {
  await put([{ user_id: u.id, access: pickAccess.value }])
  candidates.value = candidates.value.filter((c) => c.id !== u.id)
}

async function addCompany() {
  await put([{ company_id: pickCompany.value, access: pickAccess.value }])
}

async function changeAccess(s, access) {
  await put([s.company_id ? { company_id: s.company_id, access } : { user_id: s.user_id, access }])
}

async function put(targets) {
  busy.value = true
  try {
    await shareWith(props.registryId, targets)
    await load()
    emit('changed')
  } catch (e) {
    emit('error', e?.message || 'Не удалось выдать доступ')
  } finally {
    busy.value = false
  }
}

// Аватар: загруженное фото либо автоматический identicon — тот же порядок, что
// и всюду на платформе.
function avatarUrl(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function askRemove(s) {
  removing.value = s
}

async function remove() {
  const s = removing.value
  removing.value = null
  try {
    await unshare(props.registryId, { userId: s.user_id, companyId: s.company_id })
    list.value = list.value.filter((x) => x.id !== s.id)
    emit('changed')
  } catch (e) {
    emit('error', e?.message || 'Не удалось забрать доступ')
  }
}
</script>

<style scoped>
.su-add { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
.su-search, .su-pick { flex: 1; min-width: 200px; }

.su-title { font-size: 13px; font-weight: 600; color: var(--color-text-dim); }
.su-current { display: flex; flex-direction: column; gap: 8px; }
.su-empty { padding: 12px; text-align: center; font-size: 14px; color: var(--color-text-dim); }

.su-list, .su-found { display: flex; flex-direction: column; gap: 8px; margin: 0; padding: 0; list-style: none; }

.su-item, .su-found-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
  flex-wrap: wrap;
}

.su-icon { font-size: 20px; color: var(--color-text-dim); }
.su-avatar { width: 28px; height: 28px; border-radius: 50%; object-fit: cover; }
.su-name, .su-found-name { font-size: 14px; overflow-wrap: anywhere; }
.su-spacer { flex: 1; }
</style>
