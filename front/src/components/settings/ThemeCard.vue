<template>
  <!-- Компактная карточка темы: маленькая капсула-превью из трёх сегментов
       палитры и название рядом. Цвета приходят данными темы (hex), поэтому
       здесь они инлайном — как цвет тега; токенами задан только «корпус». -->
  <div class="tc" :class="{ active }">
    <button class="tc-apply" type="button" :title="`Применить тему «${name}»`" @click="$emit('apply')">
      <!-- Порядок наложения: третичный снизу, основной сверху; сегменты
           полупрозрачны — на пересечениях цвета смешиваются. -->
      <span class="tc-swatch">
        <span class="tc-seg c" :style="{ background: vars.tertiary }" />
        <span class="tc-seg b" :style="{ background: vars.secondary }" />
        <span class="tc-seg a" :style="{ background: vars.primary }" />
        <span class="tc-frost" />
      </span>

      <span class="tc-name">{{ name }}</span>

      <span v-if="active" class="material-symbols-outlined tc-check">check_circle</span>
    </button>

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
defineProps({
  name: { type: String, required: true },
  vars: { type: Object, required: true },
  active: { type: Boolean, default: false },
  editable: { type: Boolean, default: false },
})
defineEmits(['apply', 'edit', 'remove'])
</script>

<style scoped>
.tc {
  position: relative;
  display: flex;
  align-items: center;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  transition: border-color 0.18s ease;
}

.tc:hover { border-color: color-mix(in oklch, var(--color-primary) 45%, var(--acrylic-border)); }
.tc.active { border-color: var(--color-primary); }

.tc-apply {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 10px 12px;
  border: none;
  background: none;
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
}

/* Капсула-превью: три сегмента внахлёст образуют одну пилюлю. */
.tc-swatch {
  position: relative;
  display: block;
  flex-shrink: 0;
  width: 54px;
  height: 22px;
}

.tc-seg {
  position: absolute;
  top: 0;
  bottom: 0;
  border-radius: 999px;
}

.tc-seg.a { left: 0; width: 46%; opacity: 0.92; }
.tc-seg.b { left: 22%; width: 46%; opacity: 0.86; }
.tc-seg.c { left: 45%; right: 0; opacity: 0.76; }

/* Матовый слой поверх палитры. -webkit- ПЕРЕД стандартным: иначе минификатор
   LightningCSS выбрасывает стандартное свойство. */
.tc-frost {
  position: absolute;
  inset: 0;
  border-radius: 999px;
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  -webkit-backdrop-filter: blur(6px) saturate(1.15);
  backdrop-filter: blur(6px) saturate(1.15);
  pointer-events: none;
}

.tc-name {
  flex: 1;
  min-width: 0;
  font-size: 0.9rem;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tc-check {
  flex-shrink: 0;
  font-size: 20px;
  color: var(--color-primary);
}

/* Инструменты своей темы места в покое НЕ занимают: иначе от названия в
   190-пиксельной колонке оставалось около трети, и свои темы читались хуже
   встроенных. Появляются по наведению и с клавиатуры, раздвигая строку, — но
   поверх названия не ложатся: перекрытый текст здесь уже пробовали. */
.tc-tools {
  flex-shrink: 0;
  display: flex;
  gap: 4px;
  width: 0;
  padding-right: 0;
  overflow: hidden;
  opacity: 0;
  transition: width 0.18s ease, padding-right 0.18s ease, opacity 0.18s ease;
}

.tc:hover .tc-tools,
.tc:focus-within .tc-tools {
  width: 56px; /* две кнопки 26px + промежуток */
  padding-right: 10px;
  opacity: 1;
}

/* Без указателя наводить нечем — на тач-устройствах кнопки видны всегда.
   Там колонка одна на всю ширину, и названию места хватает. */
@media (hover: none) {
  .tc-tools { width: 56px; padding-right: 10px; opacity: 1; }
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
  border: none;
  border-radius: 50%;
  background: var(--color-surface);
  color: var(--color-text);
  cursor: pointer;
  box-shadow: var(--shadow-sm);
}

.tc-tool .material-symbols-outlined { font-size: 16px; }

.tc-tool:hover { background: var(--color-surface-high); }

.tc-tool.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}
</style>
