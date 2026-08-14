<template>
  <AppStack>
    <AppRow
      title="Ваше согласие"
      :hint="acceptedHint"
    >
      <AppChip :label="state?.required ? 'Требуется согласие' : 'Принято'"
               :tone="state?.required ? 'warning' : 'success'" />
    </AppRow>

    <AppRow
      v-for="doc in documents"
      :key="doc.key"
      :title="doc.title"
      :hint="doc.short"
    >
      <AppButton variant="glass" icon="open_in_new" label="Открыть" @click="open(doc.key)" />
    </AppRow>

    <AppRow
      title="Отзыв согласия"
      hint="Отозвать согласие на обработку персональных данных можно письмом на адрес оператора. Отзыв делает работу в приложении невозможной: без обработки данных оно не работает."
    >
      <AppButton variant="glass" icon="mail" :label="operatorEmail" @click="mail" />
    </AppRow>

    <AppDialog v-model="dialogOpen" :header="openedTitle" size="lg">
      <div class="legal-doc-body">
        <BrandLoader v-if="loading" :size="48" class="legal-doc-loader" />
        <MarkdownView v-else :source="text" />
      </div>
    </AppDialog>
  </AppStack>
</template>

<script setup>
/* «Правовые документы» в настройках: действующая редакция и дата согласия.

   Раздел обязателен по смыслу 152-ФЗ — субъект должен иметь доступ к тексту,
   с которым согласился, и знать, как отозвать согласие. Плашка показывается
   один раз, поэтому без этого раздела документы было бы негде перечитать. */
import { computed, onMounted, ref } from 'vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import MarkdownView from '@/components/common/MarkdownView.vue'
import { getLegalState } from '@/api/auth.js'
import { LEGAL_DATE, LEGAL_DOCUMENTS, LEGAL_VERSION, OPERATOR, legalDocument, loadLegalText } from '@/utils/legal.js'

const documents = LEGAL_DOCUMENTS
const operatorEmail = OPERATOR.email

const state = ref(null)
const dialogOpen = ref(false)
const openedKey = ref('')
const text = ref('')
const loading = ref(false)

const openedTitle = computed(() => legalDocument(openedKey.value)?.title || 'Документ')

const acceptedHint = computed(() => {
  if (!state.value) return `Действующая редакция ${LEGAL_VERSION} от ${LEGAL_DATE}`
  if (state.value.required) {
    return `Действующая редакция ${LEGAL_VERSION} от ${LEGAL_DATE} ещё не принята`
  }
  const at = state.value.accepted_at ? new Date(state.value.accepted_at) : null
  const when = at ? at.toLocaleString('ru-RU', { dateStyle: 'long', timeStyle: 'short' }) : '—'
  return `Редакция ${state.value.accepted_version} принята ${when}`
})

onMounted(async () => {
  try {
    state.value = await getLegalState()
  } catch { /* карточка обойдётся текущей редакцией из бандла */ }
})

async function open(key) {
  openedKey.value = key
  dialogOpen.value = true
  loading.value = true
  try {
    text.value = await loadLegalText(key)
  } finally {
    loading.value = false
  }
}

function mail() {
  window.location.href = `mailto:${OPERATOR.email}?subject=${encodeURIComponent('Отзыв согласия на обработку персональных данных')}`
}
</script>

<style scoped>
.legal-doc-body {
  max-height: min(64dvh, 680px);
  overflow-y: auto;
  font-size: 0.9rem;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.legal-doc-loader { margin: 32px auto; }
</style>
