<template>
  <div class="ai-settings">
    <div v-if="!companyId" class="settings-card ai-empty">
      <div class="hero-icon" data-tone="secondary">
        <span class="material-symbols-outlined">domain</span>
      </div>
      <div class="card-text">
        <h3>Сначала выберите компанию</h3>
        <p>Используйте селектор компании в шапке, чтобы перейти к её настройкам ИИ.</p>
      </div>
    </div>

    <template v-else>
      <!-- Главная карточка: включение + ключ + модели -->
      <section class="settings-card ai-card">
        <header class="ai-card-head">
          <div class="hero-icon" data-tone="tertiary">
            <span class="material-symbols-outlined">smart_toy</span>
          </div>
          <div class="card-text">
            <h3>ИИ-функции через ProxyAPI</h3>
            <p>
              Включите интеграцию и укажите свой ключ от
              <a href="https://proxyapi.ru" target="_blank" rel="noopener">ProxyAPI</a>.
              Ключ хранится в зашифрованном виде — другие сотрудники его не видят.
            </p>
          </div>
        </header>

        <div class="ai-form">
          <!-- Toggle — стиль switch-row из CompanyFormDialog -->
          <label class="switch-row">
            <span class="switch-text">
              <span class="material-symbols-outlined">power_settings_new</span>
              <span>
                <strong>Включить ИИ для компании</strong>
                <small>Без действующего ключа функции остаются выключенными.</small>
              </span>
            </span>
            <input type="checkbox" v-model="form.enabled" class="switch" />
          </label>

          <!-- API key -->
          <div class="field">
            <label class="lbl">Ключ ProxyAPI</label>
            <div class="key-row">
              <input
                v-model="form.api_key"
                :type="showKey ? 'text' : 'password'"
                :placeholder="hasKey ? `Текущий: ${currentHint}` : 'sk-…'"
                autocomplete="off"
                spellcheck="false"
                class="ctl"
              />
              <button
                type="button"
                class="icon-btn"
                @click="showKey = !showKey"
                :title="showKey ? 'Скрыть' : 'Показать'"
              >
                <span class="material-symbols-outlined">
                  {{ showKey ? 'visibility_off' : 'visibility' }}
                </span>
              </button>
              <button
                v-if="hasKey"
                type="button"
                class="icon-btn danger"
                @click="onClearKey"
                title="Удалить ключ"
              >
                <span class="material-symbols-outlined">delete</span>
              </button>
            </div>
            <div v-if="hasKey" class="hint">
              Оставьте поле пустым, чтобы не менять текущий ключ.
            </div>
          </div>

          <!-- Models -->
          <div class="row-2">
            <div class="field">
              <label class="lbl">Модель чата</label>
              <input v-model="form.model_chat" placeholder="gpt-4o-mini" class="ctl" />
            </div>
            <div class="field">
              <label class="lbl">Модель эмбеддингов</label>
              <input v-model="form.model_embedding" placeholder="text-embedding-3-small" class="ctl" />
            </div>
          </div>

          <div class="hint">
            При смене модели эмбеддингов потребуется переиндексировать задачи.
          </div>
        </div>

        <footer class="ai-actions">
          <button class="btn-outlined" :disabled="testing || !hasKey || dirty" @click="onTest">
            <span class="material-symbols-outlined">network_check</span>
            {{ testing ? 'Проверяю…' : 'Проверить связь' }}
          </button>
          <button class="btn-filled" :disabled="saving || !dirty" @click="onSave">
            <span class="material-symbols-outlined">save</span>
            {{ saving ? 'Сохраняю…' : 'Сохранить' }}
          </button>
        </footer>

        <div
          v-if="testResult"
          class="ai-test-result"
          :class="testResult.ok ? 'ok' : 'err'"
        >
          <span class="material-symbols-outlined">
            {{ testResult.ok ? 'check_circle' : 'error' }}
          </span>
          <span>{{ testResult.text }}</span>
        </div>
      </section>

      <!-- Индексация задач (для семантического поиска) -->
      <section v-if="hasKey && form.enabled" class="settings-card ai-index-card">
        <div class="hero-icon" data-tone="primary">
          <span class="material-symbols-outlined">database</span>
        </div>
        <div class="card-text">
          <h3>Индексация задач</h3>
          <p>
            Семантический поиск работает только по задачам, для которых
            посчитаны эмбеддинги. Новые задачи индексируются автоматически.
            Существующие — запусти разовый бэкфилл здесь.
          </p>
          <div v-if="indexing" class="index-stats">
            <span class="stat-pill">
              <span class="material-symbols-outlined">task_alt</span>
              {{ indexing.indexed }} / {{ indexing.total_tasks }} задач проиндексировано
            </span>
            <span v-if="indexing.pending > 0" class="stat-pill warn">
              <span class="material-symbols-outlined">pending</span>
              осталось {{ indexing.pending }}
            </span>
          </div>
        </div>
        <div class="card-actions">
          <button
            class="btn-filled"
            :disabled="reindexing || !indexing || indexing.pending === 0"
            @click="onReindex"
          >
            <span class="material-symbols-outlined">refresh</span>
            {{ reindexing ? 'Запускаю…' : 'Переиндексировать' }}
          </button>
        </div>
      </section>

      <!-- Карточка-подсказка про фичи -->
      <section class="settings-card ai-hint-card">
        <div class="hero-icon" data-tone="secondary">
          <span class="material-symbols-outlined">tips_and_updates</span>
        </div>
        <div class="card-text">
          <h3>Что работает при включённом ИИ</h3>
          <ul class="ai-feat-list">
            <li>
              <b>Факт дня в ТВ-режиме.</b>
              Один из слайдов показывает свежий познавательный или контекстный факт. Обновляется раз в час.
            </li>
            <li>
              <b>Семантический поиск задач.</b>
              Поиск ищет по смыслу запроса по всей базе задач. Без ключа — обычный точный поиск.
            </li>
          </ul>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { storeToRefs } from 'pinia'
