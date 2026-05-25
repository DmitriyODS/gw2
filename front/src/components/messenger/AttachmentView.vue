<template>
  <a v-if="isImage" :href="att.url" target="_blank" rel="noopener" class="att-image-wrap">
    <img :src="att.url" :alt="att.file_name" class="att-image" />
  </a>
  <video v-else-if="isVideo" :src="att.url" controls class="att-video" preload="metadata" />
  <audio v-else-if="isAudio" :src="att.url" controls class="att-audio" preload="metadata" />
  <a v-else :href="att.url" target="_blank" rel="noopener" class="att-file">
    <span class="material-symbols-outlined">attach_file</span>
    <span class="att-file-info">
      <span class="att-file-name">{{ att.file_name }}</span>
      <span class="att-file-size">{{ formatSize(att.size_bytes) }}</span>
    </span>
    <span class="material-symbols-outlined download-ico">download</span>
  </a>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  att: { type: Object, required: true },
})

const isImage = computed(() => props.att.mime_type?.startsWith('image/'))
const isVideo = computed(() => props.att.mime_type?.startsWith('video/'))
const isAudio = computed(() => props.att.mime_type?.startsWith('audio/'))

function formatSize(bytes) {
  if (!bytes) return ''
  const units = ['Б', 'КБ', 'МБ', 'ГБ']
  let n = bytes
  let i = 0
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++ }
  return `${n.toFixed(n < 10 && i > 0 ? 1 : 0)} ${units[i]}`
}
</script>

<style scoped>
.att-image-wrap {
  display: block;
  border-radius: var(--radius-md);
  overflow: hidden;
  max-width: 280px;
}

.att-image {
  display: block;
  width: 100%;
  height: auto;
  max-height: 280px;
  object-fit: cover;
  cursor: zoom-in;
}

.att-video {
  max-width: 320px;
  max-height: 240px;
  border-radius: var(--radius-md);
  background: black;
}

.att-audio {
  width: 260px;
  max-width: 100%;
}

.att-file {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  text-decoration: none;
  color: var(--color-text);
  min-width: 220px;
  max-width: 320px;
}

.att-file:hover {
  border-color: var(--color-primary);
}

.att-file-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.att-file-name {
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.att-file-size {
  font-size: 11px;
  color: var(--color-text-dim);
}

.download-ico {
  color: var(--color-primary);
  font-size: 20px;
}
</style>
