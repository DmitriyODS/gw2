<template>
  <AppDialog
    :model-value="modelValue"
    title="Назначить форму"
    subtitle="Уровень доступа и срок ответа настраиваются для каждого адресата"
    size="md"
    :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
  >
    <AppStack :gap="14">
      <!-- Кому назначаем — вкладками: путей всего два, и держать ради них два
           отдельных диалога незачем. -->
      <AppTabs
        v-model="mode"
        :tabs="[{ value: 'user', label: 'Людям' }, { value: 'company', label: 'Компаниям' }]"
        variant="tint"
        dense
        full-width
      />
      <!-- Компанийный режим: выбираем из своих компаний, а не из каталога людей. -->
      <div v-if="company" class="fu-add">
        <Select
          v-model="pickCompany"
          :options="companies"
          option-label="name"
          option-value="id"
          placeholder="Компания"
          class="fu-pick"
        />
        <FormAccessSelect v-model="pickAccess" />
        <AppButton
          variant="filled" icon="add" label="Назначить"
          :disabled="!pickCompany || busy" :loading="busy"
          @click="addCompany"
        />
      </div>

      <!-- Людской режим: поиск по коллегам. -->
      <div v-else class="fu-add">
        <SearchField
          v-model="query"
          class="fu-search"
          placeholder="Имя или логин коллеги…"
          @update:model-value="onSearch"
        />
        <FormAccessSelect v-model="pickAccess" />
      </div>

      <!-- Срок ответа осмыслен только у назначения: смотрящему торопиться некуда. -->
      <AppRow
        v-if="pickAccess === 'respond'"
        title="Срок ответа"
        hint="За сутки до срока напомним тем, кто ещё не ответил"
        inline
      >
        <DatePicker
          v-model="pickDue"
          class="fu-date"
          show-time hour-format="24" date-format="dd.mm.yy"
          show-icon icon-display="input" show-button-bar
          placeholder="Без срока"
        />
      </AppRow>

      <ul v-if="!company && candidates.length" class="fu-found">
        <li v-for="u in candidates" :key="u.id" class="fu-item">
          <img class="fu-avatar" :src="avatarUrl(u)" :alt="u.fio" />
          <span class="fu-name">{{ u.fio }}</span>
          <span class="fu-spacer" />
          <AppButton size="sm" variant="glass" icon="add" label="Назначить" @click="addUser(u)" />
        </li>
      </ul>

      <div class="fu-current">
        <span class="fu-title">Уже назначено</span>
        <div v-if="loading" class="fu-empty">Загрузка…</div>
        <EmptyState
          v-else-if="!list.length"
          size="sm"
          icon="person_off"
          title="Пока никого"
          subtitle="Добавьте первого — форма появится у него сразу."
        />
        <ul v-else class="fu-list">
          <li v-for="s in list" :key="s.id" class="fu-item">
            <span class="material-symbols-outlined fu-icon">
              {{ s.company_id ? 'domain' : 'person' }}
            </span>
            <span class="fu-name">
              {{ s.name }}
              <small v-if="s.due_at" class="fu-due">до {{ dueText(s.due_at) }}</small>
            </span>
            <span class="fu-spacer" />
            <FormAccessSelect
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
      :message="`${removing?.name || ''} потеряет доступ к форме.`"
      confirm-label="Забрать" danger-confirm
      @confirm="remove" @cancel="removing = null"
    />
  </AppDialog>
</template>

<script setup>
/* Адресный доступ к форме: людям либо компании целиком. Уровень «Заполнить» и
   есть НАЗНАЧЕНИЕ — адресат получает уведомление, форма появляется у него во
   вкладке «Назначены», и его считает контроль исполнения. */
import { computed, ref, watch } from 'vue'
import DatePicker from 'primevue/datepicker'
import Select from 'primevue/select'
import AppButton from '@/components/ui/AppButton.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import FormAccessSelect from './FormAccessSelect.vue'
import { getAccess, getCompanies, getDirectory, shareWith, unshare } from '@/api/forms.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  formId: { type: [Number, null], default: null },
})
const emit = defineEmits(['update:modelValue', 'error', 'changed'])

// mode — кому назначаем: конкретным людям или компании целиком.
const mode = ref('user')
const company = computed(() => mode.value === 'company')