import { useCompaniesStore } from '@/stores/companies.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import {
  getAiSettings, updateAiSettings, testAiSettings,
  getAiIndexingStatus, reindexAiTasks,
} from '@/api/ai.js'

const props = defineProps({ companyId: { type: Number, default: null } })
const companies = useCompaniesStore()
const notif = useNotificationsStore()
const { effectiveCompanyId } = storeToRefs(companies)

// Явный companyId (страница управления компанией) приоритетнее активной компании.
const companyId = computed(() => props.companyId ?? effectiveCompanyId.value)

const initial = ref(null)
const form = ref({
  enabled: false,
  api_key: '',
  model_chat: 'gpt-4o-mini',
  model_embedding: 'text-embedding-3-small',
})
const hasKey = ref(false)
const currentHint = ref('')
const showKey = ref(false)
const saving = ref(false)
const testing = ref(false)
const testResult = ref(null)
const indexing = ref(null)            // {total_tasks, indexed, pending, ai_enabled}
const reindexing = ref(false)
let indexingTimer = null

const dirty = computed(() => {
  if (!initial.value) return false
  const i = initial.value
  return (
    form.value.enabled !== i.enabled ||
    form.value.model_chat !== i.model_chat ||
    form.value.model_embedding !== i.model_embedding ||
    form.value.api_key.trim() !== ''
  )
})

async function load() {
  if (!companyId.value) return
  testResult.value = null
  try {
    const data = await getAiSettings(companyId.value)
    initial.value = {
      enabled: !!data.enabled,
      model_chat: data.model_chat || 'gpt-4o-mini',
      model_embedding: data.model_embedding || 'text-embedding-3-small',
    }
    form.value = { ...initial.value, api_key: '' }
    hasKey.value = !!data.has_key
    currentHint.value = data.key_hint || ''
    showKey.value = false
  } catch (e) {
    notif.error(e.message || 'Не удалось загрузить настройки ИИ')
  }
  loadIndexing()
}

async function loadIndexing() {
  if (!companyId.value || !hasKey.value || !form.value.enabled) {
    indexing.value = null
    return
  }
  try {
    indexing.value = await getAiIndexingStatus(companyId.value)
  } catch {
    indexing.value = null
  }
}

