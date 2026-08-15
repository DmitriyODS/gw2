<template>
  <!-- Просмотр файла прямо в разделе: картинки, видео, звук, PDF и текст
       открываются без скачивания. Всё остальное честно предлагает скачать —
       делать вид, что мы умеем показать .docx, хуже, чем не показывать. -->
  <AppDialog
    :model-value="true"
    :title="file.name"
    size="lg"
    :actions="[{ kind: 'cancel', label: 'Закрыть' }, { kind: 'confirm', label: 'Скачать', icon: 'download' }]"
    @confirm="download"
    @cancel="$emit('close')"
    @update:model-value="(v) => !v && $emit('close')"
  >
    <div class="preview" :data-kind="kind">
      <img v-if="kind === 'image'" :src="src" :alt="file.name" class="media">

      <video v-else-if="kind === 'video'" :src="src" class="media" controls preload="metadata" />

      <audio v-else-if="kind === 'audio'" :src="src" class="audio" controls preload="metadata" />

      <iframe v-else-if="kind === 'pdf'" :src="src" class="frame" :title="file.name" />

      <div v-else-if="isText" class="text-wrap">
        <BrandLoader v-if="loadingText" block :size="48" />
        <pre v-else class="text">{{ text }}</pre>
      </div>

      <EmptyState
        v-else
        icon="draft"
        title="Предпросмотр недоступен"
        :subtitle="`${formatBytes(file.size)} · скачайте файл, чтобы открыть его в приложении`"
      />
    </div>
  </AppDialog>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import * as api from '@/api/drive.js'
import { fileKind } from '@/utils/fileTypes.js'
import { formatBytes } from '@/utils/money.js'
import { saveUrl } from '@/utils/download.js'

const props = defineProps({
  file: { type: Object, required: true },
})

defineEmits(['close'])

const kind = computed(() => fileKind(props.file.mime, props.file.name))
const isText = computed(() => ['text', 'code'].includes(kind.value))
// Показываем файл по общему адресу хранилища, как это делают вложения чатов и
// картинки заметок: <img>, <video> и <iframe> ходят браузером и заголовок с
// токеном не несут — через ручку сервиса они получали бы 401.
const src = computed(() => props.file.url || api.previewURL(props.file.id))

const text = ref('')
const loadingText = ref(false)

// Текст читаем сами: iframe показал бы его в кодировке браузера, а файлы
// бывают и в UTF-8 без BOM, и просто большими — режем по объёму.
const TEXT_LIMIT = 512 * 1024

onMounted(async () => {
  if (!isText.value) return
  loadingText.value = true
  try {
    const res = await fetch(src.value)
    const raw = await res.text()
    text.value = raw.length > TEXT_LIMIT
      ? `${raw.slice(0, TEXT_LIMIT)}\n\n… файл показан не целиком`
      : raw
  } catch {
    text.value = 'Не удалось прочитать файл.'
  } finally {
    loadingText.value = false
  }
})

// Скачивание — ссылкой на тот же объект: имя файла задаёт атрибут download,
// поэтому на диск он ложится под своим именем, а не под ключом хранилища.
function download() {
  saveUrl(src.value, props.file.name)
}
</script>

<style scoped>
.preview {
  display: grid;
  place-items: center;
  min-height: 240px;
  max-height: 70dvh;
}

.media {
  max-width: 100%;
  max-height: 70dvh;
  border-radius: var(--radius-md);
}

.audio { width: 100%; }

.frame {
  width: 100%;
  height: 70dvh;
  border: none;
  border-radius: var(--radius-md);
}

.text-wrap {
  width: 100%;
  max-height: 70dvh;
  overflow: auto;
}

.text {
  margin: 0;
  padding: 12px;
  border-radius: var(--radius-md);
  background: var(--color-surface-variant);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.85rem;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
</style>
