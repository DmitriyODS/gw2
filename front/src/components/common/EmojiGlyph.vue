<template>
  <!-- Эмодзи каталога питомцев: забандленный Twemoji SVG — одинаковый у всех
       платформ; вне набора — обычный системный эмодзи. Размер наследует
       font-size родителя (как текстовый эмодзи). -->
  <img
    v-if="src"
    class="emoji-glyph"
    :src="src"
    :alt="char"
    draggable="false"
    aria-hidden="true"
  />
  <span v-else>{{ char }}</span>
</template>

<script setup>
import { computed } from 'vue'
import { emojiAsset } from '@/utils/emoji.js'

const props = defineProps({
  char: { type: String, default: '' },
})

const src = computed(() => emojiAsset(props.char))
</script>

<style scoped>
.emoji-glyph {
  width: 1em;
  height: 1em;
  display: inline-block;
  vertical-align: -0.125em;
  user-select: none;
}
</style>