async function onReindex() {
  if (!companyId.value) return
  reindexing.value = true
  try {
    const r = await reindexAiTasks(companyId.value)
    notif.success(
      r.pending > 0
        ? `Запущена индексация ${r.pending} задач — займёт пару минут`
        : 'Все задачи уже проиндексированы'
    )
    // Опрашиваем статус каждые 5 сек, пока pending не упадёт до 0.
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

async function onSave() {
  if (!companyId.value) return
  saving.value = true
  testResult.value = null
  try {
    const payload = {
      enabled: form.value.enabled,
      model_chat: form.value.model_chat.trim() || 'gpt-4o-mini',
      model_embedding: form.value.model_embedding.trim() || 'text-embedding-3-small',
    }
    const key = form.value.api_key.trim()
    if (key) payload.api_key = key
    const data = await updateAiSettings(companyId.value, payload)
    initial.value = {
      enabled: !!data.enabled,
      model_chat: data.model_chat,
      model_embedding: data.model_embedding,
    }
    form.value = { ...initial.value, api_key: '' }
    hasKey.value = !!data.has_key
    currentHint.value = data.key_hint || ''
    notif.success('Настройки ИИ сохранены')
  } catch (e) {
    notif.error(e.message || 'Не удалось сохранить настройки ИИ')
  } finally {
    saving.value = false
  }
}

async function onClearKey() {
  if (!companyId.value) return
  if (!confirm('Удалить сохранённый ключ? Это выключит все ИИ-функции.')) return
  saving.value = true
  testResult.value = null
  try {
    const data = await updateAiSettings(companyId.value, { clear_key: true })
    hasKey.value = !!data.has_key
    currentHint.value = data.key_hint || ''
    notif.success('Ключ удалён')
  } catch (e) {
    notif.error(e.message || 'Не удалось удалить ключ')
  } finally {
    saving.value = false
  }
}

async function onTest() {
  if (!companyId.value) return
  testing.value = true
  testResult.value = null
  try {
    const r = await testAiSettings(companyId.value)
    const ok = r.chat && r.embedding
    testResult.value = {
      ok,
      text: ok
        ? `Связь установлена (chat + embeddings, ${r.latency_ms} мс)`
        : `Ошибка: ${r.error || 'модель не ответила'}`,
    }
  } catch (e) {
    testResult.value = {
      ok: false,
      text: e.message || 'Не удалось проверить связь',
    }
  } finally {
    testing.value = false
  }
}

onMounted(load)
watch(companyId, load)
onBeforeUnmount(() => {
  if (indexingTimer) { clearInterval(indexingTimer); indexingTimer = null }
})
</script>

<style scoped>
/* Используем тот же визуальный язык, что и в SettingsView (.settings-card,
   .hero-icon, .btn-filled / .btn-outlined). Дублируем правила в scoped,
   потому что stylesheet родителя не доезжает в дочерний компонент. */
.ai-settings {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ── settings-card (общий каркас) ───────────────────────────── */
.settings-card {
  display: flex;
  align-items: flex-start;
  gap: 18px;
  padding: 20px 22px;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--color-outline-dim);
  border-radius: 20px;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.settings-card:hover {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--color-outline-dim));
}

.hero-icon {
  flex-shrink: 0;
  width: 56px;
  height: 56px;
  border-radius: 16px;
  display: grid;
  place-items: center;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}
.hero-icon[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.hero-icon[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.hero-icon[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }
.hero-icon[data-tone="error"]     { --tone-bg: var(--color-error-container);     --tone-fg: var(--color-on-error-container); }
.hero-icon .material-symbols-outlined { font-size: 28px; }

.card-text { flex: 1; min-width: 0; }
.card-text h3 {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text);
}
.card-text p {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-text-dim);
}
.card-text p a { color: var(--color-primary); text-decoration: none; }
.card-text p a:hover { text-decoration: underline; }
.card-text b { color: var(--color-text); }

/* ── главная AI-карточка: layout колоночный (шапка + форма + actions) ─ */
.ai-card {
  flex-direction: column;
  align-items: stretch;
  gap: 18px;
}
.ai-card-head {
  display: flex;
  align-items: flex-start;
  gap: 18px;
}

.ai-empty {
  align-items: center;
}

/* ── форма (стиль 1:1 с CompanyFormDialog: .field / .lbl / .ctl) ── */
.ai-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.field { display: flex; flex-direction: column; gap: 6px; }
.lbl { font-size: 13px; font-weight: 600; color: var(--color-on-surface-variant); }

.ctl {
  appearance: none;
  width: 100%;
  border: 1px solid var(--color-outline-variant);
  background: var(--acrylic-card-bg);
  color: var(--color-on-surface);
  padding: 10px 12px;
  border-radius: var(--radius-md, 12px);
  font: inherit;
  transition: border-color .15s, box-shadow .15s;
}
.ctl:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px color-mix(in oklab, var(--color-primary) 18%, transparent);
}
.ctl::placeholder { color: var(--color-on-surface-variant); }

