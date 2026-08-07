<template>
  <div class="hola">
    <div class="hola-inner">
      <!-- Одно поле на все вкладки: ищет, разбирает команды и отправляет
           сообщения ассистенту — в зависимости от выбранного режима. -->
      <div class="hola-field">
        <input
          ref="inputEl"
          v-model="text"
          class="hola-input"
          type="text"
          :placeholder="placeholder"
          autocomplete="off"
          spellcheck="false"
          :disabled="tab === 'chat' && chatBlocked"
          @keydown.down.prevent="move(1)"
          @keydown.up.prevent="move(-1)"
          @keydown.enter.prevent="submit"
          @keydown.esc.prevent="onEsc"
        />
        <span v-if="busy" class="hola-spinner" aria-hidden="true" />
        <button
          v-else-if="text"
          class="hola-clear"
          type="button"
          title="Очистить"
          aria-label="Очистить"
          @click="clearText"
        >
          <span class="material-symbols-outlined">close</span>
        </button>
      </div>

      <nav class="hola-tabs" role="tablist">
        <button
          v-for="t in TABS"
          :key="t.value"
          class="hola-tab"
          :class="{ active: tab === t.value }"
          type="button"
          role="tab"
          :aria-selected="tab === t.value"
          @click="tab = t.value"
        >
          <span class="material-symbols-outlined">{{ t.icon }}</span>
          <span class="hola-tab-label">{{ t.label }}</span>
        </button>
      </nav>

      <div class="hola-body">
        <!-- ── Поиск ── -->
        <template v-if="tab === 'search'">
          <button v-if="calc !== null" class="hola-calc" type="button" @click="onCopyCalc">
            <span class="material-symbols-outlined">calculate</span>
            <span class="hola-calc-value">= {{ calcText }}</span>
            <span class="hola-calc-hint">{{ copied ? 'Скопировано' : 'Enter — скопировать' }}</span>
          </button>

          <HolaResultList
            v-if="sections.length"
            :sections="sections"
            :active-key="activeKey"
            @pick="go"
            @hover="onHover"
          />

          <template v-else-if="!query">
            <section v-if="history.length" class="hola-history">
              <header class="hola-hist-head">
                <h3 class="hola-h">История поиска</h3>
                <button
                  class="hola-hist-clear"
                  type="button"
                  title="Очистить историю"
                  aria-label="Очистить историю поиска"
                  @click="forgetAll"
                >
                  <span class="material-symbols-outlined">delete_sweep</span>
                </button>
              </header>
              <div v-for="row in history" :key="row.text" class="hola-hist-row">
                <button class="hola-hist" type="button" @click="repeat(row.text)">
                  <span class="hola-hist-text">{{ row.text }}</span>
                  <span class="hola-hist-time">{{ historyTime(row.at) }}</span>
                </button>
                <button
                  class="hola-hist-del"
                  type="button"
                  title="Убрать из истории"
                  aria-label="Убрать из истории"
                  @click="forget(row.text)"
                >
                  <span class="material-symbols-outlined">close</span>
                </button>
              </div>
            </section>
            <p v-else class="hola-hint">
              Ищу по всем разделам и в интернете. Умею «создай задачу …»,
              «напомни … завтра в 9», «напиши Васе …» и «1200×3».
            </p>
          </template>

          <p v-else-if="!loading" class="hola-hint">Ничего не нашлось</p>
        </template>

        <!-- ── Команды ── -->
        <template v-else-if="tab === 'commands'">
          <HolaResultList
            v-if="commandSections.length"
            :sections="commandSections"
            :active-key="activeKey"
            @pick="go"
            @hover="onHover"
          />
          <p v-else class="hola-hint">Такой команды нет — попробуйте вкладку «Поиск»</p>
        </template>

        <!-- ── Чат ── -->
        <template v-else>
          <div v-if="chatBlocked" class="hola-locked">
            <HolaIcon :size="40" />
            <p class="hola-locked-title">Ассистент не подключён</p>
            <p class="hola-locked-sub">{{ chatBlockedReason }}</p>
            <AppButton
              variant="filled"
              icon="key"
              label="Подключить ключ"
              class="hola-locked-btn"
              @click="openAiSettings"
            />
          </div>
          <HolaChat v-else @suggest="ask" />
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
/**
 * Содержимое Hola: поиск, быстрые команды и разговор с ИИ.
 *
 * Оболочек две — всплывающая панель рабочего стола (HolaPopup) и обычный
 * экран мобильного каркаса (views/HolaView.vue), поэтому сама панель ничего
 * не знает про окна и лишь сообщает наружу, что пользователь куда-то ушёл.
 *
 * Поиск и команды — общий движок useHolaSearch (наследник строки Spotlight),
 * чат — деловой ассистент aisvc. Ассистент подключён к ПОЛЬЗОВАТЕЛЮ и работает
 * на его личном ключе: пока ключа нет, вкладка честно говорит, чего не хватает,
 * и ведёт в профиль, а не молчит ошибкой.
 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import { useRouter } from 'vue-router'
import { useHolaSearch } from '@/composables/useHolaSearch.js'
import { useAssistantStore } from '@/stores/assistant.js'
import { getMyAiSettings } from '@/api/ai.js'
import { loadHistory, pushHistory, removeHistory, clearHistory, historyTime } from '@/utils/holaHistory.js'
import HolaResultList from '@/components/hola/HolaResultList.vue'
import HolaChat from '@/components/hola/HolaChat.vue'
import HolaIcon from '@/components/common/HolaIcon.vue'

const TABS = [
  { value: 'search', label: 'Поиск', icon: 'search' },
  { value: 'commands', label: 'Команды', icon: 'deployed_code' },
  { value: 'chat', label: 'Чат', icon: 'chat' },
]

const props = defineProps({
  // С какой вкладки открыть панель (deep-link или прошлый выбор оболочки).
  startTab: { type: String, default: 'search' },
})

const emit = defineEmits(['navigate'])

// Разворачиваем сразу: в шаблоне доступны только верхнеуровневые ref'ы setup.
const {
  query: searchQuery, loading, sections, commandCatalog, calc, calcText,
  schedule, run, stop, copyCalc,
} = useHolaSearch()
const assistant = useAssistantStore()
const router = useRouter()

const tab = ref(TABS.some((t) => t.value === props.startTab) ? props.startTab : 'search')
const text = ref('')
const cursor = ref(0)
const copied = ref(false)
const history = ref(loadHistory())
const inputEl = ref(null)
// null — ещё не спрашивали; true/false — подключён ли личный ключ ассистента.
const aiConnected = ref(null)

const query = computed(() => text.value.trim())
const busy = computed(() => loading.value || assistant.sending)

const placeholder = computed(() => {
  if (tab.value === 'chat') return chatBlocked.value ? 'Чат недоступен' : 'Спросите Hola о работе компании'
  if (tab.value === 'commands') return 'Название команды'
  return 'введите свой запрос здесь'
})

/* ── Доступность чата ──
   Ассистент подключён к ПОЛЬЗОВАТЕЛЮ: работает на его личном ключе и не
   требует активной компании. Настройки тянем разово (/api/ai/my-settings),
   поэтому вкладка блокируется заранее, а не после первой неудачной отправки. */
