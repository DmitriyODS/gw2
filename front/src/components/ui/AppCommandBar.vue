<template>
  <div class="cmdbar" :class="{ block }">
    <div v-if="slots.default" class="cmd-free"><slot /></div>

    <AppButton
      v-for="c in inlineCommands"
      :key="c.key"
      :icon="c.icon"
      :label="c.label"
      :variant="c.variant || 'filled'"
      :tone="c.tone || 'primary'"
      :size="size"
      :block="block"
      :disabled="c.disabled"
      :loading="c.loading"
      @click="$emit('command', c.key)"
    />

    <AppButton
      v-if="menuCommands.length"
      variant="icon"
      icon="more_horiz"
      :size="size"
      ref="moreBtn"
      aria-label="Ещё действия"
      title="Ещё действия"
      @click="toggleMenu"
    />

    <ContextMenu
      :visible="menuOpen"
      :x="menuX"
      :y="menuY"
      :items="menuItems"
      :anchor="moreEl"
      @select="onSelect"
      @close="menuOpen = false"
    />
  </div>
</template>

<script setup>
/* Панель команд раздела. Состав строки постоянный и не зависит от ширины:
   главное действие (`primary`) — кнопкой с подписью, всё остальное — в меню
   «ещё». Кнопки не переставляются при изменении размера окна, а строке остаётся
   место под поиск и вкладки — ради этого от подгонки под ширину и отказались. */
import { computed, ref, useSlots } from 'vue'
import AppButton from './AppButton.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'

const props = defineProps({
  /** [{ key, label, icon, primary?, fab?, variant?, tone?, disabled?, loading?, danger?, hidden? }] */
  commands: { type: Array, default: () => [] },
  size: { type: String, default: 'md' },
  /** Растянуть главные команды на всю ширину — панель отдана им целиком. */
  block: { type: Boolean, default: false },
})

const emit = defineEmits(['command'])
const slots = useSlots()

const visible = computed(() => props.commands.filter((c) => !c.hidden))
const inlineCommands = computed(() => visible.value.filter((c) => c.primary))
const menuCommands = computed(() => visible.value.filter((c) => !c.primary))

const menuOpen = ref(false)
const menuX = ref(0)
const menuY = ref(0)

/* `children` — подменю: так в тесной панели прячутся переключатели раздела
   (вид периода, набор записей), которым иначе нужна отдельная строка вкладок. */
const menuItems = computed(() =>
  menuCommands.value.map((c) => ({
    label: c.label,
    icon: c.icon,
    danger: c.danger || c.tone === 'danger',
    action: c.key,
    children: c.children?.map((s) => ({ label: s.label, icon: s.icon, action: s.key })),
  })),
)

/* Кнопка «ещё» — переключатель: повторное нажатие закрывает меню. Чтобы это
   работало, ContextMenu знает про якорь и не принимает нажатие по нему за
   «клик мимо» — иначе меню закрывалось бы на pointerdown и тут же открывалось
   обратно на click. */
const moreBtn = ref(null)
const moreEl = computed(() => moreBtn.value?.$el || moreBtn.value || null)

function toggleMenu(e) {
  if (menuOpen.value) {
    menuOpen.value = false
    return
  }
  const r = e.currentTarget.getBoundingClientRect()
  menuX.value = r.left
  menuY.value = r.bottom + 6
  menuOpen.value = true
}

function onSelect(key) {
  menuOpen.value = false
  emit('command', key)
}
</script>

<style scoped>
/* Панель не сжимается: место в строке уступают поиск и заголовок. */
.cmdbar {
  display: flex;
  align-items: center;
  flex-wrap: nowrap;
  flex-shrink: 0;
  gap: 8px;
}

.cmdbar.block { width: 100%; }
.cmdbar.block > :deep(.btn:not(.v-icon)) { flex: 1; }

/* Свободная зона (поиск, вкладки) — наоборот, забирает весь остаток строки. */
.cmd-free {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}
</style>