.hint {
  font-size: 12px;
  color: var(--color-on-surface-variant);
  line-height: 1.4;
}

.row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
@media (max-width: 640px) {
  .row-2 { grid-template-columns: 1fr; }
}

/* поле «ключ» — инпут + 1-2 иконки справа в одну строку */
.key-row {
  display: flex;
  gap: 6px;
  align-items: stretch;
}
.key-row .ctl { flex: 1; min-width: 0; }

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  min-height: 40px;
  border-radius: var(--radius-md, 12px);
  border: 1px solid var(--color-outline-variant);
  background: var(--acrylic-card-bg);
  color: var(--color-on-surface-variant);
  cursor: pointer;
  transition: background 0.12s, color 0.12s, border-color 0.12s;
}
.icon-btn:hover { background: var(--color-surface-high); color: var(--color-on-surface); }
.icon-btn.danger:hover { color: var(--color-error); border-color: var(--color-error); }
.icon-btn .material-symbols-outlined { font-size: 20px; }

/* ── switch-row (1:1 с CompanyFormDialog) ─────────────────── */
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  background: var(--color-surface-container);
  border-radius: var(--radius-md, 12px);
  cursor: pointer;
  transition: background .12s;
}
.switch-row:hover { background: var(--color-surface-high); }
.switch-text {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.switch-text .material-symbols-outlined {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md, 12px);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 20px;
  flex: none;
}
.switch-text strong { display: block; font-size: 14px; color: var(--color-on-surface); }
.switch-text small { display: block; font-size: 12px; color: var(--color-on-surface-variant); }

.switch {
  appearance: none;
  width: 44px;
  height: 24px;
  border-radius: 999px;
  background: var(--color-surface-highest, var(--color-surface-high));
  border: 2px solid var(--color-outline, var(--color-outline-variant));
  box-sizing: border-box;
  position: relative;
  cursor: pointer;
  outline: none;
  transition: background .18s, border-color .18s;
  flex: none;
}
.switch::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 4px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--color-outline, var(--color-on-surface-variant));
  transform: translateY(-50%);
  transition: transform .2s cubic-bezier(0.4, 0, 0.2, 1),
              background .2s, width .2s, height .2s, left .2s;
}
.switch:checked {
  background: var(--color-primary);
  border-color: var(--color-primary);
}
.switch:checked::after {
  width: 16px;
  height: 16px;
  left: 24px;
  background: var(--color-on-primary);
}

/* ── actions: кнопки одной строкой справа (как в backup) ───── */
.ai-actions {
  display: flex;
  gap: 10px;
  justify-content: flex-end;
  flex-wrap: wrap;
}

/* кнопки — те же что в SettingsView (повторяем в scoped). */
.btn-filled, .btn-outlined {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 18px;
  border-radius: 999px;
  border: none;
  font: inherit;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s, box-shadow 0.15s;
}
.btn-filled {
  background: var(--color-primary);
  color: var(--color-on-primary);
}
.btn-filled:hover:not(:disabled) {
  background: color-mix(in oklch, var(--color-primary) 88%, var(--color-on-primary) 12%);
}
.btn-filled:disabled { opacity: 0.55; cursor: not-allowed; }
.btn-filled .material-symbols-outlined { font-size: 18px; }

