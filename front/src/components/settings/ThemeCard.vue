<template>
  <!-- Тема — «выкраска», как образец краски в салоне: крупная плашка основного
       цвета, под ней полосы второго и третьего, внизу название на фоновом
       цвете самой темы. Карточка физическая: выбранная вдавлена в поверхность.
       Цвета приходят данными темы (hex), поэтому они инлайном — как цвет
       тега; токенами задан только «корпус». -->
  <div class="tc" :class="{ active }">
    <button
      class="tc-apply"
      type="button"
      :title="`Применить тему «${name}»`"
      :aria-pressed="active"
      @click="$emit('apply')"
    >
      <span class="tc-sample" :style="{ backgroundColor: base }">
        <span class="tc-main" :style="{ backgroundColor: vars.primary }" />
        <span class="tc-strips">
          <span :style="{ backgroundColor: vars.secondary }" />
          <span :style="{ backgroundColor: vars.tertiary }" />
        </span>
        <span class="tc-name" :style="{ color: ink }">{{ name }}</span>
      </span>
    </button>

    <span v-if="active" class="tc-check material-symbols-outlined" aria-hidden="true">check</span>

    <div v-if="editable" class="tc-tools">
      <button class="tc-tool" type="button" title="Изменить цвета" @click.stop="$emit('edit')">
        <span class="material-symbols-outlined">tune</span>
      </button>
      <button class="tc-tool danger" type="button" title="Удалить тему" @click.stop="$emit('remove')">
        <span class="material-symbols-outlined">delete</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: { type: String, required: true },
  vars: { type: Object, required: true },
  active: { type: Boolean, default: false },
  editable: { type: Boolean, default: false },
})
defineEmits(['apply', 'edit', 'remove'])

// Подложка выкраски — фоновый цвет темы; у тем без него — белый лист.
const base = computed(() => props.vars.neutral || '#ffffff')

/* Подпись на подложке: тёмная на светлом фоне темы и светлая на тёмном —
   по относительной яркости. */
const ink = computed(() => {
  const hex = /^#([0-9a-f]{6})$/i.exec(base.value)?.[1]
  if (!hex) return '#1d1b19'
  const [r, g, b] = [0, 2, 4].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255)
  return 0.2126 * r + 0.7152 * g + 0.0722 * b > 0.55 ? '#1d1b19' : '#f6f4f1'
})
</script>

<style scoped>
.tc {
  position: relative;
  padding: 6px;
  border: 1px solid var(--sk-edge);
  border-radius: var(--radius-lg);
  background: var(--sk-raised-bg);
  box-shadow: var(--sk-raised-shadow);
  transition: transform 0.15s ease;
}

.tc:hover { transform: translateY(-1px); }

/* Выбранная — нажатая и зафиксированная: вдавлена, обведена акцентом. */
.tc.active {
  background: var(--sk-pressed-bg);
  box-shadow: var(--sk-pressed-shadow), 0 0 0 2px var(--color-primary);
  transform: none;
}

.tc-apply {
  display: block;
  width: 100%;
  padding: 0;
  border: none;
  background: none;
  cursor: pointer;
}

.tc-sample {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: var(--radius-md);
  box-shadow: inset 0 0 0 1px var(--sk-edge);
}

/* Краска матовая: лёгкий свет сверху и зерно поверх цвета. */
.tc-main,
.tc-strips > span {
  background-image: var(--grain), linear-gradient(180deg,
    color-mix(in oklch, white 16%, transparent),
    transparent 60%,
    color-mix(in oklch, black 8%, transparent));
}

.tc-main { height: 58px; }

.tc-strips {
  display: grid;
  grid-template-columns: 1fr 1fr;
  height: 18px;
}

.tc-name {
  padding: 8px 10px 9px;
  font-size: 0.85rem;
  font-weight: 650;
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Отметка выбранной — круглая выпуклая «кнопка» в углу образца. */
.tc-check {
  position: absolute;
  top: 12px;
  right: 12px;
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--sk-raised-bg);
  box-shadow: var(--sk-raised-shadow);
  color: var(--color-primary);
  font-size: 16px;
  font-weight: 700;
  pointer-events: none;
}

/* Инструменты своей темы — по наведению поверх плашки основного цвета; на
   тач-устройствах видны всегда. */
.tc-tools {
  position: absolute;
  top: 12px;
  left: 12px;
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.tc:hover .tc-tools,
.tc:focus-within .tc-tools { opacity: 1; }

@media (hover: none) {
  .tc-tools { opacity: 1; }
}

.tc-tool {
  display: grid;
  place-items: center;
  width: 26px;
  min-width: 26px;
  max-width: 26px;
  height: 26px;
  min-height: 26px;
  max-height: 26px;
  padding: 0;
  border: 1px solid var(--sk-edge);
  border-radius: 50%;
  background: var(--sk-raised-bg);
  box-shadow: var(--sk-raised-shadow);
  color: var(--color-text);
  cursor: pointer;
}

.tc-tool .material-symbols-outlined { font-size: 15px; }
.tc-tool:hover { background: var(--sk-raised-hover-bg); }
.tc-tool:active { box-shadow: var(--sk-pressed-shadow); }

.tc-tool.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}
</style>
