<template>
  <!-- Платформенный ИИ: ОДИН ключ proxy-api на всю платформу и каталог
       моделей. Цена модели задаёт стоимость обращения в токенах доступа —
       правка каталога сразу меняет тарификацию для всех пользователей. -->
  <div class="tab">
    <BrandLoader v-if="loading" block :size="64" />

    <AppStack v-else :gap="14">
      <AppCard>
        <AppRow
          title="Ключ платформы"
          :hint="settings.has_key ? `Подключён ключ ${settings.key_hint}` : 'Ключ не задан — ИИ недоступен никому'"
          inline
        >
          <label class="toggle">
            <InputSwitch v-model="form.enabled" />
            <span>ИИ включён</span>
          </label>
        </AppRow>

        <div class="form-row">
          <label class="field grow">
            <span class="field-label">Новый ключ API</span>
            <InputText v-model="form.apiKey" type="password" autocomplete="off" placeholder="sk-…" />
          </label>
          <label class="field grow">
            <span class="field-label">Адрес API</span>
            <InputText v-model="form.baseURL" placeholder="https://api.proxyapi.ru/openai/v1" />
          </label>
        </div>

        <div class="form-row">
          <label class="field">
            <span class="field-label">Модель по умолчанию</span>
            <InputText v-model="form.modelChat" />
          </label>
          <label class="field">
            <span class="field-label">Модель эмбеддингов</span>
            <InputText v-model="form.modelEmbedding" />
          </label>
          <label class="field">
            <span class="field-label">Модель техподдержки</span>
            <InputText v-model="form.modelSupport" />
          </label>
        </div>

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
            :loading="testing"
            @click="test"
          />
          <AppButton v-if="settings.has_key" label="Удалить ключ" tone="danger" @click="clearKey" />
          <span v-if="testResult" class="test" :data-ok="testResult.chat">
            {{ testResult.chat ? `Связь есть · ${testResult.latency_ms} мс` : (testResult.error || 'Не отвечает') }}
          </span>
        </AppStack>
      </AppCard>

      <h3 class="group-title">Модели и цены</h3>
      <p class="note">
        Токен доступа = 1000 токенов самой дешёвой модели каталога. Остальные
        расходуют его пропорционально цене — коэффициент считается сам.
      </p>

      <AppCard v-for="m in models" :key="m.code" :gap="10">
        <div class="model-head">
          <strong>{{ m.code }}</strong>
          <AppChip size="sm" tone="primary" :label="m.kind === 'embedding' ? 'эмбеддинги' : 'чат'" />
          <span class="rate">×{{ m.rate.toFixed(2) }}</span>
        </div>
        <div class="form-row">
          <label class="field">
            <span class="field-label">Название в интерфейсе</span>
            <InputText v-model="m.title" />
          </label>
          <label class="field">
            <span class="field-label">Цена, руб. за 1 млн токенов</span>
            <InputNumber
              :model-value="m.price_per_mtok / 100"
              :min="0"
              :max-fraction-digits="2"
              @update:model-value="v => m.price_per_mtok = Math.round((v || 0) * 100)"
            />
          </label>
          <label class="toggle">
            <InputSwitch v-model="m.selectable" />
            <span>Выбирают пользователи</span>
          </label>
          <label class="toggle">
            <InputSwitch v-model="m.is_active" />
            <span>Доступна</span>
          </label>
        </div>
      </AppCard>

      <AppButton
        class="self-start"
        label="Сохранить каталог"
        icon="check"
        variant="filled"
        :loading="saving"
        @click="save"
      />
    </AppStack>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import InputSwitch from 'primevue/inputswitch'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import * as aiApi from '@/api/ai.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const notif = useNotificationsStore()

const loading = ref(true)
const saving = ref(false)
const testing = ref(false)
const settings = ref({})
const models = ref([])
const testResult = ref(null)
const form = ref({
  enabled: false, apiKey: '', baseURL: '',
  modelChat: '', modelEmbedding: '', modelSupport: '',
})

onMounted(load)

async function load() {
  loading.value = true
  try {
    settings.value = await aiApi.getPlatformAi()
    models.value = settings.value.models ?? []
    form.value = {
      enabled: settings.value.enabled,
      apiKey: '',
      baseURL: settings.value.base_url || '',
      modelChat: settings.value.model_chat || '',
      modelEmbedding: settings.value.model_embedding || '',
      modelSupport: settings.value.model_support || '',
    }
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    settings.value = await aiApi.updatePlatformAi({
      enabled: form.value.enabled,
      base_url: form.value.baseURL,
      model_chat: form.value.modelChat,
      model_embedding: form.value.modelEmbedding,
      model_support: form.value.modelSupport,
      models: models.value,
      ...(form.value.apiKey ? { api_key: form.value.apiKey } : {}),
    })
    models.value = settings.value.models ?? []
    form.value.apiKey = ''
    notif.notify({ severity: 'success', summary: 'Настройки ИИ сохранены', life: 3000 })
  } catch (e) {
    notif.notify({
      severity: 'error',
      summary: 'Не сохранилось',
      detail: e?.data?.message || '',
      life: 5000,
    })
  } finally {
    saving.value = false
  }
}

async function clearKey() {
  saving.value = true
  try {
    settings.value = await aiApi.updatePlatformAi({ clear_key: true })
    models.value = settings.value.models ?? []
    notif.notify({ severity: 'info', summary: 'Ключ удалён', life: 3000 })
  } finally {
    saving.value = false
  }
}

async function test() {
  testing.value = true
  testResult.value = null
  try {
    testResult.value = await aiApi.testPlatformAi()
  } catch (e) {
    testResult.value = { chat: false, error: e?.data?.message || 'Не удалось проверить' }
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.tab { display: flex; flex-direction: column; gap: 14px; }
.group-title { margin: 4px 0 0; font-size: 1.1rem; font-weight: 700; }
.note { margin: 0; font-size: 0.85rem; color: var(--color-text-dim); }
.field-label { font-size: 0.85rem; color: var(--color-text-dim); }
.form-row { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 12px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.grow { flex: 1; min-width: 220px; }
.toggle { display: flex; align-items: center; gap: 8px; font-size: 0.85rem; }
.test { font-size: 0.82rem; color: var(--color-error); }
.test[data-ok='true'] { color: var(--color-success, var(--color-primary)); }
.model-head { display: flex; align-items: center; gap: 10px; }
.rate { margin-left: auto; font-weight: 700; }
.self-start { align-self: flex-start; }
</style>
