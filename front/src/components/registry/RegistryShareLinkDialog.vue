<template>
  <AppDialog
    :model-value="modelValue"
    title="Поделиться по ссылке"
    subtitle="Кто знает ссылку — открывает реестр в браузере"
    size="md"
    :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
  >
    <AppStack :gap="14">
      <!-- Создание новой ссылки -->
      <div class="sl-new">
        <div class="sl-new-row">
          <InputText v-model="draft.name" class="sl-name" placeholder="Название ссылки, например «для подрядчика»" maxlength="120" />
          <RegistryAccessSelect v-model="draft.access" />
        </div>
        <div class="sl-new-row">
          <AppSwitch
            v-model="draft.require_auth"
            label="Требовать авторизацию"
          />
          <span class="sl-hint">
            С галочкой человек сначала войдёт в аккаунт или зарегистрируется — тогда в журнале
            переходов будет видно имя, а не только адрес.
          </span>
        </div>
        <AppButton
          variant="filled"
          icon="link"
          label="Создать ссылку"
          :loading="busy"
          @click="create"
        />
      </div>

      <div v-if="loading" class="sl-empty">Загрузка…</div>
      <EmptyState
        v-else-if="!shares.length"
        size="sm"
        icon="link_off"
        title="Ссылок пока нет"
        subtitle="Создайте первую — её можно будет отозвать в любой момент."
      />
      <ul v-else class="sl-list">
        <li v-for="s in shares" :key="s.id" class="sl-item">
          <div class="sl-item-head">
            <span class="sl-item-name">{{ s.name || 'Без названия' }}</span>
            <AppChip :label="accessLabel(s.access)" :tone="accessTone(s.access)" />
            <AppChip v-if="s.require_auth" icon="lock" label="только для вошедших" />
            <span class="sl-spacer" />
            <AppButton
              variant="icon" size="sm" icon="history"
              :title="`Переходов: ${s.visits}`"
              aria-label="Журнал переходов"
              @click="openVisits(s)"
            />
            <AppButton
              variant="icon" size="sm" icon="content_copy"
              title="Копировать" aria-label="Копировать"
              @click="copy(s.code)"
            />
            <AppButton
              tag="a" variant="icon" size="sm" icon="open_in_new"
              title="Открыть" aria-label="Открыть"
              :href="shareUrl(s.code)" target="_blank" rel="noopener"
            />
            <AppButton
              variant="icon" size="sm" tone="danger" icon="delete"
              title="Отозвать" aria-label="Отозвать"
              @click="askRevoke(s)"
            />
          </div>
          <input class="sl-url" :value="shareUrl(s.code)" readonly @focus="$event.target.select()" />
          <span class="sl-meta">
            Переходов: {{ s.visits }}<template v-if="s.last_visit_at">, последний {{ when(s.last_visit_at) }}</template>
          </span>
        </li>
      </ul>
    </AppStack>

    <ConfirmDialog
      :visible="!!revoking"
      header="Отозвать ссылку?"
      message="Ссылка перестанет открываться у всех, кому вы её давали."
      confirm-label="Отозвать" danger-confirm
      @confirm="revoke" @cancel="revoking = null"
    />

    <!-- Журнал переходов: у вошедшего — карточка человека, у гостя только адрес. -->
    <AppDialog
      v-model="visitsOpen"
      title="Кто открывал ссылку"
      :subtitle="visitsShare?.name || ''"
      size="md"
      :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
      @cancel="visitsOpen = false"
    >
      <div v-if="visitsLoading" class="sl-empty">Загрузка…</div>
      <EmptyState
        v-else-if="!visits.length"
        size="sm"
        icon="visibility_off"
        title="Переходов не было"
        subtitle="Никто ещё не открывал эту ссылку."
      />
      <ul v-else class="sl-visits">
        <li v-for="v in visits" :key="v.id" class="sl-visit">
          <template v-if="v.user_id">
            <button class="sl-visit-user" type="button" @click="openUser(v)">
              {{ v.user_name || 'Аккаунт' }}
            </button>
            <span class="sl-visit-login">@{{ v.user_login }}</span>
          </template>
          <span v-else class="sl-visit-guest">Гость · {{ v.ip || 'адрес неизвестен' }}</span>
          <span class="sl-spacer" />
          <span class="sl-visit-when">{{ when(v.visited_at) }}</span>
        </li>
      </ul>
    </AppDialog>

    <!-- Карточка перешедшего: тот же вид, что и в «Сотрудниках», — оттуда ему
         можно сразу написать. -->
    <EmployeeProfileDialog v-model="userOpen" :user="userCard" elevated />
  </AppDialog>
</template>

