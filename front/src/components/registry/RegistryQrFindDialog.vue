<template>
  <!-- Сканер: закрывается сам, как только код распознан — дальше ищем запись. -->
  <QrScanDialog
    v-model="scanOpen"
    title="Поиск по QR-коду"
    subtitle="Наведите камеру на код записи"
    hint="Код распознаётся автоматически — запись откроется сама."
    @decoded="onDecoded"
    @update:model-value="onScanToggle"
  />

  <!-- Итог поиска показываем, только когда запись не нашлась или идёт запрос. -->
  <AppDialog
    v-model="resultOpen"
    title="Поиск по QR-коду"
    icon="qr_code_scanner"
    size="sm"
    :tone="notFound ? 'warning' : 'primary'"
    :actions="notFound
      ? [{ kind: 'cancel', label: 'Закрыть' }, { kind: 'confirm', label: 'Сканировать снова', icon: 'qr_code_scanner' }]
      : []"
    @cancel="closeAll"
    @confirm="rescan"
  >
    <div class="qf">
      <template v-if="searching">
        <span class="material-symbols-outlined spin">progress_activity</span>
        <p class="qf-text">Ищем запись…</p>
      </template>
      <template v-else-if="notFound">
        <span class="material-symbols-outlined qf-warn">search_off</span>
        <p class="qf-text">Запись не найдена</p>
        <p class="qf-code">{{ code }}</p>
      </template>
    </div>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import QrScanDialog from '@/components/common/QrScanDialog.vue'
import { getRecords } from '@/api/registries.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { hasQr, qrValue } from '@/utils/registryFields.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  registry: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue', 'found'])

const notif = useNotificationsStore()

const scanOpen = ref(false)
const resultOpen = ref(false)
const searching = ref(false)
const notFound = ref(false)
const code = ref('')

watch(() => props.modelValue, (open) => {
  if (open) {
    code.value = ''
    notFound.value = false
    searching.value = false
    resultOpen.value = false
    scanOpen.value = true
  } else {
    scanOpen.value = false
    resultOpen.value = false
  }
})

// Закрытие сканера пользователем (без распознавания) закрывает весь поиск.
function onScanToggle(open) {
  if (!open && !searching.value && !notFound.value) emit('update:modelValue', false)
}

function closeAll() {
  resultOpen.value = false
  emit('update:modelValue', false)
}
function rescan() {
  notFound.value = false
  resultOpen.value = false
  scanOpen.value = true
}

async function onDecoded(raw) {
  code.value = String(raw).trim()
  searching.value = true
  notFound.value = false
  resultOpen.value = true
  try {
    const record = await findRecord(code.value)
    if (record) {
      resultOpen.value = false
      emit('found', record)
      emit('update:modelValue', false)
      return
    }
    notFound.value = true
  } catch (e) {
    resultOpen.value = false
    emit('update:modelValue', false)
    notif.error(e?.message || 'Не удалось выполнить поиск')
  } finally {
    searching.value = false
  }
}

// Сквозной поиск сужает выборку на сервере (search_text), а точное совпадение
// проверяем по QR-полям — иначе подстрочное совпадение в чужом поле дало бы
// не ту запись.
async function findRecord(value) {
  const qrFields = (props.registry?.fields || []).filter(hasQr)
  if (!qrFields.length || !value) return null
  const data = await getRecords(props.registry.id, { search: value, per_page: 100, page: 1 })
  const needle = value.toLowerCase()
  return (data.items ?? []).find((rec) => qrFields.some(
    (f) => qrValue(rec.data?.[String(f.id)]).toLowerCase() === needle,
  )) || null
}
</script>

<style scoped>
.qf { display: flex; flex-direction: column; align-items: center; gap: 10px; padding: 12px 0; text-align: center; }
.qf-text { margin: 0; font-size: 15px; font-weight: 600; color: var(--color-text); }
.qf-code { margin: 0; font-size: 13px; color: var(--color-text-dim); word-break: break-all; }
.qf-warn { font-size: 44px; color: var(--color-warning, var(--color-error)); }
.spin { animation: qfspin 1s linear infinite; font-size: 36px; color: var(--color-primary); }
@keyframes qfspin { to { transform: rotate(360deg); } }
</style>
