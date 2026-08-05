<template>
  <!-- ИИ-возможности КОМПАНИИ. Своего ключа у компании нет: умный поиск задач и
       факты ТВ-режима работают на ключе платформы и тратят токены СОЗДАТЕЛЯ
       компании — поэтому разрешение на трату даёт только он. -->
  <div class="ai-settings">
    <AppCard v-if="!companyId">
      <AppRow
        title="Сначала выберите компанию"
        hint="ИИ-возможности настраиваются для конкретной компании."
      />
    </AppCard>

    <template v-else>
      <BrandLoader v-if="loading" :size="48" />

      <template v-else>
        <AppCard
          title="ИИ возможности компании"
          hint="Работают на ключе платформы. Токены списываются с баланса
            создателя компании — участникам свои ключи не нужны."
        >
          <AppInfoBar
            v-if="!settings.platform_ready"
            tone="warning"
            message="ИИ на платформе не настроен — тумблеры ниже пока ничего не включат."
          />

          <AppSwitchRow
            v-model="form.enabled"
            title="ИИ включён в компании"
            hint="Общий выключатель компанийных возможностей"
          />
          <AppSwitchRow
            v-model="form.shared"
            title="Тратить мои токены на компанию"
            :hint="ownerHint"
            :disabled="!isOwner"
          />
          <AppSwitchRow
            v-model="form.featSearch"
            title="Умный поиск по задачам"
            hint="Выключен — поиск ищет по словам, без учёта регистра"
          />
          <AppSwitchRow
            v-model="form.featTVFact"
            title="Интересные факты в ТВ-режиме"
            hint="Короткая заметка дня на табло"
          />

          <p v-if="settings.owner_tokens_left >= 0" class="ai-note">
            У создателя компании осталось {{ formatCount(settings.owner_tokens_left) }} токенов.
          </p>

          <AppStack row :gap="10">
            <AppButton
              :label="saving ? 'Сохраняем…' : 'Сохранить'"
              icon="check"
              variant="filled"
              :loading="saving"
              @click="save"
            />
            <AppButton
              :label="testing ? 'Проверяем…' : 'Проверить связь'"
              icon="wifi_tethering"
              :loading="testing"
              @click="test"
            />
            <span v-if="testResult" class="ai-test" :data-ok="testResult.chat">
              {{ testResult.chat ? `Связь есть · ${testResult.latency_ms} мс` : (testResult.error || 'Не отвечает') }}
            </span>
          </AppStack>
        </AppCard>

        <!-- Индексация задач нужна умному поиску: без эмбеддингов он пуст. -->
        <AppCard v-if="indexing">
          <AppRow title="Индексация задач">
            <template #hint>
              Проиндексировано {{ indexing.indexed }} из {{ indexing.total_tasks }}
              <template v-if="indexing.pending > 0"> · осталось {{ indexing.pending }}</template>
            </template>
            <AppButton
              :label="reindexing ? 'Запускаем…' : 'Переиндексировать'"
              size="sm"
              :loading="reindexing"
              @click="reindex"
            />
          </AppRow>
        </AppCard>
      </template>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import BrandLoader from '@/components/common/BrandLoader.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'
import { useCompaniesStore } from '@/stores/companies.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import {
  getAiSettings, updateAiSettings, testAiSettings,
  getAiIndexingStatus, reindexAiTasks,
} from '@/api/ai.js'
import { formatCount } from '@/utils/money.js'

const props = defineProps({
  companyId: { type: Number, default: null },
  // ownerId — создатель компании: только он разрешает тратить свои токены.
  ownerId: { type: Number, default: null },
})

const companies = useCompaniesStore()
const auth = useAuthStore()
const notif = useNotificationsStore()
const { effectiveCompanyId } = storeToRefs(companies)

const companyId = computed(() => props.companyId ?? effectiveCompanyId.value)
const isOwner = computed(() => !props.ownerId || props.ownerId === auth.user?.id)
const ownerHint = computed(() => (isOwner.value
  ? 'Без разрешения ИИ компании не работает'
  : 'Разрешение даёт создатель компании'))

const loading = ref(true)
const saving = ref(false)
const testing = ref(false)
const reindexing = ref(false)
const settings = ref({})
const indexing = ref(null)
const testResult = ref(null)
const form = ref({ enabled: false, shared: false, featSearch: true, featTVFact: true })

let indexingTimer = null

onMounted(load)
onUnmounted(() => clearInterval(indexingTimer))
watch(companyId, load)

async function load() {
  if (!companyId.value) {
    loading.value = false
    return
  }
  loading.value = true
  testResult.value = null
  try {
    settings.value = await getAiSettings(companyId.value)
    form.value = {
      enabled: !!settings.value.enabled,
      shared: !!settings.value.shared,
      featSearch: !!settings.value.feat_search,
      featTVFact: !!settings.value.feat_tv_fact,
    }
    await loadIndexing()
  } catch (e) {
    notif.error(e.message || 'Не удалось загрузить настройки ИИ')
  } finally {
    loading.value = false
  }
}

async function loadIndexing() {
  if (!companyId.value || !form.value.enabled) {
    indexing.value = null
    return
  }
  try {
    indexing.value = await getAiIndexingStatus(companyId.value)
  } catch {
    indexing.value = null
  }
}

async function save() {
  saving.value = true
  try {
    const payload = {
      enabled: form.value.enabled,
      feat_search: form.value.featSearch,
      feat_tv_fact: form.value.featTVFact,
    }
    // shared отправляем только от создателя — иначе сервер ответит отказом.
    if (isOwner.value) payload.shared = form.value.shared
    settings.value = await updateAiSettings(companyId.value, payload)
    notif.success('Настройки ИИ сохранены')
    await loadIndexing()
  } catch (e) {
    notif.error(e.message || 'Не удалось сохранить настройки ИИ')
  } finally {
    saving.value = false
  }
}

async function test() {
  testing.value = true
  testResult.value = null
  try {
    testResult.value = await testAiSettings(companyId.value)
  } catch (e) {
    testResult.value = { chat: false, error: e?.data?.message || e.message || 'Не удалось проверить' }
  } finally {
    testing.value = false
  }
}

async function reindex() {
  reindexing.value = true
  try {
    const r = await reindexAiTasks(companyId.value)
    notif.success(r.pending > 0
      ? `Запущена индексация ${r.pending} задач — займёт пару минут`
      : 'Все задачи уже проиндексированы')
    // Опрашиваем статус, пока очередь не опустеет.
    clearInterval(indexingTimer)
    indexingTimer = setInterval(async () => {
      await loadIndexing()
      if (!indexing.value || indexing.value.pending === 0) {
        clearInterval(indexingTimer)
        indexingTimer = null
      }
    }, 5000)
  } catch (e) {
    notif.error(e.message || 'Не удалось запустить переиндексацию')
  } finally {
    reindexing.value = false
  }
}
</script>

<style scoped>
.ai-settings { display: flex; flex-direction: column; gap: 14px; }
.ai-note { margin: 0; font-size: 0.85rem; color: var(--color-text-dim); }
.ai-test { font-size: 0.82rem; color: var(--color-error); }
.ai-test[data-ok='true'] { color: var(--color-success, var(--color-primary)); }
</style>
