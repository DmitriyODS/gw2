<template>
  <!-- «ИИ возможности» пользователя: что включено, какой моделью отвечать,
       сколько токенов осталось и — отдельно, для тех, у кого свой доступ —
       собственный ключ со своим адресом API. -->
  <div class="ai-sec">
    <BrandLoader v-if="loading" :size="48" />

    <AppStack v-else :gap="16">
      <AppCard
        title="ИИ возможности"
        hint="Работают на ключе платформы: каждому выдаётся дневная норма токенов,
          и она тратится на ответы модели. Свой ключ подключать не обязательно."
      >
        <template #head>
          <AppChip
            :tone="form.enabled ? 'success' : 'neutral'"
            :label="form.enabled ? 'Включено' : 'Выключено'"
          />
        </template>

        <AppInfoBar
          v-if="!settings.platform_ready"
          tone="warning"
          message="ИИ на платформе пока не настроен — обратитесь к администратору платформы
            или подключите свой ключ ниже."
        />

        <AppSwitchRow
          v-model="form.enabled"
          title="ИИ включён"
          hint="Общий выключатель: без него не работает ни одна возможность"
        />
        <AppSwitchRow
          v-model="form.featAssistant"
          title="Hola-ассистент — чат"
          hint="Ответы на вопросы по задачам и статистике во вкладке «Чат»"
        />
        <AppSwitchRow
          v-model="form.featNotes"
          title="ИИ в заметках"
          hint="Правка выделенного текста, корректура и продолжение записи"
        />

        <!-- Выбор модели закрыт (settings.model_locked) — показываем, чем
             отвечает ИИ, без возможности переключиться. -->
        <AppRow
          v-if="modelLocked"
          title="Модель"
          hint="Пока все работают на модели платформы. Выбор откроется позже."
        >
          <AppChip :label="modelTitle" />
        </AppRow>
        <AppRow
          v-else
          title="Модель"
          hint="Модели отвечают по-разному и расходуют разное количество токенов — выберите ту, что больше нравится."
          stack
        >
          <div class="model-row">
            <button
              v-for="m in settings.models"
              :key="m.code"
              class="model-btn"
              :class="{ 'is-active': form.modelChat === m.code }"
              type="button"
              @click="form.modelChat = m.code"
            >
              <strong>{{ m.title }}</strong>
            </button>
          </div>
        </AppRow>

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

      <!-- Сколько токенов израсходовано и сколько осталось на сегодня. -->
      <AppCard>
        <AppRow title="Токены ИИ" :hint="tokensHint">
          <strong class="tokens-left">{{ formatCount(settings.tokens_left) }}</strong>
        </AppRow>
        <div v-if="!ownKeyActive && usage && usage.tokens_limit > 0" class="bar">
          <span :style="{ width: usedPct }" />
        </div>
        <ul v-if="featureUsage.length" class="usage-list">
          <li v-for="row in featureUsage" :key="row.key">
            <span>{{ row.label }}</span>
            <b>{{ formatCount(row.value) }}</b>
          </li>
        </ul>
      </AppCard>

      <!-- Свой ключ — отдельная настройка: она нужна единицам, поэтому свёрнута. -->
      <AppCard>
        <button class="own-toggle" type="button" @click="ownOpen = !ownOpen">
          <span class="material-symbols-outlined">{{ ownOpen ? 'expand_less' : 'expand_more' }}</span>
          <span>
            <span class="own-title">Свой ключ и сервер модели</span>
            <span class="own-hint">
              {{ settings.has_key ? `Подключён ключ ${settings.key_hint || ''}` : 'Не обязательно: без него работает ключ платформы' }}
            </span>
          </span>
        </button>

        <div v-if="ownOpen" class="own-body">
          <label class="ai-label" for="ai-key">Ключ API</label>
          <InputText
            id="ai-key"
            v-model="form.apiKey"
            class="ai-input"
            type="password"
            autocomplete="off"
            :placeholder="settings.has_key ? (settings.key_hint || 'Ключ сохранён') : 'sk-…'"
          />
          <label class="ai-label" for="ai-url">Адрес API</label>
          <InputText
            id="ai-url"
            v-model="form.apiBaseURL"
            class="ai-input"
            autocomplete="off"
            placeholder="https://api.proxyapi.ru/openai/v1"
          />
          <p class="ai-note">
            С собственным ключом запросы уходят на ваш сервер, а токены тарифа
            не тратятся. Ключ шифруется на сервере и наружу не отдаётся.
          </p>
          <AppStack row :gap="10">
            <AppButton label="Сохранить ключ" icon="check" variant="filled" :loading="saving" @click="save" />
            <AppButton
              v-if="settings.has_key"
              label="Отключить ключ"
              icon="link_off"
              tone="danger"
              :disabled="saving"
              @click="clearKey"
            />
          </AppStack>
        </div>
      </AppCard>
    </AppStack>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import BrandLoader from '@/components/common/BrandLoader.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'
