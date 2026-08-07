<template>
  <!-- «Хранилище»: сколько места занято, чем именно и как его освободить.
       Разбивку по разделам и журнал файлов ведёт биллинг, удаляет каждый файл
       его сервис-владелец (снять вложение с сообщения умеет только он). -->
  <div class="storage-sec">
    <BrandLoader v-if="loading" block :size="64" />

    <AppStack v-else :gap="16">
      <AppCard title="Занятое место" :hint="quotaHint">
        <template #head>
          <AppChip :tone="quotaTone" :label="quotaLabel" />
        </template>

        <!-- Полоса имеет смысл только при конечной квоте: без ограничения ей
             не от чего заполняться. -->
        <div v-if="state.limit >= 0" class="quota">
          <div class="bar" :data-tone="quotaTone">
            <span :style="{ width: usedPct }" />
          </div>
          <span class="quota-share">{{ usedShare }}</span>
        </div>

        <AppInfoBar
          v-if="quotaTone === 'error'"
          tone="error"
          message="Место закончилось — новые файлы не загрузятся. Удалите лишнее ниже."
        />

        <ul v-if="services.length" class="svc-list">
          <li v-for="row in services" :key="row.service">
            <span class="svc-dot" :style="{ background: sectionColor(row.service) }" />
            <span class="svc-name">{{ sectionTitle(row.service) }}</span>
            <b>{{ formatBytes(row.bytes) }}</b>
          </li>
        </ul>
        <p v-else class="storage-empty">Пока ничего не занято — загруженных файлов нет.</p>
      </AppCard>

      <AppCard
        title="Что занимает место"
        hint="Самые крупные файлы сверху; список обновляется сам при открытии раздела.
          Удаление снимает файл с его записи — сообщение, заметка или публикация
          остаются на месте."
      >
        <template #head>
          <span v-if="indexing" class="indexing">
            <ProgressSpinner style="width: 18px; height: 18px" stroke-width="5" />
            Обновляем список…
          </span>
          <!-- Кнопка ничего не удаляет из пользовательских записей: она заново
               сверяет список с разделами. Название об этом и говорит — иначе
               её страшно нажимать. -->
          <AppButton
            :label="sweeping ? 'Обновляем…' : 'Обновить список'"
            icon="refresh"
            :loading="sweeping"
            :disabled="indexing"
            @click="sweep"
          />
        </template>

        <AppStack v-if="files.length" :gap="10">
          <div class="filter-row">
            <AppChip
              label="Все разделы"
              interactive
              :selected="filter === ''"
              @click="filter = ''"
            />
            <AppChip
              v-for="row in services"
              :key="row.service"
              :label="sectionTitle(row.service)"
              interactive
              :selected="filter === row.service"
              @click="filter = filter === row.service ? '' : row.service"
            />
          </div>

          <ul class="file-list">
            <li v-for="file in visibleFiles" :key="file.key" :class="{ 'is-picked': picked.has(file.key) }">
              <Checkbox
                :model-value="picked.has(file.key)"
                binary
                :aria-label="`Выбрать ${fileName(file)}`"
                @update:model-value="toggle(file.key)"
              />
              <span class="file-main">
                <span class="file-name">{{ fileName(file) }}</span>
                <span class="file-meta">
                  {{ file.ref_title || sectionTitle(file.service) }}
                  <template v-if="file.created_at"> · {{ formatDate(file.created_at) }}</template>
                </span>
              </span>
              <b class="file-size">{{ formatBytes(file.size) }}</b>
            </li>
          </ul>

          <AppStack row :gap="10" class="file-actions">
            <AppButton
              :label="`Удалить выбранное${pickedBytes ? ` — ${formatBytes(pickedBytes)}` : ''}`"
              icon="delete"
              tone="danger"
              variant="filled"
              :disabled="!picked.size"
              :loading="deleting"
              @click="confirmDelete = true"
            />
            <AppButton v-if="picked.size" label="Снять выбор" variant="text" @click="picked.clear()" />
          </AppStack>
        </AppStack>

        <p v-else class="storage-empty">
          {{ indexing ? 'Собираем список файлов по разделам…' : 'Загруженных файлов нет.' }}
        </p>
      </AppCard>
    </AppStack>

    <ConfirmDialog
      :visible="confirmDelete"
      header="Удалить выбранные файлы?"
      :message="`Файлов: ${picked.size}. Их не восстановить: вложения исчезнут из сообщений,
        картинки — из заметок и досок. Сами записи останутся.`"
      confirm-label="Удалить"
      danger-confirm
      @confirm="removePicked"
      @cancel="confirmDelete = false"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import Checkbox from 'primevue/checkbox'
import ProgressSpinner from 'primevue/progressspinner'
import BrandLoader from '@/components/common/BrandLoader.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppStack from '@/components/ui/AppStack.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import * as billingApi from '@/api/billing.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { formatBytes, usageRatio } from '@/utils/money.js'
import { sectionColor, sectionTitle } from '@/utils/storageSections.js'

const notif = useNotificationsStore()

const loading = ref(true)
const deleting = ref(false)
const sweeping = ref(false)
// Фоновая сверка при входе — тихая, но не невидимая: пока она идёт, список
// может быть неполным, и человек должен понимать, почему.
const indexing = ref(false)
const confirmDelete = ref(false)
const filter = ref('')

const state = reactive({ limit: -1, used: 0, services: [], files: [] })
const picked = reactive(new Set())

const services = computed(() => state.services)
const files = computed(() => state.files)
const visibleFiles = computed(() =>
  filter.value ? files.value.filter((f) => f.service === filter.value) : files.value)

/* Ширина заполнения. Занятое место обычно куда меньше квоты (12 Мб из 5 Гб —
   это 0,2%), и округление до целых процентов давало пустую полосу: выглядело
   так, будто она не работает. Поэтому дробные проценты и заметный минимум,
   пока занято хоть что-то. */
const usedPct = computed(() => {
  const ratio = usageRatio(state.used, state.limit)
  if (ratio <= 0) return '0%'
  return `${Math.max(1.5, ratio * 100).toFixed(2)}%`
})

// Подпись у полосы: доля числом — по ней видно, что счётчик живой.
const usedShare = computed(() => {
  const ratio = usageRatio(state.used, state.limit)
  if (state.limit < 0) return ''
  if (ratio > 0 && ratio < 0.01) return 'меньше 1%'
  return `${Math.round(ratio * 100)}%`
})

const quotaLabel = computed(() => state.limit < 0
  ? formatBytes(state.used)
  : `${formatBytes(state.used)} из ${formatBytes(state.limit)}`)

const quotaHint = computed(() => state.limit < 0
  ? 'Место не ограничено.'
  : 'Место общее на все разделы: переписку, заметки, доски, реестры, календари и портал.')

// Красным — когда свободного места уже нет, жёлтым — когда его почти нет.
const quotaTone = computed(() => {
  if (state.limit < 0) return 'neutral'
  const ratio = usageRatio(state.used, state.limit)
  if (ratio >= 1) return 'error'
  return ratio >= 0.9 ? 'warning' : 'success'
})

const pickedBytes = computed(() => files.value
  .filter((f) => picked.has(f.key))
  .reduce((sum, f) => sum + (f.size || 0), 0))

function fileName(file) {
  return file.name || file.key.split('/').pop()
}

function formatDate(iso) {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })
}

function toggle(key) {
  if (picked.has(key)) picked.delete(key)
  else picked.add(key)
}

function apply(data) {
  state.limit = data.limit_bytes ?? -1
  state.used = data.used_bytes ?? 0
  state.services = data.services || data.items || []
  state.files = data.files || []
  // Выбор чистим: удалённых ключей в новом списке уже нет.
  picked.clear()
}

async function load() {
  try {
    apply(await billingApi.getStorage())
  } catch (e) {
    notif.error(e.message || 'Не удалось загрузить сведения о хранилище')
  } finally {
    loading.value = false
  }
}

async function removePicked() {
  deleting.value = true
  try {
    const res = await billingApi.deleteStorageFiles([...picked])
    notif.success(res.deleted_files
      ? `Удалено файлов: ${res.deleted_files}, освободилось ${formatBytes(res.freed_bytes)}`
      : 'Удалять было нечего')
    await load()
  } catch (e) {
    notif.error(e.message || 'Не удалось удалить файлы')
  } finally {
    deleting.value = false
  }
}

async function sweep() {
  sweeping.value = true
  try {
    const res = await billingApi.sweepStorage()
    notif.success(res.deleted_files || res.added_files
      ? `Список обновлён: добавлено ${res.added_files}, снято ${res.deleted_files}`
      : 'Список актуален — расхождений нет')
    await load()
  } catch (e) {
    notif.error(e.message || 'Не удалось проверить хранилище')
  } finally {
    sweeping.value = false
  }
}

/* Сверка при открытии раздела: список должен показывать то, что лежит в
   разделах на самом деле, а не то, что успел записать журнал. Идёт ФОНОМ и
   молча — данные уже на экране, а по готовности просто обновляются.

   Не чаще раза в AUTO_SWEEP_MS: обход всех сервисов-владельцев не бесплатен,
   а за пять минут расхождению взяться неоткуда. Отметка личная и на
   устройстве — это подсказка интерфейса, а не состояние учёта. */
const AUTO_SWEEP_KEY = 'gw_storage_swept_at'
const AUTO_SWEEP_MS = 5 * 60 * 1000

async function autoSweep() {
  try {
    if (Number(localStorage.getItem(AUTO_SWEEP_KEY)) > Date.now() - AUTO_SWEEP_MS) return
  } catch { /* приватный режим — просто сверимся ещё раз */ }
  indexing.value = true
  try {
    await billingApi.sweepStorage()
    localStorage.setItem(AUTO_SWEEP_KEY, String(Date.now()))
    apply(await billingApi.getStorage())
  } catch { /* молча: раздел уже показан, а руками сверку запускает кнопка */ } finally {
    indexing.value = false
  }
}

onMounted(async () => {
  await load()
  autoSweep()
})
</script>

<style scoped>
.quota {
  display: flex;
  align-items: center;
  gap: 10px;
}

.quota-share {
  flex: none;
  font-size: 0.82rem;
  color: var(--color-text-dim);
  font-variant-numeric: tabular-nums;
}

.bar {
  width: 100%;
  height: 10px;
  border-radius: 999px;
  background: var(--color-surface-variant);
  overflow: hidden;
}

.bar span {
  display: block;
  height: 100%;
  background: var(--color-primary);
  transition: width 0.3s ease;
}

.bar[data-tone='warning'] span { background: var(--color-warning, var(--color-tertiary)); }
.bar[data-tone='error'] span { background: var(--color-error); }

.svc-list,
.file-list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.svc-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(200px, 100%), 1fr));
  gap: 8px;
}

.svc-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-variant);
}

.svc-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex: none;
}

.svc-name {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
}

.filter-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.file-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 420px;
  overflow-y: auto;
}

.file-list li {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
}

.file-list li.is-picked { background: var(--color-surface-variant); }

.file-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.file-name {
  overflow-wrap: anywhere;
  font-weight: 500;
}

.file-meta {
  font-size: 0.82rem;
  color: var(--color-text-secondary);
  overflow-wrap: anywhere;
}

.file-size {
  flex: none;
  font-variant-numeric: tabular-nums;
}

.file-actions { flex-wrap: wrap; }

.storage-empty {
  margin: 0;
  color: var(--color-text-secondary);
}

.indexing {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.85rem;
  color: var(--color-text-dim);
}
</style>