<script setup>
/* Внешние ссылки реестра: создание, уровень доступа, требование входа и журнал
   переходов.

   Журнал — ответ на вопрос «кто это видел»: у вошедшего есть имя и логин (по
   ним открывается карточка сотрудника), у гостя остаются только адрес и время. */
import { ref, watch } from 'vue'
import InputText from 'primevue/inputtext'
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitch from '@/components/ui/AppSwitch.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import EmployeeProfileDialog from '@/components/common/EmployeeProfileDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RegistryAccessSelect from './RegistryAccessSelect.vue'
import { createShare, getShareVisits, getShares, revokeShare } from '@/api/registries.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  registryId: { type: [Number, null], default: null },
})
const emit = defineEmits(['update:modelValue', 'error', 'copied'])

const shares = ref([])
const loading = ref(false)
const busy = ref(false)
const revoking = ref(null)
const draft = ref({ name: '', access: 'view', require_auth: false })

const userOpen = ref(false)
const userCard = ref(null)

/* Журнал знает про перешедшего ровно имя и логин — карточке этого хватает,
   остальное она добирает сама. */
function openUser(visit) {
  userCard.value = { id: visit.user_id, fio: visit.user_name, login: visit.user_login }
  userOpen.value = true
}

const visitsOpen = ref(false)
const visitsShare = ref(null)
const visits = ref([])
const visitsLoading = ref(false)

function shareUrl(code) {
  return `${location.origin}/registry/${code}`
}

function accessLabel(access) {
  return { edit: 'редактирование', admin: 'администрирование' }[access] || 'просмотр'
}
function accessTone(access) {
  return { edit: 'primary', admin: 'warning' }[access] || 'neutral'
}

function when(iso) {
  const d = new Date(iso)
  return isNaN(d) ? '' : d.toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' })
}

watch(() => props.modelValue, (open) => {
  if (open) load()
})

async function load() {
  loading.value = true
  try {
    const d = await getShares(props.registryId)
    shares.value = d.shares ?? []
  } catch (e) {
    emit('error', e?.message || 'Не удалось загрузить ссылки')
  } finally {
    loading.value = false
  }
}

async function create() {
  busy.value = true
  try {
    const s = await createShare(props.registryId, { ...draft.value })
    shares.value.unshift({ ...s, visits: 0 })
    draft.value = { name: '', access: 'view', require_auth: false }
  } catch (e) {
    emit('error', e?.message || 'Не удалось создать ссылку')
  } finally {
    busy.value = false
  }
}

function askRevoke(s) {
  revoking.value = s
}

async function revoke() {
  const s = revoking.value
  revoking.value = null
  try {
    await revokeShare(props.registryId, s.id)
    shares.value = shares.value.filter((x) => x.id !== s.id)
  } catch (e) {
    emit('error', e?.message || 'Не удалось отозвать ссылку')
  }
}

async function copy(code) {
  try {
    await navigator.clipboard.writeText(shareUrl(code))
    emit('copied')
  } catch { /* буфер недоступен — человек скопирует из поля руками */ }
}

async function openVisits(s) {
  visitsShare.value = s
  visitsOpen.value = true
  visitsLoading.value = true
  try {
    const d = await getShareVisits(props.registryId, s.id)
    visits.value = d.visits ?? []
  } catch (e) {
    emit('error', e?.message || 'Не удалось загрузить журнал')
  } finally {
    visitsLoading.value = false
  }
}
</script>

<style scoped>
.sl-new { display: flex; flex-direction: column; gap: 10px; align-items: flex-start; }
.sl-new-row { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; width: 100%; }
.sl-name { flex: 1; min-width: 0; }
.sl-hint { flex: 1; min-width: 200px; font-size: 12px; line-height: 1.4; color: var(--color-text-dim); }

.sl-empty { padding: 16px; text-align: center; font-size: 14px; color: var(--color-text-dim); }

.sl-list, .sl-visits { display: flex; flex-direction: column; gap: 10px; margin: 0; padding: 0; list-style: none; }

.sl-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
}

.sl-item-head { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.sl-item-name { font-size: 14px; font-weight: 600; overflow-wrap: anywhere; }
.sl-spacer { flex: 1; }
.sl-meta { font-size: 12px; color: var(--color-text-dim); }

.sl-url {
  width: 100%;
  height: 36px;
  padding: 0 12px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  color: var(--color-text);
  font-size: 13px;
}

.sl-visit { display: flex; align-items: center; gap: 8px; padding: 8px 0; border-bottom: 1px solid var(--acrylic-border); }
.sl-visit:last-child { border-bottom: none; }

.sl-visit-user {
  border: none;
  background: none;
  padding: 0;
  color: var(--color-primary);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

.sl-visit-login, .sl-visit-when, .sl-visit-guest { font-size: 12px; color: var(--color-text-dim); }
</style>
