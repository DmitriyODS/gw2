<template>
  <!-- Публичная страница правовых документов: тексты обязаны быть доступны до
       регистрации, а не только в плашке согласия внутри приложения. -->
  <AuthShell title="Правовые документы" :subtitle="`Редакция ${version} от ${date}`" size="lg" back="/welcome">
    <div class="legal-page">
      <AppTabs v-model="activeKey" :tabs="tabs" variant="tint" dense full-width />

      <div class="legal-page-doc">
        <BrandLoader v-if="loading" :size="48" class="legal-page-loader" />
        <MarkdownView v-else :source="text" />
      </div>
    </div>
  </AuthShell>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AuthShell from '@/components/auth/AuthShell.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import MarkdownView from '@/components/common/MarkdownView.vue'
import { LEGAL_DATE, LEGAL_DOCUMENTS, LEGAL_VERSION, legalDocument, loadLegalText } from '@/utils/legal.js'

const route = useRoute()
const router = useRouter()

const version = LEGAL_VERSION
const date = LEGAL_DATE
// Якорь /legal#privacy открывает нужный документ сразу: на него ссылаются
// письма и внешние страницы.
const anchor = route.hash.replace('#', '')
const activeKey = ref(legalDocument(anchor) ? anchor : LEGAL_DOCUMENTS[0].key)
const text = ref('')
const loading = ref(true)

const tabs = computed(() => LEGAL_DOCUMENTS.map((d) => ({ value: d.key, label: d.title })))

watch(activeKey, async (key) => {
  loading.value = true
  try {
    text.value = await loadLegalText(key)
  } finally {
    loading.value = false
  }
  router.replace({ hash: `#${key}` })
}, { immediate: true })
</script>

<style scoped>
.legal-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
}

/* Ряд вкладок высоту не отдаёт: сжимать её вправе только текст документа
   (иначе вкладки срезаются им сверху). */
.legal-page > .app-tabs { flex: none; }

.legal-page-doc {
  max-height: min(60dvh, 640px);
  overflow-y: auto;
  padding: 16px 18px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface-variant);
  font-size: 0.9rem;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.legal-page-loader { margin: 32px auto; }
</style>
