<template>
  <!-- Выбор цвета оформления при создании аккаунта — плитками, как в
       первоначальной настройке Windows 8. Цвета — данные темы (hex), поэтому
       здесь они инлайном, как в карточке темы в настройках; корпус плитки
       нарисован токенами. -->
  <div class="att">
    <div class="att-head">
      <span class="att-title">цвет оформления</span>
      <span class="att-name">{{ theme.presetLabels[theme.currentPreset] || theme.currentPreset }}</span>
    </div>
    <div class="att-grid">
      <button
        v-for="name in theme.presetNames"
        :key="name"
        type="button"
        class="att-tile"
        :class="{ active: theme.currentPreset === name }"
        :title="theme.presetLabels[name] || name"
        :aria-label="theme.presetLabels[name] || name"
        :aria-pressed="theme.currentPreset === name"
        @click="pick(name)"
      >
        <span class="att-fill" :style="{ background: theme.getVars(name).primary }" />
        <span class="att-corner" :style="{ background: theme.getVars(name).secondary }" />
        <span class="att-dot" :style="{ background: theme.getVars(name).tertiary }" />
      </button>
    </div>
  </div>
</template>

<script setup>
import { useThemeStore } from '@/stores/theme.js'

const theme = useThemeStore()

function pick(name) {
  theme.applyTheme(name)
}
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
  grid-template-columns: repeat(auto-fill, minmax(38px, 1fr));
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
