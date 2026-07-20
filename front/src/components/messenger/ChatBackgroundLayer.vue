<script setup>
import { computed } from 'vue'
import { chatBgStyles } from '@/utils/chatBackgrounds.js'

const props = defineProps({
  // Нормализованный рецепт или null (тогда — базовый фон токена).
  recipe: { type: Object, default: null },
})

const styles = computed(() => chatBgStyles(props.recipe))
</script>

<template>
  <div class="chat-bg" :style="styles.gradient">
    <div v-if="styles.image" class="chat-bg-image" :style="styles.image" />
    <div v-if="styles.pattern" class="chat-bg-pattern" :style="styles.pattern" />
  </div>
</template>

<style scoped>
/* Подложка чата: непрозрачный базовый фон + слои градиента поверх. Лежит под
   контентом ленты (messages-area делается прозрачной) через z-index:-1 —
   поэтому контент НЕ нужно поднимать z-index'ом (иначе меню/поля ловятся в
   stacking-контекст). Родитель обязан быть стекинг-контекстом (isolation:isolate),
   чтобы отрицательный слой не ушёл за родителя. --chat-grad-dim гасит
   насыщённость пятен в тёмной теме, как --bg-grad-dim у фона приложения. */
.chat-bg {
  position: absolute;
  inset: 0;
  z-index: -1;
  overflow: hidden;
  background-color: var(--color-bg);
  --chat-grad-dim: 1;
  pointer-events: none;
}

:global([data-dark='true']) .chat-bg { --chat-grad-dim: 0.62; }

/* Картинка-фон: upscale-запас под размытие (scale задаётся инлайном по степени). */
.chat-bg-image {
  position: absolute;
  inset: 0;
  transform-origin: center;
  will-change: transform;
}

.chat-bg-pattern {
  position: absolute;
  inset: 0;
}
</style>