import * as aiApi from '@/api/ai.js'
import * as billingApi from '@/api/billing.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { formatCount } from '@/utils/money.js'

const notif = useNotificationsStore()

const FEATURE_LABELS = {
  assistant: 'Чат Hola',
  notes: 'Заметки',
  search: 'Поиск по задачам',
  tv_fact: 'Факты ТВ-режима',
  support: 'Техподдержка',
}

const loading = ref(true)
const saving = ref(false)
const testing = ref(false)
const ownOpen = ref(false)
const settings = ref({ models: [], tokens_left: -1 })
const usage = ref(null)
const testResult = ref(null)

const form = ref({
  enabled: true,
  featAssistant: true,
  featNotes: true,
  modelChat: '',
  apiKey: '',
  apiBaseURL: '',
})

const ownKeyActive = computed(() => Boolean(settings.value.has_key))

/* Закрыт ли выбор модели, решает сервер (model_locked): на своём ключе он
   открыт даже когда для остальных закрыт — там платит сам пользователь. */
const modelLocked = computed(() => Boolean(settings.value.model_locked))

const modelTitle = computed(() => {
  const code = settings.value.model_chat
  return settings.value.models?.find((m) => m.code === code)?.title || code || '—'
})

const tokensHint = computed(() => {
  if (ownKeyActive.value) return 'Работает ваш ключ — дневная норма не расходуется.'
  if (!usage.value) return ''
  return `Использовано ${formatCount(usage.value.tokens_used)} из ${formatCount(usage.value.tokens_limit)} за сегодня`
})

const usedPct = computed(() => {
  const limit = usage.value?.tokens_limit || 0
  if (limit <= 0) return '0%'
  return `${Math.min(100, Math.round((usage.value.tokens_used / limit) * 100))}%`
})

const featureUsage = computed(() => {
  const by = usage.value?.by_feature || {}
  return Object.entries(by)
    .filter(([, value]) => value > 0)
    .map(([key, value]) => ({ key, label: FEATURE_LABELS[key] || key, value }))
})

onMounted(load)

async function load() {
  loading.value = true
  try {
    settings.value = await aiApi.getMyAiSettings()
    form.value = {
      enabled: settings.value.enabled,
      featAssistant: settings.value.feat_assistant,
      featNotes: settings.value.feat_notes,
      modelChat: settings.value.model_chat,
      apiKey: '',
      apiBaseURL: settings.value.api_base_url || '',
    }
    // Расход по возможностям считает биллинг — он же ведёт баланс токенов.
    usage.value = await billingApi.getAiUsage().catch(() => null)
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    settings.value = await aiApi.updateMyAiSettings({
      enabled: form.value.enabled,
      feat_assistant: form.value.featAssistant,
      feat_notes: form.value.featNotes,
      model_chat: form.value.modelChat,
      api_base_url: form.value.apiBaseURL,
      ...(form.value.apiKey ? { api_key: form.value.apiKey } : {}),
    })
    form.value.apiKey = ''
    notif.notify({ severity: 'success', summary: 'Настройки сохранены', life: 3000 })
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
    settings.value = await aiApi.updateMyAiSettings({ clear_key: true })
    form.value.apiKey = ''
    form.value.apiBaseURL = ''
    notif.notify({ severity: 'info', summary: 'Ключ отключён', life: 3000 })
  } finally {
    saving.value = false
  }
}

async function test() {
  testing.value = true
  testResult.value = null
  try {
    testResult.value = await aiApi.testMyAiSettings()
  } catch (e) {
    testResult.value = { chat: false, error: e?.data?.message || 'Не удалось проверить' }
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.ai-sec { display: flex; flex-direction: column; gap: 14px; }
.ai-label { font-size: 0.85rem; font-weight: 600; }
.ai-input { width: 100%; }
.ai-note { margin: 0; font-size: 0.8rem; color: var(--color-text-dim); }

.model-row { display: flex; flex-wrap: wrap; gap: 10px; width: 100%; }

.model-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 130px;
  padding: 14px 18px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--color-surface-variant);
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
}

.model-btn.is-active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}


.ai-test { font-size: 0.82rem; color: var(--color-error); }
.ai-test[data-ok='true'] { color: var(--color-success, var(--color-primary)); }

.tokens-left { font-size: 1.2rem; font-weight: 700; white-space: nowrap; }

.bar {
  width: 100%;
  height: 10px;
  border-radius: 999px;
  background: var(--color-surface-variant);
  overflow: hidden;
}

.bar span { display: block; height: 100%; background: var(--color-primary); }

.usage-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.usage-list li {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border-radius: var(--radius-md);
  background: var(--color-surface-variant);
  font-size: 0.82rem;
}

.own-toggle {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 0;
  border: none;
  background: none;
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
}

.own-toggle > span:last-child { display: flex; flex-direction: column; }
.own-title { font-weight: 600; }
.own-hint { font-size: 0.85rem; color: var(--color-text-dim); }
.own-body { display: flex; flex-direction: column; gap: 10px; margin-top: 14px; }
</style>
