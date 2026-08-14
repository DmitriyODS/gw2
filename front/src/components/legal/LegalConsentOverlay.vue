<template>
  <!-- Плашка правовых документов поверх всего приложения. Отменить её нельзя:
       пока действующая редакция не принята, все сервисы отвечают 403, поэтому
       из состояния «не согласен» есть ровно два выхода — принять или выйти. -->
  <Teleport to="body">
    <div class="legal" role="dialog" aria-modal="true" aria-label="Правовые документы">
      <AuthWave class="legal-wave" />

      <div class="legal-card">
        <header class="legal-head">
          <BrandWordmark class="legal-brand" />
          <h1 class="legal-title">{{ returning ? 'Документы обновились' : 'Прежде чем начать' }}</h1>
          <p class="legal-lead">
            {{ returning
              ? 'Мы обновили правовые документы. Прочитайте новую редакцию и подтвердите согласие — без этого работа в приложении недоступна.'
              : 'Прочитайте документы и подтвердите согласие. Без этого работа в приложении недоступна.' }}
          </p>
          <p class="legal-meta">Редакция {{ version }} от {{ date }}</p>
        </header>

        <AppTabs v-model="activeKey" :tabs="tabs" variant="tint" dense full-width />

        <div ref="scrollEl" class="legal-doc" tabindex="0" @scroll.passive="onScroll">
          <BrandLoader v-if="loading" :size="48" class="legal-loader" />
          <MarkdownView v-else :source="text" />
        </div>

        <p v-if="!activeRead" class="legal-scroll-hint">
          <span class="material-symbols-outlined">arrow_downward</span>
          Пролистайте документ до конца
        </p>

        <div class="legal-checks">
          <label
            v-for="g in groups"
            :key="g.key"
            class="legal-check"
            :class="{ locked: !groupRead(g.key) }"
          >
            <Checkbox v-model="agreed[g.key]" binary :disabled="!groupRead(g.key)" />
            <span class="legal-check-text">{{ g.label }}</span>
          </label>
        </div>

        <p v-if="error" class="legal-error">{{ error }}</p>

        <div class="legal-actions">
          <AppButton variant="text" tone="danger" label="Выйти" :disabled="busy" @click="logout" />
          <AppButton
            variant="filled"
            icon="check"
            label="Принять и продолжить"
            :loading="busy"
            :disabled="!canAccept"
            @click="accept"
          />
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import Checkbox from 'primevue/checkbox'
import AuthWave from '@/components/auth/AuthWave.vue'
import BrandWordmark from '@/components/common/BrandWordmark.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import MarkdownView from '@/components/common/MarkdownView.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import { useAuthStore } from '@/stores/auth.js'
import {
  LEGAL_DATE, LEGAL_DOCUMENTS, LEGAL_DOC_KEYS, LEGAL_GROUPS, LEGAL_VERSION, loadLegalText,
} from '@/utils/legal.js'

const auth = useAuthStore()

const version = LEGAL_VERSION
const date = LEGAL_DATE
const groups = LEGAL_GROUPS
// «Документы обновились» — прежнюю редакцию человек уже принимал.
const returning = computed(() => !!auth.user)

const activeKey = ref(LEGAL_DOCUMENTS[0].key)
const text = ref('')
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const scrollEl = ref(null)

// Прочитанные документы: галочку открывает только пролистанный до конца текст —
// иначе «ознакомлен» ставится не глядя, а согласие должно быть информированным.
const read = reactive({})
const agreed = reactive(Object.fromEntries(LEGAL_GROUPS.map((g) => [g.key, false])))

const tabs = computed(() => LEGAL_DOCUMENTS.map((d) => ({
  value: d.key,
  label: d.title,
  icon: read[d.key] ? 'check_circle' : 'description',
})))

const activeRead = computed(() => !!read[activeKey.value])

// Галочка группы доступна, когда прочитаны ВСЕ входящие в неё документы.
function groupRead(groupKey) {
  return LEGAL_DOCUMENTS.filter((d) => d.group === groupKey).every((d) => read[d.key])
}

const canAccept = computed(() => LEGAL_GROUPS.every((g) => agreed[g.key]))

function markRead() {
  read[activeKey.value] = true
}

