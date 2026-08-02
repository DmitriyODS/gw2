<template>
  <div class="cp">
    <AppRow
      title="Фон переписки"
      hint="Общий градиент и узор для всех чатов. В отдельном чате фон переопределяется через меню «⋮ → Оформление чата»."
      clickable
      @click="chatBgOpen = true"
    >
      <template #lead>
        <span class="cp-preview">
          <ChatBackgroundLayer v-if="chatRecipe" :recipe="chatRecipe" />
          <span class="cp-bubble left" />
          <span class="cp-bubble right" />
        </span>
      </template>
    </AppRow>

    <AppRow
      v-if="hasCompany"
      title="Фон ленты портала"
      hint="Оформление корпоративной ленты. Личное — коллеги видят свой фон."
      clickable
      @click="portalBgOpen = true"
    >
      <template #lead>
        <span class="cp-preview">
          <ChatBackgroundLayer v-if="portalRecipe" :recipe="portalRecipe" />
          <span class="cp-post" />
          <span class="cp-post short" />
        </span>
      </template>
    </AppRow>

    <ChatBackgroundDialog v-model="chatBgOpen" :conversation="null" />
    <PortalBackgroundDialog v-if="hasCompany" v-model="portalBgOpen" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import ChatBackgroundLayer from '@/components/common/ChatBackgroundLayer.vue'
import AppRow from '@/components/ui/AppRow.vue'
import ChatBackgroundDialog from '@/components/messenger/ChatBackgroundDialog.vue'
import PortalBackgroundDialog from '@/components/portal/PortalBackgroundDialog.vue'
import { useMessengerStore } from '@/stores/messenger.js'
import { usePortalStore } from '@/stores/portal.js'
import { isBlankRecipe } from '@/utils/chatBackgrounds.js'

const props = defineProps({
  hasCompany: { type: Boolean, default: false },
})

const messenger = useMessengerStore()
const portal = usePortalStore()

const chatBgOpen = ref(false)
const portalBgOpen = ref(false)

// Пустой рецепт рисовать незачем — под превью останется поверхность карточки.
const chatRecipe = computed(() => (isBlankRecipe(messenger.chatBgDefault) ? null : messenger.chatBgDefault))
const portalRecipe = computed(() => (isBlankRecipe(portal.background) ? null : portal.background))

onMounted(() => {
  messenger.fetchChatBackgrounds()
  if (props.hasCompany) portal.fetchBackground()
})
</script>

<style scoped>
.cp {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* Мини-сцена чата/ленты поверх реального фона пользователя. */
.cp-preview {
  position: relative;
  display: block;
  width: 108px;
  min-width: 108px;
  height: 68px;
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  border: 1px solid var(--color-outline-dim);
  overflow: hidden;
  isolation: isolate;
}

.cp-bubble, .cp-post {
  position: absolute;
  border-radius: 999px;
}

.cp-bubble.left {
  left: 10px;
  top: 16px;
  width: 46px;
  height: 13px;
  background: var(--color-surface-high);
}

.cp-bubble.right {
  right: 10px;
  bottom: 16px;
  width: 56px;
  height: 13px;
  background: var(--color-primary);
}

.cp-post {
  left: 12px;
  right: 12px;
  top: 16px;
  height: 15px;
  border-radius: var(--radius-md);
  background: var(--color-surface-high);
}

.cp-post.short {
  top: 38px;
  right: 34px;
  height: 12px;
  background: var(--color-secondary-container);
}

/* На узком экране превью ужимается, но строка остаётся строкой: текст рядом,
   а не под баннером во всю ширину. */
@media (max-width: 480px) {
  .cp-preview { width: 76px; min-width: 76px; height: 52px; }
}
</style>
