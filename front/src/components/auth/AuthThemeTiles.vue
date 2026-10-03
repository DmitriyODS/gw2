<template>
  <!-- Выбор оформления при создании аккаунта — цвет плитками и светлый/тёмный
       вид. Цвета — данные темы (hex),
       поэтому здесь они инлайном, как в карточке темы в настройках; корпус
       плитки нарисован токенами. Выбор — черновик: закрепит его только
       созданный аккаунт (см. примерку в stores/theme.js). -->
  <div class="att">
    <div class="att-head">
      <span class="att-title">Оформление</span>
      <span class="att-name">{{ theme.presetLabels[theme.activePreset] || theme.activePreset }}</span>
    </div>
    <div class="att-grid">
      <button
        v-for="name in theme.presetNames"
        :key="name"
        type="button"
        class="att-tile"
        :class="{ active: theme.activePreset === name }"
        :title="theme.presetLabels[name] || name"
        :aria-label="theme.presetLabels[name] || name"
        :aria-pressed="theme.activePreset === name"
        @click="theme.applyTheme(name)"
      >
        <span class="att-fill" :style="{ background: theme.getVars(name).primary }" />
        <span class="att-corner" :style="{ background: theme.getVars(name).secondary }" />
        <span class="att-dot" :style="{ background: theme.getVars(name).tertiary }" />
      </button>
    </div>

    <!-- Светлая/тёмная: пока не выбрали, показываем системный вид — выбрана
         та вкладка, которая на экране сейчас. -->
    <AppTabs
      :model-value="theme.dark ? 'dark' : 'light'"
      :tabs="MODES"
      variant="tint"
      full-width
      dense
      @change="theme.setMode"
    />
  </div>
</template>

<script setup>
import { useThemeStore } from '@/stores/theme.js'
import AppTabs from '@/components/ui/AppTabs.vue'

const theme = useThemeStore()

const MODES = [
  { value: 'light', icon: 'light_mode', label: 'Светлая' },
  { value: 'dark', icon: 'dark_mode', label: 'Тёмная' },
]
</script>

<style scoped>
.att {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.att-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}

.att-title {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--color-text);
}

.att-name {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-primary);
}

.att-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(38px, 100%), 1fr));
  gap: 8px;
}

.att-tile {
  position: relative;
  width: 100%;
  aspect-ratio: 1 / 1;
  min-width: 0;
  padding: 0;
  border: none;
  border-radius: var(--radius-sm);
  overflow: hidden;
  cursor: pointer;
  background: none;
  box-shadow: var(--sk-raised-shadow);
  transition: box-shadow 0.14s, transform 0.14s;
}

/* Плитка — приподнятая клавиша палитры: под курсором чуть поднимается. */
.att-tile:hover { transform: translateY(-1px); }

/* Активная плитка помечена обводкой ВНУТРЕННЕЙ тенью: внешнюю срезает
   overflow плитки и скролл панели. */
.att-tile.active {
  box-shadow:
    inset 0 0 0 2px var(--color-surface),
    inset 0 0 0 4px var(--color-primary),
    var(--sk-pressed-shadow);
}

.att-fill {
  position: absolute;
  inset: 0;
}

/* Уголок и точка показывают вторичный и третичный цвета темы — плитка
   читается как палитра, а не как один плоский цвет. */
.att-corner {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 55%;
  height: 55%;
  clip-path: polygon(100% 0, 100% 100%, 0 100%);
}

.att-dot {
  position: absolute;
  left: 22%;
  top: 22%;
  width: 26%;
  height: 26%;
  border-radius: 50%;
  transform: translate(-50%, -50%);
}
</style>