function onScroll() {
  const el = scrollEl.value
  if (!el) return
  // 24px запаса: у дробного зума и инерционной прокрутки низ не сходится точно.
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 24) markRead()
}

watch(activeKey, async (key) => {
  loading.value = true
  error.value = ''
  try {
    text.value = await loadLegalText(key)
  } catch {
    text.value = ''
    error.value = 'Не удалось загрузить документ — проверьте соединение и обновите страницу'
  } finally {
    loading.value = false
    if (scrollEl.value) scrollEl.value.scrollTop = 0
    // Текст короче области прокрутки виден целиком — листать нечего.
    requestAnimationFrame(() => {
      const el = scrollEl.value
      if (el && el.scrollHeight <= el.clientHeight + 8) markRead()
    })
  }
}, { immediate: true })

async function accept() {
  if (!canAccept.value || busy.value) return
  busy.value = true
  error.value = ''
  try {
    await auth.acceptLegal(version, LEGAL_DOC_KEYS)
  } catch (e) {
    // Документы успели обновиться, пока человек читал: перезагрузка покажет
    // актуальную редакцию — принимать старую нельзя.
    error.value = e?.error === 'LEGAL_VERSION_MISMATCH'
      ? 'Документы обновились — обновите страницу и прочитайте новую редакцию'
      : e?.message || 'Не удалось сохранить согласие — попробуйте ещё раз'
  } finally {
    busy.value = false
  }
}

async function logout() {
  await auth.logout()
}
</script>

<style scoped>
.legal {
  position: fixed;
  inset: 0;
  /* Ниже запертого экрана (20000), выше всего остального: за плашкой
     приложением пользоваться нельзя. */
  z-index: 19000;
  display: grid;
  place-items: center;
  padding: 16px;
  background: var(--color-surface);
  overflow: hidden;
}

.legal-wave {
  position: absolute;
  inset: 0;
}

.legal-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: min(760px, 100%);
  max-height: min(880px, calc(100dvh - 32px));
  /* Страховка для совсем низких экранов: когда даже сжатый документ не
     оставляет места, прокручивается карточка целиком — но не срезается. */
  overflow-y: auto;
  padding: 24px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
}

/* Сжиматься по высоте вправе ТОЛЬКО текст документа: у остальных частей
   карточки высота своя. Без этого браузер отбирал высоту у ряда вкладок
   (у него горизонтальный скролл, но не фиксированная высота), и вкладки
   оказывались срезаны текстом соглашения. */
.legal-card > *:not(.legal-doc) { flex: none; }

.legal-head {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  text-align: center;
}

.legal-brand { margin-bottom: 4px; }

.legal-title {
  margin: 0;
  font-size: 1.25rem;
}

.legal-lead {
  margin: 0;
  max-width: 56ch;
  color: var(--color-text-dim);
  font-size: 0.92rem;
}

.legal-meta {
  margin: 0;
  color: var(--color-text-dim);
  font-size: 0.8rem;
}

.legal-doc {
  /* Сжимается до 30% высоты экрана: min-height в пикселях на низком экране
     (ландшафт телефона) снова отобрал бы высоту у вкладок. Ниже — не даём:
     документ обязан оставаться читаемым, его же требуется пролистать. */
  flex: 1 1 0;
  min-height: min(220px, 30dvh);
  overflow-y: auto;
  padding: 16px 18px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface-variant);
  font-size: 0.9rem;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.legal-doc:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

.legal-loader {
  margin: 32px auto;
}

.legal-scroll-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin: 0;
  color: var(--color-text-dim);
  font-size: 0.82rem;
}

.legal-scroll-hint .material-symbols-outlined { font-size: 18px; }

.legal-checks {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.legal-check {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  cursor: pointer;
  font-size: 0.9rem;
}

.legal-check.locked {
  cursor: default;
  color: var(--color-text-dim);
}

.legal-check-text { padding-top: 2px; }

.legal-error {
  margin: 0;
  color: var(--color-error);
  font-size: 0.86rem;
}

.legal-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

@media (max-width: 640px) {
  .legal { padding: 0; }

  .legal-card {
    width: 100%;
    max-height: 100dvh;
    height: 100dvh;
    border: none;
    border-radius: 0;
    padding: 16px 14px calc(16px + env(safe-area-inset-bottom));
  }

  .legal-lead { font-size: 0.86rem; }
}
</style>
