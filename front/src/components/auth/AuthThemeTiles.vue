<template>
  <!-- Выбор оформления при создании аккаунта — цвет плитками и светлый/тёмный
       вид. Цвета — данные темы (hex),
       поэтому здесь они инлайном, как в карточке темы в настройках; корпус
       плитки нарисован токенами. Выбор — черновик: закрепит его только
       созданный аккаунт (см. примерку в stores/theme.js). -->
  <div class="att">
    <div class="att-head">
      <span class="att-title">цвет оформления</span>
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

    <!-- Светлая/тёмная: пока не выбрали, показываем системный вид — активна
         та кнопка, которая на экране сейчас. -->
    <div class="att-modes" role="tablist">
      <button
        v-for="m in MODES"
        :key="m.value"
        type="button"
        class="att-mode"
        :class="{ active: theme.dark === m.dark }"
        role="tab"
        :aria-selected="theme.dark === m.dark"
        @click="theme.setMode(m.value)"
      >
        <span class="material-symbols-outlined">{{ m.icon }}</span>
        {{ m.label }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { useThemeStore } from '@/stores/theme.js'

const theme = useThemeStore()

const MODES = [
  { value: 'light', dark: false, icon: 'light_mode', label: 'светлая' },
  { value: 'dark',  dark: true,  icon: 'dark_mode',  label: 'тёмная' },
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
  font-size: 12.5px;
  font-weight: 600;
  color: var(--color-text-dim);
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

.att-modes {
  display: flex;
  gap: 4px;
  padding: 4px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg), color-mix(in oklch, var(--color-surface) 45%, transparent);
  box-shadow: var(--glass-edge);
}

.att-mode {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 0;
  height: 32px;
  padding: 0 12px;
  border: none;
  border-radius: var(--radius-full);
  background: none;
  color: var(--color-text-dim);
  font: inherit;
  font-size: 12.5px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.att-mode .material-symbols-outlined { font-size: 17px; }

.att-mode:hover:not(.active) { color: var(--color-primary); }

.att-mode.active {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
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
  box-shadow: var(--shadow-sm);
  transition: box-shadow 0.14s;
}

/* Выделение — стеклянный блик по кромке, плитка не смещается. */
.att-tile:hover {
  box-shadow: inset 0 0 0 2px color-mix(in oklch, white 55%, transparent), var(--shadow-md);
}

/* Активная плитка помечена обводкой ВНУТРЕННЕЙ тенью: внешнюю срезает
   overflow плитки и скролл панели. */
.att-tile.active {
  box-shadow:
    inset 0 0 0 2px var(--color-surface),
    inset 0 0 0 4px var(--color-primary),
    var(--shadow-md);
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
