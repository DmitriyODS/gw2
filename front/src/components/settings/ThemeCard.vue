<template>
  <!-- Карточка темы: капсула-превью из трёх наложенных сегментов палитры и
       название под ней. Цвета приходят данными темы (hex), поэтому здесь они
       инлайном — как цвет тега; токенами задан только «корпус» карточки. -->
  <div class="tc" :class="{ active }">
    <button class="tc-apply" type="button" :title="`Применить тему «${name}»`" @click="$emit('apply')">
      <!-- Капсула лежит ПОД матовым стеклом: слой .tc-frost размывает и
           осветляет сегменты, отсюда мягкие края палитры, как в макете. -->
      <span class="tc-swatch">
        <!-- Порядок наложения: третичный снизу, основной сверху. -->
        <span class="tc-seg c" :style="{ background: vars.tertiary }" />
        <span class="tc-seg b" :style="{ background: vars.secondary }" />
        <span class="tc-seg a" :style="{ background: vars.primary }" />
        <span class="tc-frost" />
        <!-- Активная тема помечена обводкой левого сегмента, а не галочкой. -->
        <span v-if="active" class="tc-ring" />
      </span>
      <span class="tc-name">{{ name }}</span>
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
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  /* Стекло: градиент-иней поверх акриловой подложки + блик по верхней кромке. */
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
}

.tc-apply {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  padding: 16px;
  border: none;
  background: none;
  color: var(--color-text);
  cursor: pointer;
}

/* Капсула-превью: три сегмента внахлёст образуют одну сплошную пилюлю —
   подложки под ними нет. */
.tc-swatch {
  position: relative;
  display: block;
  width: 100%;
  aspect-ratio: 3.4 / 1;
  min-height: 58px;
}

.tc-seg {
  position: absolute;
  top: 0;
  bottom: 0;
  border-radius: 999px;
}

/* Сегменты полупрозрачны — на пересечениях цвета смешиваются, как в макете. */
.tc-seg.a { left: 0; width: 46%; opacity: 0.92; }
.tc-seg.b { left: 22%; width: 46%; opacity: 0.86; }
.tc-seg.c { left: 45%; right: 0; opacity: 0.76; }

/* Матовый слой поверх палитры. backdrop-filter здесь размывает именно
   сегменты под ним (ближайший backdrop root — панель раздела), поэтому
   капсула выглядит как под стеклом. -webkit- ПЕРЕД стандартным: иначе
   минификатор LightningCSS выбрасывает стандартное свойство. */
.tc-frost {
  position: absolute;
  inset: 0;
  border-radius: 999px;
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  -webkit-backdrop-filter: blur(7px) saturate(1.15);
  backdrop-filter: blur(7px) saturate(1.15);
  pointer-events: none;
}

.tc-ring {
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  width: 46%;
  border: 2px solid var(--color-primary);
  border-radius: 999px;
  pointer-events: none;
}

.tc-name {
  font-size: 0.95rem;
  font-weight: 700;
  text-align: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Инструменты своей темы — появляются при наведении/фокусе внутри карточки. */
.tc-tools {
  position: absolute;
  top: 10px;
  right: 10px;
  display: flex;
  gap: 6px;
  opacity: 0;
  transition: opacity 0.18s ease;
}

.tc:hover .tc-tools,
.tc:focus-within .tc-tools { opacity: 1; }

.tc-tool {
  display: grid;
  place-items: center;
  width: 28px;
  min-width: 28px;
  max-width: 28px;
  height: 28px;
  min-height: 28px;
  max-height: 28px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: var(--color-surface);
  color: var(--color-text);
  cursor: pointer;
  box-shadow: var(--shadow-sm);
}

.tc-tool .material-symbols-outlined { font-size: 17px; }

.tc-tool:hover { background: var(--color-surface-high); }

.tc-tool.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

/* На тач-устройствах наведения нет — инструменты видны всегда. */
@media (hover: none) {
  .tc-tools { opacity: 1; }
}
</style>
