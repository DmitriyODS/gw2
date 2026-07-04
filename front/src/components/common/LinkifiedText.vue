<template>
  <span class="linkified">
    <template v-for="(part, i) in parts" :key="i">
      <a
        v-if="part.url"
        :href="part.url"
        target="_blank"
        rel="noopener noreferrer"
        class="linkified-a"
        @click.stop
      >{{ part.text }}</a>
      <template v-else>{{ part.text }}</template>
    </template>
  </span>
</template>

<script setup>
import { computed } from 'vue'

// Кликабельные ссылки в произвольном тексте без v-html (XSS-safe): текст
// режется на части, URL-части рендерятся <a target="_blank">.
const props = defineProps({
  text: { type: String, default: '' },
})

const URL_RE = /((?:https?:\/\/|www\.)[^\s<>"']+)/gi

const parts = computed(() => {
  const out = []
  const text = props.text || ''
  let last = 0
  for (const m of text.matchAll(URL_RE)) {
    if (m.index > last) out.push({ text: text.slice(last, m.index) })
    // Хвостовую пунктуацию не включаем в ссылку («см. https://a.ru.» → точка — текст).
    // Закрывающую скобку отрезаем только когда она непарная (нет открывающей
    // в самой ссылке) — иначе рвём валидные URL вида …/Foo_(bar).
    let url = m[0]
    let tail = ''
    for (;;) {
      const last = url.slice(-1)
      if (/[.,;:!?»”]/.test(last)) { tail = last + tail; url = url.slice(0, -1); continue }
      if (last === ')' && (url.match(/\)/g)?.length || 0) > (url.match(/\(/g)?.length || 0)) {
        tail = last + tail; url = url.slice(0, -1); continue
      }
      break
    }
    out.push({ text: url, url: url.startsWith('www.') ? `https://${url}` : url })
    if (tail) out.push({ text: tail })
    last = m.index + m[0].length
  }
  if (last < text.length) out.push({ text: text.slice(last) })
  return out
})
</script>

<style scoped>
.linkified { white-space: inherit; }
.linkified-a { color: var(--color-primary); text-decoration: underline; word-break: break-all; }
.linkified-a:hover { opacity: 0.85; }
</style>
