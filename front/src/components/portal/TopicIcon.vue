<template>
  <span class="ti" :style="colorStyle" :class="{ lg }">
    <EmojiGlyph v-if="isEmoji" :char="icon" class="ti-emoji" />
    <span v-else class="material-symbols-outlined">{{ icon || 'label' }}</span>
  </span>
</template>

<script setup>
// Иконка раздела портала: material-ключ ИЛИ эмодзи — одно поле topic.icon
// хранит и то, и другое (различаем по составу строки, см. isEmojiIcon).
import { computed } from 'vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import { isEmojiIcon } from '@/utils/topicIcons.js'

const props = defineProps({
  icon: { type: String, default: '' },
  color: { type: String, default: null },
  lg: { type: Boolean, default: false },
})

const isEmoji = computed(() => isEmojiIcon(props.icon))
const colorStyle = computed(() => (props.color
  ? { background: `var(--tag-${props.color}-surface)`, color: `var(--tag-${props.color}-accent)` }
  : {}))
</script>

<style scoped>
.ti {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--color-surface-high);
  flex-shrink: 0;
}
.ti .material-symbols-outlined { font-size: 19px; }
.ti-emoji { font-size: 18px; line-height: 1; }

.ti.lg { width: 44px; height: 44px; }
.ti.lg .material-symbols-outlined { font-size: 22px; }
.ti.lg .ti-emoji { font-size: 22px; }
</style>
