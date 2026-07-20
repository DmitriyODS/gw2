<template>
  <AppDialog
    :model-value="modelValue"
    icon="devices"
    size="sm"
    title="Авторизовать устройство"
    subtitle="Введите код с другого устройства или отсканируйте его QR."
    :busy="loading"
    :actions="dialogActions"
    @update:modelValue="close"
    @confirm="onConfirm"
  >
    <div v-if="done" class="ad-done">
      <span class="material-symbols-outlined ad-done-ico">check_circle</span>
      <p>{{ doneMessage }}</p>
    </div>

    <div v-else class="ad-body">
      <div class="ad-field">
        <label class="ad-label">Код устройства</label>
        <InputText
          v-model="codeInput"
          class="ad-code-input w-full"
          placeholder="Например, ABC-123"
          autocapitalize="characters"
          autocomplete="off"
          @keyup.enter="approve"
        />
      </div>

      <button type="button" class="btn-glass ad-scan-btn" @click="openScanner">
        <span class="material-symbols-outlined">qr_code_scanner</span>
        Сканировать QR
      </button>

      <div v-if="info" class="ad-preview" :class="{ warn: needsCompanyWarn }">
        <template v-if="info.kind === 'tv'">
          <span class="material-symbols-outlined">tv</span>
          <span v-if="needsCompanyWarn">
            Сначала выберите компанию — ТВ-киоск входит под активной компанией.
          </span>
          <span v-else>
            ТВ-киоск войдёт под компанией «{{ authStore.companyName }}».
          </span>
        </template>
        <template v-else>
          <span class="material-symbols-outlined">login</span>
          <span>Подтверждение входа на другом устройстве под вашим аккаунтом.</span>
        </template>
      </div>

      <p v-if="error" class="ad-error">{{ error }}</p>
    </div>

    <QrScanDialog
      v-model="scannerOpen"
      subtitle="Наведите камеру на QR-код входа"
      :decode="extractLinkCode"
      @decoded="onScanned"
    />
  </AppDialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import InputText from 'primevue/inputtext'
import { useAuthStore } from '@/stores/auth.js'
import { linkInfo, linkApprove } from '@/api/devicelink.js'
import { extractLinkCode, normalizeLinkCode } from '@/utils/deviceLink.js'
import AppDialog from '@/components/common/AppDialog.vue'
import QrScanDialog from '@/components/common/QrScanDialog.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const authStore = useAuthStore()

const codeInput = ref('')
const info = ref(null)
const loading = ref(false)
const error = ref('')
const done = ref(false)
const doneMessage = ref('')
const scannerOpen = ref(false)

const normalized = computed(() => normalizeLinkCode(codeInput.value))
const isCodeValid = computed(() => /^[A-Z2-9]{6}$/.test(normalized.value))
const needsCompanyWarn = computed(
  () => info.value?.kind === 'tv' && authStore.companyId == null,
)

const dialogActions = computed(() => {
  if (done.value) {
    return [{ kind: 'confirm', label: 'Готово', icon: 'check' }]
  }
  return [
    { kind: 'cancel', label: 'Отмена' },
    {
      kind: 'confirm',
      label: 'Авторизовать',
      icon: 'check',
      disabled: !isCodeValid.value || needsCompanyWarn.value,
    },
  ]
})

// Как только код стал валидным — подтягиваем тип (вход/ТВ) для превью.
watch(normalized, async (code) => {
  info.value = null
  error.value = ''
  if (!/^[A-Z2-9]{6}$/.test(code)) return
  try {
    info.value = await linkInfo(code)
  } catch {
    /* код мог не существовать — approve сам вернёт понятную ошибку */
  }
})

function onConfirm() {
  if (done.value) {
    close()
    return
  }
  approve()
}

function openScanner() {
  scannerOpen.value = true
}

function onScanned(code) {
  const c = extractLinkCode(code)
  if (c) codeInput.value = c
}

async function approve() {
  if (!isCodeValid.value || loading.value || needsCompanyWarn.value) return
  loading.value = true
  error.value = ''
  try {
    await linkApprove(normalized.value)
    done.value = true
    doneMessage.value =
      info.value?.kind === 'tv'
        ? 'ТВ-киоск авторизован. Он войдёт в систему автоматически.'
        : 'Вход подтверждён. Устройство войдёт в систему автоматически.'
  } catch (e) {
    error.value = errText(e)
  } finally {
    loading.value = false
  }
}

function errText(e) {
  switch (e?.error) {
    case 'LINK_EXPIRED':
      return 'Код устарел. Обновите его на устройстве и попробуйте снова.'
    case 'LINK_ALREADY_USED':
      return 'Этот код уже подтверждён другим аккаунтом.'
    case 'LINK_NEED_COMPANY':
      return 'Сначала выберите компанию, под которой авторизовать ТВ-киоск.'
    default:
      return e?.message || 'Не удалось авторизовать устройство.'
  }
}

function reset() {
  codeInput.value = ''
  info.value = null
  error.value = ''
  loading.value = false
  done.value = false
  doneMessage.value = ''
}

function close() {
  emit('update:modelValue', false)
}

// Сброс состояния при каждом открытии.
watch(
  () => props.modelValue,
  (open) => { if (open) reset() },
)
</script>

<style scoped>
.ad-body { display: flex; flex-direction: column; gap: 16px; }
.ad-field { display: flex; flex-direction: column; gap: 8px; }
.ad-label {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--color-text-secondary);
}
.ad-code-input {
  text-transform: uppercase;
  letter-spacing: 0.16em;
  text-align: center;
  font-weight: 600;
}
.ad-scan-btn {
  width: 100%;
  justify-content: center;
}
.ad-preview {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  color: var(--color-text-secondary);
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  padding: 10px 12px;
  border-radius: var(--radius-md, 12px);
}
.ad-preview.warn { color: var(--color-warning, var(--color-error)); }
.ad-preview .material-symbols-outlined { font-size: 20px; }
.ad-error { color: var(--color-error); font-size: 0.85rem; text-align: center; }
.ad-done {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  text-align: center;
  padding: 12px 0;
  color: var(--color-text-secondary);
}
.ad-done-ico {
  font-size: 48px;
  color: var(--color-success, var(--color-primary));
}
</style>
