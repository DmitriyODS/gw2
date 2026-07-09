<script setup>
import { computed } from 'vue'
import { renderMarkdown } from '@/utils/markdown.js'

const props = defineProps({
  source: { type: String, default: '' },
})

const html = computed(() => renderMarkdown(props.source))
</script>

<template>
  <div class="markdown-view" v-html="html" />
</template>

<style scoped>
.markdown-view {
  color: var(--color-on-surface);
  line-height: 1.55;
  word-break: break-word;
  overflow-wrap: anywhere;
}
.markdown-view :deep(p) {
  margin: 0 0 8px;
}
.markdown-view :deep(p:last-child) {
  margin-bottom: 0;
}
.markdown-view :deep(.md-h) {
  font-weight: 700;
  color: var(--color-on-surface);
  line-height: 1.25;
  margin: 12px 0 6px;
}
.markdown-view :deep(.md-h1) { font-size: 1.25rem; }
.markdown-view :deep(.md-h2) { font-size: 1.1rem; }
.markdown-view :deep(.md-h3) { font-size: 1rem; }
.markdown-view :deep(strong) { font-weight: 700; }
.markdown-view :deep(em) { font-style: italic; }
.markdown-view :deep(s) { opacity: 0.7; }
.markdown-view :deep(.md-link) {
  color: var(--color-primary);
  text-decoration: underline;
  text-underline-offset: 2px;
}
.markdown-view :deep(.md-link:hover) {
  filter: brightness(0.92);
}
.markdown-view :deep(.md-code) {
  background: var(--color-surface-high);
  padding: 1px 6px;
  border-radius: var(--radius-xs, 6px);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.92em;
}
.markdown-view :deep(.md-pre) {
  background: var(--color-surface-high);
  padding: 10px 12px;
  border-radius: var(--radius-md, 12px);
  overflow-x: auto;
  margin: 8px 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.9em;
  line-height: 1.5;
}
.markdown-view :deep(.md-pre code) {
  background: transparent;
  padding: 0;
}
.markdown-view :deep(.md-list) {
  margin: 4px 0 8px;
  padding-left: 22px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.markdown-view :deep(.md-task) {
  list-style: none;
  margin-left: -22px;
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.markdown-view :deep(.md-task input) {
  accent-color: var(--color-primary);
  transform: translateY(1px);
}
.markdown-view :deep(.md-quote) {
  margin: 8px 0;
  padding: 6px 12px;
  border-left: 3px solid var(--color-primary);
  background: var(--color-surface-high);
  border-radius: 0 var(--radius-sm, 8px) var(--radius-sm, 8px) 0;
  color: var(--color-text-dim);
}
.markdown-view :deep(.md-hr) {
  border: none;
  height: 1px;
  background: var(--color-outline-dim);
  margin: 12px 0;
}
.markdown-view :deep(.md-img) {
  max-width: 100%;
  border-radius: var(--radius-md, 12px);
  display: block;
  margin: 8px 0;
}
.markdown-view :deep(.md-table-wrap) {
  overflow-x: auto;
  margin: 8px 0;
}
.markdown-view :deep(.md-table) {
  border-collapse: collapse;
  font-size: 0.95em;
  min-width: 60%;
}
.markdown-view :deep(.md-table th),
.markdown-view :deep(.md-table td) {
  border: 1px solid var(--color-outline-dim);
  padding: 6px 10px;
  text-align: left;
}
.markdown-view :deep(.md-table th) {
  background: var(--color-surface-high);
  font-weight: 700;
}
</style>
