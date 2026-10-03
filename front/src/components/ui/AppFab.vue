<template>
  <!-- Хост — экран/окно своего раздела, иначе body (см. useFloatHost). -->
  <Teleport :to="host">
    <button
      v-if="visible"
      class="fab float-fade"
      :class="[
        `fab--${tone}`,
        { 'fab--collapsed': collapsed, 'fab--icon-only': !label, 'float-hidden': floatingHidden },
      ]"
      :aria-label="ariaLabel || label || icon"
      @click="$emit('click', $event)"
    >
      <span class="material-symbols-outlined">{{ icon }}</span>
      <span v-if="label" class="fab-label">{{ label }}</span>
    </button>
  </Teleport>
</template>

<script setup>
import { onMounted } from 'vue'
import { floatingHidden, installFloatingHide } from '@/composables/useFloatingHide.js'
import { useFloatHost } from '@/desktop/windowHost.js'

const host = useFloatHost()

onMounted(installFloatingHide)

defineProps({
  icon: { type: String, default: 'add' },
  label: { type: String, default: '' },
  collapsed: { type: Boolean, default: false },
  visible: { type: Boolean, default: true },
  ariaLabel: { type: String, default: '' },
  /* primary | tertiary */
  tone: { type: String, default: 'primary' },
})
defineEmits(['click'])
</script>

<style scoped>
/* Плавающая кнопка позиционируется по СВОЕМУ хосту — экрану раздела в мобильном
   каркасе или телу окна рабочего стола (оба позиционированы, см. useFloatHost).
   Раньше кнопка пряталась медиазапросом по ширине ЭКРАНА и в узком окне на
   десктопе исчезала вместе с действием, которое больше негде было вызвать. */
.fab {
  position: absolute;
  right: 16px;
  bottom: calc(16px + env(safe-area-inset-bottom, 0px));
  height: 56px;
  min-width: 56px;
  padding: 0 22px 0 18px;
  border: none;
  border-radius: 28px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font: inherit;
  font-size: 15px;
  font-weight: 650;
  letter-spacing: 0.2px;
  z-index: 150;
  transition: padding 0.26s cubic-bezier(0.34, 1.36, 0.64, 1),
              min-width 0.26s cubic-bezier(0.34, 1.36, 0.64, 1),
              border-radius 0.26s cubic-bezier(0.34, 1.36, 0.64, 1),
              background 0.15s, box-shadow 0.2s, transform 0.12s,
              opacity 0.22s ease;
}

/* Плавающая кнопка — главное действие экрана: выпуклое тело своего цвета,
   как filled-кнопка, но с тенью повыше — она парит над страницей. */
.fab--primary,
.fab--tertiary { background: var(--color-primary); color: var(--color-on-primary); box-shadow: var(--shadow-lg); }
@supports (color: color-mix(in oklch, red, blue)) {
  .fab--primary,
  .fab--tertiary {
    --sk-tone: var(--color-primary);
    border: 1px solid color-mix(in oklch, var(--sk-tone) 70%, black);
    background:
      var(--grain),
      linear-gradient(180deg,
        color-mix(in oklch, var(--sk-tone) 82%, white),
        var(--sk-tone) 55%,
        color-mix(in oklch, var(--sk-tone) 88%, black));
    color: var(--color-on-primary);
    box-shadow:
      inset 0 1px 0 color-mix(in oklch, white 40%, transparent),
      inset 0 -1px 0 color-mix(in oklch, black 18%, transparent),
      0 2px 4px color-mix(in oklch, black 18%, transparent),
      0 10px 24px color-mix(in oklch, var(--sk-tone) 30%, transparent);
  }
}

.fab--tertiary { --sk-tone: var(--color-tertiary); color: var(--color-on-tertiary); }

.fab--primary:active,
.fab--tertiary:active {
  transform: scale(0.97);
  box-shadow:
    inset 0 2px 5px color-mix(in oklch, black 28%, transparent),
    0 1px 2px color-mix(in oklch, black 18%, transparent);
}

.fab .material-symbols-outlined {
  font-size: 24px;
  font-variation-settings: 'FILL' 0, 'wght' 500;
}

.fab-label {
  white-space: nowrap;
  overflow: hidden;
  max-width: 160px;
  opacity: 1;
  transition: max-width 0.26s cubic-bezier(0.4, 0, 0.2, 1),
              opacity 0.18s ease;
}

.fab--collapsed,
.fab--icon-only {
  padding: 0;
  min-width: 56px;
  width: 56px;
  border-radius: 50%;
  justify-content: center;
  gap: 0;
}

.fab--collapsed .fab-label {
  max-width: 0;
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .fab { transition: opacity 0.22s ease; }
}
</style>
