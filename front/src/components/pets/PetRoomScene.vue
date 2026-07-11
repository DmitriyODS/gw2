<template>
  <!-- Комната грувика как фон-сцена мини-игры: градиент темы — всегда
       (вместо серой заглушки), декор — по расстановке house_placed
       (координаты в % сцены, как в домике). -->
  <div class="prs" :style="{ background: themeBackground }" aria-hidden="true">
    <span
      v-for="item in items"
      :key="item.key"
      class="prs-item"
      :style="{ left: item.x + '%', top: item.y + '%' }"
      :title="decorTitle(item.key)"
    ><EmojiGlyph :char="decorEmoji(item.key)" /></span>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import { decorEmoji, decorTitle } from '@/utils/pets.js'
import { houseThemeBackground } from '@/utils/houseThemes.js'

const props = defineProps({
  pet: { type: Object, default: null },
})

const items = computed(() => props.pet?.house_placed || [])
const themeBackground = computed(() => houseThemeBackground(props.pet?.house_theme))
</script>

<style scoped>
.prs {
  position: absolute;
  inset: 0;
  border-radius: inherit;
  overflow: hidden;
  pointer-events: none;
}

.prs-item {
  position: absolute;
  transform: translate(-50%, -50%);
  font-size: 22px;
  line-height: 1;
  opacity: 0.9;
}
</style>