const list = ref([])
const loading = ref(false)
const busy = ref(false)
const removing = ref(null)

const query = ref('')
const candidates = ref([])
const pickAccess = ref('respond')
const pickDue = ref(null)
const pickCompany = ref(null)
const companies = ref([])

watch(() => props.modelValue, (open) => {
  if (!open) return
  mode.value = 'user'
  reload()
})

// Смена вкладки — другой список адресатов и другие кандидаты.
watch(mode, reload)

function reload() {
  if (!props.modelValue) return
  query.value = ''
  candidates.value = []
  pickAccess.value = 'respond'
  pickDue.value = null
  load()
  if (company.value) loadCompanies()
  else search('')
}

async function load() {
  loading.value = true
  try {
    const d = await getAccess(props.formId)
    // В своей модалке показываем только «своих» адресатов: людей или компании.
    list.value = (d.access ?? []).filter((s) => (company.value ? s.company_id : s.user_id))
  } catch (e) {
    emit('error', e?.message || 'Не удалось загрузить список доступа')
  } finally {
    loading.value = false
  }
}

async function loadCompanies() {
  try {
    const d = await getCompanies()
    companies.value = d.companies ?? []
    if (companies.value.length === 1) pickCompany.value = companies.value[0].id
  } catch (e) {
    emit('error', e?.message || 'Не удалось загрузить компании')
  }
}

// Поиск с задержкой: каталог тянется на паузу в наборе, а не на каждый символ.
let searchTimer = null
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => search(query.value.trim()), 300)
}

async function search(q) {
  try {
    const d = await getDirectory(q)
    const already = new Set(list.value.map((s) => s.user_id))
    candidates.value = (d.users ?? []).filter((u) => !already.has(u.id))
  } catch {
    candidates.value = []
  }
}

function addUser(u) {
  put([{ user_id: u.id, access: pickAccess.value, due_at: dueValue() }])
  candidates.value = candidates.value.filter((c) => c.id !== u.id)
}

function addCompany() {
  put([{ company_id: pickCompany.value, access: pickAccess.value, due_at: dueValue() }])
}

// Срок нужен только назначению; у прочих уровней он не имеет смысла.
function dueValue() {
  if (pickAccess.value !== 'respond' || !pickDue.value) return ''
  return new Date(pickDue.value).toISOString()
}

function changeAccess(s, access) {
  const target = s.company_id ? { company_id: s.company_id } : { user_id: s.user_id }
  put([{ ...target, access, due_at: s.due_at || '' }])
}

async function put(targets) {
  busy.value = true
  try {
    await shareWith(props.formId, targets)
    await load()
    emit('changed')
  } catch (e) {
    emit('error', e?.message || 'Не удалось выдать доступ')
  } finally {
    busy.value = false
  }
}

function avatarUrl(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function dueText(value) {
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleDateString('ru-RU')
}

function askRemove(s) {
  removing.value = s
}

async function remove() {
  const s = removing.value
  removing.value = null
  try {
    await unshare(props.formId, { userId: s.user_id, companyId: s.company_id })
    list.value = list.value.filter((x) => x.id !== s.id)
    emit('changed')
  } catch (e) {
    emit('error', e?.message || 'Не удалось забрать доступ')
  }
}
</script>

<style scoped>
.fu-add { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
.fu-search, .fu-pick { flex: 1; min-width: 200px; }
.fu-date { width: 230px; }

.fu-title { font-size: 13px; font-weight: 600; color: var(--color-text-dim); }
.fu-current { display: flex; flex-direction: column; gap: 8px; }
.fu-empty { padding: 12px; text-align: center; font-size: 14px; color: var(--color-text-dim); }

.fu-list, .fu-found { display: flex; flex-direction: column; gap: 8px; margin: 0; padding: 0; list-style: none; }

.fu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
  flex-wrap: wrap;
}

.fu-icon { font-size: 20px; color: var(--color-text-dim); }
.fu-avatar { width: 28px; height: 28px; border-radius: 50%; object-fit: cover; }
.fu-name { display: flex; flex-direction: column; font-size: 14px; overflow-wrap: anywhere; }
.fu-due { font-size: 12px; color: var(--color-text-dim); }
.fu-spacer { flex: 1; }
</style>
