<template>
  <!-- «ИИ возможности» пользователя: что включено, какой моделью отвечать,
       сколько токенов осталось и — отдельно, для тех, у кого свой доступ —
       собственный ключ со своим адресом API. -->
  <div class="ai-sec">
    <BrandLoader v-if="loading" :size="48" />

    <template v-else>
      <section class="gw-card ai-card">
        <header class="ai-head">
          <div class="ai-head-text">
            <h3 class="gw-h">ИИ возможности</h3>
            <p class="gw-sub">
              Работают на ключе платформы: тариф выдаёт токены, и они тратятся
              на ответы модели. Свой ключ подключать не обязательно.
            </p>
          </div>
          <span class="ai-state" :data-on="form.enabled">{{ form.enabled ? 'Включено' : 'Выключено' }}</span>
        </header>

        <p v-if="!settings.platform_ready" class="ai-warn">
          ИИ на платформе пока не настроен — обратитесь к администратору
          платформы или подключите свой ключ ниже.
        </p>

        <SwitchRow
          v-model="form.enabled"
          title="ИИ включён"
          hint="Общий выключатель: без него не работает ни одна возможность"
        />
        <SwitchRow
          v-model="form.featAssistant"
          title="Hola-ассистент — чат"
          hint="Ответы на вопросы по задачам и статистике во вкладке «Чат»"
        />
        <SwitchRow
          v-model="form.featNotes"
          title="ИИ в заметках"
          hint="Правка выделенного текста, корректура и продолжение записи"
        />

        <SettingRow
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
        </SettingRow>

        <div class="ai-actions">
          <button class="btn-grad" type="button" :disabled="saving" @click="save">
            <span class="material-symbols-outlined">check</span>
            {{ saving ? 'Сохраняем…' : 'Сохранить' }}
          </button>
          <button class="btn-glass" type="button" :disabled="testing" @click="test">
            <span class="material-symbols-outlined">wifi_tethering</span>
            {{ testing ? 'Проверяем…' : 'Проверить связь' }}
          </button>
          <span v-if="testResult" class="ai-test" :data-ok="testResult.chat">
            {{ testResult.chat ? `Связь есть · ${testResult.latency_ms} мс` : (testResult.error || 'Не отвечает') }}
          </span>
        </div>
      </section>

      <!-- Токены тарифа: сколько осталось и где докупить. -->
      <section class="gw-card tokens-card">
        <SettingRow title="Токены ИИ" :hint="tokensHint">
          <strong class="tokens-left">{{ formatCount(settings.tokens_left) }}</strong>
          <button class="gw-chip" type="button" @click="goToStore">Перейти в магазин</button>
        </SettingRow>
        <div v-if="!ownKeyActive && usage && usage.tokens_limit > 0" class="bar">
          <span :style="{ width: usedPct }" />
        </div>
        <ul v-if="featureUsage.length" class="usage-list">
          <li v-for="row in featureUsage" :key="row.key">
            <span>{{ row.label }}</span>
            <b>{{ formatCount(row.value) }}</b>
          </li>
        </ul>
      </section>

      <!-- Свой ключ — отдельная настройка: она нужна единицам, поэтому свёрнута. -->
      <section class="gw-card">
        <button class="own-toggle" type="button" @click="ownOpen = !ownOpen">
          <span class="material-symbols-outlined">{{ ownOpen ? 'expand_less' : 'expand_more' }}</span>
          <span>
            <span class="gw-h">Свой ключ и сервер модели</span>
            <span class="gw-sub">
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
          <div class="ai-actions">
            <button class="btn-grad" type="button" :disabled="saving" @click="save">
              <span class="material-symbols-outlined">check</span>
              Сохранить ключ
            </button>
            <button
              v-if="settings.has_key"
              class="btn-glass"
              type="button"
              :disabled="saving"
              @click="clearKey"
            >
              <span class="material-symbols-outlined">link_off</span>
              Отключить ключ
            </button>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import InputText from 'primevue/inputtext'
import BrandLoader from '@/components/common/BrandLoader.vue'
import SettingRow from '@/components/common/SettingRow.vue'
import SwitchRow from '@/components/common/SwitchRow.vue'
import * as aiApi from '@/api/ai.js'
import * as billingApi from '@/api/billing.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { formatCount } from '@/utils/money.js'

const router = useRouter()
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

const tokensHint = computed(() => {
  if (ownKeyActive.value) return 'Работает ваш ключ — токены тарифа не расходуются.'
  if (!usage.value) return ''
  return `Использовано ${formatCount(usage.value.tokens_used)} из ${formatCount(usage.value.tokens_limit)} за текущий месяц`
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

function goToStore() {
  router.push('/store?tab=subs').catch(() => {})
}
</script>

<style scoped>
.ai-sec { display: flex; flex-direction: column; gap: 14px; }
.ai-card { display: flex; flex-direction: column; gap: 12px; }

.ai-head { display: flex; align-items: flex-start; gap: 12px; }

.ai-head-text { flex: 1; min-width: 0; }

.ai-state {
  padding: 6px 12px;
  border-radius: 999px;
  background: var(--color-surface-variant);
  font-size: 0.78rem;
  font-weight: 600;
}

.ai-state[data-on='true'] {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.ai-warn {
  margin: 0;
  padding: 10px 12px;
  border-radius: var(--radius-md);
  background: var(--color-warning-container, var(--color-surface-variant));
  font-size: 0.85rem;
}

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

.ai-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; }

.ai-test { font-size: 0.82rem; color: var(--color-error); }
.ai-test[data-ok='true'] { color: var(--color-success, var(--color-primary)); }

.tokens-card { display: flex; flex-direction: column; gap: 10px; }
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
.own-body { display: flex; flex-direction: column; gap: 10px; margin-top: 14px; }
</style>