.btn-outlined {
  background: transparent;
  color: var(--color-text);
  border: 1px solid var(--color-outline-variant);
}
.btn-outlined:hover:not(:disabled) {
  background: color-mix(in oklch, var(--color-primary) 6%, transparent);
  border-color: var(--color-primary);
}
.btn-outlined:disabled { opacity: 0.55; cursor: not-allowed; }
.btn-outlined .material-symbols-outlined { font-size: 18px; }

/* ── результат теста ───────────────────────────────────────── */
.ai-test-result {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 14px;
}
.ai-test-result.ok {
  background: var(--color-success-container, color-mix(in oklch, var(--color-primary) 18%, transparent));
  color: var(--color-on-success-container, var(--color-text));
}
.ai-test-result.err {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}
.ai-test-result .material-symbols-outlined { font-size: 20px; }

/* ── карточка индексации ───────────────────────────────────── */
.ai-index-card { align-items: center; }
.card-actions { flex-shrink: 0; }
.index-stats {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 10px;
}
.stat-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 999px;
  background: var(--color-surface-container);
  color: var(--color-on-surface);
  font-size: 12px;
  font-weight: 500;
}
.stat-pill .material-symbols-outlined { font-size: 16px; }
.stat-pill.warn {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}

/* ── карточка-подсказка ────────────────────────────────────── */
.ai-feat-list { margin: 6px 0 0; padding-left: 18px; }
.ai-feat-list li {
  margin: 6px 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-text-dim);
}
.ai-feat-list li b { color: var(--color-text); }

/* ── Adaptive: мобильный ≤768 ───────────────────────────────── */
@media (max-width: 768px) {
  .ai-settings { gap: 12px; }

  .settings-card {
    padding: 16px;
    gap: 12px;
    border-radius: 18px;
  }

  /* Шапка главной карточки: иконка + текст в столбик слева */
  .ai-card { gap: 14px; }
  .ai-card-head {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .hero-icon {
    width: 48px;
    height: 48px;
    border-radius: 14px;
  }
  .hero-icon .material-symbols-outlined { font-size: 24px; }

  .card-text h3 { font-size: 15px; }
  .card-text p { font-size: 12px; }

  .ai-form { gap: 14px; }

  /* Switch-row: текст уже плотнее, иконка-плашка чуть меньше */
  .switch-row { padding: 10px; }
  .switch-text { gap: 10px; }
  .switch-text .material-symbols-outlined {
    width: 32px;
    height: 32px;
    font-size: 18px;
  }
  .switch-text strong { font-size: 13px; }
  .switch-text small { font-size: 11px; }

  /* Кнопки действий — на всю ширину в столбик */
  .ai-actions {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
  .ai-actions .btn-filled,
  .ai-actions .btn-outlined {
    width: 100%;
  }

  /* Карточка индексации — иконка сверху, текст и кнопка ниже */
  .ai-index-card {
    flex-direction: column;
    align-items: flex-start;
  }
  .ai-index-card .card-actions { width: 100%; }
  .ai-index-card .card-actions .btn-filled { width: 100%; }
  .index-stats { gap: 6px; }
  .stat-pill { font-size: 11px; padding: 5px 10px; }

  /* Карточка-подсказка — иконка сверху */
  .ai-hint-card {
    flex-direction: column;
    align-items: flex-start;
  }
  .ai-feat-list { padding-left: 16px; }
  .ai-feat-list li { font-size: 12px; }

  /* Empty-state */
  .ai-empty {
    flex-direction: column;
    text-align: center;
    align-items: center;
  }

  .ai-test-result {
    width: 100%;
    box-sizing: border-box;
    font-size: 13px;
  }
}

/* ── Adaptive: очень узкий мобильный <380 ───────────────────── */
@media (max-width: 380px) {
  .settings-card { padding: 14px; }
  .card-text h3 { font-size: 14px; }
  .card-text p { font-size: 11.5px; }
  .ctl { padding: 9px 11px; font-size: 13px; }
  .icon-btn { width: 36px; min-height: 36px; }
}
</style>