const chatBlocked = computed(() => aiConnected.value === false || assistant.disabled)

const chatBlockedReason = computed(() =>
  'Ассистент работает на вашем личном ключе ИИ. Подключите его в профиле — ' +
  'ключ останется с вами в любой компании.')

/* ── Строка ввода и режимы ──
   В поиске и командах поле фильтрует выдачу, в чате — набирает сообщение,
   поэтому запрос уезжает в движок поиска только вне чата. */
watch(text, () => {
  if (tab.value === 'chat') return
  searchQuery.value = text.value
  cursor.value = 0
  schedule()
})

watch(tab, async (t) => {
  cursor.value = 0
  if (t === 'chat') {
    stop()
    if (!chatBlocked.value && !assistant.loaded) assistant.fetchHistory()
  } else {
    searchQuery.value = text.value
    schedule()
  }
  await nextTick()
  inputEl.value?.focus()
})

const commandSections = computed(() =>
  (commandCatalog.value.length ? [{ key: 'commands', label: 'Быстрые команды', items: commandCatalog.value }] : []))

/* Плоский список для навигации клавишами — секции листаются насквозь. */
const flat = computed(() => {
  const source = tab.value === 'commands' ? commandSections.value : sections.value
  return source.flatMap((s) => s.items)
})

const activeKey = computed(() => flat.value[cursor.value]?.key || null)

