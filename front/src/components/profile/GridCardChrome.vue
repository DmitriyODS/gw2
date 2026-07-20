<template>
  <!-- Ручка перетаскивания + три хвата ресайза (право/низ/угол). Крепится
       абсолютно к карточке-родителю (у неё position: relative). -->
  <button
    class="card-grip"
    type="button"
    title="Перетащить карточку"
    aria-label="Перетащить карточку"
    @pointerdown="$emit('grip', $event, cardId)"
  >
    <span class="material-symbols-outlined">drag_indicator</span>
  </button>
  <span
    class="card-resize card-resize-e"
    title="Изменить ширину"
    @pointerdown="$emit('resize', $event, cardId, 'x')"
  ></span>
  <span
    class="card-resize card-resize-s"
    title="Изменить высоту"
    @pointerdown="$emit('resize', $event, cardId, 'y')"
  ></span>
  <span
    class="card-resize card-resize-se"
    title="Изменить размер"
    @pointerdown="$emit('resize', $event, cardId, 'xy')"
  ></span>
</template>

<script setup>
defineProps({ cardId: { type: String, required: true } })
defineEmits(['grip', 'resize'])
</script>

<style scoped>
/* Ручка перетаскивания — «пилюля» из точек над шапкой карточки (в зоне
   верхнего паддинга, не спорит с содержимым). */
.card-grip {
  position: absolute;
  top: 3px;
  left: 50%;
  transform: translateX(-50%);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 18px;
  padding: 0 10px;
  border: none;
  border-radius: var(--radius-full);
  background: color-mix(in oklch, var(--color-text) 6%, transparent);
  color: var(--color-text-dim);
  cursor: grab;
  opacity: 0.5;
  touch-action: none;
  transition: opacity 0.15s, background 0.15s, color 0.15s;
  z-index: 3;
}
.card-grip:hover {
  opacity: 1;
  background: color-mix(in oklch, var(--color-primary) 16%, transparent);
  color: var(--color-primary);
}
.card-grip .material-symbols-outlined {
  font-size: 16px;
  transform: rotate(90deg);
}

/* Хваты ресайза — в зоне паддинга карточки, не перекрывают содержимое.
   Слегка видны всегда, ярче на наведении. */
.card-resize {
  position: absolute;
  z-index: 3;
  touch-action: none;
  opacity: 0.35;
  transition: opacity 0.15s;
}
.card-resize::after {
  content: '';
  position: absolute;
  background: color-mix(in oklch, var(--color-text) 22%, transparent);
  transition: background 0.15s, border-color 0.15s;
}
.card-resize:hover { opacity: 1; }
.card-resize:hover::after { background: var(--color-primary); }

/* Правый край — ширина. */
.card-resize-e {
  top: 12px;
  bottom: 20px;
  right: 0;
  width: 12px;
  cursor: ew-resize;
}
.card-resize-e::after {
  top: 50%;
  right: 3px;
  transform: translateY(-50%);
  width: 3px;
  height: 30px;
  border-radius: var(--radius-full);
}

/* Нижний край — высота. */
.card-resize-s {
  left: 12px;
  right: 20px;
  bottom: 0;
  height: 12px;
  cursor: ns-resize;
}
.card-resize-s::after {
  left: 50%;
  bottom: 3px;
  transform: translateX(-50%);
  height: 3px;
  width: 30px;
  border-radius: var(--radius-full);
}

/* Угол — обе оси. */
.card-resize-se {
  right: 0;
  bottom: 0;
  width: 22px;
  height: 22px;
  cursor: nwse-resize;
  opacity: 0.5;
}
.card-resize-se::after {
  right: 5px;
  bottom: 5px;
  width: 9px;
  height: 9px;
  background: none;
  border-right: 2px solid var(--color-text-dim);
  border-bottom: 2px solid var(--color-text-dim);
  border-bottom-right-radius: 4px;
}
.card-resize-se:hover::after {
  background: none;
  border-color: var(--color-primary);
}

/* На мобиле кастомизация раскладки выключена. */
@media (max-width: 768px) {
  .card-grip,
  .card-resize { display: none; }
}
</style>
