<template>
  <div class="cp">
    <button class="cp-card" type="button" @click="chatBgOpen = true">
      <span class="cp-preview">
        <ChatBackgroundLayer v-if="chatRecipe" :recipe="chatRecipe" />
        <span class="cp-bubble left" />
        <span class="cp-bubble right" />
      </span>
      <span class="cp-text">
        <strong>Фон переписки</strong>
        <small>
          Общий градиент и узор для всех чатов. В отдельном чате фон
          переопределяется через меню «⋮ → Оформление чата».
        </small>
      </span>
      <span class="material-symbols-outlined cp-chev">chevron_right</span>
    </button>

    <button v-if="hasCompany" class="cp-card" type="button" @click="portalBgOpen = true">
      <span class="cp-preview">
        <ChatBackgroundLayer v-if="portalRecipe" :recipe="portalRecipe" />
        <span class="cp-post" />
        <span class="cp-post short" />
      </span>
      <span class="cp-text">
        <strong>Фон ленты портала</strong>
        <small>Оформление корпоративной ленты. Личное — коллеги видят свой фон.</small>
      </span>
      <span class="material-symbols-outlined cp-chev">chevron_right</span>
    </button>

    <ChatBackgroundDialog v-model="chatBgOpen" :conversation="null" />
    <PortalBackgroundDialog v-if="hasCompany" v-model="portalBgOpen" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import ChatBackgroundLayer from '@/components/common/ChatBackgroundLayer.vue'
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

.cp-card {
  display: flex;
  align-items: center;
  gap: 16px;
  width: 100%;
  padding: 14px 16px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.2s ease;
}

.cp-card:hover { border-color: var(--color-primary); }

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

.cp-text {
  display: flex;
  flex-direction: column;
  gap: 3px;
  flex: 1;
  min-width: 0;
}

.cp-text strong { font-size: 0.95rem; font-weight: 600; }

.cp-text small {
  font-size: 0.82rem;
  color: var(--color-text-dim);
  line-height: 1.35;
}

.cp-chev { color: var(--color-text-dim); }

@container (max-width: 560px) {
  .cp-card { flex-wrap: wrap; }
  .cp-preview { width: 100%; min-width: 0; }
}

@media (max-width: 560px) {
  .cp-card { flex-wrap: wrap; }
  .cp-preview { width: 100%; min-width: 0; }
}
</style>
