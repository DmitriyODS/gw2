<template>
  <AppDialog
    :model-value="modelValue"
    title="Поделиться расписанием"
    subtitle="Открыть можно только на чтение — вести занятия остаётесь вы"
    size="md"
    :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
  >
    <AppStack :gap="14">
      <AppTabs
        v-model="mode"
        :tabs="[{ value: 'link', label: 'Ссылка' }, { value: 'people', label: 'Люди и компании' }]"
        variant="tint"
        dense
        full-width
      />

      <!-- Публичная ссылка: код в адресе и есть доступ, вход не нужен. -->
      <template v-if="mode === 'link'">
        <EmptyState
          v-if="!links.length && !loading"
          size="sm"
          icon="link"
          title="Ссылок нет"
          subtitle="По ссылке расписание открывается без входа в аккаунт."
        />
        <ul v-else class="sh-list">
          <li v-for="s in links" :key="s.id" class="sh-item">
            <span class="material-symbols-outlined sh-icon">link</span>
            <span class="sh-name">{{ linkOf(s) }}</span>
            <span class="sh-spacer" />
            <AppButton variant="icon" size="sm" icon="content_copy" label="Скопировать" @click="copy(s)" />
            <AppButton variant="icon" size="sm" tone="danger" icon="close" label="Отозвать" @click="revoke(s)" />
          </li>
        </ul>
        <AppButton variant="glass" icon="add_link" label="Создать ссылку" :loading="busy" @click="addLink" />
      </template>

      <!-- Адресный доступ: человеку либо компании целиком. -->
      <template v-else>
        <div class="sh-add">
          <SearchField v-model="query" class="sh-search" placeholder="Имя или логин коллеги…" @update:model-value="onSearch" />
        </div>
        <ul v-if="candidates.length" class="sh-list">
          <li v-for="u in candidates" :key="u.id" class="sh-item">
            <span class="material-symbols-outlined sh-icon">person</span>
            <span class="sh-name">{{ u.fio }}<small v-if="u.post"> · {{ u.post }}</small></span>
            <span class="sh-spacer" />
            <AppButton size="sm" variant="glass" icon="add" label="Открыть" @click="addUser(u)" />
          </li>
        </ul>

        <div v-if="companies.length" class="sh-add">
          <select v-model="pickCompany" class="ctl">
            <option :value="null">Компания целиком…</option>
            <option v-for="c in companies" :key="c.id" :value="c.id">{{ c.name }}</option>
          </select>
          <AppButton
            variant="filled"
            icon="add"
            label="Открыть"
            :disabled="!pickCompany || busy"
            @click="addCompany"
          />
        </div>

        <div class="sh-current">
          <span class="sh-title">Уже открыто</span>
          <EmptyState
            v-if="!access.length && !loading"
            size="sm"
            icon="person_off"
            title="Пока никому"
            subtitle="Открытое расписание появится у человека во вкладке «Поделились»."
          />
          <ul v-else class="sh-list">
            <li v-for="s in access" :key="s.id" class="sh-item">
              <span class="material-symbols-outlined sh-icon">{{ s.company_id ? 'domain' : 'person' }}</span>
              <span class="sh-name">{{ s.name }}</span>
              <span class="sh-spacer" />
              <AppButton variant="icon" size="sm" tone="danger" icon="close" label="Забрать доступ" @click="drop(s)" />
            </li>
          </ul>
        </div>
      </template>

      <span v-if="error" class="sh-error">{{ error }}</span>
    </AppStack>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import * as api from '@/api/schedules.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  schedule: { type: Object, required: true },
})

defineEmits(['update:modelValue'])

const notif = useNotificationsStore()

const mode = ref('link')
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const links = ref([])
const access = ref([])
const candidates = ref([])
const companies = ref([])
const query = ref('')
const pickCompany = ref(null)

watch(() => props.modelValue, async (open) => {
  if (!open) return
  error.value = ''
  candidates.value = []
  query.value = ''
  loading.value = true
  try {
    const [shares, users, comps] = await Promise.all([
      api.getShares(props.schedule.id),
      api.getUserShares(props.schedule.id),
      api.getCompanies(),
    ])
    links.value = shares.shares || []
    access.value = users.access || []
    companies.value = comps.companies || []
  } catch (e) {
    error.value = e?.message || 'Не удалось загрузить доступы'
  } finally {
    loading.value = false
  }
})

function linkOf(share) {
  return `${window.location.origin}/schedule/s/${share.code}`
}

async function copy(share) {
  try {
    await navigator.clipboard.writeText(linkOf(share))
    notif.success('Ссылка скопирована')
  } catch {
    error.value = 'Браузер не дал скопировать — выделите адрес вручную'
  }
}

async function addLink() {
  busy.value = true
  try {
    links.value = [await api.createShare(props.schedule.id), ...links.value]
  } catch (e) {
    error.value = e?.message || 'Не удалось создать ссылку'
  } finally {
    busy.value = false
  }
}

async function revoke(share) {
  try {
    await api.revokeShare(props.schedule.id, share.id)
    links.value = links.value.filter((s) => s.id !== share.id)
  } catch (e) {
    error.value = e?.message || 'Не удалось отозвать ссылку'
  }
}

let searchSeq = 0
async function onSearch(value) {
  const seq = ++searchSeq
  if (!value?.trim()) { candidates.value = []; return }
  try {
    const data = await api.getDirectory(value.trim())
    if (seq === searchSeq) candidates.value = data.users || []
  } catch { /* поиск коллег не критичен */ }
}

async function addUser(user) {
  await grant({ user_id: user.id })
  candidates.value = candidates.value.filter((u) => u.id !== user.id)
}

async function addCompany() {
  await grant({ company_id: pickCompany.value })
  pickCompany.value = null
}

async function grant(target) {
  busy.value = true
  try {
    await api.shareWith(props.schedule.id, target)
    // Имя адресата приходит только списком, поэтому перечитываем его: иначе
    // свежая строка осталась бы безымянной.
    access.value = (await api.getUserShares(props.schedule.id)).access || []
  } catch (e) {
    error.value = e?.message || 'Не удалось открыть доступ'
  } finally {
    busy.value = false
  }
}

async function drop(share) {
  try {
    await api.unshare(props.schedule.id, share.company_id
      ? { company_id: share.company_id }
      : { user_id: share.user_id })
    access.value = access.value.filter((s) => s.id !== share.id)
  } catch (e) {
    error.value = e?.message || 'Не удалось забрать доступ'
  }
}
</script>

<style scoped>
.sh-list { display: flex; flex-direction: column; gap: 4px; margin: 0; padding: 0; list-style: none; }
.sh-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-container-lowest);
}
.sh-icon { font-size: 18px; color: var(--color-on-surface-variant); }
.sh-name { min-width: 0; overflow-wrap: anywhere; font-size: 13px; }
.sh-name small { color: var(--color-on-surface-variant); }
.sh-spacer { flex: 1; }

.sh-add { display: flex; align-items: center; gap: 8px; }
.sh-search, .sh-add .ctl { flex: 1; min-width: 0; }
.sh-current { display: flex; flex-direction: column; gap: 6px; }
.sh-title { font-size: 12px; color: var(--color-on-surface-variant); }
.sh-error { font-size: 13px; color: var(--color-error); }
</style>
