<script setup>
defineProps({
  label: { type: String, required: true },
})
</script>

<template>
  <div class="msg-date-divider">
    <span class="msg-date-pill">{{ label }}</span>
  </div>
</template>

<style scoped>
.msg-date-divider {
  position: sticky;
  top: 6px;
  z-index: 2;
  display: flex;
  justify-content: center;
  padding: 4px 0;
  pointer-events: none;
}

.msg-date-pill {
  padding: 3px 12px;
  border-radius: var(--radius-full);
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  box-shadow: var(--glass-edge);
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-dim);
  user-select: none;
}
</style>