function move(delta) {
  if (tab.value === 'chat' || !flat.value.length) return
  const next = cursor.value + delta
  cursor.value = (next + flat.value.length) % flat.value.length
}

function onHover(item) {
  const i = flat.value.findIndex((x) => x.key === item.key)
  if (i >= 0) cursor.value = i
}

async function submit() {
  if (tab.value === 'chat') return send()
  const item = flat.value[cursor.value]
  if (item) return go(item)
  if (calc.value !== null) onCopyCalc()
}

/* Выполненный поиск попадает в историю: это след работы пользователя, а не
   каждый набранный символ, поэтому пишем на переходе, а не на вводе. */
async function go(item) {
  const q = query.value
  const ok = await run(item)
  if (!ok) return
  if (q && tab.value === 'search') history.value = pushHistory(q)
  emit('navigate')
}

// Ключ ассистента живёт в настройках («ИИ возможности») — ведём туда и
// закрываем панель, как при любом другом переходе из Hola.
function openAiSettings() {
  router.push('/settings?section=ai')
  emit('navigate')
}

function repeat(value) {
  text.value = value
  inputEl.value?.focus()
}

function forget(value) {
  history.value = removeHistory(value)
}

function forgetAll() {
  history.value = clearHistory()
}

async function onCopyCalc() {
  if (await copyCalc()) {
    copied.value = true
    setTimeout(() => { copied.value = false }, 1500)
  }
}

/* ── Чат ── */
async function send() {
  const value = query.value
  if (!value || chatBlocked.value) return
  text.value = ''
  await assistant.send(value)
}

function ask(value) {
  text.value = value
  send()
}

function clearText() {
  text.value = ''
  inputEl.value?.focus()
}

function onEsc() {
  if (text.value) clearText()
}

onMounted(async () => {
  inputEl.value?.focus()
  try {
    const my = await getMyAiSettings()
    aiConnected.value = !!my?.has_key && my?.enabled !== false
  } catch {
    // Настройки не ответили — не запираем вкладку заранее: попытка отправки
    // сама расскажет, доступен ли ассистент.
    aiConnected.value = null
  }
  if (tab.value === 'chat' && !chatBlocked.value) assistant.fetchHistory()
})

onBeforeUnmount(stop)

defineExpose({ focus: () => inputEl.value?.focus() })
</script>

<style scoped>
.hola {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  padding: 16px;
}

.hola-inner {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ── Строка запроса ── */
.hola-field {
  position: relative;
  display: flex;
  align-items: center;
  flex-shrink: 0;
  padding: 0 16px;
  height: 56px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  transition: border-color 0.15s;
}

.hola-field:focus-within { border-color: color-mix(in oklch, var(--color-primary) 40%, var(--acrylic-border)); }

.hola-input {
  flex: 1;
  min-width: 0;
  height: 100%;
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-text);
  font-size: 16px;
  font-family: inherit;
  text-align: center;
}

.hola-input::placeholder { color: color-mix(in oklch, var(--color-text-dim) 85%, transparent); }
.hola-input:disabled { cursor: not-allowed; }

.hola-clear {
  width: 32px;
  min-width: 32px;
  max-width: 32px;
  height: 32px;
  min-height: 32px;
  max-height: 32px;
  display: grid;
  place-items: center;
  padding: 0;
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
}

.hola-clear:hover { background: color-mix(in oklch, var(--color-primary) 12%, transparent); color: var(--color-primary); }

.hola-spinner {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  border: 2px solid color-mix(in oklch, var(--color-primary) 30%, transparent);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: holaSpin 0.7s linear infinite;
}

@keyframes holaSpin { to { rotate: 360deg; } }

/* ── Вкладки-плитки ── */
.hola-tabs {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  flex-shrink: 0;
}

.hola-tab {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 14px 8px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text-dim);
  font-size: 13.5px;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}

.hola-tab:hover { border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border)); }

.hola-tab.active {
  background: var(--grad-primary-soft);
  border-color: color-mix(in oklch, var(--color-primary) 34%, transparent);
  color: var(--color-primary);
}

.hola-tab .material-symbols-outlined { font-size: 24px; }

