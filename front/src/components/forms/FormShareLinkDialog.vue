<template>
  <AppDialog
    :model-value="modelValue"
    title="Ссылки на форму"
    subtitle="Кто знает код — открывает форму и отвечает"
    size="md"
    :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
  >
    <AppStack :gap="14">
      <div class="sl-add">
        <InputText v-model="name" class="sl-name" placeholder="Название ссылки (для себя)" maxlength="120" />
        <AppSwitch v-model="requireAuth" label="Только для вошедших" />
        <AppButton
          variant="filled" icon="add_link" label="Создать"
          :loading="busy" @click="create"
        />
      </div>

      <AppInfoBar
        v-if="form?.status === 'draft'"
        tone="warning"
        icon="visibility_off"
        message="Форма в черновике — по ссылке она пока не открывается. Включите приём ответов в настройках."
      />

      <div v-if="loading" class="sl-empty">Загрузка…</div>
      <EmptyState
        v-else-if="!shares.length"
        size="sm"
        icon="link_off"
        title="Ссылок нет"
        subtitle="Создайте первую — её можно отправить кому угодно."
      />

      <AppStack v-else :gap="8">
        <AppCard v-for="s in shares" :key="s.id" :gap="8">
          <div class="sl-row">
            <span class="sl-title">{{ s.name || 'Без названия' }}</span>
            <AppChip v-if="s.require_auth" size="sm" icon="lock" label="Для своих" />
            <span class="sl-spacer" />
            <AppButton
              variant="icon" size="sm" icon="content_copy"
              title="Скопировать ссылку" aria-label="Скопировать ссылку"
              @click="copy(s)"
            />
            <AppButton
              variant="icon" size="sm" icon="history"
              title="Журнал переходов" aria-label="Журнал переходов"
              @click="openVisits(s)"
            />
            <AppButton
              variant="icon" size="sm" tone="danger" icon="delete"
              title="Отозвать ссылку" aria-label="Отозвать ссылку"
              @click="removing = s"
            />
          </div>
          <span class="sl-url">{{ linkOf(s) }}</span>
          <div class="sl-stats">
            <AppChip size="sm" icon="visibility" :label="`${s.visits} переходов`" />
            <AppChip size="sm" icon="forum" :label="`${s.responses} ответов`" />
            <span v-if="s.last_visit_at" class="sl-note">
              последний — {{ dateText(s.last_visit_at) }}
            </span>
          </div>

          <div v-if="visitsOf === s.id" class="sl-visits">
            <div v-if="!visits.length" class="sl-empty">Переходов пока нет</div>
            <div v-for="v in visits" :key="v.id" class="sl-visit">
              <span class="sl-visit-who">{{ v.user_name || 'Гость' }}</span>
              <span class="sl-note">{{ v.ip }} · {{ dateText(v.visited_at) }}</span>
            </div>
          </div>
        </AppCard>
      </AppStack>
    </AppStack>

    <ConfirmDialog
      :visible="!!removing"
      header="Отозвать ссылку?"
      message="Форма перестанет открываться по этому адресу. Уже полученные ответы останутся."
      confirm-label="Отозвать" danger-confirm
      @confirm="revoke" @cancel="removing = null"
    />
  </AppDialog>
</template>

<script setup>
/* Внешние ссылки на заполнение. Код в адресе — capability: кто его знает, тот
   и отвечает; флажок «только для вошедших» требует входа, и тогда в журнале у
   перехода есть имя, а не только адрес. */
import { ref, watch } from 'vue'
import InputText from 'primevue/inputtext'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitch from '@/components/ui/AppSwitch.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { createShare, getShareVisits, getShares, revokeShare } from '@/api/forms.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  formId: { type: [Number, null], default: null },
  form: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue', 'error', 'copied'])

const shares = ref([])
const loading = ref(false)
const busy = ref(false)
const name = ref('')
const requireAuth = ref(false)
const removing = ref(null)
const visitsOf = ref(null)
const visits = ref([])

watch(() => props.modelValue, (open) => {
  if (!open) return
  name.value = ''
  requireAuth.value = false
  visitsOf.value = null
  load()
})

async function load() {
  loading.value = true
  try {
    const d = await getShares(props.formId)
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
    await createShare(props.formId, { name: name.value.trim(), require_auth: requireAuth.value })
    name.value = ''
    await load()
  } catch (e) {
    emit('error', e?.message || 'Не удалось создать ссылку')
  } finally {
    busy.value = false
  }
}

async function revoke() {
  const s = removing.value
  removing.value = null
  try {
    await revokeShare(props.formId, s.id)
    shares.value = shares.value.filter((x) => x.id !== s.id)
  } catch (e) {
    emit('error', e?.message || 'Не удалось отозвать ссылку')
  }
}

function linkOf(s) {
  return `${window.location.origin}/form/${s.code}`
}

async function copy(s) {
  try {
    await navigator.clipboard.writeText(linkOf(s))
    emit('copied')
  } catch {
    emit('error', 'Браузер не дал скопировать — выделите адрес вручную')
  }
}

async function openVisits(s) {
  if (visitsOf.value === s.id) {
    visitsOf.value = null
    return
  }
  visitsOf.value = s.id
  visits.value = []
  try {
    const d = await getShareVisits(props.formId, s.id, 50)
    visits.value = d.visits ?? []
  } catch (e) {
    emit('error', e?.message || 'Не удалось загрузить журнал')
  }
}

function dateText(value) {
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleString('ru-RU')
}
</script>

<style scoped>
.sl-add { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
.sl-name { flex: 1; min-width: 200px; }

.sl-row { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.sl-title { font-size: 14px; font-weight: 600; overflow-wrap: anywhere; }
.sl-spacer { flex: 1; }

.sl-url {
  font-size: 12px;
  color: var(--color-text-dim);
  overflow-wrap: anywhere;
}

.sl-stats { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.sl-note { font-size: 12px; color: var(--color-text-dim); }
.sl-empty { padding: 10px; text-align: center; font-size: 13px; color: var(--color-text-dim); }

.sl-visits {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-top: 8px;
  border-top: 1px solid var(--acrylic-border);
}
.sl-visit { display: flex; gap: 8px; align-items: baseline; flex-wrap: wrap; }
.sl-visit-who { font-size: 13px; }
</style>
