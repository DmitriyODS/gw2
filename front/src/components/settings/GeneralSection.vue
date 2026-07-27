<template>
  <div class="gs">
    <section class="gs-block">
      <h3 class="gs-h">Интерфейс</h3>
      <SwitchRow
        :model-value="themeStore.hubFabEnabled"
        title="Плавающая кнопка ассистента"
        hint="Кнопка ИИ-ассистента и мини-чата поверх всех разделов; её можно перетащить в удобное место."
        icon="smart_toy"
        @update:model-value="themeStore.setHubFabEnabled"
      />

      <button class="gs-row" type="button" @click="tutorial.open()">
        <span class="gs-row-icon">
          <span class="material-symbols-outlined">tour</span>
        </span>
        <span class="gs-row-text">
          <strong>Тур по интерфейсу</strong>
          <small>Короткое знакомство с разделами и горячими действиями.</small>
        </span>
        <span class="material-symbols-outlined gs-row-chev">chevron_right</span>
      </button>
    </section>

    <!-- Личный ключ YouGile: только для рядовых участников компании —
         администратор подключает компанию целиком в карточке компании. -->
    <section v-if="showYougile" class="gs-block">
      <h3 class="gs-h">Интеграция с YouGile</h3>
      <YougileUserSettings />
    </section>
  </div>
</template>

<script setup>
import SwitchRow from '@/components/common/SwitchRow.vue'
import YougileUserSettings from '@/components/settings/YougileUserSettings.vue'
import { useThemeStore } from '@/stores/theme.js'
import { useTutorial } from '@/composables/useTutorial.js'

defineProps({
  showYougile: { type: Boolean, default: false },
})

const themeStore = useThemeStore()
const tutorial = useTutorial()
</script>

<style scoped>
.gs {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.gs-block {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.gs-h {
  margin: 0;
  padding: 0 4px;
  font-size: 1.05rem;
  font-weight: 600;
}

.gs-row {
  display: flex;
  align-items: center;
  gap: 14px;
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

.gs-row:hover { border-color: var(--color-primary); }

.gs-row-icon {
  display: grid;
  place-items: center;
  width: 40px;
  min-width: 40px;
  max-width: 40px;
  height: 40px;
  min-height: 40px;
  max-height: 40px;
  border-radius: var(--radius-md);
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}

.gs-row-icon .material-symbols-outlined { font-size: 22px; }

.gs-row-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.gs-row-text strong { font-size: 0.95rem; font-weight: 600; }

.gs-row-text small {
  font-size: 0.82rem;
  color: var(--color-text-dim);
  line-height: 1.35;
}

.gs-row-chev { color: var(--color-text-dim); }
</style>