/* ── Содержимое вкладки ── */
.hola-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-y: auto;
  scrollbar-width: thin;
}

.hola-h {
  margin: 0;
  padding: 0 4px;
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text);
}

/* ── История поиска ── */
.hola-history { display: flex; flex-direction: column; gap: 8px; }

.hola-hist-head { display: flex; align-items: center; gap: 8px; }
.hola-hist-head .hola-h { flex: 1; min-width: 0; }

.hola-hist-clear {
  width: 32px;
  min-width: 32px;
  max-width: 32px;
  height: 32px;
  min-height: 32px;
  max-height: 32px;
  display: grid;
  place-items: center;
  padding: 0;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.hola-hist-clear:hover { background: color-mix(in oklch, var(--color-primary) 12%, transparent); color: var(--color-primary); }
.hola-hist-clear .material-symbols-outlined { font-size: 20px; }

.hola-hist-row { position: relative; display: flex; align-items: center; }

.hola-hist {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.14s;
}

.hola-hist:hover { border-color: color-mix(in oklch, var(--color-primary) 34%, var(--acrylic-border)); }

/* Место под крестик держим только там, где он вообще появляется — по
   наведению. На тач-экране крестика нет, и время стоит симметрично тексту. */
@media (hover: hover) {
  .hola-hist { padding-right: 44px; }
}

.hola-hist-text {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hola-hist-time { font-size: 12.5px; color: var(--color-text-dim); font-variant-numeric: tabular-nums; }

/* Крестик поверх строки: сама строка кликабельна целиком (повтор запроса). */
.hola-hist-del {
  position: absolute;
  right: 8px;
  width: 28px;
  min-width: 28px;
  max-width: 28px;
  height: 28px;
  min-height: 28px;
  max-height: 28px;
  display: grid;
  place-items: center;
  padding: 0;
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text-dim);
  opacity: 0;
  cursor: pointer;
  transition: opacity 0.14s, color 0.14s;
}

.hola-hist-row:hover .hola-hist-del { opacity: 1; }
.hola-hist-del:hover { color: var(--color-error); }
.hola-hist-del .material-symbols-outlined { font-size: 17px; }

/* ── Калькулятор строки ── */
.hola-calc {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
  padding: 14px 16px;
  border: 1px solid color-mix(in oklch, var(--color-primary) 26%, transparent);
  border-radius: var(--radius-lg);
  background: color-mix(in oklch, var(--color-primary) 8%, transparent);
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
}

.hola-calc .material-symbols-outlined { font-size: 22px; color: var(--color-primary); }
.hola-calc-value { flex: 1; font-size: 20px; font-weight: 700; font-variant-numeric: tabular-nums; }
.hola-calc-hint { font-size: 12px; color: var(--color-text-dim); }

/* ── Заглушки ── */
.hola-hint {
  margin: 0;
  padding: 26px 22px;
  text-align: center;
  font-size: 13.5px;
  line-height: 1.5;
  color: var(--color-text-dim);
}

.hola-locked {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 24px;
  text-align: center;
  color: var(--color-text-dim);
}

.hola-locked :deep(.hola-icon) { color: color-mix(in oklch, var(--color-text-dim) 70%, transparent); }
.hola-locked-title { margin: 0; font-size: 15px; font-weight: 600; color: var(--color-text); }
.hola-locked-sub { margin: 0; max-width: 380px; font-size: 13px; line-height: 1.45; }

.hola-locked-btn {
  margin-top: 6px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.hola-locked-btn .material-symbols-outlined { font-size: 18px; }

/* Узкое окно (и мобильный экран): подписи вкладок ужимаются, поля становятся
   компактнее. Дублируем media-запросом — заводской WebView старых Android
   не знает @container. */
@container (max-width: 520px) {
  .hola { padding: 12px; }
  .hola-field { height: 50px; padding: 0 12px; }
  .hola-input { font-size: 15px; }
  .hola-tabs { gap: 8px; }
  .hola-tab { padding: 10px 6px; font-size: 12.5px; }
}

@media (max-width: 520px) {
  .hola { padding: 12px; }
  .hola-field { height: 50px; padding: 0 12px; }
  .hola-input { font-size: 15px; }
  .hola-tabs { gap: 8px; }
  .hola-tab { padding: 10px 6px; font-size: 12.5px; }
}
</style>
